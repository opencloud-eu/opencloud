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
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/pkg/xattr"
	"github.com/rs/zerolog/log"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

type ScanDebouncer struct {
	after      time.Duration
	f          func(item scanItem)
	pending    sync.Map
	inProgress sync.Map

	mutex sync.Mutex
}

type EventAction int

const (
	ActionCreate EventAction = iota
	ActionUpdate
	ActionMove
	ActionDelete
	ActionMoveFrom
)

type queueItem struct {
	item  scanItem
	timer *time.Timer
}

const dirtyFlag = "user.oc.dirty"

type assimilationNode struct {
	path    string
	nodeId  string
	spaceID string
}

func (d assimilationNode) GetID() string {
	return d.nodeId
}

func (d assimilationNode) GetSpaceID() string {
	return d.spaceID
}

func (d assimilationNode) InternalPath() string {
	return d.path
}

// NewScanDebouncer returns a new SpaceDebouncer instance
func NewScanDebouncer(d time.Duration, f func(item scanItem)) *ScanDebouncer {
	return &ScanDebouncer{
		after:      d,
		f:          f,
		pending:    sync.Map{},
		inProgress: sync.Map{},
	}
}

// Debounce restarts the debounce timer for the given space
func (d *ScanDebouncer) Debounce(item scanItem) {
	if d.after == 0 {
		d.f(item)
		return
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	path := item.Path
	recurse := item.Recurse
	if i, ok := d.pending.Load(item.Path); ok {
		AssimilationPendingTasks.Dec()
		queueItem := i.(*queueItem)
		recurse = recurse || queueItem.item.Recurse
		queueItem.timer.Stop()
	}

	AssimilationPendingTasks.Inc()
	d.pending.Store(item.Path, &queueItem{
		item: item,
		timer: time.AfterFunc(d.after, func() {
			if _, ok := d.inProgress.Load(path); ok {
				// Reschedule this run for when the previous run has finished
				d.mutex.Lock()
				if i, ok := d.pending.Load(path); ok {
					i.(*queueItem).timer.Reset(d.after)
				}

				d.mutex.Unlock()
				return
			}

			AssimilationPendingTasks.Dec()
			d.pending.Delete(path)
			d.inProgress.Store(path, true)
			defer d.inProgress.Delete(path)
			d.f(scanItem{
				Path:    path,
				Recurse: recurse,
			})
		}),
	})
}

// InProgress returns true if the given path is currently being processed
func (d *ScanDebouncer) InProgress(path string) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if _, ok := d.pending.Load(path); ok {
		return true
	}

	_, ok := d.inProgress.Load(path)
	return ok
}

// Pending returns true if the given path is currently pending to be processed
func (d *ScanDebouncer) Pending(path string) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	_, ok := d.pending.Load(path)
	return ok
}

func (t *Tree) workScanQueue() {
	for i := 0; i < t.options.MaxConcurrency; i++ {
		go func() {
			for {
				item := <-t.scanQueue

				err := t.assimilate(item)
				if err != nil {
					log.Error().Err(err).Str("path", item.Path).Msg("failed to assimilate item")
					continue
				}

				if item.Recurse {
					err = t.WarmupIDCache(item.Path, true, false)
					if err != nil {
						log.Error().Err(err).Str("path", item.Path).Msg("failed to warmup id cache")
					}
				}
			}
		}()
	}
}

// Scan scans the given path and updates the id chache
func (t *Tree) Scan(path string, action EventAction, isDir bool) error {
	// cases:
	switch action {
	case ActionCreate:
		t.log.Debug().Str("path", path).Bool("isDir", isDir).Msg("scanning path (ActionCreate)")
		if !isDir {
			// 1. New file (could be emitted as part of a new directory)
			//	 -> assimilate file
			//   -> scan parent directory recursively to update tree size and catch nodes that weren't covered by an event
			AssimilationCounter.WithLabelValues(_labelFile, _labelAdded).Inc()
			if !t.scanDebouncer.Pending(filepath.Dir(path)) {
				t.scanDebouncer.Debounce(scanItem{
					Path: path,
				})
			}
			if err := t.setDirty(filepath.Dir(path), true); err != nil {
				t.log.Error().Err(err).Str("path", path).Bool("isDir", isDir).Msg("failed to mark directory as dirty")
			}
			t.scanDebouncer.Debounce(scanItem{
				Path:    filepath.Dir(path),
				Recurse: true,
			})
		} else {
			// 2. New directory
			//  -> scan directory
			if err := t.setDirty(path, true); err != nil {
				t.log.Error().Err(err).Str("path", path).Bool("isDir", isDir).Msg("failed to mark directory as dirty")
			}
			AssimilationCounter.WithLabelValues(_labelDir, _labelAdded).Inc()
			t.scanDebouncer.Debounce(scanItem{
				Path:    path,
				Recurse: true,
			})
		}

	case ActionUpdate:
		t.log.Debug().Str("path", path).Bool("isDir", isDir).Msg("scanning path (ActionUpdate)")
		// 3. Updated file
		//   -> update file unless parent directory is being rescanned
		if !t.scanDebouncer.InProgress(filepath.Dir(path)) {
			t.scanDebouncer.Debounce(scanItem{
				Path: path,
			})
		}

		if !isDir {
			AssimilationCounter.WithLabelValues(_labelFile, _labelUpdated).Inc()
		} else {
			AssimilationCounter.WithLabelValues(_labelDir, _labelUpdated).Inc()
		}

	case ActionMove:
		t.log.Debug().Str("path", path).Bool("isDir", isDir).Msg("scanning path (ActionMove)")
		// 4. Moved file
		//   -> update file
		// 5. Moved directory
		//   -> update directory and all children
		t.scanDebouncer.Debounce(scanItem{
			Path:    path,
			Recurse: isDir,
		})

		if !isDir {
			AssimilationCounter.WithLabelValues(_labelFile, _labelMoved).Inc()
		} else {
			AssimilationCounter.WithLabelValues(_labelDir, _labelMoved).Inc()
		}

	case ActionMoveFrom:
		t.log.Debug().Str("path", path).Bool("isDir", isDir).Msg("scanning path (ActionMoveFrom)")
		// 6. file/directory moved out of the watched directory
		//   -> remove from caches

		// remember the id of the moved away item
		spaceID, nodeID, err := t.lookup.IDsForPath(context.Background(), path)
		if err == nil {
			err = t.lookup.IDHistoryCache.Set(context.Background(), spaceID, nodeID, path)
			if err != nil {
				t.log.Error().Err(err).Str("path", path).Msg("failed to cache the id of the moved item")
			}
		}

		err = t.HandleFileDelete(path, false) // Do not send a item-trashed SSE in case of moves. They trigger a item-renamed event instead.
		if err != nil {
			t.log.Error().Err(err).Str("path", path).Bool("isDir", isDir).Msg("failed to handle moved away item")
		}

		// We do not do metrics here because this has been handled in `ActionMove`

	case ActionDelete:
		t.log.Debug().Str("path", path).Bool("isDir", isDir).Msg("handling deleted item")

		// 7. Deleted file or directory
		//   -> update parent and all children

		err := t.HandleFileDelete(path, true)
		if err != nil {
			t.log.Error().Err(err).Str("path", path).Bool("isDir", isDir).Msg("failed to handle deleted item")
		}

		t.scanDebouncer.Debounce(scanItem{
			Path:    filepath.Dir(path),
			Recurse: true,
		})

		if !isDir {
			AssimilationCounter.WithLabelValues(_labelFile, _labelDeleted).Inc()
		} else {
			AssimilationCounter.WithLabelValues(_labelDir, _labelDeleted).Inc()
		}
	}

	return nil
}

func (t *Tree) HandleFileDelete(path string, sendSSE bool) error {
	spaceID, id, err := t.lookup.IDsForPath(context.Background(), path)
	if err != nil {
		return err
	}
	n := node.NewBaseNode(spaceID, id, t.lookup)
	if n.InternalPath() != path {
		return fmt.Errorf("internal path does not match path")
	}
	_, err = os.Stat(path)
	if err == nil || !os.IsNotExist(err) {
		t.log.Info().Str("path", path).Msg("file that was about to be cleared still exists/exists again. We'll leave it alone")
		return nil
	}

	// purge metadata
	if err := t.lookup.IDCache.DeleteByPath(context.Background(), path); err != nil {
		t.log.Error().Err(err).Str("path", path).Msg("could not delete id cache entry by path")
	}
	if err := t.lookup.MetadataBackend().Purge(context.Background(), n); err != nil {
		t.log.Error().Err(err).Str("path", path).Msg("could not purge metadata")
	}

	if !sendSSE {
		return nil
	}

	parentNode, err := t.getNodeForPath(filepath.Dir(path))
	if err != nil {
		return err
	}
	t.PublishEvent(events.ItemTrashed{
		Owner:     parentNode.Owner(),
		Executant: parentNode.Owner(),
		Ref: &provider.Reference{
			ResourceId: &provider.ResourceId{
				StorageId: t.options.MountID,
				SpaceId:   n.SpaceID,
				OpaqueId:  parentNode.ID,
			},
			Path: filepath.Base(path),
		},
		ID: &provider.ResourceId{
			StorageId: t.options.MountID,
			SpaceId:   n.SpaceID,
			OpaqueId:  n.ID,
		},
		Timestamp: utils.TSNow(),
	})

	return nil
}

func (t *Tree) getNodeForPath(path string) (*node.Node, error) {
	spaceID, nodeID, err := t.lookup.IDsForPath(context.Background(), path)
	if err != nil {
		return nil, err
	}

	return node.ReadNode(context.Background(), t.lookup, spaceID, nodeID, false, nil, false)
}

func (t *Tree) findSpaceId(path string) (string, error) {
	// find the space id, scope by the according user
	spaceCandidate := path
	for strings.HasPrefix(spaceCandidate, t.options.Root) {
		spaceID, _, err := t.lookup.IDsForPath(context.Background(), spaceCandidate)
		if err == nil && len(spaceID) > 0 {
			if t.options.UseSpaceGroups {
				// set the uid and gid for the space
				fi, err := os.Stat(spaceCandidate)
				if err != nil {
					return "", err
				}
				sys := fi.Sys().(*syscall.Stat_t)
				gid := int(sys.Gid)
				_, err = t.userMapper.ScopeUserByIds(-1, gid)
				if err != nil {
					return "", err
				}
			}

			return spaceID, nil
		}
		spaceCandidate = filepath.Dir(spaceCandidate)
	}
	return "", fmt.Errorf("could not find space for path %s", path)
}

func (t *Tree) assimilate(item scanItem) error {
	t.log.Debug().Str("path", item.Path).Bool("recurse", item.Recurse).Msg("assimilate")
	var err error

	spaceID, id, parentID, mtime, err := t.lookup.MetadataBackend().IdentifyPath(context.Background(), item.Path)
	if err != nil {
		return err
	}

	if spaceID == "" {
		// node didn't have a space ID attached. try to find it by walking up the path on disk
		spaceID, err = t.findSpaceId(filepath.Dir(item.Path))
		if err != nil {
			return err
		}
	}

	if id != "" {
		// the file has an id set, we already know it from the past

		// lock the file for re-assimilation
		assimilationNode := &assimilationNode{
			spaceID: spaceID,
			nodeId:  id,
			path:    item.Path,
		}

		unlock, err := t.lookup.MetadataBackend().Lock(assimilationNode)
		if err != nil {
			return errors.Wrap(err, "failed to lock item for assimilation")
		}
		defer func() {
			_ = unlock()
		}()

		previousPath, ok := t.lookup.GetCachedID(context.Background(), spaceID, id)
		if previousPath == "" || !ok {
			previousPath, ok = t.lookup.IDHistoryCache.Get(context.Background(), spaceID, id)
		}

		// compare metadata mtime with actual mtime. if it matches AND the path hasn't changed (move operation)
		// we can skip the assimilation because the file was handled by us
		fi, err := os.Stat(item.Path)
		if err == nil && previousPath == item.Path {
			if mtime.Equal(fi.ModTime()) {
				return nil
			}
		}

		// was it moved or copied/restored with a clashing id?
		if ok && len(parentID) > 0 && previousPath != item.Path {
			_, err := os.Stat(previousPath)
			if err == nil {
				// this id clashes with an existing item -> clear metadata and re-assimilate
				t.log.Debug().Str("path", item.Path).Msg("ID clash detected, purging metadata and re-assimilating")

				if err := t.lookup.MetadataBackend().Purge(context.Background(), assimilationNode); err != nil {
					t.log.Error().Err(err).Str("path", item.Path).Msg("could not purge metadata")
				}
				go func() {
					if err := t.assimilate(scanItem{Path: item.Path}); err != nil {
						t.log.Error().Err(err).Str("path", item.Path).Msg("could not re-assimilate")
					}
				}()
			} else {
				// this is a move
				t.log.Debug().Str("path", item.Path).Msg("move detected")

				if err := t.lookup.CacheID(context.Background(), spaceID, id, item.Path); err != nil {
					t.log.Error().Err(err).Str("spaceID", spaceID).Str("id", id).Str("path", item.Path).Msg("could not cache id")
				}
				_, attrs, err := t.updateFile(item.Path, id, spaceID, fi)
				if err != nil {
					return err
				}

				// Delete the path entry using DeletePath(reverse lookup), not the whole entry pair.
				if err := t.lookup.IDCache.DeletePath(context.Background(), previousPath); err != nil {
					t.log.Error().Err(err).Str("path", previousPath).Msg("could not delete id cache entry by path")
				}

				if fi.IsDir() {
					// if it was moved and it is a directory we need to propagate the move
					go func() {
						if err := t.WarmupIDCache(item.Path, false, true); err != nil {
							t.log.Error().Err(err).Str("path", item.Path).Msg("could not warmup id cache")
						}
					}()
				}

				newParentID := attrs.String(prefixes.ParentidAttr)
				if len(parentID) > 0 {
					ref := &provider.Reference{
						ResourceId: &provider.ResourceId{
							StorageId: t.options.MountID,
							SpaceId:   spaceID,
							OpaqueId:  newParentID,
						},
						Path: filepath.Base(item.Path),
					}
					oldRef := &provider.Reference{
						ResourceId: &provider.ResourceId{
							StorageId: t.options.MountID,
							SpaceId:   spaceID,
							OpaqueId:  parentID,
						},
						Path: filepath.Base(previousPath),
					}
					t.PublishEvent(events.ItemMoved{
						Ref:          ref,
						OldReference: oldRef,
						Timestamp:    utils.TSNow(),
					})
				}
			}
		} else {
			// This item had already been assimilated in the past. Update the path
			t.log.Debug().Str("path", item.Path).Msg("updating cached path")
			if err := t.lookup.CacheID(context.Background(), spaceID, id, item.Path); err != nil {
				t.log.Error().Err(err).Str("spaceID", spaceID).Str("id", id).Str("path", item.Path).Msg("could not cache id")
			}

			_, _, err := t.updateFile(item.Path, id, spaceID, fi)
			if err != nil {
				return err
			}
		}
	} else {
		t.log.Debug().Str("path", item.Path).Msg("new item detected")
		assimilationNode := &assimilationNode{
			spaceID: spaceID,
			// Use the path as the node ID (which is used for calculating the lock file path) since we do not have an ID yet
			nodeId: strings.ReplaceAll(strings.TrimPrefix(item.Path, "/"), "/", "-"),
		}
		unlock, err := t.lookup.MetadataBackend().Lock(assimilationNode)
		if err != nil {
			return err
		}
		defer func() { _ = unlock() }()

		// check if the file got an ID while we were waiting for the lock
		_, id, _, _, err = t.lookup.MetadataBackend().IdentifyPath(context.Background(), item.Path)
		if err != nil {
			return err
		}
		if id != "" {
			// file was assimilated by another thread while we were waiting for the lock
			t.log.Debug().Str("path", item.Path).Msg("file was assimilated by another thread")
			return nil
		}

		// assimilate new file
		newId := uuid.New().String()
		fi, attrs, err := t.updateFile(item.Path, newId, spaceID, nil)
		if err != nil {
			return err
		}

		var parentId *provider.ResourceId
		if len(attrs[prefixes.ParentidAttr]) > 0 {
			parentId = &provider.ResourceId{
				StorageId: t.options.MountID,
				SpaceId:   spaceID,
				OpaqueId:  string(attrs[prefixes.ParentidAttr]),
			}
		}

		ref := &provider.Reference{
			ResourceId: &provider.ResourceId{
				StorageId: t.options.MountID,
				SpaceId:   spaceID,
				OpaqueId:  newId,
			},
		}
		if fi.IsDir() {
			t.PublishEvent(events.ContainerCreated{
				Ref:       ref,
				ParentID:  parentId,
				Timestamp: utils.TSNow(),
			})
		} else {
			if fi.Size() == 0 {
				t.PublishEvent(events.FileTouched{
					Ref:       ref,
					ParentID:  parentId,
					Timestamp: utils.TSNow(),
				})
			} else {
				t.PublishEvent(events.UploadReady{
					FileRef:   ref,
					ParentID:  parentId,
					Timestamp: utils.TSNow(),
				})
			}
		}
	}
	return nil
}

func (t *Tree) updateFile(path, id, spaceID string, fi fs.FileInfo) (fs.FileInfo, node.Attributes, error) {
	retries := 1
	parentID := ""
	bn := assimilationNode{spaceID: spaceID, nodeId: id, path: path}
assimilate:
	if id != spaceID {
		// read parent
		var err error
		_, parentID, err = t.lookup.IDsForPath(context.Background(), filepath.Dir(path))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read parent id")
		}
		parentAttribs, err := t.lookup.MetadataBackend().All(context.Background(), node.NewBaseNode(spaceID, parentID, t.lookup))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read parent item attributes")
		}

		if len(parentAttribs) == 0 || len(parentAttribs[prefixes.IDAttr]) == 0 {
			if retries == 0 {
				return nil, nil, fmt.Errorf("got empty parent attribs even after assimilating")
			}

			// assimilate parent first
			err = t.assimilate(scanItem{Path: filepath.Dir(path)})
			if err != nil {
				return nil, nil, err
			}

			// retry
			retries--
			goto assimilate
		}
		parentID = string(parentAttribs[prefixes.IDAttr])
	}

	// assimilate file
	if fi == nil {
		var err error
		fi, err = os.Stat(path)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to stat item")
		}
	}

	attrs, err := t.lookup.MetadataBackend().All(context.Background(), bn)
	if err != nil && !metadata.IsAttrUnset(err) {
		return nil, nil, errors.Wrap(err, "failed to get item attribs")
	}
	previousAttribs := node.Attributes(attrs)

	attributes := node.Attributes{
		prefixes.IDAttr:   []byte(id),
		prefixes.NameAttr: []byte(filepath.Base(path)),
	}
	if len(parentID) > 0 {
		attributes[prefixes.ParentidAttr] = []byte(parentID)
	}

	var n *node.Node
	if fi.IsDir() {
		attributes.SetInt64(prefixes.TypeAttr, int64(provider.ResourceType_RESOURCE_TYPE_CONTAINER))
		attributes.SetInt64(prefixes.TreesizeAttr, 0)
		if previousAttribs != nil && previousAttribs[prefixes.TreesizeAttr] != nil {
			attributes[prefixes.TreesizeAttr] = previousAttribs[prefixes.TreesizeAttr]
		}
		attributes[prefixes.PropagationAttr] = []byte("1")
		treeSize, err := attributes.Int64(prefixes.TreesizeAttr)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse treesize")
		}
		n = node.New(spaceID, id, parentID, filepath.Base(path), treeSize, "", provider.ResourceType_RESOURCE_TYPE_CONTAINER, nil, t.lookup)
	} else {
		sha1h, md5h, adler32h, err := node.CalculateChecksums(context.Background(), path)
		if err == nil {
			attributes[prefixes.ChecksumPrefix+"sha1"] = sha1h.Sum(nil)
			attributes[prefixes.ChecksumPrefix+"md5"] = md5h.Sum(nil)
			attributes[prefixes.ChecksumPrefix+"adler32"] = adler32h.Sum(nil)
		}

		blobID := uuid.NewString()
		attributes.SetString(prefixes.BlobIDAttr, blobID)
		attributes.SetInt64(prefixes.BlobsizeAttr, fi.Size())
		attributes.SetInt64(prefixes.TypeAttr, int64(provider.ResourceType_RESOURCE_TYPE_FILE))
		n = node.New(spaceID, id, parentID, filepath.Base(path), fi.Size(), blobID, provider.ResourceType_RESOURCE_TYPE_FILE, nil, t.lookup)
	}
	attributes.SetTime(prefixes.MTimeAttr, fi.ModTime())

	n.SpaceRoot = &node.Node{BaseNode: node.BaseNode{SpaceID: spaceID, ID: spaceID}}

	if t.options.EnableFSRevisions {
		go func() {
			// Copy the previous current version to a revision
			currentNode := node.NewBaseNode(n.SpaceID, n.ID+node.CurrentIDDelimiter, t.lookup)
			currentPath := currentNode.InternalPath()
			stat, err := os.Stat(currentPath)
			if err != nil {
				t.log.Error().Err(err).Str("path", path).Str("currentPath", currentPath).Msg("could not stat current path")
				return
			}
			revisionPath := t.lookup.VersionPath(n.SpaceID, n.ID, stat.ModTime().UTC().Format(time.RFC3339Nano))

			err = os.Rename(currentPath, revisionPath)
			if err != nil {
				t.log.Error().Err(err).Str("path", path).Str("revisionPath", revisionPath).Msg("could not create revision")
				return
			}

			// Copy the new version to the current version
			w, err := os.OpenFile(currentPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
			if err != nil {
				t.log.Error().Err(err).Str("path", path).Str("currentPath", currentPath).Msg("could not open current path for writing")
				return
			}
			defer w.Close()
			r, err := os.OpenFile(n.InternalPath(), os.O_RDONLY, 0600)
			if err != nil {
				t.log.Error().Err(err).Str("path", path).Msg("could not open file for reading")
				return
			}
			defer r.Close()

			_, err = io.Copy(w, r)
			if err != nil {
				t.log.Error().Err(err).Str("currentPath", currentPath).Str("path", path).Msg("could not copy new version to current version")
				return
			}

			err = t.lookup.CopyMetadata(context.Background(), n, currentNode, func(attributeName string, value []byte) (newValue []byte, copy bool) {
				return value, strings.HasPrefix(attributeName, prefixes.ChecksumPrefix) ||
					attributeName == prefixes.TypeAttr ||
					attributeName == prefixes.BlobIDAttr ||
					attributeName == prefixes.BlobsizeAttr
			}, false)
			if err != nil {
				t.log.Error().Err(err).Str("currentPath", currentPath).Str("path", path).Msg("failed to copy xattrs to 'current' file")
				return
			}
		}()
	}

	err = t.Propagate(context.Background(), n, 0)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to propagate")
	}

	t.log.Debug().Str("path", path).Interface("attributes", attributes).Msg("setting attributes")
	err = t.lookup.MetadataBackend().SetMultiple(context.Background(), bn, attributes, false)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to set attributes")
	}

	if err := t.lookup.CacheID(context.Background(), spaceID, id, path); err != nil {
		t.log.Error().Err(err).Str("spaceID", spaceID).Str("id", id).Str("path", path).Msg("could not cache id")
	}

	return fi, attributes, nil
}

// WarmupIDCache warms up the id cache
func (t *Tree) WarmupIDCache(root string, assimilate, onlyDirty bool) error {
	root = filepath.Clean(root)
	spaceID := ""

	scopeSpace := func(spaceCandidate string) error {
		if !t.options.UseSpaceGroups {
			return nil
		}

		// set the uid and gid for the space
		fi, err := os.Stat(spaceCandidate)
		if err != nil {
			return err
		}
		sys := fi.Sys().(*syscall.Stat_t)
		gid := int(sys.Gid)
		_, err = t.userMapper.ScopeUserByIds(-1, gid)
		return err
	}

	sizes := make(map[string]int64)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// skip lock and upload files
		if t.isIndex(path) || isTrash(path) || t.isUpload(path) {
			return filepath.SkipDir
		}
		if t.isInternal(path) || isLockFile(path) {
			return nil
		}

		if err != nil {
			return err
		}

		// calculate tree sizes
		if !info.IsDir() {
			dir := path
			for dir != root {
				dir = filepath.Clean(filepath.Dir(dir))
				sizes[dir] += info.Size()
			}
		} else if onlyDirty {
			dirty, err := t.isDirty(path)
			if err != nil {
				return err
			}
			if !dirty {
				return filepath.SkipDir
			}
			sizes[path] += 0 // Make sure to set the size to 0 for empty directories
		}

		nodeSpaceID, id, _, _, err := t.lookup.MetadataBackend().IdentifyPath(context.Background(), path)
		if err == nil && len(id) > 0 {
			if len(nodeSpaceID) > 0 {
				spaceID = nodeSpaceID

				err = scopeSpace(path)
				if err != nil {
					return err
				}
			} else {
				// try to find space
				spaceCandidate := path
				for strings.HasPrefix(spaceCandidate, t.options.Root) {
					spaceID, _, err = t.lookup.IDsForPath(context.Background(), spaceCandidate)
					if err == nil && len(spaceID) > 0 {
						err = scopeSpace(path)
						if err != nil {
							return err
						}
						break
					}

					spaceID, _, _, _, err = t.lookup.MetadataBackend().IdentifyPath(context.Background(), spaceCandidate)
					if err == nil && len(spaceID) > 0 {
						err = scopeSpace(path)
						if err != nil {
							return err
						}
						break
					}
					spaceCandidate = filepath.Dir(spaceCandidate)
				}
			}
			if len(spaceID) == 0 {
				return nil // no space found
			}

			if id != "" {
				// Check if the item on the previous still exists. In this case it might have been a copy with extended attributes -> set new ID
				previousPath, ok := t.lookup.GetCachedID(context.Background(), spaceID, id)
				if ok && previousPath != path {
					// this id clashes with an existing id -> re-assimilate
					_, err := os.Stat(previousPath)
					if err == nil {
						_ = t.assimilate(scanItem{Path: path})
					}
				}
				if err := t.lookup.CacheID(context.Background(), spaceID, id, path); err != nil {
					t.log.Error().Err(err).Str("spaceID", spaceID).Str("id", id).Str("path", path).Msg("could not cache id")
				}
			}
		} else if assimilate {
			if err := t.assimilate(scanItem{Path: path}); err != nil {
				t.log.Error().Err(err).Str("path", path).Msg("could not assimilate item")
			}
		}

		if info.IsDir() {
			return t.setDirty(path, false)
		}
		return nil
	})

	for dir, size := range sizes {
		spaceID, id, err := t.lookup.IDsForPath(context.Background(), dir)
		if err != nil {
			t.log.Error().Err(err).Str("path", dir).Msg("could not get ids for path")
			continue
		}
		n, err := node.ReadNode(context.Background(), t.lookup, spaceID, id, true, nil, false)
		if err != nil {
			t.log.Error().Err(err).Str("path", dir).Msg("could not read directory node")
			continue
		}
		if dir == root {
			// Propagate the size diff further up the tree
			if err := t.propagateSizeDiff(n, size); err != nil {
				t.log.Error().Err(err).Str("path", dir).Msg("could not propagate size diff")
			}
		}
		if err := t.lookup.MetadataBackend().Set(context.Background(), n, prefixes.TreesizeAttr, []byte(fmt.Sprintf("%d", size))); err != nil {
			t.log.Error().Err(err).Str("path", dir).Int64("size", size).Msg("could not set tree size")
		}
	}

	if err != nil {
		return err
	}

	return nil
}

func (t *Tree) propagateSizeDiff(n *node.Node, size int64) error {
	attrs, err := t.lookup.MetadataBackend().All(context.Background(), n)
	if err != nil {
		return err
	}

	oldSize, err := node.Attributes(attrs).Int64(prefixes.TreesizeAttr)
	if err != nil {
		return err
	}
	return t.Propagate(context.Background(), n, size-oldSize)
}

func (t *Tree) setDirty(path string, dirty bool) error {
	return xattr.Set(path, dirtyFlag, []byte(fmt.Sprintf("%t", dirty)))
}

func (t *Tree) isDirty(path string) (bool, error) {
	dirtyAttr, err := xattr.Get(path, dirtyFlag)
	if err != nil {
		if metadata.IsAttrUnset(err) {
			return true, nil
		}
		return false, err
	}
	return string(dirtyAttr) == "true", nil
}
