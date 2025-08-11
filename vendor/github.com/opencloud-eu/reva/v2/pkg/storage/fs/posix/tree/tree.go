// Copyright 2018-2021 CERN
// Copyright 2025 OpenCloud GmbH <mail@opencloud.eu>
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

package tree

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/pkg/xattr"
	"github.com/rs/zerolog"
	"go-micro.dev/v4/store"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/lookup"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/trashbin"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/permissions"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/tree/propagator"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/usermapper"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/cs3org/reva/pkg/storage/pkg/decomposedfs/tree")
}

type Watcher interface {
	Watch(path string)
}

type scanItem struct {
	Path    string
	Recurse bool
}

// Tree manages a hierarchical tree
type Tree struct {
	lookup      *lookup.Lookup
	blobstore   node.Blobstore
	trashbin    *trashbin.Trashbin
	propagator  propagator.Propagator
	permissions permissions.Permissions

	options            *options.Options
	personalSpacesRoot string
	projectSpacesRoot  string

	userMapper    usermapper.Mapper
	idCache       store.Store
	watcher       Watcher
	scanQueue     chan scanItem
	scanDebouncer *ScanDebouncer

	es  events.Stream
	log *zerolog.Logger
}

// PermissionCheckFunc defined a function used to check resource permissions
type PermissionCheckFunc func(rp *provider.ResourcePermissions) bool

// New returns a new instance of Tree
func New(lu node.PathLookup, bs node.Blobstore, um usermapper.Mapper, trashbin *trashbin.Trashbin, permissions permissions.Permissions, o *options.Options, es events.Stream, cache store.Store, log *zerolog.Logger) (*Tree, error) {
	scanQueue := make(chan scanItem)

	t := &Tree{
		lookup:      lu.(*lookup.Lookup),
		blobstore:   bs,
		userMapper:  um,
		trashbin:    trashbin,
		permissions: permissions,
		options:     o,
		idCache:     cache,
		propagator:  propagator.New(lu, &o.Options, log),
		scanQueue:   scanQueue,
		scanDebouncer: NewScanDebouncer(o.ScanDebounceDelay, func(item scanItem) {
			scanQueue <- item
		}),
		es:                 es,
		log:                log,
		personalSpacesRoot: filepath.Clean(filepath.Join(o.Root, templates.Base(o.PersonalSpacePathTemplate))),
		projectSpacesRoot:  filepath.Clean(filepath.Join(o.Root, templates.Base(o.GeneralSpacePathTemplate))),
	}
	if err := t.checkStorage(); err != nil {
		return nil, errors.Wrap(err, "tree: unfit storage '"+o.Root+"'")
	}

	// Start watching for fs events and put them into the queue
	if o.WatchFS {
		watchPath := o.WatchPath
		var err error

		t.log.Info().Str("watch type", o.WatchType).Str("path", o.WatchPath).Str("root", o.WatchRoot).
			Str("brokers", o.WatchNotificationBrokers).Msg("Watching fs")
		switch o.WatchType {
		case "gpfswatchfolder":
			t.watcher, err = NewGpfsWatchFolderWatcher(t, strings.Split(o.WatchNotificationBrokers, ","), log)
			if err != nil {
				return nil, err
			}
		case "gpfsfileauditlogging":
			t.watcher, err = NewGpfsFileAuditLoggingWatcher(t, o.WatchPath, log)
			if err != nil {
				return nil, err
			}
		case "cephfs":
			t.watcher, err = NewCephfsWatcher(t, strings.Split(o.WatchNotificationBrokers, ","), log)
			if err != nil {
				return nil, err
			}
		default:
			t.watcher, err = NewInotifyWatcher(t, o, log)
			if err != nil {
				return nil, err
			}
			watchPath = o.Root
		}

		go t.watcher.Watch(watchPath)
		go t.workScanQueue()
	}
	if o.ScanFS {
		go func() {
			_ = t.WarmupIDCache(o.Root, true, false)
		}()
	}

	return t, nil
}

func (t *Tree) checkStorage() error {
	// check if the root path is a directory
	err := os.MkdirAll(t.options.Root, 0700)
	if err != nil {
		return errors.Wrap(err, "could not create root path")
	}
	fi, err := os.Stat(t.options.Root)
	if err != nil {
		return errors.Wrap(err, "root path does not exist")
	}
	if !fi.IsDir() {
		return errors.New("root path is not a directory")
	}

	// check if extended attributes are supported
	f, err := os.CreateTemp(t.options.Root, "posixfs-xattr-check-")
	if err != nil {
		return errors.Wrap(err, "could not create file in root path")
	}
	err = f.Close()
	if err != nil {
		return errors.Wrap(err, "could not close temp file")
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.log.Error().Err(err).Str("path", f.Name()).Msg("could not remove temp file")
		}
	}()

	attrKey := "user.posixfs.test"
	attrVal := []byte("test")
	if err := xattr.Set(f.Name(), attrKey, attrVal); err != nil {
		return errors.Wrap(err, "extended attributes not supported")
	}

	val, err := xattr.Get(f.Name(), attrKey)
	if err != nil {
		return errors.Wrap(err, "extended attributes not supported")
	}
	if string(val) != string(attrVal) {
		return errors.New("extended attribute mismatch")
	}
	return nil
}

func (t *Tree) PublishEvent(ev interface{}) {
	if t.es == nil {
		return
	}

	if err := events.Publish(context.Background(), t.es, ev); err != nil {
		t.log.Error().Err(err).Interface("event", ev).Msg("failed to publish event")
	}
}

// Setup prepares the tree structure
func (t *Tree) Setup() error {
	err := os.MkdirAll(t.options.Root, 0700)
	if err != nil {
		return err
	}

	err = os.MkdirAll(t.options.UploadDirectory, 0700)
	if err != nil {
		return err
	}

	return nil
}

// GetMD returns the metadata of a node in the tree
func (t *Tree) GetMD(ctx context.Context, n *node.Node) (os.FileInfo, error) {
	md, err := os.Stat(n.InternalPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errtypes.NotFound(n.ID)
		}
		return nil, errors.Wrap(err, "tree: error stating "+n.ID)
	}

	return md, nil
}

// TouchFile creates a new empty file
func (t *Tree) TouchFile(ctx context.Context, n *node.Node, markprocessing bool, mtime string) error {
	if n.Exists {
		if markprocessing {
			return n.SetXattr(ctx, prefixes.StatusPrefix, []byte(node.ProcessingStatus))
		}

		return errtypes.AlreadyExists(n.ID)
	}

	parentPath := n.ParentPath()
	nodePath := filepath.Join(parentPath, n.Name)

	// lock the meta file
	unlock, err := t.lookup.MetadataBackend().Lock(n)
	if err != nil {
		return err
	}
	defer func() {
		_ = unlock()
	}()

	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	n.SetType(provider.ResourceType_RESOURCE_TYPE_FILE)

	// Set id in cache
	if err := t.lookup.CacheID(context.Background(), n.SpaceID, n.ID, nodePath); err != nil {
		t.log.Error().Err(err).Str("spaceID", n.SpaceID).Str("id", n.ID).Str("path", nodePath).Msg("could not cache id")
	}

	if err := os.MkdirAll(filepath.Dir(nodePath), 0700); err != nil {
		return errors.Wrap(err, "posixfs: error creating node")
	}
	f, err := os.Create(nodePath)
	if err != nil {
		return errors.Wrap(err, "posixfs: error creating node")
	}

	attributes := n.NodeMetadata(ctx)
	attributes[prefixes.IDAttr] = []byte(n.ID)
	if markprocessing {
		attributes[prefixes.StatusPrefix] = []byte(node.ProcessingStatus)
	}
	if mtime != "" {
		nodeMTime, err := utils.MTimeToTime(mtime)
		if err != nil {
			return err
		}
		err = t.lookup.TimeManager().OverrideMtime(ctx, n, &attributes, nodeMTime)
		if err != nil {
			return err
		}
	} else {
		fi, err := f.Stat()
		if err != nil {
			return err
		}
		mtime := fi.ModTime()
		attributes[prefixes.MTimeAttr] = []byte(mtime.UTC().Format(time.RFC3339Nano))
	}

	err = n.SetXattrsWithContext(ctx, attributes, false)
	if err != nil {
		return err
	}

	return t.Propagate(ctx, n, 0)
}

// CreateDir creates a new directory entry in the tree
func (t *Tree) CreateDir(ctx context.Context, n *node.Node) (err error) {
	ctx, span := tracer.Start(ctx, "CreateDir")
	defer span.End()
	if n.Exists {
		return errtypes.AlreadyExists(n.ID) // path?
	}

	// create a directory node
	n.SetType(provider.ResourceType_RESOURCE_TYPE_CONTAINER)
	if n.ID == "" {
		n.ID = uuid.New().String()
	}

	err = t.createDirNode(ctx, n)
	if err != nil {
		return
	}

	return t.Propagate(ctx, n, 0)
}

// Move replaces the target with the source
func (t *Tree) Move(ctx context.Context, oldNode *node.Node, newNode *node.Node) (err error) {
	lockFilePath := oldNode.LockFilePath()

	if oldNode.SpaceID != newNode.SpaceID {
		// WebDAV RFC https://www.rfc-editor.org/rfc/rfc4918#section-9.9.4 says to use
		// > 502 (Bad Gateway) - This may occur when the destination is on another
		// > server and the destination server refuses to accept the resource.
		// > This could also occur when the destination is on another sub-section
		// > of the same server namespace.
		// but we only have a not supported error
		return errtypes.NotSupported("cannot move across spaces")
	}
	// if target exists delete it without trashing it
	if newNode.Exists {
		// TODO make sure all children are deleted
		if err := os.RemoveAll(newNode.InternalPath()); err != nil {
			return errors.Wrap(err, "posixfs: Move: error deleting target node "+newNode.ID)
		}
	}
	oldParent := oldNode.ParentPath()
	newParent := newNode.ParentPath()
	if newNode.ID == "" {
		newNode.ID = oldNode.ID
	}

	// update the id cache
	// invalidate old tree
	err = t.lookup.IDCache.DeleteByPath(ctx, filepath.Join(oldNode.ParentPath(), oldNode.Name))
	if err != nil {
		return err
	}
	if err := t.lookup.CacheID(ctx, newNode.SpaceID, newNode.ID, filepath.Join(newNode.ParentPath(), newNode.Name)); err != nil {
		t.log.Error().Err(err).Str("spaceID", newNode.SpaceID).Str("id", newNode.ID).Str("path", filepath.Join(newNode.ParentPath(), newNode.Name)).Msg("could not cache id")
	}

	// rename node
	err = os.Rename(
		filepath.Join(oldParent, oldNode.Name),
		filepath.Join(newParent, newNode.Name),
	)
	if err != nil {
		if err := t.lookup.CacheID(ctx, oldNode.SpaceID, oldNode.ID, filepath.Join(oldNode.ParentPath(), oldNode.Name)); err != nil {
			t.log.Error().Err(err).Str("spaceID", oldNode.SpaceID).Str("id", oldNode.ID).Str("path", filepath.Join(oldNode.ParentPath(), oldNode.Name)).Msg("could not reset cached id after failed move")
		}
		if err := t.WarmupIDCache(filepath.Join(oldNode.ParentPath(), oldNode.Name), false, false); err != nil {
			t.log.Error().Err(err).Str("spaceID", oldNode.SpaceID).Str("id", oldNode.ID).Str("path", filepath.Join(oldNode.ParentPath(), oldNode.Name)).Msg("could not warum cached after failed move")
		}
		return errors.Wrap(err, "posixfs: could not move child")
	}

	// update target parentid and name
	attribs := node.Attributes{}
	attribs.SetString(prefixes.ParentidAttr, newNode.ParentID)
	attribs.SetString(prefixes.NameAttr, newNode.Name)
	if err := newNode.SetXattrsWithContext(ctx, attribs, true); err != nil {
		return errors.Wrap(err, "posixfs: could not update node attributes")
	}

	// rename the lock (if it exists)
	if _, err := os.Stat(lockFilePath); err == nil {
		err = os.Rename(lockFilePath, newNode.LockFilePath())
		if err != nil {
			return errors.Wrap(err, "posixfs: could not move lock")
		}
	}

	// update id cache for the moved subtree.
	if oldNode.IsDir(ctx) {
		err = t.WarmupIDCache(filepath.Join(newNode.ParentPath(), newNode.Name), false, false)
		if err != nil {
			return err
		}
	}

	// the size diff is the current treesize or blobsize of the old/source node
	var sizeDiff int64
	if oldNode.IsDir(ctx) {
		treeSize, err := oldNode.GetTreeSize(ctx)
		if err != nil {
			return err
		}
		sizeDiff = int64(treeSize)
	} else {
		sizeDiff = oldNode.Blobsize
	}

	err = t.Propagate(ctx, oldNode, -sizeDiff)
	if err != nil {
		return errors.Wrap(err, "posixfs: Move: could not propagate old node")
	}
	err = t.Propagate(ctx, newNode, sizeDiff)
	if err != nil {
		return errors.Wrap(err, "posixfs: Move: could not propagate new node")
	}
	return nil
}

// ListFolder lists the content of a folder node
func (t *Tree) ListFolder(ctx context.Context, n *node.Node) ([]*node.Node, error) {
	ctx, span := tracer.Start(ctx, "ListFolder")
	defer span.End()
	dir := n.InternalPath()

	_, subspan := tracer.Start(ctx, "os.Open")
	f, err := os.Open(dir)
	subspan.End()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errtypes.NotFound(dir)
		}
		return nil, errors.Wrap(err, "tree: error listing "+dir)
	}
	defer f.Close()

	_, subspan = tracer.Start(ctx, "f.Readdirnames")
	names, err := f.Readdirnames(0)
	subspan.End()
	if err != nil {
		return nil, err
	}

	numWorkers := t.options.MaxConcurrency
	if len(names) < numWorkers {
		numWorkers = len(names)
	}
	work := make(chan string)
	results := make(chan *node.Node)

	g, ctx := errgroup.WithContext(ctx)

	// Distribute work
	g.Go(func() error {
		defer close(work)
		for _, name := range names {
			if t.isInternal(name) || isLockFile(name) || isTrash(name) {
				continue
			}

			select {
			case work <- name:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// Spawn workers that'll concurrently work the queue
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			// switch user if necessary
			spaceGID, ok := ctx.Value(decomposedfs.CtxKeySpaceGID).(uint32)
			if ok {
				unscope, err := t.userMapper.ScopeUserByIds(-1, int(spaceGID))
				if err != nil {
					return errors.Wrap(err, "failed to scope user")
				}
				defer func() { _ = unscope() }()
			}

			for name := range work {
				path := filepath.Join(dir, name)

				_, nodeID, err := t.lookup.IDsForPath(ctx, path)
				if err != nil {
					t.log.Error().Err(err).Str("path", path).Msg("failed to get ids for entry")
					continue
				}

				child, err := node.ReadNode(ctx, t.lookup, n.SpaceID, nodeID, false, n.SpaceRoot, true)
				if err != nil {
					t.log.Error().Err(err).Str("path", path).Msg("failed to read node")
					continue
				}

				// prevent listing denied resources
				if !child.IsDenied(ctx) {
					if child.SpaceRoot == nil {
						child.SpaceRoot = n.SpaceRoot
					}
					select {
					case results <- child:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			return nil
		})
	}
	// Wait for things to settle down, then close results chan
	go func() {
		_ = g.Wait() // error is checked later
		close(results)
	}()

	retNodes := []*node.Node{}
	for n := range results {
		retNodes = append(retNodes, n)
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return retNodes, nil
}

// Delete deletes a node in the tree by moving it to the trash
func (t *Tree) Delete(ctx context.Context, n *node.Node) error {
	path := n.InternalPath()

	if !strings.HasPrefix(path, t.options.Root) {
		return errtypes.InternalError("invalid internal path")
	}

	// remove entry from cache immediately to avoid inconsistencies
	defer func() {
		if err := t.idCache.Delete(path); err != nil {
			t.log.Error().Err(err).Str("path", path).Msg("could not delete id from cache")
		}
	}()

	if appctx.DeletingSharedResourceFromContext(ctx) {
		src := filepath.Join(n.ParentPath(), n.Name)
		return os.RemoveAll(src)
	}

	var sizeDiff int64
	if n.IsDir(ctx) {
		treesize, err := n.GetTreeSize(ctx)
		if err != nil {
			return err // TODO calculate treesize if it is not set
		}
		sizeDiff = -int64(treesize)
	} else {
		sizeDiff = -n.Blobsize
	}

	// Remove lock file if it exists
	if err := os.Remove(n.LockFilePath()); err != nil {
		t.log.Error().Err(err).Str("path", n.LockFilePath()).Msg("could not remove lock file")
	}

	err := t.trashbin.MoveToTrash(ctx, n, path)
	if err != nil {
		return err
	}

	return t.Propagate(ctx, n, sizeDiff)
}

// Propagate propagates changes to the root of the tree
func (t *Tree) Propagate(ctx context.Context, n *node.Node, sizeDiff int64) (err error) {
	// We do not propagate size diffs here but rely on the assimilation to take care of the tree sizes instead
	return t.propagator.Propagate(ctx, n, sizeDiff)
}

// WriteBlob writes a blob to the blobstore
func (t *Tree) WriteBlob(n *node.Node, source string) error {
	var currentPath string
	var err error

	if t.options.EnableFSRevisions {
		currentPath = t.lookup.CurrentPath(n.SpaceID, n.ID)

		defer func() {
			attrs := node.Attributes{}
			attrs.SetInt64(prefixes.TypeAttr, int64(n.Type(context.Background())))
			attrs.SetString(prefixes.BlobIDAttr, n.BlobID)
			attrs.SetInt64(prefixes.BlobsizeAttr, n.Blobsize)

			err := t.lookup.MetadataBackend().SetMultiple(context.Background(), node.NewBaseNode(n.SpaceID, n.ID+node.CurrentIDDelimiter, t.lookup), attrs, true)
			if err != nil {
				t.log.Error().Err(err).Str("spaceID", n.SpaceID).Str("id", n.ID).Msg("could not copy metadata to current revision")
			}
		}()
	}

	err = t.blobstore.Upload(n, source, currentPath)
	return err
}

// ReadBlob reads a blob from the blobstore
func (t *Tree) ReadBlob(node *node.Node) (io.ReadCloser, error) {
	return t.blobstore.Download(node)
}

// DeleteBlob deletes a blob from the blobstore
func (t *Tree) DeleteBlob(node *node.Node) error {
	if node == nil {
		return fmt.Errorf("could not delete blob, nil node was given")
	}
	return t.blobstore.Delete(node)
}

// BuildSpaceIDIndexEntry returns the entry for the space id index
func (t *Tree) BuildSpaceIDIndexEntry(spaceID string) string {
	return spaceID
}

// ResolveSpaceIDIndexEntry returns the node id for the space id index entry
func (t *Tree) ResolveSpaceIDIndexEntry(spaceID string) (string, error) {
	return spaceID, nil
}

// InitNewNode initializes a new node
func (t *Tree) InitNewNode(ctx context.Context, n *node.Node, fsize uint64) (metadata.UnlockFunc, error) {
	_, span := tracer.Start(ctx, "InitNewNode")
	defer span.End()
	// create folder structure (if needed)
	if err := os.MkdirAll(filepath.Dir(n.InternalPath()), 0700); err != nil {
		return nil, err
	}

	// create and write lock new node metadata
	unlock, err := t.lookup.MetadataBackend().Lock(n)
	if err != nil {
		return nil, err
	}

	// we also need to touch the actual node file here it stores the mtime of the resource
	h, err := os.OpenFile(n.InternalPath(), os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			return unlock, errtypes.AlreadyExists(n.InternalPath())
		}
		return unlock, err
	}
	h.Close()

	if _, err := node.CheckQuota(ctx, n.SpaceRoot, false, 0, fsize); err != nil {
		return unlock, err
	}

	return unlock, nil
}

// TODO check if node exists?
func (t *Tree) createDirNode(ctx context.Context, n *node.Node) (err error) {
	ctx, span := tracer.Start(ctx, "createDirNode")
	defer span.End()

	idcache := t.lookup.IDCache
	// create a directory node
	parentPath, ok := idcache.Get(ctx, n.SpaceID, n.ParentID)
	if !ok {
		return errtypes.NotFound(n.ParentID)
	}
	path := filepath.Join(parentPath, n.Name)

	// lock the meta file
	unlock, err := t.lookup.MetadataBackend().Lock(n)
	if err != nil {
		return err
	}
	defer func() {
		_ = unlock()
	}()

	if err := os.MkdirAll(path, 0700); err != nil {
		return errors.Wrap(err, "posixfs: error creating node")
	}

	if err := idcache.Set(ctx, n.SpaceID, n.ID, path); err != nil {
		t.log.Error().Err(err).Str("spaceID", n.SpaceID).Str("id", n.ID).Str("path", path).Msg("could not cache id")
	}

	attributes := n.NodeMetadata(ctx)
	attributes[prefixes.IDAttr] = []byte(n.ID)
	attributes[prefixes.TreesizeAttr] = []byte("0") // initialize as empty, TODO why bother? if it is not set we could treat it as 0?
	if t.options.TreeTimeAccounting || t.options.TreeSizeAccounting {
		attributes[prefixes.PropagationAttr] = []byte("1") // mark the node for propagation
	}
	return n.SetXattrsWithContext(ctx, attributes, false)
}

func (t *Tree) isIgnored(path string) bool {
	return isLockFile(path) || isTrash(path) || t.isUpload(path) || t.isInternal(path) || t.isRootPath(path)
}

func (t *Tree) isUpload(path string) bool {
	return strings.HasPrefix(path, t.options.UploadDirectory)
}

func (t *Tree) isIndex(path string) bool {
	return strings.HasPrefix(path, filepath.Join(t.options.Root, "indexes"))
}

func (t *Tree) isRootPath(path string) bool {
	return path == t.options.Root ||
		path == t.personalSpacesRoot ||
		path == t.projectSpacesRoot
}

func (t *Tree) isInternal(path string) bool {
	return t.isIndex(path) || strings.Contains(path, lookup.MetadataDir)
}

func isLockFile(path string) bool {
	return strings.HasSuffix(path, ".lock") || strings.HasSuffix(path, ".flock") || strings.HasSuffix(path, ".mlock")
}

func isTrash(path string) bool {
	return strings.HasSuffix(path, ".trashinfo") || strings.HasSuffix(path, ".trashitem") || strings.Contains(path, ".Trash")
}
