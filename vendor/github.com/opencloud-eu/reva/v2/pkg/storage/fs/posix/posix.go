//go:build linux
// +build linux

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

	"github.com/rs/zerolog"
	tusd "github.com/tus/tusd/v2/pkg/handler"
	microstore "go-micro.dev/v4/store"

	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/blobstore"
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
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/pkg/errors"
)

func init() {
	registry.Register("posix", New)
}

type posixFS struct {
	storage.FS

	um usermapper.Mapper
}

// New returns an implementation to of the storage.FS interface that talk to
// a local filesystem.
func New(m map[string]interface{}, stream events.Stream, log *zerolog.Logger) (storage.FS, error) {
	o, err := options.New(m)
	if err != nil {
		return nil, err
	}

	fs := &posixFS{}
	um := usermapper.NewUnixMapper()

	var lu *lookup.Lookup
	switch o.MetadataBackend {
	case "xattrs":
		lu = lookup.New(metadata.NewXattrsBackend(o.FileMetadataCache), um, o, &timemanager.Manager{})
	case "hybrid":
		lu = lookup.New(metadata.NewHybridBackend(1024, // start offloading grants after 1KB
			func(n metadata.MetadataNode) string {
				spaceRoot, _ := lu.IDCache.Get(context.Background(), n.GetSpaceID(), n.GetSpaceID())
				if len(spaceRoot) == 0 {
					return ""
				}

				return filepath.Join(spaceRoot, lookup.MetadataDir)
			},
			o.FileMetadataCache), um, o, &timemanager.Manager{})
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

	switch o.IDCache.Store {
	case "", "memory", "noop":
		return nil, fmt.Errorf("the posix driver requires a shared id cache, e.g. nats-js-kv or redis")
	}

	tp, err := tree.New(lu, bs, um, trashbin, p, o, stream, store.Create(
		store.Store(o.IDCache.Store),
		store.TTL(o.IDCache.TTL),
		store.Size(o.IDCache.Size),
		microstore.Nodes(o.IDCache.Nodes...),
		microstore.Database(o.IDCache.Database),
		microstore.Table(o.IDCache.Table),
		store.DisablePersistence(o.IDCache.DisablePersistence),
		store.Authentication(o.IDCache.AuthUsername, o.IDCache.AuthPassword),
	), log)
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
	fs.um = um

	return fs, nil
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
