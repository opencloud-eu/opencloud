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

package lookup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/usermapper"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/lockedfile"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

const MetadataDir = ".oc-nodes"

var _spaceTypePersonal = "personal"
var _spaceTypeProject = "project"

func init() {
	tracer = otel.Tracer("github.com/cs3org/reva/pkg/storage/pkg/decomposedfs/lookup")
}

// IDCache is a cache for node ids
type IDCache interface {
	Get(ctx context.Context, spaceID, nodeID string) (string, bool)
	GetByPath(ctx context.Context, path string) (string, string, bool)

	Set(ctx context.Context, spaceID, nodeID, val string) error

	Delete(ctx context.Context, spaceID, nodeID string) error
	DeleteByPath(ctx context.Context, path string) error

	DeletePath(ctx context.Context, path string) error
}

// Lookup implements transformations from filepath to node and back
type Lookup struct {
	Options *options.Options

	IDCache         IDCache
	IDHistoryCache  IDCache
	metadataBackend metadata.Backend
	userMapper      usermapper.Mapper
	tm              node.TimeManager
}

// New returns a new Lookup instance
func New(b metadata.Backend, um usermapper.Mapper, o *options.Options, tm node.TimeManager) *Lookup {
	idHistoryConf := o.Options.IDCache
	idHistoryConf.Database = o.Options.IDCache.Table + "_history"
	idHistoryConf.TTL = 1 * time.Minute

	lu := &Lookup{
		Options:         o,
		metadataBackend: b,
		IDCache:         NewStoreIDCache(o.Options.IDCache),
		IDHistoryCache:  NewStoreIDCache(idHistoryConf),
		userMapper:      um,
		tm:              tm,
	}

	return lu
}

// CacheID caches the path for the given space and node id
func (lu *Lookup) CacheID(ctx context.Context, spaceID, nodeID, val string) error {
	return lu.IDCache.Set(ctx, spaceID, nodeID, val)
}

// GetCachedID returns the cached path for the given space and node id
func (lu *Lookup) GetCachedID(ctx context.Context, spaceID, nodeID string) (string, bool) {
	return lu.IDCache.Get(ctx, spaceID, nodeID)
}

func (lu *Lookup) IDsForPath(ctx context.Context, path string) (string, string, error) {
	// IDsForPath returns the space and opaque id for the given path
	spaceID, nodeID, ok := lu.IDCache.GetByPath(ctx, path)
	if !ok {
		return "", "", errtypes.NotFound("path not found in cache:" + path)
	}
	return spaceID, nodeID, nil
}

// NodeFromPath returns the node for the given path
func (lu *Lookup) NodeIDFromParentAndName(ctx context.Context, parent *node.Node, name string) (string, error) {
	parentPath, ok := lu.GetCachedID(ctx, parent.SpaceID, parent.ID)
	if !ok {
		return "", errtypes.NotFound(parent.ID)
	}

	childPath := filepath.Join(parentPath, name)
	_, childID, err := lu.IDsForPath(ctx, childPath)
	if err != nil {
		return "", err
	}
	return childID, nil
}

// MetadataBackend returns the metadata backend
func (lu *Lookup) MetadataBackend() metadata.Backend {
	return lu.metadataBackend
}

func (lu *Lookup) ReadBlobIDAndSizeAttr(ctx context.Context, n metadata.MetadataNode, _ node.Attributes) (string, int64, error) {
	fi, err := os.Stat(n.InternalPath())
	if err != nil {
		return "", 0, errors.Wrap(err, "error stating file")
	}
	return "", fi.Size(), nil
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
	return node.ReadNode(ctx, lu, id.SpaceId, id.OpaqueId, false, nil, false)
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
	node, err := node.ReadNode(ctx, lu, spaceID, spaceID, false, nil, false)
	if err != nil {
		return nil, err
	}

	node.SpaceRoot = node
	return node, nil
}

// Path returns the path for node
func (lu *Lookup) Path(ctx context.Context, n *node.Node, hasPermission node.PermissionFunc) (p string, err error) {
	root := n.SpaceRoot
	for n.ID != root.ID {
		p = filepath.Join(n.Name, p)
		if n, err = n.Parent(ctx); err != nil {
			appctx.GetLogger(ctx).
				Error().Err(err).
				Str("path", p).
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
	return lu.InternalPath(spaceID, spaceID)
}

// InternalPath returns the internal path for a given ID
func (lu *Lookup) InternalPath(spaceID, nodeID string) string {
	if strings.Contains(nodeID, node.RevisionIDDelimiter) || strings.HasSuffix(nodeID, node.CurrentIDDelimiter) {
		spaceRoot, _ := lu.IDCache.Get(context.Background(), spaceID, spaceID)
		if len(spaceRoot) == 0 {
			return ""
		}
		return filepath.Join(spaceRoot, MetadataDir, Pathify(nodeID, 4, 2))
	}

	path, _ := lu.IDCache.Get(context.Background(), spaceID, nodeID)

	return path
}

// LockfilePaths returns the paths(s) to the lockfile of the node
func (lu *Lookup) LockfilePaths(spaceID, nodeID string) []string {
	spaceRoot, _ := lu.IDCache.Get(context.Background(), spaceID, spaceID)
	if len(spaceRoot) == 0 {
		return nil
	}
	paths := []string{filepath.Join(spaceRoot, MetadataDir, Pathify(nodeID, 4, 2)+".lock")}

	nodepath := lu.InternalPath(spaceID, nodeID)
	if len(nodepath) > 0 {
		paths = append(paths, nodepath+".lock")
	}

	return paths
}

// VersionPath returns the path to the version of the node
func (lu *Lookup) VersionPath(spaceID, nodeID, version string) string {
	spaceRoot, _ := lu.IDCache.Get(context.Background(), spaceID, spaceID)
	if len(spaceRoot) == 0 {
		return ""
	}

	return filepath.Join(spaceRoot, MetadataDir, Pathify(nodeID, 4, 2)+node.RevisionIDDelimiter+version)
}

// VersionPath returns the "current" path of the node
func (lu *Lookup) CurrentPath(spaceID, nodeID string) string {
	spaceRoot, _ := lu.IDCache.Get(context.Background(), spaceID, spaceID)
	if len(spaceRoot) == 0 {
		return ""
	}

	return filepath.Join(spaceRoot, MetadataDir, Pathify(nodeID, 4, 2)+node.CurrentIDDelimiter)
}

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
func (lu *Lookup) CopyMetadata(ctx context.Context, src, target metadata.MetadataNode, filter func(attributeName string, value []byte) (newValue []byte, copy bool), acquireTargetLock bool) (err error) {
	// Acquire a read log on the source node
	// write lock existing node before reading treesize or tree time
	lock, err := lockedfile.OpenFile(lu.MetadataBackend().LockfilePath(src), os.O_RDONLY|os.O_CREATE, 0600)
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

	return lu.CopyMetadataWithSourceLock(ctx, src, target, filter, lock, acquireTargetLock)
}

// CopyMetadataWithSourceLock copies all extended attributes from source to target.
// The optional filter function can be used to filter by attribute name, e.g. by checking a prefix
// For the source file, a matching lockedfile is required.
// NOTE: target resource will be write locked!
func (lu *Lookup) CopyMetadataWithSourceLock(ctx context.Context, src, target metadata.MetadataNode, filter func(attributeName string, value []byte) (newValue []byte, copy bool), lockedSource *lockedfile.File, acquireTargetLock bool) (err error) {
	switch {
	case lockedSource == nil:
		return errors.New("no lock provided")
	case lockedSource.File.Name() != lu.MetadataBackend().LockfilePath(src):
		return errors.New("lockpath does not match filepath")
	}

	attrs, err := lu.metadataBackend.All(ctx, src)
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

	return lu.MetadataBackend().SetMultiple(ctx, target, newAttrs, acquireTargetLock)
}

// GenerateSpaceID generates a space id for the given space type and owner
func (lu *Lookup) GenerateSpaceID(spaceType string, owner *user.User) (string, error) {
	switch spaceType {
	case _spaceTypeProject:
		return uuid.New().String(), nil
	case _spaceTypePersonal:
		relPath := templates.WithUser(owner, lu.Options.PersonalSpacePathTemplate)
		path := filepath.Join(lu.Options.Root, relPath)

		// do we already know about this space?
		spaceID, _, err := lu.IDsForPath(context.TODO(), path)
		if err != nil {
			// check if the space exists on disk incl. attributes
			spaceID, _, _, _, err := lu.metadataBackend.IdentifyPath(context.TODO(), path)
			if err != nil {
				if metadata.IsNotExist(err) || metadata.IsAttrUnset(err) {
					return uuid.New().String(), nil
				} else {
					return "", err
				}
			}
			if len(spaceID) == 0 {
				return "", errtypes.InternalError("encountered empty space id on disk")
			}
			return spaceID, nil
		}
		return spaceID, nil
	default:
		return "", fmt.Errorf("unsupported space type: %s", spaceType)
	}
}

func (lu *Lookup) PurgeNode(n *node.Node) error {
	rerr := os.RemoveAll(n.InternalPath())
	if cerr := lu.IDCache.Delete(context.Background(), n.SpaceID, n.ID); cerr != nil {
		return cerr
	}
	return rerr
}

// TimeManager returns the time manager
func (lu *Lookup) TimeManager() node.TimeManager {
	return lu.tm
}
