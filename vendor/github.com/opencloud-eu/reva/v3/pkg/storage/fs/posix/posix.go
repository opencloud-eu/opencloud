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

// Package posix implements the "posix" storage driver, which keeps a regular,
// browsable directory tree on disk and picks up changes made to it directly.
// It reuses the decomposedfs core with its own lookup, tree, trashbin and blobstore
// and wraps the result, see pkg/storage/pkg/decomposedfs/README.md for how it
// builds itself and how it delegates to and overrides the shared code.
package posix

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v3/pkg/storage"
	"github.com/opencloud-eu/reva/v3/pkg/storage/cache"
	"github.com/opencloud-eu/reva/v3/pkg/storage/fs/posix/blobstore"
	"github.com/opencloud-eu/reva/v3/pkg/storage/fs/posix/idcache"
	"github.com/opencloud-eu/reva/v3/pkg/storage/fs/posix/lookup"
	"github.com/opencloud-eu/reva/v3/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v3/pkg/storage/fs/posix/timemanager"
	"github.com/opencloud-eu/reva/v3/pkg/storage/fs/posix/trashbin"
	"github.com/opencloud-eu/reva/v3/pkg/storage/fs/posix/tree"
	"github.com/opencloud-eu/reva/v3/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/aspects"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/permissions"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/revisioncleanup"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/usermapper"
	"github.com/opencloud-eu/reva/v3/pkg/storage/utils/middleware"
	"github.com/pkg/errors"
)

func init() {
	registry.Register("posix", NewDefault)
}

type posixFS struct {
	storage.FS

	tree *tree.Tree
	um   usermapper.Mapper
	// the inner Decomposedfs, for the on-demand revision cleanup: storage.FS is the
	// middleware wrapping it, which doesn't pass optional interfaces through
	revisions revisioncleanup.OnDemand
	// resolves Request.FSPath for the on-demand revision cleanup
	fsPaths fsPathResolver
}

type fsPathResolver interface {
	SpacePathFromFSPath(ctx context.Context, p string) (string, string, error)
}

var _ fsPathResolver = (*lookup.Lookup)(nil)

var _ revisioncleanup.OnDemand = (*posixFS)(nil)

// New returns an implementation to of the storage.FS interface that talk to
// a local filesystem.
func NewDefault(m map[string]interface{}, stream events.Stream, log *zerolog.Logger) (storage.FS, error) {
	o, err := options.New(m)
	if err != nil {
		return nil, err
	}

	// Check this before connecting, otherwise a non-nats store keeps us retrying for minutes
	if err := checkIDCacheStore(o); err != nil {
		return nil, err
	}

	o.IDCache.Database += "_v2" // Use a versioned bucket name to avoid conflicts with previous implementations
	o.IDCache.TTL = 0           // Disable TTL for the ID cache, as the posix driver relies on it for caching file IDs and we don't want them to expire
	kv, err := cache.NewNatsKeyValue(o.IDCache, log)
	if err != nil {
		return nil, errors.Wrap(err, "could not create nats key value store")
	}
	c, err := idcache.NewStoreIDCache(kv)
	if err != nil {
		return nil, err
	}

	o.IDCache.Database += "_history" // Use a versioned bucket name to avoid conflicts with previous implementations
	o.IDCache.TTL = 24 * 60 * time.Minute
	historyKv, err := cache.NewNatsKeyValue(o.IDCache, log)
	if err != nil {
		return nil, errors.Wrap(err, "could not create nats key value store")
	}
	historyCache, err := idcache.NewStoreIDCache(historyKv)
	if err != nil {
		return nil, err
	}

	return New(o, stream, c, historyCache, log)
}

func checkIDCacheStore(o *options.Options) error {
	if o.IDCache.Store != "nats-js-kv" {
		return fmt.Errorf("the posix driver requires a nats-js-kv id cache, got '%s'", o.IDCache.Store)
	}
	return nil
}

func New(o *options.Options, stream events.Stream, cache, historyCache *idcache.IDCache, log *zerolog.Logger) (storage.FS, error) {
	if err := checkIDCacheStore(o); err != nil {
		return nil, err
	}
	if o.FileMetadataCache.Store == "noop" {
		return nil, fmt.Errorf("the posix driver requires a file metadata cache")
	}
	if err := prefixes.SetOcPrefix(o.MetadataPrefix); err != nil {
		return nil, err
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
		lu, err = lookup.New(metadata.NewXattrsBackend(o.FileMetadataCache), um, o, &timemanager.Manager{}, cache, historyCache, log)
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
			o.FileMetadataCache), um, o, &timemanager.Manager{}, cache, historyCache, log)
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
		Lookup:      lu,
		Tree:        tp,
		Permissions: p,
		EventStream: stream,
		UserMapper:  um,
		Trashbin:    trashbin,
		// the tree already created the revision cleanup for its own use in assimilation
		RevisionsCleanup: tp.RevisionsCleanup(),
	}

	dfs, err := decomposedfs.New(&o.Options, aspects, log)
	if err != nil {
		// the tree already started the revision cleanup, stop its workers again
		_ = tp.RevisionsCleanup().Shutdown(context.Background())
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
	if onDemand, ok := dfs.(revisioncleanup.OnDemand); ok {
		fs.revisions = onDemand
	}
	fs.fsPaths = lu
	fs.tree = tp
	fs.um = um

	return fs, nil
}

func (fs *posixFS) WithDisabledSpaces() storage.FS {
	f := *fs
	f.FS = fs.FS.(storage.SpaceDisabledListableFS).WithDisabledSpaces()
	return &f
}

// CleanupRevisionsUnchecked cleans up revisions on demand, see revisioncleanup.OnDemand. It goes to
// the inner Decomposedfs directly, bypassing the middleware, whose space group hooks only switch to
// the group of the space of a user request: the cleanup switches to the group of each space it
// cleans up itself, see revisioncleanup.SpaceScoper.
//
// A path on disk (Request.FSPath) is resolved here to its space and its path within the space,
// which the Decomposedfs doesn't know about.
func (fs *posixFS) CleanupRevisionsUnchecked(ctx context.Context, req revisioncleanup.Request, report func(revisioncleanup.Result)) error {
	if fs.revisions == nil {
		return errtypes.NotSupported("on-demand revision cleanup")
	}
	if req.DryRun {
		// a dry run changes nothing on disk, not even the lock files of what it reads
		ctx = metadata.WithReadOnly(ctx)
	}
	if req.FSPath != "" {
		if err := req.Validate(); err != nil {
			return err
		}
		spaceID, path, err := fs.fsPaths.SpacePathFromFSPath(ctx, req.FSPath)
		if err != nil {
			return err
		}
		req.SpaceIDs, req.Path, req.FSPath = []string{spaceID}, path, ""
	}
	return fs.revisions.CleanupRevisionsUnchecked(ctx, req, report)
}

// WarmupIDCache allows triggering a posix fs scan and id cache warmup manually.
func (fs *posixFS) WarmupIDCache(root string, assimilate, onlyDirty bool) error {
	return fs.tree.WarmupIDCache(root, assimilate, onlyDirty)
}

// PrepareCreatesNode reports whether the wrapped driver's PrepareUpload creates a
// new file's node.
func (fs *posixFS) PrepareCreatesNode() bool {
	nc, ok := fs.FS.(storage.NodeCreator)
	return ok && nc.PrepareCreatesNode()
}

// HandleUploadFinished passes the end of an upload on to the wrapped driver, see
// storage.UploadFinishedHandler.
func (fs *posixFS) HandleUploadFinished(ctx context.Context, ref *provider.Reference, sessionID string, versionCreated bool) {
	if h, ok := fs.FS.(storage.UploadFinishedHandler); ok {
		h.HandleUploadFinished(ctx, ref, sessionID, versionCreated)
	}
}

// IsOrphaned reports whether the referenced resource exists but its metadata is unreadable.
func (fs *posixFS) IsOrphaned(ctx context.Context, ref *provider.Reference) bool {
	return fs.FS.(storage.OrphanChecker).IsOrphaned(ctx, ref)
}
