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

package node

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"hash/adler32"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v2/internal/grpc/services/storageprovider"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/mime"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/ace"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/grants"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/lockedfile"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/cs3org/reva/pkg/storage/utils/decomposedfs/node")
}

// Define keys and values used in the node metadata
const (
	LockdiscoveryKey = "lockdiscovery"
	FavoriteKey      = "http://owncloud.org/ns/favorite"
	ShareTypesKey    = "http://owncloud.org/ns/share-types"
	ChecksumsKey     = "http://owncloud.org/ns/checksums"
	UserShareType    = "0"
	QuotaKey         = "quota"

	QuotaUnlimited    = "0"
	QuotaUncalculated = "-1"
	QuotaUnknown      = "-2"

	// TrashIDDelimiter represents the characters used to separate the nodeid and the deletion time.
	TrashIDDelimiter    = ".T."
	RevisionIDDelimiter = ".REV."
	CurrentIDDelimiter  = ".CURRENT"

	// RootID defines the root node's ID
	RootID = "root"

	// ProcessingStatus is the name of the status when processing a file
	ProcessingStatus = "processing:"
)

// Blobstore defines an interface for storing blobs in a blobstore
type Blobstore interface {
	Upload(node *Node, source, copyTarget string) error
	Download(node *Node) (io.ReadCloser, error)
	Delete(node *Node) error
}

type TimeManager interface {
	// OverrideMTime overrides the mtime of the node, either on the node itself or in the given attributes, depending on the implementation
	OverrideMtime(ctx context.Context, n *Node, attrs *Attributes, mtime time.Time) error

	// MTime returns the mtime of the node
	MTime(ctx context.Context, n *Node) (time.Time, error)
	// SetMTime sets the mtime of the node
	SetMTime(ctx context.Context, n *Node, mtime *time.Time) error

	// TMTime returns the tmtime of the node
	TMTime(ctx context.Context, n *Node) (time.Time, error)
	// SetTMTime sets the tmtime of the node
	SetTMTime(ctx context.Context, n *Node, tmtime *time.Time) error

	// CTime returns the ctime of the node
	CTime(ctx context.Context, n *Node) (time.Time, error)

	// DTime returns the deletion time of the node
	DTime(ctx context.Context, n *Node) (time.Time, error)
	// SetDTime sets the deletion time of the node
	SetDTime(ctx context.Context, n *Node, mtime *time.Time) error
}

// Tree is used to manage a tree hierarchy
type Tree interface {
	Setup() error

	GetMD(ctx context.Context, node *Node) (os.FileInfo, error)
	ListFolder(ctx context.Context, node *Node) ([]*Node, error)
	// CreateHome(owner *userpb.UserId) (n *Node, err error)
	CreateDir(ctx context.Context, node *Node) (err error)
	TouchFile(ctx context.Context, node *Node, markprocessing bool, mtime string) error
	// CreateReference(ctx context.Context, node *Node, targetURI *url.URL) error
	Move(ctx context.Context, oldNode *Node, newNode *Node) (err error)
	Delete(ctx context.Context, node *Node) (err error)

	InitNewNode(ctx context.Context, n *Node, fsize uint64) (metadata.UnlockFunc, error)
	RestoreRevision(ctx context.Context, source, target metadata.MetadataNode, mtime time.Time) (err error)

	WriteBlob(node *Node, source string) error
	ReadBlob(node *Node) (io.ReadCloser, error)
	DeleteBlob(node *Node) error

	BuildSpaceIDIndexEntry(spaceID string) string
	ResolveSpaceIDIndexEntry(spaceID string) (string, error)

	CreateRevision(ctx context.Context, n *Node, version string, f *lockedfile.File) (string, error)
	ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error)
	DownloadRevision(ctx context.Context, ref *provider.Reference, revisionKey string, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error)

	Propagate(ctx context.Context, node *Node, sizeDiff int64) (err error)
}

// PathLookup defines the interface for the lookup component
type PathLookup interface {
	NodeFromSpaceID(ctx context.Context, spaceID string) (n *Node, err error)
	NodeFromResource(ctx context.Context, ref *provider.Reference) (*Node, error)
	NodeFromID(ctx context.Context, id *provider.ResourceId) (n *Node, err error)

	NodeIDFromParentAndName(ctx context.Context, n *Node, name string) (string, error)

	GenerateSpaceID(spaceType string, owner *userpb.User) (string, error)

	InternalRoot() string
	InternalSpaceRoot(spaceID string) string
	InternalPath(spaceID, nodeID string) string
	LockfilePaths(spaceID, nodeID string) []string
	VersionPath(spaceID, nodeID, version string) string
	Path(ctx context.Context, n *Node, hasPermission PermissionFunc) (path string, err error)
	MetadataBackend() metadata.Backend
	TimeManager() TimeManager
	ReadBlobIDAndSizeAttr(ctx context.Context, n metadata.MetadataNode, attrs Attributes) (string, int64, error)
	CopyMetadataWithSourceLock(ctx context.Context, sourceNode, targetNode metadata.MetadataNode, filter func(attributeName string, value []byte) (newValue []byte, copy bool), lockedSource *lockedfile.File, acquireTargetLock bool) (err error)
	CopyMetadata(ctx context.Context, sourceNode, targetNode metadata.MetadataNode, filter func(attributeName string, value []byte) (newValue []byte, copy bool), acquireTargetLock bool) (err error)

	PurgeNode(n *Node) error
}

type IDCacher interface {
	CacheID(ctx context.Context, spaceID, nodeID, val string) error
	GetCachedID(ctx context.Context, spaceID, nodeID string) (string, bool)
}

type BaseNode struct {
	SpaceID string
	ID      string

	lu             PathLookup
	internalPathID string
	internalPath   string
}

func NewBaseNode(spaceID, nodeID string, lu PathLookup) *BaseNode {
	return &BaseNode{
		SpaceID: spaceID,
		ID:      nodeID,
		lu:      lu,
	}
}

func (n *BaseNode) GetSpaceID() string { return n.SpaceID }
func (n *BaseNode) GetID() string      { return n.ID }

// InternalPath returns the internal path of the Node
func (n *BaseNode) InternalPath() string {
	if len(n.internalPath) > 0 && n.ID == n.internalPathID {
		return n.internalPath
	}

	n.internalPath = n.lu.InternalPath(n.SpaceID, n.ID)
	n.internalPathID = n.ID
	return n.internalPath
}

// Node represents a node in the tree and provides methods to get a Parent or Child instance
type Node struct {
	BaseNode
	ParentID  string
	Name      string
	Blobsize  int64
	BlobID    string
	owner     *userpb.UserId
	Exists    bool
	SpaceRoot *Node

	xattrsCache map[string][]byte
	disabled    *bool
	nodeType    *provider.ResourceType
}

// New returns a new instance of Node
func New(spaceID, id, parentID, name string, blobsize int64, blobID string, t provider.ResourceType, owner *userpb.UserId, lu PathLookup) *Node {
	if blobID == "" {
		blobID = uuid.New().String()
	}
	return &Node{
		BaseNode: BaseNode{
			SpaceID: spaceID,
			ID:      id,
			lu:      lu,
		},
		ParentID: parentID,
		Name:     name,
		Blobsize: blobsize,
		owner:    owner,
		BlobID:   blobID,
		nodeType: &t,
	}
}

func (n *Node) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name     string `json:"name"`
		ID       string `json:"id"`
		SpaceID  string `json:"spaceID"`
		ParentID string `json:"parentID"`
		BlobID   string `json:"blobID"`
		BlobSize int64  `json:"blobSize"`
		Exists   bool   `json:"exists"`
	}{
		Name:     n.Name,
		ID:       n.ID,
		SpaceID:  n.SpaceID,
		ParentID: n.ParentID,
		BlobID:   n.BlobID,
		BlobSize: n.Blobsize,
		Exists:   n.Exists,
	})
}

// Type returns the node's resource type
func (n *Node) Type(ctx context.Context) provider.ResourceType {
	_, span := tracer.Start(ctx, "Type")
	defer span.End()
	if n.nodeType != nil {
		return *n.nodeType
	}

	t := provider.ResourceType_RESOURCE_TYPE_INVALID

	// Try to read from xattrs
	typeAttr, err := n.XattrInt32(ctx, prefixes.TypeAttr)
	if err == nil {
		t = provider.ResourceType(typeAttr)
		n.nodeType = &t
		return t
	}

	// Fall back to checking on disk
	fi, err := os.Lstat(n.InternalPath())
	if err != nil {
		return t
	}

	switch {
	case fi.IsDir():
		if _, err = n.Xattr(ctx, prefixes.ReferenceAttr); err == nil {
			t = provider.ResourceType_RESOURCE_TYPE_REFERENCE
		} else {
			t = provider.ResourceType_RESOURCE_TYPE_CONTAINER
		}
	case fi.Mode().IsRegular():
		t = provider.ResourceType_RESOURCE_TYPE_FILE
	case fi.Mode()&os.ModeSymlink != 0:
		t = provider.ResourceType_RESOURCE_TYPE_SYMLINK
		// TODO reference using ext attr on a symlink
		// nodeType = provider.ResourceType_RESOURCE_TYPE_REFERENCE
	}
	n.nodeType = &t
	return t
}

// SetType sets the type of the node.
func (n *Node) SetType(t provider.ResourceType) {
	n.nodeType = &t
}

// NodeMetadata writes the Node metadata to disk and allows passing additional attributes
func (n *Node) NodeMetadata(ctx context.Context) Attributes {
	attribs := Attributes{}
	attribs.SetInt64(prefixes.TypeAttr, int64(n.Type(ctx)))
	attribs.SetString(prefixes.ParentidAttr, n.ParentID)
	attribs.SetString(prefixes.NameAttr, n.Name)
	if n.Type(ctx) == provider.ResourceType_RESOURCE_TYPE_FILE {
		attribs.SetString(prefixes.BlobIDAttr, n.BlobID)
		attribs.SetInt64(prefixes.BlobsizeAttr, n.Blobsize)
	}
	return attribs
}

// SetOwner sets the space owner on the node
func (n *Node) SetOwner(owner *userpb.UserId) {
	n.SpaceRoot.owner = owner
}

// SpaceOwnerOrManager returns the space owner of the space. If no owner is set
// one of the space managers is returned instead.
func (n *Node) SpaceOwnerOrManager(ctx context.Context) *userpb.UserId {
	owner := n.Owner()
	if owner != nil && owner.Type != userpb.UserType_USER_TYPE_SPACE_OWNER {
		return owner
	}

	// We don't have an owner set. Find a manager instead.
	grants, err := n.SpaceRoot.ListGrants(ctx)
	if err != nil {
		return nil
	}
	for _, grant := range grants {
		if grant.Permissions.Stat && grant.Permissions.ListContainer && grant.Permissions.InitiateFileDownload {
			return grant.GetGrantee().GetUserId()
		}
	}

	return nil
}

// ReadNode creates a new instance from an id and checks if it exists
func ReadNode(ctx context.Context, lu PathLookup, spaceID, nodeID string, canListDisabledSpace bool, spaceRoot *Node, skipParentCheck bool) (*Node, error) {
	ctx, span := tracer.Start(ctx, "ReadNode")
	defer span.End()
	var err error

	if spaceRoot == nil {
		// read space root
		spaceRoot = &Node{
			BaseNode: BaseNode{
				SpaceID: spaceID,
				lu:      lu,
				ID:      spaceID,
			},
		}
		spaceRoot.SpaceRoot = spaceRoot
		spaceRoot.owner, err = spaceRoot.readOwner(ctx)
		switch {
		case metadata.IsNotExist(err):
			return spaceRoot, nil // swallow not found, the node defaults to exists = false
		case err != nil:
			return nil, err
		}
		spaceRoot.Exists = true

		// lookup name in extended attributes
		spaceRoot.Name, err = spaceRoot.XattrString(ctx, prefixes.NameAttr)
		if err != nil {
			return nil, err
		}
	}

	// TODO ReadNode should not check permissions
	if !canListDisabledSpace && spaceRoot.IsDisabled(ctx) {
		// no permission = not found
		return nil, errtypes.NotFound(spaceID)
	}

	// if current user cannot stat the root return not found?
	// no for shares the root might be a different resource

	// check if this is a space root
	if spaceID == nodeID {
		return spaceRoot, nil
	}

	// are we reading a revision?
	revisionSuffix := ""
	if strings.Contains(nodeID, RevisionIDDelimiter) {
		// verify revision key format
		kp := strings.SplitN(nodeID, RevisionIDDelimiter, 2)
		if len(kp) == 2 {
			// use the actual node for the metadata lookup
			nodeID = kp[0]
			// remember revision for blob metadata
			revisionSuffix = kp[1]
		}
	}

	// read node
	n := &Node{
		BaseNode: BaseNode{
			SpaceID: spaceID,
			lu:      lu,
			ID:      nodeID,
		},
		SpaceRoot: spaceRoot,
	}

	// append back revision to nodeid, even when returning a not existing node
	defer func() {
		// when returning errors n is nil
		if n != nil && revisionSuffix != "" {
			n.ID += RevisionIDDelimiter + revisionSuffix
		}
	}()

	attrs, err := n.Xattrs(ctx)
	switch {
	case metadata.IsNotExist(err):
		return n, nil // swallow not found, the node defaults to exists = false
	case err != nil:
		return nil, err
	}
	n.Exists = true

	n.Name = attrs.String(prefixes.NameAttr)
	n.ParentID = attrs.String(prefixes.ParentidAttr)
	if n.ParentID == "" {
		d, _ := os.ReadFile(lu.MetadataBackend().MetadataPath(n))
		if _, ok := lu.MetadataBackend().(metadata.MessagePackBackend); ok {
			appctx.GetLogger(ctx).Error().Str("path", n.InternalPath()).Str("nodeid", n.ID).Interface("attrs", attrs).Bytes("messagepack", d).Msg("missing parent id")
		}
		return nil, errtypes.InternalError("Missing parent ID on node")
	}

	if n.Type(ctx) == provider.ResourceType_RESOURCE_TYPE_FILE {
		if revisionSuffix == "" {
			var (
				idOk, sizeOk bool
				d            []byte
				err          error
			)

			if d, idOk = attrs[prefixes.BlobIDAttr]; idOk {
				n.BlobID = string(d)
			}
			if d, sizeOk = attrs[prefixes.BlobsizeAttr]; sizeOk {
				n.Blobsize, err = strconv.ParseInt(string(d), 10, 64)
			}
			if err != nil || !idOk || !sizeOk {
				n.BlobID, n.Blobsize, err = lu.ReadBlobIDAndSizeAttr(ctx, n, attrs)
				if err != nil {
					return nil, err
				}
			}
		} else {
			versionNode := NewBaseNode(spaceID, nodeID+RevisionIDDelimiter+revisionSuffix, lu)
			n.BlobID, n.Blobsize, err = lu.ReadBlobIDAndSizeAttr(ctx, versionNode, nil)
			if err != nil {
				return nil, err
			}
		}
	}

	return n, nil
}

// Child returns the child node with the given name
func (n *Node) Child(ctx context.Context, name string) (*Node, error) {
	ctx, span := tracer.Start(ctx, "Child")
	defer span.End()

	spaceID := n.SpaceID
	if spaceID == "" && n.ParentID == "root" {
		spaceID = n.ID
	} else if n.SpaceRoot != nil {
		spaceID = n.SpaceRoot.ID
	}
	c := &Node{
		BaseNode: BaseNode{
			SpaceID: spaceID,
			lu:      n.lu,
		},
		ParentID:  n.ID,
		Name:      name,
		SpaceRoot: n.SpaceRoot,
	}

	nodeID, err := n.lu.NodeIDFromParentAndName(ctx, n, name)
	switch {
	case metadata.IsNotExist(err) || metadata.IsNotDir(err):
		return c, nil // if the file does not exist we return a node that has Exists = false
	case err != nil:
		return nil, err
	}

	readNode, err := ReadNode(ctx, n.lu, spaceID, nodeID, false, n.SpaceRoot, true)
	if err != nil {
		return nil, errors.Wrap(err, "could not read child node")
	}
	if !readNode.Exists {
		// if the node does not exist we return the manually crafted node instead which contains more
		// information than the read node, e.g. the parent id and the name
		return c, nil
	}

	return readNode, nil
}

// ParentWithReader returns the parent node
func (n *Node) ParentWithReader(ctx context.Context, r io.Reader) (*Node, error) {
	_, span := tracer.Start(ctx, "ParentWithReader")
	defer span.End()
	if n.ParentID == "" {
		return nil, fmt.Errorf("decomposedfs: root has no parent")
	}
	p := &Node{
		BaseNode: BaseNode{
			SpaceID: n.SpaceID,
			lu:      n.lu,
			ID:      n.ParentID,
		},
		SpaceRoot: n.SpaceRoot,
	}

	// fill metadata cache using the reader
	attrs, err := p.XattrsWithReader(ctx, r)
	switch {
	case metadata.IsNotExist(err):
		return p, nil // swallow not found, the node defaults to exists = false
	case err != nil:
		return nil, err
	}
	p.Exists = true

	p.Name = attrs.String(prefixes.NameAttr)
	p.ParentID = attrs.String(prefixes.ParentidAttr)

	return p, err
}

// Parent returns the parent node
func (n *Node) Parent(ctx context.Context) (p *Node, err error) {
	return n.ParentWithReader(ctx, nil)
}

// Owner returns the space owner
func (n *Node) Owner() *userpb.UserId {
	return n.SpaceRoot.owner
}

// readOwner reads the owner from the extended attributes of the space root
// in case either owner id or owner idp are unset we return an error and an empty owner object
func (n *Node) readOwner(ctx context.Context) (*userpb.UserId, error) {
	owner := &userpb.UserId{}

	// lookup parent id in extended attributes
	var attr string
	var err error
	// lookup ID in extended attributes
	attr, err = n.SpaceRoot.XattrString(ctx, prefixes.OwnerIDAttr)
	switch {
	case err == nil:
		owner.OpaqueId = attr
	case metadata.IsAttrUnset(err):
		// ignore
	default:
		return nil, err
	}

	// lookup IDP in extended attributes
	attr, err = n.SpaceRoot.XattrString(ctx, prefixes.OwnerIDPAttr)
	switch {
	case err == nil:
		owner.Idp = attr
	case metadata.IsAttrUnset(err):
		// ignore
	default:
		return nil, err
	}

	// lookup type in extended attributes
	attr, err = n.SpaceRoot.XattrString(ctx, prefixes.OwnerTypeAttr)
	switch {
	case err == nil:
		owner.Type = utils.UserTypeMap(attr)
	case metadata.IsAttrUnset(err):
		// ignore
	default:
		return nil, err
	}

	// owner is an optional property
	if owner.Idp == "" && owner.OpaqueId == "" {
		return nil, nil
	}
	return owner, nil
}

// PermissionSet returns the permission set and an accessDenied flag
// for the current user
// the parent nodes are not taken into account
// accessDenied is separate from the resource permissions
// because we only support full denials
func (n *Node) PermissionSet(ctx context.Context) (*provider.ResourcePermissions, bool) {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		appctx.GetLogger(ctx).Debug().Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Msg("no user in context, returning default permissions")
		return NoPermissions(), false
	}
	if utils.UserEqual(u.Id, n.SpaceRoot.Owner()) {
		return OwnerPermissions(), false
	}
	// read the permissions for the current user from the acls of the current node
	if np, accessDenied, err := n.ReadUserPermissions(ctx, u); err == nil {
		return np, accessDenied
	}
	// be defensive, we could have access via another grant
	return NoPermissions(), true
}

// ParentPath returns the internal path of the parent of the current node
func (n *Node) ParentPath() string {
	return n.lu.InternalPath(n.SpaceID, n.ParentID)
}

// LockfilePaths returns the paths(s) to the lockfile of the node
// Returning multiple paths allows for supporting legacy lockfiles
// while migrating to a new lockfile scheme. The first element is always the
// path to use for new locks.
// In the future only one path should remain at which point the function can return a single string.
func (n *Node) LockFilePaths() []string {
	return n.lu.LockfilePaths(n.SpaceID, n.ID)
}

// CalculateEtag returns a hash of fileid + tmtime (or mtime)
func CalculateEtag(id string, tmTime time.Time) (string, error) {
	h := md5.New()
	if _, err := io.WriteString(h, id); err != nil {
		return "", err
	}
	/* TODO we could strengthen the etag by adding the blobid, but then all etags would change. we would need a legacy etag check as well
	if _, err := io.WriteString(h, n.BlobID); err != nil {
		return "", err
	}
	*/
	if tb, err := tmTime.UTC().MarshalBinary(); err == nil {
		if _, err := h.Write(tb); err != nil {
			return "", err
		}
	} else {
		return "", err
	}
	return fmt.Sprintf(`"%x"`, h.Sum(nil)), nil
}

// SetMtimeString sets the mtime and atime of a node to the unixtime parsed from the given string
func (n *Node) SetMtimeString(ctx context.Context, mtime string) error {
	mt, err := utils.MTimeToTime(mtime)
	if err != nil {
		return err
	}
	return n.SetMtime(ctx, &mt)
}

// SetMTime writes the UTC mtime to the extended attributes or removes the attribute if nil is passed
func (n *Node) SetMtime(ctx context.Context, t *time.Time) (err error) {
	if t == nil {
		return n.RemoveXattr(ctx, prefixes.MTimeAttr, true)
	}
	return n.SetXattrString(ctx, prefixes.MTimeAttr, t.UTC().Format(time.RFC3339Nano))
}

// SetEtag sets the temporary etag of a node if it differs from the current etag
func (n *Node) SetEtag(ctx context.Context, val string) (err error) {
	sublog := appctx.GetLogger(ctx).With().Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Logger()
	var tmTime time.Time
	if tmTime, err = n.GetTMTime(ctx); err != nil {
		return
	}
	var etag string
	if etag, err = CalculateEtag(n.ID, tmTime); err != nil {
		return
	}

	// sanitize etag
	val = fmt.Sprintf("\"%s\"", strings.Trim(val, "\""))
	if etag == val {
		sublog.Debug().
			Str("etag", val).
			Msg("ignoring request to update identical etag")
		return nil
	}
	// etag is only valid until the calculated etag changes, is part of propagation
	return n.SetXattrString(ctx, prefixes.TmpEtagAttr, val)
}

// SetFavorite sets the favorite for the current user
// TODO we should not mess with the user here ... the favorites is now a user specific property for a file
// that cannot be mapped to extended attributes without leaking who has marked a file as a favorite
// it is a specific case of a tag, which is user individual as well
// TODO there are different types of tags
// 1. public that are managed by everyone
// 2. private tags that are only visible to the user
// 3. system tags that are only visible to the system
// 4. group tags that are only visible to a group ...
// urgh ... well this can be solved using different namespaces
// 1. public = p:
// 2. private = u:<uid>: for user specific
// 3. system = s: for system
// 4. group = g:<gid>:
// 5. app? = a:<aid>: for apps?
// obviously this only is secure when the u/s/g/a namespaces are not accessible by users in the filesystem
// public tags can be mapped to extended attributes
func (n *Node) SetFavorite(ctx context.Context, uid *userpb.UserId, val string) error {
	// the favorite flag is specific to the user, so we need to incorporate the userid
	fa := fmt.Sprintf("%s:%s:%s@%s", prefixes.FavPrefix, utils.UserTypeToString(uid.GetType()), uid.GetOpaqueId(), uid.GetIdp())
	return n.SetXattrString(ctx, fa, val)
}

// IsDir returns true if the node is a directory
func (n *Node) IsDir(ctx context.Context) bool {
	attr, _ := n.XattrInt32(ctx, prefixes.TypeAttr)
	return attr == int32(provider.ResourceType_RESOURCE_TYPE_CONTAINER)
}

// AsResourceInfo return the node as CS3 ResourceInfo
func (n *Node) AsResourceInfo(ctx context.Context, rp *provider.ResourcePermissions, mdKeys, fieldMask []string, returnBasename bool) (ri *provider.ResourceInfo, err error) {
	sublog := appctx.GetLogger(ctx).With().Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Logger()

	var fn string
	nodeType := n.Type(ctx)

	var target string
	if nodeType == provider.ResourceType_RESOURCE_TYPE_REFERENCE {
		target, _ = n.XattrString(ctx, prefixes.ReferenceAttr)
	}

	id := &provider.ResourceId{SpaceId: n.SpaceID, OpaqueId: n.ID}

	switch {
	case n.IsSpaceRoot(ctx):
		fn = "." // space roots do not have a path as they are referencing themselves
	case returnBasename:
		fn = n.Name
	default:
		fn, err = n.lu.Path(ctx, n, NoCheck)
		if err != nil {
			return nil, err
		}
	}

	ri = &provider.ResourceInfo{
		Id:            id,
		Path:          fn,
		Type:          nodeType,
		MimeType:      mime.Detect(nodeType == provider.ResourceType_RESOURCE_TYPE_CONTAINER, fn),
		Size:          uint64(n.Blobsize),
		Target:        target,
		PermissionSet: rp,
		Owner:         n.Owner(),
		ParentId: &provider.ResourceId{
			SpaceId:  n.SpaceID,
			OpaqueId: n.ParentID,
		},
		Name: n.Name,
	}

	if n.IsProcessing(ctx) {
		ri.Opaque = utils.AppendPlainToOpaque(ri.Opaque, "status", "processing")
	}

	if nodeType == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		ts, err := n.GetTreeSize(ctx)
		if err == nil {
			ri.Size = ts
		} else {
			ri.Size = 0 // make dirs always return 0 if it is unknown
			sublog.Debug().Err(err).Msg("could not read treesize")
		}
	}

	// TODO make etag of files use fileid and checksum

	var tmTime time.Time
	if tmTime, err = n.GetTMTime(ctx); err != nil {
		sublog.Debug().Err(err).Msg("could not get tmtime")
	}

	// use temporary etag if it is set
	if b, err := n.XattrString(ctx, prefixes.TmpEtagAttr); err == nil && b != "" {
		ri.Etag = fmt.Sprintf(`"%x"`, b)
	} else if ri.Etag, err = CalculateEtag(n.ID, tmTime); err != nil {
		sublog.Debug().Err(err).Msg("could not calculate etag")
	}

	// mtime uses tmtime if present
	// TODO expose mtime and tmtime separately?
	un := tmTime.UnixNano()
	ri.Mtime = &types.Timestamp{
		Seconds: uint64(un / 1000000000),
		Nanos:   uint32(un % 1000000000),
	}

	mdKeysMap := make(map[string]struct{})
	for _, k := range mdKeys {
		mdKeysMap[k] = struct{}{}
	}

	var returnAllMetadata bool
	if _, ok := mdKeysMap["*"]; len(mdKeys) == 0 || ok {
		returnAllMetadata = true
	}

	metadata := map[string]string{}

	fieldMaskKeysMap := make(map[string]struct{})
	for _, k := range fieldMask {
		fieldMaskKeysMap[k] = struct{}{}
	}

	var returnAllFields bool
	if _, ok := fieldMaskKeysMap["*"]; len(fieldMask) == 0 || ok {
		returnAllFields = true
	}

	// read favorite flag for the current user
	if _, ok := mdKeysMap[FavoriteKey]; returnAllMetadata || ok {
		favorite := ""
		if u, ok := ctxpkg.ContextGetUser(ctx); ok {
			// the favorite flag is specific to the user, so we need to incorporate the userid
			if uid := u.GetId(); uid != nil {
				fa := fmt.Sprintf("%s:%s:%s@%s", prefixes.FavPrefix, utils.UserTypeToString(uid.GetType()), uid.GetOpaqueId(), uid.GetIdp())
				if val, err := n.XattrString(ctx, fa); err == nil {
					sublog.Debug().
						Str("favorite", fa).
						Msg("found favorite flag")
					favorite = val
				}
			} else {
				sublog.Error().Err(errtypes.UserRequired("userrequired")).Msg("user has no id")
			}
		} else {
			sublog.Error().Err(errtypes.UserRequired("userrequired")).Msg("error getting user from ctx")
		}
		metadata[FavoriteKey] = favorite
	}
	// read locks
	// FIXME move to fieldmask
	if _, ok := mdKeysMap[LockdiscoveryKey]; returnAllMetadata || ok {
		if n.hasLocks(ctx) {
			err = readLocksIntoOpaque(ctx, n, ri)
			if err != nil {
				sublog.Debug().Err(errtypes.InternalError("lockfail"))
			}
		}
	}

	// share indicator
	if _, ok := fieldMaskKeysMap["share-types"]; returnAllFields || ok {
		granteeTypes := n.getGranteeTypes(ctx)
		if len(granteeTypes) > 0 {
			// TODO add optional property to CS3 ResourceInfo to transport grants?
			var s strings.Builder
			first := true
			for _, t := range granteeTypes {
				if !first {
					s.WriteString(",")
				} else {
					first = false
				}
				s.WriteString(strconv.Itoa(int(t)))
			}
			ri.Opaque = utils.AppendPlainToOpaque(ri.Opaque, "share-types", s.String())
		}
	}

	// checksums
	// FIXME move to fieldmask
	if _, ok := mdKeysMap[ChecksumsKey]; (nodeType == provider.ResourceType_RESOURCE_TYPE_FILE) && (returnAllMetadata || ok) {
		// TODO which checksum was requested? sha1 adler32 or md5? for now hardcode sha1?
		// TODO make ResourceInfo carry multiple checksums
		n.readChecksumIntoResourceChecksum(ctx, storageprovider.XSSHA1, ri)
		n.readChecksumIntoOpaque(ctx, storageprovider.XSMD5, ri)
		n.readChecksumIntoOpaque(ctx, storageprovider.XSAdler32, ri)
	}
	// quota
	// FIXME move to fieldmask
	if _, ok := mdKeysMap[QuotaKey]; (nodeType == provider.ResourceType_RESOURCE_TYPE_CONTAINER) && returnAllMetadata || ok {
		if n.SpaceRoot != nil && n.SpaceRoot.InternalPath() != "" {
			n.SpaceRoot.readQuotaIntoOpaque(ctx, ri)
		}
	}

	// only read the requested metadata attributes
	attrs, err := n.Xattrs(ctx)
	if err != nil {
		sublog.Error().Err(err).Msg("error getting list of extended attributes")
	} else {
		for key, value := range attrs {
			// filter out non-custom properties
			if !strings.HasPrefix(key, prefixes.MetadataPrefix) {
				continue
			}
			// only read when key was requested
			k := key[len(prefixes.MetadataPrefix):]
			if _, ok := mdKeysMap[k]; returnAllMetadata || ok {
				metadata[k] = string(value)
			}

		}
	}
	ri.ArbitraryMetadata = &provider.ArbitraryMetadata{
		Metadata: metadata,
	}

	// add virusscan information
	if scanned, _, date := n.ScanData(ctx); scanned {
		ri.Opaque = utils.AppendPlainToOpaque(ri.Opaque, "scantime", date.Format(time.RFC3339Nano))
	}

	sublog.Trace().
		Interface("ri", ri).
		Msg("AsResourceInfo")

	return ri, nil
}

func (n *Node) readChecksumIntoResourceChecksum(ctx context.Context, algo string, ri *provider.ResourceInfo) {
	v, err := n.Xattr(ctx, prefixes.ChecksumPrefix+algo)
	switch {
	case err == nil:
		ri.Checksum = &provider.ResourceChecksum{
			Type: storageprovider.PKG2GRPCXS(algo),
			Sum:  hex.EncodeToString(v),
		}
	case metadata.IsAttrUnset(err):
		appctx.GetLogger(ctx).Debug().Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Str("nodepath", n.InternalPath()).Str("algorithm", algo).Msg("checksum not set")
	default:
		appctx.GetLogger(ctx).Error().Err(err).Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Str("nodepath", n.InternalPath()).Str("algorithm", algo).Msg("could not read checksum")
	}
}

func (n *Node) readChecksumIntoOpaque(ctx context.Context, algo string, ri *provider.ResourceInfo) {
	v, err := n.Xattr(ctx, prefixes.ChecksumPrefix+algo)
	switch {
	case err == nil:
		if ri.Opaque == nil {
			ri.Opaque = &types.Opaque{
				Map: map[string]*types.OpaqueEntry{},
			}
		}
		ri.Opaque.Map[algo] = &types.OpaqueEntry{
			Decoder: "plain",
			Value:   []byte(hex.EncodeToString(v)),
		}
	case metadata.IsAttrUnset(err):
		appctx.GetLogger(ctx).Debug().Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Str("nodepath", n.InternalPath()).Str("algorithm", algo).Msg("checksum not set")
	default:
		appctx.GetLogger(ctx).Error().Err(err).Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Str("nodepath", n.InternalPath()).Str("algorithm", algo).Msg("could not read checksum")
	}
}

// quota is always stored on the root node
func (n *Node) readQuotaIntoOpaque(ctx context.Context, ri *provider.ResourceInfo) {
	v, err := n.XattrString(ctx, prefixes.QuotaAttr)
	switch {
	case err == nil:
		// make sure we have a proper signed int
		// we use the same magic numbers to indicate:
		// -1 = uncalculated
		// -2 = unknown
		// -3 = unlimited
		if _, err := strconv.ParseInt(v, 10, 64); err == nil {
			if ri.Opaque == nil {
				ri.Opaque = &types.Opaque{
					Map: map[string]*types.OpaqueEntry{},
				}
			}
			ri.Opaque.Map[QuotaKey] = &types.OpaqueEntry{
				Decoder: "plain",
				Value:   []byte(v),
			}
		} else {
			appctx.GetLogger(ctx).Error().Err(err).Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Str("nodepath", n.InternalPath()).Str("quota", v).Msg("malformed quota")
		}
	case metadata.IsAttrUnset(err):
		appctx.GetLogger(ctx).Trace().Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Str("nodepath", n.InternalPath()).Msg("quota not set")
	default:
		appctx.GetLogger(ctx).Error().Err(err).Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Str("nodepath", n.InternalPath()).Msg("could not read quota")
	}
}

// HasPropagation checks if the propagation attribute exists and is set to "1"
func (n *Node) HasPropagation(ctx context.Context) (propagation bool) {
	if b, err := n.XattrString(ctx, prefixes.PropagationAttr); err == nil {
		return b == "1"
	}
	return false
}

// IsDisabled returns true when the node has a dmtime attribute set
// only used to check if a space is disabled
// FIXME confusing with the trash logic
func (n *Node) IsDisabled(ctx context.Context) bool {
	if n.disabled != nil {
		return *n.disabled
	}
	if _, err := n.GetDTime(ctx); err == nil {
		v := true
		n.disabled = &v
	} else {
		v := false
		n.disabled = &v
	}
	return *n.disabled
}

// GetTreeSize reads the treesize from the extended attributes
func (n *Node) GetTreeSize(ctx context.Context) (treesize uint64, err error) {
	ctx, span := tracer.Start(ctx, "GetTreeSize")
	defer span.End()
	s, err := n.XattrUint64(ctx, prefixes.TreesizeAttr)
	if err != nil {
		return 0, err
	}
	return s, nil
}

// SetTreeSize writes the treesize to the extended attributes
func (n *Node) SetTreeSize(ctx context.Context, ts uint64) (err error) {
	return n.SetXattrString(ctx, prefixes.TreesizeAttr, strconv.FormatUint(ts, 10))
}

// GetBlobSize reads the blobsize from the extended attributes
func (n *Node) GetBlobSize(ctx context.Context) (treesize uint64, err error) {
	s, err := n.XattrInt64(ctx, prefixes.BlobsizeAttr)
	if err != nil {
		return 0, err
	}
	return uint64(s), nil
}

// SetChecksum writes the checksum with the given checksum type to the extended attributes
func (n *Node) SetChecksum(ctx context.Context, csType string, h hash.Hash) (err error) {
	return n.SetXattr(ctx, prefixes.ChecksumPrefix+csType, h.Sum(nil))
}

// UnsetTempEtag removes the temporary etag attribute
func (n *Node) UnsetTempEtag(ctx context.Context) (err error) {
	return n.RemoveXattr(ctx, prefixes.TmpEtagAttr, true)
}

func isGrantExpired(g *provider.Grant) bool {
	if g.Expiration == nil {
		return false
	}
	return time.Now().After(time.Unix(int64(g.Expiration.Seconds), int64(g.Expiration.Nanos)))
}

// ReadUserPermissions will assemble the permissions for the current user on the given node without parent nodes
// we indicate if the access was denied by setting a grant with no permissions
func (n *Node) ReadUserPermissions(ctx context.Context, u *userpb.User) (ap *provider.ResourcePermissions, accessDenied bool, err error) {
	// check if the current user is the owner
	if utils.UserEqual(u.Id, n.Owner()) {
		appctx.GetLogger(ctx).Debug().Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Msg("user is owner, returning owner permissions")
		return OwnerPermissions(), false, nil
	}

	ap = &provider.ResourcePermissions{}

	// for an efficient group lookup convert the list of groups to a map
	// groups are just strings ... groupnames ... or group ids ??? AAARGH !!!
	groupsMap := make(map[string]bool, len(u.Groups))
	for i := range u.Groups {
		groupsMap[u.Groups[i]] = true
	}

	var g *provider.Grant

	// we read all grantees from the node
	var grantees []string
	if grantees, err = n.ListGrantees(ctx); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Msg("error listing grantees")
		return NoPermissions(), true, err
	}

	// instead of making n getxattr syscalls we are going to list the acls and filter them here
	// we have two options here:
	// 1. we can start iterating over the acls / grants on the node or
	// 2. we can iterate over the number of groups
	// The current implementation tries to be defensive for cases where users have hundreds or thousands of groups, so we iterate over the existing acls.
	userace := prefixes.GrantPrefix + ace.UserAce(u.Id)
	userFound := false
	for i := range grantees {
		switch {
		// we only need to find the user once
		case !userFound && grantees[i] == userace:
			g, err = n.ReadGrant(ctx, grantees[i])
		case strings.HasPrefix(grantees[i], prefixes.GrantGroupAcePrefix): // only check group grantees
			gr := strings.TrimPrefix(grantees[i], prefixes.GrantGroupAcePrefix)
			if groupsMap[gr] {
				g, err = n.ReadGrant(ctx, grantees[i])
			} else {
				// no need to check attribute
				continue
			}
		default:
			// no need to check attribute
			continue
		}

		switch {
		case err == nil:
			if isGrantExpired(g) {
				continue
			}

			// If all permissions are set to false we have a deny grant
			if grants.PermissionsEqual(g.Permissions, &provider.ResourcePermissions{}) {
				return NoPermissions(), true, nil
			}
			AddPermissions(ap, g.GetPermissions())
		case metadata.IsAttrUnset(err):
			appctx.GetLogger(ctx).Error().Err(err).Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Str("path", n.InternalPath()).Str("grant", grantees[i]).Interface("grantees", grantees).Msg("grant vanished from node after listing")
			// continue with next segment
		default:
			appctx.GetLogger(ctx).Error().Err(err).Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Str("path", n.InternalPath()).Str("grant", grantees[i]).Msg("error reading permissions")
			// continue with next segment
		}
	}

	appctx.GetLogger(ctx).Trace().Interface("permissions", ap).Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Interface("user", u).Msg("returning aggregated permissions")
	return ap, false, nil
}

// IsDenied checks if the node was denied to that user
func (n *Node) IsDenied(ctx context.Context) bool {
	gs, err := n.ListGrants(ctx)
	if err != nil {
		// be paranoid, resource is denied
		return true
	}

	u := ctxpkg.ContextMustGetUser(ctx)
	isExecutant := func(g *provider.Grantee) bool {
		switch g.GetType() {
		case provider.GranteeType_GRANTEE_TYPE_USER:
			return g.GetUserId().GetOpaqueId() == u.GetId().GetOpaqueId()
		case provider.GranteeType_GRANTEE_TYPE_GROUP:
			// check gid
			gid := g.GetGroupId().GetOpaqueId()
			for _, group := range u.Groups {
				if gid == group {
					return true
				}

			}
			return false
		default:
			return false
		}

	}

	for _, g := range gs {
		if !isExecutant(g.Grantee) {
			continue
		}

		if grants.PermissionsEqual(g.Permissions, &provider.ResourcePermissions{}) {
			// resource is denied
			return true
		}
	}

	// no deny grants
	return false
}

// ListGrantees lists the grantees of the current node
// We don't want to wast time and memory by creating grantee objects.
// The function will return a list of opaque strings that can be used to make a ReadGrant call
func (n *Node) ListGrantees(ctx context.Context) (grantees []string, err error) {
	attrs, err := n.Xattrs(ctx)
	if err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Msg("error listing attributes")
		return nil, err
	}
	for name := range attrs {
		if strings.HasPrefix(name, prefixes.GrantPrefix) {
			grantees = append(grantees, name)
		}
	}
	return
}

// ReadGrant reads a CS3 grant
func (n *Node) ReadGrant(ctx context.Context, grantee string) (g *provider.Grant, err error) {
	xattr, err := n.Xattr(ctx, grantee)
	if err != nil {
		return nil, err
	}
	var e *ace.ACE
	if e, err = ace.Unmarshal(strings.TrimPrefix(grantee, prefixes.GrantPrefix), xattr); err != nil {
		return nil, err
	}
	return e.Grant(), nil
}

// ReadGrant reads a CS3 grant
func (n *Node) DeleteGrant(ctx context.Context, g *provider.Grant, acquireLock bool) (err error) {

	var attr string
	if g.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_GROUP {
		attr = prefixes.GrantGroupAcePrefix + g.Grantee.GetGroupId().OpaqueId
	} else {
		attr = prefixes.GrantUserAcePrefix + g.Grantee.GetUserId().OpaqueId
	}

	if err = n.RemoveXattr(ctx, attr, acquireLock); err != nil {
		return err
	}

	return nil
}

// Purge removes a node from disk. It does not move it to the trash
func (n *Node) Purge(ctx context.Context) error {
	return n.lu.PurgeNode(n)
}

// ListGrants lists all grants of the current node.
func (n *Node) ListGrants(ctx context.Context) ([]*provider.Grant, error) {
	grantees, err := n.ListGrantees(ctx)
	if err != nil {
		return nil, err
	}

	grants := make([]*provider.Grant, 0, len(grantees))
	for _, g := range grantees {
		grant, err := n.ReadGrant(ctx, g)
		if err != nil {
			appctx.GetLogger(ctx).
				Error().
				Err(err).
				Str("spaceid", n.SpaceID).
				Str("nodeid", n.ID).
				Str("grantee", g).
				Msg("error reading grant")
			continue
		}
		grants = append(grants, grant)
	}
	return grants, nil
}

func (n *Node) getGranteeTypes(ctx context.Context) []provider.GranteeType {
	types := []provider.GranteeType{}
	if g, err := n.ListGrantees(ctx); err == nil {
		hasUserShares, hasGroupShares := false, false
		for i := range g {
			switch {
			case !hasUserShares && strings.HasPrefix(g[i], prefixes.GrantUserAcePrefix):
				hasUserShares = true
			case !hasGroupShares && strings.HasPrefix(g[i], prefixes.GrantGroupAcePrefix):
				hasGroupShares = true
			case hasUserShares && hasGroupShares:
				break
			}
		}
		if hasUserShares {
			types = append(types, provider.GranteeType_GRANTEE_TYPE_USER)
		}
		if hasGroupShares {
			types = append(types, provider.GranteeType_GRANTEE_TYPE_GROUP)
		}
	}
	return types
}

// FindStorageSpaceRoot calls n.Parent() and climbs the tree
// until it finds the space root node and adds it to the node
func (n *Node) FindStorageSpaceRoot(ctx context.Context) error {
	if n.SpaceRoot != nil {
		return nil
	}
	var err error
	// remember the node we ask for and use parent to climb the tree
	parent := n
	for {
		if parent.IsSpaceRoot(ctx) {
			n.SpaceRoot = parent
			break
		}
		if parent, err = parent.Parent(ctx); err != nil {
			return err
		}
	}
	return nil
}

// UnmarkProcessing removes the processing flag from the node
func (n *Node) UnmarkProcessing(ctx context.Context, uploadID string) error {
	// we currently have to decrease the counter for every processing run to match the incrases
	metrics.UploadProcessing.Sub(1)

	v, _ := n.XattrString(ctx, prefixes.StatusPrefix)
	if v != ProcessingStatus+uploadID {
		// file started another postprocessing later - do not remove
		return nil
	}
	return n.RemoveXattr(ctx, prefixes.StatusPrefix, true)
}

// IsProcessing returns true if the node is currently being processed
func (n *Node) IsProcessing(ctx context.Context) bool {
	v, err := n.XattrString(ctx, prefixes.StatusPrefix)
	return err == nil && strings.HasPrefix(v, ProcessingStatus)
}

// ProcessingID returns the latest upload session id
func (n *Node) ProcessingID(ctx context.Context) (string, error) {
	v, err := n.XattrString(ctx, prefixes.StatusPrefix)
	return strings.TrimPrefix(v, ProcessingStatus), err
}

// IsSpaceRoot checks if the node is a space root
func (n *Node) IsSpaceRoot(ctx context.Context) bool {
	return n.ID == n.SpaceID
}

// SetScanData sets the virus scan info to the node
func (n *Node) SetScanData(ctx context.Context, info string, date time.Time) error {
	attribs := Attributes{}
	attribs.SetString(prefixes.ScanStatusPrefix, info)
	attribs.SetString(prefixes.ScanDatePrefix, date.Format(time.RFC3339Nano))
	return n.SetXattrsWithContext(ctx, attribs, true)
}

// ScanData returns scanning information of the node
func (n *Node) ScanData(ctx context.Context) (scanned bool, virus string, scantime time.Time) {
	ti, _ := n.XattrString(ctx, prefixes.ScanDatePrefix)
	if ti == "" {
		return // not scanned yet
	}

	t, err := time.Parse(time.RFC3339Nano, ti)
	if err != nil {
		return
	}

	i, err := n.XattrString(ctx, prefixes.ScanStatusPrefix)
	if err != nil {
		return
	}

	return true, i, t
}

// CheckQuota checks if both disk space and available quota are sufficient
// Overwrite must be set to true if the new file replaces the old file e.g.
// when creating a new file version. In such a case the function will
// reduce the used bytes by the old file size and then add the new size.
// If overwrite is false oldSize will be ignored.
var CheckQuota = func(ctx context.Context, spaceRoot *Node, overwrite bool, oldSize, newSize uint64) (quotaSufficient bool, err error) {
	used, _ := spaceRoot.GetTreeSize(ctx)
	if !enoughDiskSpace(spaceRoot.InternalPath(), newSize) {
		return false, errtypes.InsufficientStorage("disk full")
	}
	quotaByteStr, _ := spaceRoot.XattrString(ctx, prefixes.QuotaAttr)
	switch quotaByteStr {
	case "":
		// if quota is not set, it means unlimited
		return true, nil
	case QuotaUnlimited:
		return true, nil
	case QuotaUncalculated:
		// treat it as unlimited
		return true, nil
	case QuotaUnknown:
		// treat it as unlimited
		return true, nil
	}
	quotaByte, _ := strconv.ParseUint(quotaByteStr, 10, 64)
	if overwrite {
		if quotaByte < used-oldSize+newSize {
			return false, errtypes.InsufficientStorage("quota exceeded")
		}
		// if total is smaller than used, total-used could overflow and be bigger than fileSize
	} else if newSize > quotaByte-used || quotaByte < used {
		return false, errtypes.InsufficientStorage("quota exceeded")
	}
	return true, nil
}

func enoughDiskSpace(path string, fileSize uint64) bool {
	avalB, err := GetAvailableSize(path)
	if err != nil {
		return false
	}
	return avalB > fileSize
}

// CalculateChecksums calculates the sha1, md5 and adler32 checksums of a file
func CalculateChecksums(ctx context.Context, path string) (hash.Hash, hash.Hash, hash.Hash32, error) {
	sha1h := sha1.New()
	md5h := md5.New()
	adler32h := adler32.New()

	_, subspan := tracer.Start(ctx, "os.Open")
	f, err := os.Open(path)
	subspan.End()
	if err != nil {
		return nil, nil, nil, err
	}
	defer f.Close()

	r1 := io.TeeReader(f, sha1h)
	r2 := io.TeeReader(r1, md5h)

	_, subspan = tracer.Start(ctx, "io.Copy")
	_, err = io.Copy(adler32h, r2)
	subspan.End()
	if err != nil {
		return nil, nil, nil, err
	}

	return sha1h, md5h, adler32h, nil
}

// GetMTime reads the mtime from the extended attributes
func (n *Node) GetMTime(ctx context.Context) (time.Time, error) {
	return n.lu.TimeManager().MTime(ctx, n)
}

// GetTMTime reads the tmtime from the extended attributes
func (n *Node) GetTMTime(ctx context.Context) (time.Time, error) {
	return n.lu.TimeManager().TMTime(ctx, n)
}

// SetTMTime writes the UTC tmtime to the extended attributes or removes the attribute if nil is passed
func (n *Node) SetTMTime(ctx context.Context, t *time.Time) (err error) {
	return n.lu.TimeManager().SetTMTime(ctx, n, t)
}

// GetDTime reads the dmtime from the extended attributes
func (n *Node) GetDTime(ctx context.Context) (time.Time, error) {
	return n.lu.TimeManager().DTime(ctx, n)
}

// SetDTime writes the UTC dmtime to the extended attributes or removes the attribute if nil is passed
func (n *Node) SetDTime(ctx context.Context, t *time.Time) (err error) {
	return n.lu.TimeManager().SetDTime(ctx, n, t)
}

// ReadChildNodeFromLink reads the child node id from a link
func ReadChildNodeFromLink(ctx context.Context, path string) (string, error) {
	_, span := tracer.Start(ctx, "readChildNodeFromLink")
	defer span.End()
	link, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	nodeID := strings.TrimLeft(link, "/.")
	nodeID = strings.ReplaceAll(nodeID, "/", "")
	return nodeID, nil
}
