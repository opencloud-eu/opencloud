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

package lookup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/options"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/lockedfile"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

const (
	_spaceTypePersonal = "personal"
)

func init() {
	tracer = otel.Tracer("github.com/cs3org/reva/pkg/storage/utils/decomposedfs/lookup")
}

// Lookup implements transformations from filepath to node and back
type Lookup struct {
	Options *options.Options

	metadataBackend metadata.Backend
	tm              node.TimeManager
}

// New returns a new Lookup instance
func New(b metadata.Backend, o *options.Options, tm node.TimeManager) *Lookup {
	return &Lookup{
		Options:         o,
		metadataBackend: b,
		tm:              tm,
	}
}

// MetadataBackend returns the metadata backend
func (lu *Lookup) MetadataBackend() metadata.Backend {
	return lu.metadataBackend
}

func (lu *Lookup) ReadBlobIDAndSizeAttr(ctx context.Context, n metadata.MetadataNode, attrs node.Attributes) (string, int64, error) {
	blobID := ""
	blobSize := int64(0)
	var err error

	if attrs != nil {
		blobID = attrs.String(prefixes.BlobIDAttr)
		if blobID != "" {
			blobSize, err = attrs.Int64(prefixes.BlobsizeAttr)
			if err != nil {
				return "", 0, err
			}
		}
	} else {
		attrs, err := lu.metadataBackend.All(ctx, n)
		if err != nil {
			return "", 0, errors.Wrapf(err, "error reading blobid xattr")
		}
		nodeAttrs := node.Attributes(attrs)
		blobID = nodeAttrs.String(prefixes.BlobIDAttr)
		blobSize, err = nodeAttrs.Int64(prefixes.BlobsizeAttr)
		if err != nil {
			return "", 0, errors.Wrapf(err, "error reading blobsize xattr")
		}
	}
	return blobID, blobSize, nil
}

func (lu *Lookup) NodeIDFromParentAndName(ctx context.Context, parent *node.Node, name string) (string, error) {
	nodeID, err := node.ReadChildNodeFromLink(ctx, filepath.Join(parent.InternalPath(), name))
	if err != nil {
		return "", errors.Wrap(err, "decomposedfs: Wrap: readlink error")
	}
	return nodeID, nil
}

// NodeFromResource takes in a request path or request id and converts it to a Node
func (lu *Lookup) NodeFromResource(ctx context.Context, ref *provider.Reference) (*node.Node, error) {
	ctx, span := tracer.Start(ctx, "NodeFromResource")
	defer span.End()

	if ref.ResourceId != nil {
		// check if a storage space reference is used
		// currently, the decomposed fs uses the root node id as the space id
		n, err := lu.NodeFromID(ctx, ref.ResourceId)
		if err != nil {
			return nil, err
		}
		// is this a relative reference?
		if ref.Path != "" {
			p := filepath.Clean(ref.Path)
			if p != "." && p != "/" {
				// walk the relative path
				n, err = lu.WalkPath(ctx, n, p, false, func(ctx context.Context, n *node.Node) error { return nil })
				if err != nil {
					return nil, err
				}
				n.SpaceID = ref.ResourceId.SpaceId
			}
		}
		return n, nil
	}

	// reference is invalid
	return nil, fmt.Errorf("invalid reference %+v. resource_id must be set", ref)
}

// NodeFromID returns the internal path for the id
func (lu *Lookup) NodeFromID(ctx context.Context, id *provider.ResourceId) (n *node.Node, err error) {
	ctx, span := tracer.Start(ctx, "NodeFromID")
	defer span.End()
	if id == nil {
		return nil, fmt.Errorf("invalid resource id %+v", id)
	}
	if id.OpaqueId == "" {
		// The Resource references the root of a space
		return lu.NodeFromSpaceID(ctx, id.SpaceId)
	}
	return node.ReadNode(ctx, lu, id.SpaceId, id.OpaqueId, "", false, nil, false)
}

// Pathify segments the beginning of a string into depth segments of width length
// Pathify("aabbccdd", 3, 1) will return "a/a/b/bccdd"
func Pathify(id string, depth, width int) string {
	b := strings.Builder{}
	i := 0
	for ; i < depth; i++ {
		if len(id) <= i*width+width {
			break
		}
		b.WriteString(id[i*width : i*width+width])
		b.WriteRune(filepath.Separator)
	}
	b.WriteString(id[i*width:])
	return b.String()
}

// NodeFromSpaceID converts a resource id into a Node
func (lu *Lookup) NodeFromSpaceID(ctx context.Context, spaceID string) (n *node.Node, err error) {
	node, err := node.ReadNode(ctx, lu, spaceID, spaceID, "", false, nil, false)
	if err != nil {
		return nil, err
	}

	node.SpaceRoot = node
	return node, nil
}

// GenerateSpaceID generates a new space id and alias
func (lu *Lookup) GenerateSpaceID(spaceType string, owner *user.User) (string, error) {
	switch spaceType {
	case _spaceTypePersonal:
		return owner.Id.OpaqueId, nil
	default:
		return uuid.New().String(), nil
	}
}

// Path returns the path for node
func (lu *Lookup) Path(ctx context.Context, n *node.Node, hasPermission node.PermissionFunc) (p string, err error) {
	root := n.SpaceRoot
	var child *node.Node
	for n.ID != root.ID {
		p = filepath.Join(n.Name, p)
		child = n
		if n, err = n.Parent(ctx); err != nil {
			appctx.GetLogger(ctx).
				Error().Err(err).
				Str("path", p).
				Str("spaceid", child.SpaceID).
				Str("nodeid", child.ID).
				Str("parentid", child.ParentID).
				Msg("Path()")
			return
		}

		if !hasPermission(n) {
			break
		}
	}
	p = filepath.Join("/", p)
	return
}

// WalkPath calls n.Child(segment) on every path segment in p starting at the node r.
// If a function f is given it will be executed for every segment node, but not the root node r.
// If followReferences is given the current visited reference node is replaced by the referenced node.
func (lu *Lookup) WalkPath(ctx context.Context, r *node.Node, p string, followReferences bool, f func(ctx context.Context, n *node.Node) error) (*node.Node, error) {
	segments := strings.Split(strings.Trim(p, "/"), "/")
	var err error
	for i := range segments {
		if r, err = r.Child(ctx, segments[i]); err != nil {
			return r, err
		}

		if followReferences {
			if attrBytes, err := r.Xattr(ctx, prefixes.ReferenceAttr); err == nil {
				realNodeID := attrBytes
				ref, err := refFromCS3(realNodeID)
				if err != nil {
					return nil, err
				}

				r, err = lu.NodeFromID(ctx, ref.ResourceId)
				if err != nil {
					return nil, err
				}
			}
		}
		if r.IsSpaceRoot(ctx) {
			r.SpaceRoot = r
		}

		if !r.Exists && i < len(segments)-1 {
			return r, errtypes.NotFound(segments[i])
		}
		if f != nil {
			if err = f(ctx, r); err != nil {
				return r, err
			}
		}
	}
	return r, nil
}

// InternalRoot returns the internal storage root directory
func (lu *Lookup) InternalRoot() string {
	return lu.Options.Root
}

// InternalSpaceRoot returns the internal path for a space
func (lu *Lookup) InternalSpaceRoot(spaceID string) string {
	return filepath.Join(lu.Options.Root, "spaces", Pathify(spaceID, 1, 2))
}

// InternalPath returns the internal path for a given ID
func (lu *Lookup) InternalPath(spaceID, nodeID string) string {
	return filepath.Join(lu.Options.Root, "spaces", Pathify(spaceID, 1, 2), "nodes", Pathify(nodeID, 4, 2))
}

// LockfilePaths returns the paths(s) to the lockfile of the node
func (lu *Lookup) LockfilePaths(n *node.Node) []string {
	return []string{n.InternalPath() + ".lock"}
}

// VersionPath returns the internal path for a version of a node
// Deprecated: use InternalPath instead
func (lu *Lookup) VersionPath(spaceID, nodeID, version string) string {
	return lu.InternalPath(spaceID, nodeID) + node.RevisionIDDelimiter + version
}

// // ReferenceFromAttr returns a CS3 reference from xattr of a node.
// // Supported formats are: "cs3:storageid/nodeid"
// func ReferenceFromAttr(b []byte) (*provider.Reference, error) {
// 	return refFromCS3(b)
// }

// refFromCS3 creates a CS3 reference from a set of bytes. This method should remain private
// and only be called after validation because it can potentially panic.
func refFromCS3(b []byte) (*provider.Reference, error) {
	parts := string(b[4:])
	return &provider.Reference{
		ResourceId: &provider.ResourceId{
			StorageId: strings.Split(parts, "/")[0],
			OpaqueId:  strings.Split(parts, "/")[1],
		},
	}, nil
}

// CopyMetadata copies all extended attributes from source to target.
// The optional filter function can be used to filter by attribute name, e.g. by checking a prefix
// For the source file, a shared lock is acquired.
// NOTE: target resource will be write locked!
func (lu *Lookup) CopyMetadata(ctx context.Context, sourceNode, targetNode metadata.MetadataNode, filter func(attributeName string, value []byte) (newValue []byte, copy bool), acquireTargetLock bool) (err error) {
	// Acquire a read log on the source node
	// write lock existing node before reading treesize or tree time
	lock, err := lockedfile.OpenFile(lu.MetadataBackend().LockfilePath(sourceNode), os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	if err != nil {
		return errors.Wrap(err, "xattrs: Unable to lock source to read")
	}
	defer func() {
		rerr := lock.Close()

		// if err is non nil we do not overwrite that
		if err == nil {
			err = rerr
		}
	}()

	return lu.CopyMetadataWithSourceLock(ctx, sourceNode, targetNode, filter, lock, acquireTargetLock)
}

// CopyMetadataWithSourceLock copies all extended attributes from source to target.
// The optional filter function can be used to filter by attribute name, e.g. by checking a prefix
// For the source file, a matching lockedfile is required.
// NOTE: target resource will be write locked!
func (lu *Lookup) CopyMetadataWithSourceLock(ctx context.Context, sourceNode, targetNode metadata.MetadataNode, filter func(attributeName string, value []byte) (newValue []byte, copy bool), lockedSource *lockedfile.File, acquireTargetLock bool) (err error) {
	switch {
	case lockedSource == nil:
		return errors.New("no lock provided")
	case lockedSource.File.Name() != lu.MetadataBackend().LockfilePath(sourceNode):
		return errors.New("lockpath does not match filepath")
	}

	attrs, err := lu.metadataBackend.All(ctx, sourceNode)
	if err != nil {
		return err
	}

	newAttrs := make(map[string][]byte, 0)
	for attrName, val := range attrs {
		if filter != nil {
			var ok bool
			if val, ok = filter(attrName, val); !ok {
				continue
			}
		}
		newAttrs[attrName] = val
	}

	return lu.MetadataBackend().SetMultiple(ctx, targetNode, newAttrs, acquireTargetLock)
}

func (lu *Lookup) PurgeNode(n *node.Node) error {
	// remove node
	if err := utils.RemoveItem(n.InternalPath()); err != nil {
		return err
	}

	// remove child entry in parent
	src := filepath.Join(n.ParentPath(), n.Name)
	return os.Remove(src)
}

// TimeManager returns the time manager
func (lu *Lookup) TimeManager() node.TimeManager {
	return lu.tm
}

// DetectBackendOnDisk returns the name of the metadata backend being used on disk
func DetectBackendOnDisk(root string) string {
	matches, _ := filepath.Glob(filepath.Join(root, "spaces", "*", "*"))
	if len(matches) > 0 {
		base := matches[len(matches)-1]
		spaceid := strings.ReplaceAll(
			strings.TrimPrefix(base, filepath.Join(root, "spaces")),
			"/", "")
		spaceRoot := Pathify(spaceid, 4, 2)
		_, err := os.Stat(filepath.Join(base, "nodes", spaceRoot+".mpk"))
		if err == nil {
			return "mpk"
		}
		_, err = os.Stat(filepath.Join(base, "nodes", spaceRoot+".ini"))
		if err == nil {
			return "ini"
		}
	}
	return "xattrs"
}
