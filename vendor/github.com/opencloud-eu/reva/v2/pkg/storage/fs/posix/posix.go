// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package posix

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	tusd "github.com/tus/tusd/v2/pkg/handler"

	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/cache"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/blobstore"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/idcache"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/lookup"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/timemanager"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/trashbin"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/tree"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/aspects"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/permissions"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/upload"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/usermapper"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/middleware"
	"github.com/pkg/errors"
)

func init() {
	registry.Register("posix", NewDefault)
}

type posixFS struct {
	storage.FS

	tree *tree.Tree
	um   usermapper.Mapper
}

// New returns an implementation to of the storage.FS interface that talk to
// a local filesystem.
func NewDefault(m map[string]interface{}, stream events.Stream, log *zerolog.Logger) (storage.FS, error) {
	o, err := options.New(m)
	if err != nil {
		return nil, err
	}

	o.IDCache.Database += "_v2" // Use a versioned bucket name to avoid conflicts with previous implementations
	o.IDCache.TTL = 0           // Disable TTL for the ID cache, as the posix driver relies on it for caching file IDs and we don't want them to expire
	kv, err := cache.NewNatsKeyValue(o.IDCache)
	if err != nil {
		return nil, errors.Wrap(err, "could not create nats key value store")
	}
	c, err := idcache.NewStoreIDCache(kv)
	if err != nil {
		return nil, err
	}

	o.IDCache.Database += "_history" // Use a versioned bucket name to avoid conflicts with previous implementations
	o.IDCache.TTL = 24 * 60 * time.Minute
	historyKv, err := cache.NewNatsKeyValue(o.IDCache)
	if err != nil {
		return nil, errors.Wrap(err, "could not create nats key value store")
	}
	historyCache, err := idcache.NewStoreIDCache(historyKv)
	if err != nil {
		return nil, err
	}

	return New(o, stream, c, historyCache, log)
}

func New(o *options.Options, stream events.Stream, cache, historyCache *idcache.IDCache, log *zerolog.Logger) (storage.FS, error) {
	if o.IDCache.Store != "nats-js-kv" {
		return nil, fmt.Errorf("the posix driver requires a nats-js-kv cache")
	}

	var err error
	if log == nil {
		log = &zerolog.Logger{}
	}
	posixLog := log.With().Str("driver", "posix").Logger()
	log = &posixLog

	fs := &posixFS{}
	var um usermapper.Mapper
	if o.UseSpaceGroups {
		um, err = usermapper.NewUnixMapper()
		if err != nil {
			return nil, err
		}
	} else {
		um = &usermapper.NullMapper{}
	}

	var lu *lookup.Lookup
	switch o.MetadataBackend {
	case "xattrs":
		lu, err = lookup.New(metadata.NewXattrsBackend(o.FileMetadataCache), um, o, &timemanager.Manager{}, cache, historyCache)
		if err != nil {
			return nil, err
		}
	case "hybrid":
		lu, err = lookup.New(metadata.NewHybridBackend(1024, // start offloading grants after 1KB
			func(n metadata.MetadataNode) string {
				spaceRoot, _ := lu.IDCache.Get(context.Background(), n.GetSpaceID(), n.GetSpaceID())
				if len(spaceRoot) == 0 {
					return ""
				}

				return filepath.Join(spaceRoot, lookup.MetadataDir)
			},
			o.FileMetadataCache), um, o, &timemanager.Manager{}, cache, historyCache)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown metadata backend %s, only 'xattrs' or 'hybrid' (default) supported", o.MetadataBackend)
	}

	permissionsSelector, err := pool.PermissionsSelector(o.PermissionsSVC, pool.WithTLSMode(o.PermTLSMode))
	if err != nil {
		return nil, err
	}
	p := permissions.NewPermissions(node.NewPermissions(lu), permissionsSelector)

	trashbin, err := trashbin.New(o, p, lu, log)
	if err != nil {
		return nil, err
	}
	err = trashbin.Setup(fs)
	if err != nil {
		return nil, err
	}

	bs, err := blobstore.New(o.Root)
	if err != nil {
		return nil, err
	}

	tp, err := tree.New(lu, bs, um, trashbin, p, o, stream, cache, log)
	if err != nil {
		return nil, err
	}

	aspects := aspects.Aspects{
		Lookup:            lu,
		Tree:              tp,
		Permissions:       p,
		EventStream:       stream,
		UserMapper:        um,
		DisableVersioning: o.DisableVersioning,
		Trashbin:          trashbin,
	}

	dfs, err := decomposedfs.New(&o.Options, aspects, log)
	if err != nil {
		return nil, err
	}

	hooks := []middleware.Hook{}
	if o.UseSpaceGroups {
		resolveSpaceHook := func(methodName string, ctx context.Context, spaceID string) (context.Context, middleware.UnHook, error) {
			if spaceID == "" {
				return ctx, nil, nil
			}

			spaceRoot := lu.InternalPath(spaceID, spaceID)
			fi, err := os.Stat(spaceRoot)
			if err != nil {
				return ctx, nil, err
			}

			ctx = context.WithValue(ctx, decomposedfs.CtxKeySpaceGID, fi.Sys().(*syscall.Stat_t).Gid)

			return ctx, nil, err
		}
		scopeSpaceGroupHook := func(methodName string, ctx context.Context, spaceID string) (context.Context, middleware.UnHook, error) {
			spaceGID, ok := ctx.Value(decomposedfs.CtxKeySpaceGID).(uint32)
			if !ok {
				return ctx, nil, nil
			}

			unscope, err := um.ScopeUserByIds(-1, int(spaceGID))
			if err != nil {
				return ctx, nil, errors.Wrap(err, "failed to scope user")
			}

			return ctx, unscope, nil
		}
		hooks = append(hooks, resolveSpaceHook, scopeSpaceGroupHook)
	}

	mw := middleware.NewFS(dfs, hooks...)
	fs.FS = mw
	fs.tree = tp
	fs.um = um

	return fs, nil
}

// WarmupIDCache allows triggering a posix fs scan and id cache warmup manually.
func (fs *posixFS) WarmupIDCache(root string, assimilate, onlyDirty bool) error {
	return fs.tree.WarmupIDCache(root, assimilate, onlyDirty)
}

// ListUploadSessions returns the upload sessions matching the given filter
func (fs *posixFS) ListUploadSessions(ctx context.Context, filter storage.UploadSessionFilter) ([]storage.UploadSession, error) {
	return fs.FS.(storage.UploadSessionLister).ListUploadSessions(ctx, filter)
}

// UseIn tells the tus upload middleware which extensions it supports.
func (fs *posixFS) UseIn(composer *tusd.StoreComposer) {
	fs.FS.(storage.ComposableFS).UseIn(composer)
}

// NewUpload returns a new tus Upload instance
func (fs *posixFS) NewUpload(ctx context.Context, info tusd.FileInfo) (upload tusd.Upload, err error) {
	return fs.FS.(tusd.DataStore).NewUpload(ctx, info)
}

// NewUpload returns a new tus Upload instance
func (fs *posixFS) GetUpload(ctx context.Context, id string) (upload tusd.Upload, err error) {
	return fs.FS.(tusd.DataStore).GetUpload(ctx, id)
}

// AsTerminatableUpload returns a TerminatableUpload
// To implement the termination extension as specified in https://tus.io/protocols/resumable-upload.html#termination
// the storage needs to implement AsTerminatableUpload
func (fs *posixFS) AsTerminatableUpload(up tusd.Upload) tusd.TerminatableUpload {
	return up.(*upload.DecomposedFsSession)
}

// AsLengthDeclarableUpload returns a LengthDeclarableUpload
// To implement the creation-defer-length extension as specified in https://tus.io/protocols/resumable-upload.html#creation
// the storage needs to implement AsLengthDeclarableUpload
func (fs *posixFS) AsLengthDeclarableUpload(up tusd.Upload) tusd.LengthDeclarableUpload {
	return up.(*upload.DecomposedFsSession)
}

// AsConcatableUpload returns a ConcatableUpload
// To implement the concatenation extension as specified in https://tus.io/protocols/resumable-upload.html#concatenation
// the storage needs to implement AsConcatableUpload
func (fs *posixFS) AsConcatableUpload(up tusd.Upload) tusd.ConcatableUpload {
	return up.(*upload.DecomposedFsSession)
}
