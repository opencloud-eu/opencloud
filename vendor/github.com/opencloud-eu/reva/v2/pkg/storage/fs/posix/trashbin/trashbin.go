// Copyright 2018-2024 CERN
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

package trashbin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"

	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/lookup"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

var (
	tracer trace.Tracer
)

func init() {
	tracer = otel.Tracer("github.com/cs3org/reva/pkg/storage/fs/posix/trashbin")
}

type Trashbin struct {
	fs  storage.FS
	o   *options.Options
	p   Permissions
	lu  *lookup.Lookup
	log *zerolog.Logger
}

// trashNode is a helper struct to make trash items available for manipulation in the metadata backend
type trashNode struct {
	spaceID string
	id      string
	path    string
}

func (tn *trashNode) GetSpaceID() string {
	return tn.spaceID
}

func (tn *trashNode) GetID() string {
	return tn.id
}

func (tn *trashNode) InternalPath() string {
	return tn.path
}

const (
	trashHeader = `[Trash Info]`
	timeFormat  = "2006-01-02T15:04:05"
)

type Permissions interface {
	AssembleTrashPermissions(ctx context.Context, n *node.Node) (*provider.ResourcePermissions, error)
}

// New returns a new Trashbin
func New(o *options.Options, p Permissions, lu *lookup.Lookup, log *zerolog.Logger) (*Trashbin, error) {
	return &Trashbin{
		o:   o,
		p:   p,
		lu:  lu,
		log: log,
	}, nil
}

func (tb *Trashbin) writeInfoFile(trashPath, id, path string) error {
	c := trashHeader
	c += "\nPath=" + path
	c += "\nDeletionDate=" + time.Now().Format(timeFormat)

	return os.WriteFile(filepath.Join(trashPath, "info", id+".trashinfo"), []byte(c), 0644)
}

func (tb *Trashbin) readInfoFile(trashPath, id string) (string, *typesv1beta1.Timestamp, error) {
	c, err := os.ReadFile(filepath.Join(trashPath, "info", id+".trashinfo"))
	if err != nil {
		return "", nil, err
	}

	var (
		path string
		ts   *typesv1beta1.Timestamp
	)

	for _, line := range strings.Split(string(c), "\n") {
		if strings.HasPrefix(line, "DeletionDate=") {
			t, err := time.ParseInLocation(timeFormat, strings.TrimSpace(strings.TrimPrefix(line, "DeletionDate=")), time.Local)
			if err != nil {
				return "", nil, err
			}
			ts = utils.TimeToTS(t)
		}
		if strings.HasPrefix(line, "Path=") {
			path = strings.TrimPrefix(line, "Path=")
		}
	}

	return path, ts, nil
}

// Setup the trashbin
func (tb *Trashbin) Setup(fs storage.FS) error {
	if tb.fs != nil {
		return nil
	}

	tb.fs = fs
	return nil
}

func trashRootForNode(n *node.Node) string {
	return filepath.Join(n.SpaceRoot.InternalPath(), ".Trash")
}

func (tb *Trashbin) MoveToTrash(ctx context.Context, n *node.Node, path string) error {
	key := n.ID
	trashPath := trashRootForNode(n)

	err := os.MkdirAll(filepath.Join(trashPath, "info"), 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(trashPath, "files"), 0755)
	if err != nil {
		return err
	}

	relPath := strings.TrimPrefix(path, n.SpaceRoot.InternalPath())
	relPath = strings.TrimPrefix(relPath, "/")
	err = tb.writeInfoFile(trashPath, key, relPath)
	if err != nil {
		return err
	}

	// 1. "Forget" the node
	if err = tb.lu.IDCache.DeleteByPath(ctx, path); err != nil {
		return err
	}

	// 2. Move the node to the trash
	itemTrashPath := filepath.Join(trashPath, "files", key+".trashitem")
	err = os.Rename(path, itemTrashPath)
	if err != nil {
		return err
	}

	// 3. Purge the node from the metadata backend. This will not delete the xattrs from the
	// node as it has already been moved but still remove it from the file metadata cache so
	// that the metadata is no longer available when reading the node.
	return tb.lu.MetadataBackend().Purge(ctx, n)
}

// ListRecycle returns the list of available recycle items
// ref -> the space (= resourceid), key -> deleted node id, relativePath = relative to key
func (tb *Trashbin) ListRecycle(ctx context.Context, spaceID string, key, relativePath string) ([]*provider.RecycleItem, error) {
	_, span := tracer.Start(ctx, "ListRecycle")
	defer span.End()

	trashRoot := filepath.Join(tb.lu.InternalPath(spaceID, spaceID), ".Trash")
	base := filepath.Join(trashRoot, "files")

	var originalPath string
	var ts *typesv1beta1.Timestamp
	if key != "" && relativePath == "" {
		// this is listing a specific item/folder
		base = filepath.Join(base, key+".trashitem")
		var err error
		originalPath, ts, err = tb.readInfoFile(trashRoot, key)
		if err != nil {
			return nil, err
		}

		fi, err := os.Stat(base)
		if err != nil {
			return nil, err
		}
		item := &provider.RecycleItem{
			Key:  key,
			Size: uint64(fi.Size()),
			Ref: &provider.Reference{
				ResourceId: &provider.ResourceId{
					SpaceId:  spaceID,
					OpaqueId: spaceID,
				},
				Path: originalPath,
			},
			DeletionTime: ts,
			Type:         provider.ResourceType_RESOURCE_TYPE_FILE,
		}
		if fi.IsDir() {
			item.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER
		} else {
			item.Type = provider.ResourceType_RESOURCE_TYPE_FILE
		}
		return []*provider.RecycleItem{item}, nil
	} else if key != "" {
		// this is listing a specific item/folder
		base = filepath.Join(base, key+".trashitem", relativePath)
		var err error
		originalPath, ts, err = tb.readInfoFile(trashRoot, key)
		if err != nil {
			return nil, err
		}
		originalPath = filepath.Join(originalPath, relativePath)
	}

	items := []*provider.RecycleItem{}
	entries, err := os.ReadDir(filepath.Clean(base))
	if err != nil {
		switch err.(type) {
		case *os.PathError:
			return items, nil
		default:
			return nil, err
		}
	}

	for _, entry := range entries {
		var fi os.FileInfo
		var entryOriginalPath string
		var entryKey string
		if strings.HasSuffix(entry.Name(), ".trashitem") {
			entryKey = strings.TrimSuffix(entry.Name(), ".trashitem")
			entryOriginalPath, ts, err = tb.readInfoFile(trashRoot, entryKey)
			if err != nil {
				continue
			}

			fi, err = entry.Info()
			if err != nil {
				continue
			}
		} else {
			fi, err = os.Stat(filepath.Join(base, entry.Name()))
			entryKey = entry.Name()
			entryOriginalPath = filepath.Join(originalPath, entry.Name())
			if err != nil {
				continue
			}
		}

		item := &provider.RecycleItem{
			Key:  filepath.Join(key, relativePath, entryKey),
			Size: uint64(fi.Size()),
			Ref: &provider.Reference{
				ResourceId: &provider.ResourceId{
					SpaceId:  spaceID,
					OpaqueId: spaceID,
				},
				Path: entryOriginalPath,
			},
			DeletionTime: ts,
		}
		if entry.IsDir() {
			item.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER
		} else {
			item.Type = provider.ResourceType_RESOURCE_TYPE_FILE
		}

		items = append(items, item)
	}

	return items, nil
}

// RestoreRecycleItem restores the specified item
func (tb *Trashbin) RestoreRecycleItem(ctx context.Context, spaceID string, key, relativePath string, restoreRef *provider.Reference) (*node.Node, error) {
	_, span := tracer.Start(ctx, "RestoreRecycleItem")
	defer span.End()

	trashRoot := filepath.Join(tb.lu.InternalPath(spaceID, spaceID), ".Trash")
	trashPath := filepath.Clean(filepath.Join(trashRoot, "files", key+".trashitem", relativePath))

	restorePath := ""
	// TODO why can we not use NodeFromResource here? It will use walk path. Do trashed items have a problem with that?
	if restoreRef != nil {
		restoreBaseNode, err := tb.lu.NodeFromID(ctx, restoreRef.GetResourceId())
		if err != nil {
			return nil, err
		}
		restorePath = filepath.Join(restoreBaseNode.InternalPath(), restoreRef.GetPath())
	} else {
		originalPath, _, err := tb.readInfoFile(trashRoot, key)
		if err != nil {
			return nil, err
		}
		restorePath = filepath.Join(tb.lu.InternalPath(spaceID, spaceID), originalPath, relativePath)
	}
	// TODO the decomposed trash also checks the permissions on the restore node

	_, id, _, _, err := tb.lu.MetadataBackend().IdentifyPath(ctx, trashPath)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errtypes.NotFound("trashbin: item not found")

	}

	// update parent id in case it was restored to a different location
	_, parentID, _, _, err := tb.lu.MetadataBackend().IdentifyPath(ctx, filepath.Dir(restorePath))
	if err != nil {
		return nil, err
	}
	if len(parentID) == 0 {
		return nil, fmt.Errorf("trashbin: parent id not found for %s", restorePath)
	}

	trashedNode := &trashNode{spaceID: spaceID, id: id, path: trashPath}
	if err = tb.lu.MetadataBackend().SetMultiple(ctx, trashedNode, map[string][]byte{
		prefixes.NameAttr:     []byte(filepath.Base(restorePath)),
		prefixes.ParentidAttr: []byte(parentID),
	}, true); err != nil {
		return nil, fmt.Errorf("posixfs: failed to update trashed node metadata: %w", err)
	}

	// restore the item
	err = os.Rename(trashPath, restorePath)
	if err != nil {
		return nil, err
	}
	if err := tb.lu.CacheID(ctx, spaceID, id, restorePath); err != nil {
		tb.log.Error().Err(err).Str("spaceID", spaceID).Str("id", id).Str("path", restorePath).Msg("trashbin: error caching id")
	}

	restoredNode, err := tb.lu.NodeFromID(ctx, &provider.ResourceId{SpaceId: spaceID, OpaqueId: id})
	if err != nil {
		return nil, err
	}

	// cleanup trash info
	if relativePath == "" || relativePath == "." || relativePath == "/" {
		return restoredNode, os.Remove(filepath.Join(trashRoot, "info", key+".trashinfo"))
	} else {
		return restoredNode, nil
	}

}

// PurgeRecycleItem purges the specified item, all its children and all their revisions
func (tb *Trashbin) PurgeRecycleItem(ctx context.Context, spaceID, key, relativePath string) error {
	_, span := tracer.Start(ctx, "PurgeRecycleItem")
	defer span.End()

	trashRoot := filepath.Join(tb.lu.InternalPath(spaceID, spaceID), ".Trash")
	err := os.RemoveAll(filepath.Clean(filepath.Join(trashRoot, "files", key+".trashitem", relativePath)))
	if err != nil {
		return err
	}

	cleanPath := filepath.Clean(relativePath)
	if cleanPath == "." || cleanPath == "/" {
		return os.Remove(filepath.Join(trashRoot, "info", key+".trashinfo"))
	}
	return nil
}

// EmptyRecycle empties the trash
func (tb *Trashbin) EmptyRecycle(ctx context.Context, spaceID string) error {
	_, span := tracer.Start(ctx, "EmptyRecycle")
	defer span.End()

	trashRoot := filepath.Join(tb.lu.InternalPath(spaceID, spaceID), ".Trash")
	err := os.RemoveAll(filepath.Clean(filepath.Join(trashRoot, "files")))
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Clean(filepath.Join(trashRoot, "info")))
}

func (tb *Trashbin) IsEmpty(ctx context.Context, spaceID string) bool {
	_, span := tracer.Start(ctx, "HasTrashedItems")
	defer span.End()
	trashRoot := filepath.Join(tb.lu.InternalPath(spaceID, spaceID), ".Trash", "info")
	trash, err := os.Open(filepath.Clean(trashRoot))
	if err != nil {
		// there is no trash for this space, so no trashed items
		return true
	}
	dirItems, err := trash.ReadDir(1)
	if err != nil {
		// if we cannot read the trash, we assume there are no trashed items
		tb.log.Error().Err(err).Str("spaceID", spaceID).Msg("trashbin: error reading trash directory")
		return true
	}
	if len(dirItems) > 0 {
		// if we can read the trash and there are items, we assume there are trashed items
		return false
	}
	// if we cannot read the trash, we assume there are no trashed items
	return true
}
