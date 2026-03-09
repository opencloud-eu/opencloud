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

package localfs

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/mime"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/acl"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/chunking"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/grants"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
)

// Config holds the configuration details for the local fs.
type Config struct {
	Root                string `mapstructure:"root"`
	DisableHome         bool   `mapstructure:"disable_home"`
	UserLayout          string `mapstructure:"user_layout"`
	ShareFolder         string `mapstructure:"share_folder"`
	DataTransfersFolder string `mapstructure:"data_transfers_folder"`
	Uploads             string `mapstructure:"uploads"`
	DataDirectory       string `mapstructure:"data_directory"`
	RecycleBin          string `mapstructure:"recycle_bin"`
	Versions            string `mapstructure:"versions"`
	Shadow              string `mapstructure:"shadow"`
	References          string `mapstructure:"references"`
}

func (c *Config) init() {
	if c.Root == "" {
		c.Root = "/var/tmp/reva"
	}

	if c.UserLayout == "" {
		c.UserLayout = "{{.Username}}"
	}

	if c.ShareFolder == "" {
		c.ShareFolder = "/MyShares"
	}

	if c.DataTransfersFolder == "" {
		c.DataTransfersFolder = "/DataTransfers"
	}

	// ensure share folder always starts with slash
	c.ShareFolder = path.Join("/", c.ShareFolder)

	c.DataDirectory = path.Join(c.Root, "data")
	c.Uploads = path.Join(c.Root, ".uploads")
	c.Shadow = path.Join(c.Root, ".shadow")

	c.References = path.Join(c.Shadow, "references")
	c.RecycleBin = path.Join(c.Shadow, "recycle_bin")
	c.Versions = path.Join(c.Shadow, "versions")

}

type localfs struct {
	conf         *Config
	db           *sql.DB
	chunkHandler *chunking.ChunkHandler
}

// NewLocalFS returns a storage.FS interface implementation that controls then
// local filesystem.
func NewLocalFS(c *Config) (storage.FS, error) {
	c.init()

	// create namespaces if they do not exist
	namespaces := []string{c.DataDirectory, c.Uploads, c.Shadow, c.References, c.RecycleBin, c.Versions}
	for _, v := range namespaces {
		if err := os.MkdirAll(v, 0755); err != nil {
			return nil, errors.Wrap(err, "could not create home dir "+v)
		}
	}

	dbName := "localfs.db"
	if !c.DisableHome {
		dbName = "localhomefs.db"
	}

	db, err := initializeDB(c.Root, dbName)
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error initializing db")
	}

	return &localfs{
		conf:         c,
		db:           db,
		chunkHandler: chunking.NewChunkHandler(c.Uploads),
	}, nil
}

func (fs *localfs) Shutdown(ctx context.Context) error {
	err := fs.db.Close()
	if err != nil {
		return errors.Wrap(err, "localfs: error closing db connection")
	}
	return nil
}

func (fs *localfs) resolve(ctx context.Context, ref *provider.Reference) (p string, err error) {
	if ref.ResourceId != nil {
		if p, err = fs.GetPathByID(ctx, ref.ResourceId); err != nil {
			return "", err
		}
		return path.Join(p, path.Join("/", ref.Path)), nil
	}

	if ref.Path != "" {
		return path.Join("/", ref.Path), nil
	}

	// reference is invalid
	return "", fmt.Errorf("invalid reference %+v. at least resource_id or path must be set", ref)
}

func getUser(ctx context.Context) (*userpb.User, error) {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		err := errors.Wrap(errtypes.UserRequired(""), "local: error getting user from ctx")
		return nil, err
	}
	return u, nil
}

func (fs *localfs) wrap(ctx context.Context, p string) string {
	// This is to prevent path traversal.
	// With this p can't break out of its parent folder
	p = path.Join("/", p)
	var internal string
	if !fs.conf.DisableHome {
		layout, err := fs.GetHome(ctx)
		if err != nil {
			panic(err)
		}
		internal = path.Join(fs.conf.DataDirectory, layout, p)
	} else {
		internal = path.Join(fs.conf.DataDirectory, p)
	}
	return internal
}

func (fs *localfs) wrapReferences(ctx context.Context, p string) string {
	var internal string
	if !fs.conf.DisableHome {
		layout, err := fs.GetHome(ctx)
		if err != nil {
			panic(err)
		}
		internal = path.Join(fs.conf.References, layout, p)
	} else {
		internal = path.Join(fs.conf.References, p)
	}
	return internal
}

func (fs *localfs) wrapRecycleBin(ctx context.Context, p string) string {
	var internal string
	if !fs.conf.DisableHome {
		layout, err := fs.GetHome(ctx)
		if err != nil {
			panic(err)
		}
		internal = path.Join(fs.conf.RecycleBin, layout, p)
	} else {
		internal = path.Join(fs.conf.RecycleBin, p)
	}
	return internal
}

func (fs *localfs) wrapVersions(ctx context.Context, p string) string {
	p = path.Join("/", p)
	var internal string
	if !fs.conf.DisableHome {
		layout, err := fs.GetHome(ctx)
		if err != nil {
			panic(err)
		}
		internal = path.Join(fs.conf.Versions, layout, p)
	} else {
		internal = path.Join(fs.conf.Versions, p)
	}
	return internal
}

func (fs *localfs) unwrap(ctx context.Context, np string) string {
	ns := fs.getNsMatch(np, []string{fs.conf.DataDirectory, fs.conf.References, fs.conf.RecycleBin, fs.conf.Versions})
	var external string
	if !fs.conf.DisableHome {
		layout, err := fs.GetHome(ctx)
		if err != nil {
			panic(err)
		}
		ns = path.Join(ns, layout)
	}

	external = strings.TrimPrefix(np, ns)
	if external == "" {
		external = "/"
	}
	return external
}

func (fs *localfs) getNsMatch(internal string, nss []string) string {
	var match string
	for _, ns := range nss {
		if strings.HasPrefix(internal, ns) && len(ns) > len(match) {
			match = ns
		}
	}
	if match == "" {
		panic(fmt.Sprintf("local: path is outside namespaces: path=%s namespaces=%+v", internal, nss))
	}

	return match
}

func (fs *localfs) isShareFolder(ctx context.Context, p string) bool {
	return strings.HasPrefix(p, fs.conf.ShareFolder)
}

func (fs *localfs) isDataTransfersFolder(ctx context.Context, p string) bool {
	return strings.HasPrefix(p, fs.conf.DataTransfersFolder)
}

func (fs *localfs) isShareFolderRoot(ctx context.Context, p string) bool {
	return path.Clean(p) == fs.conf.ShareFolder
}

func (fs *localfs) isShareFolderChild(ctx context.Context, p string) bool {
	p = path.Clean(p)
	vals := strings.Split(p, fs.conf.ShareFolder+"/")
	return len(vals) > 1 && vals[1] != ""
}

// permissionSet returns the permission set for the current user
func (fs *localfs) permissionSet(ctx context.Context, owner *userpb.UserId) *provider.ResourcePermissions {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		return &provider.ResourcePermissions{
			// no permissions
		}
	}
	if u.Id == nil {
		return &provider.ResourcePermissions{
			// no permissions
		}
	}
	if u.Id.OpaqueId == owner.OpaqueId && u.Id.Idp == owner.Idp {
		return &provider.ResourcePermissions{
			// owner has all permissions
			AddGrant:             true,
			CreateContainer:      true,
			Delete:               true,
			GetPath:              true,
			GetQuota:             true,
			InitiateFileDownload: true,
			InitiateFileUpload:   true,
			ListContainer:        true,
			ListFileVersions:     true,
			ListGrants:           true,
			ListRecycle:          true,
			Move:                 true,
			PurgeRecycle:         true,
			RemoveGrant:          true,
			RestoreFileVersion:   true,
			RestoreRecycleItem:   true,
			Stat:                 true,
			UpdateGrant:          true,
		}
	}
	// TODO fix permissions for share recipients by traversing reading acls up to the root? cache acls for the parent node and reuse it
	return &provider.ResourcePermissions{
		AddGrant:             true,
		CreateContainer:      true,
		Delete:               true,
		GetPath:              true,
		GetQuota:             true,
		InitiateFileDownload: true,
		InitiateFileUpload:   true,
		ListContainer:        true,
		ListFileVersions:     true,
		ListGrants:           true,
		ListRecycle:          true,
		Move:                 true,
		PurgeRecycle:         true,
		RemoveGrant:          true,
		RestoreFileVersion:   true,
		RestoreRecycleItem:   true,
		Stat:                 true,
		UpdateGrant:          true,
	}
}

func (fs *localfs) normalize(ctx context.Context, fi os.FileInfo, fn string, mdKeys []string) (*provider.ResourceInfo, error) {
	fp := fs.unwrap(ctx, path.Join("/", fn))
	owner, err := getUser(ctx)
	if err != nil {
		return nil, err
	}
	metadata, err := fs.retrieveArbitraryMetadata(ctx, fn, mdKeys)
	if err != nil {
		return nil, err
	}

	var layout string
	if !fs.conf.DisableHome {
		layout, err = fs.GetHome(ctx)
		if err != nil {
			return nil, err
		}
	}

	// A fileid is constructed like `fileid-url_encoded_path`. See GetPathByID for the inverse conversion
	md := &provider.ResourceInfo{
		Id:            &provider.ResourceId{OpaqueId: "fileid-" + url.QueryEscape(path.Join(layout, fp))},
		Path:          fp,
		Type:          getResourceType(fi.IsDir()),
		Etag:          calcEtag(ctx, fi),
		MimeType:      mime.Detect(fi.IsDir(), fp),
		Size:          uint64(fi.Size()),
		PermissionSet: fs.permissionSet(ctx, owner.Id),
		Mtime: &types.Timestamp{
			Seconds: uint64(fi.ModTime().Unix()),
		},
		Owner:             owner.Id,
		ArbitraryMetadata: metadata,
	}

	return md, nil
}

func (fs *localfs) convertToFileReference(ctx context.Context, fi os.FileInfo, fn string, mdKeys []string) (*provider.ResourceInfo, error) {
	info, err := fs.normalize(ctx, fi, fn, mdKeys)
	if err != nil {
		return nil, err
	}
	info.Type = provider.ResourceType_RESOURCE_TYPE_REFERENCE
	target, err := fs.getReferenceEntry(ctx, fn)
	if err != nil {
		return nil, err
	}
	info.Target = target
	return info, nil
}

func getResourceType(isDir bool) provider.ResourceType {
	if isDir {
		return provider.ResourceType_RESOURCE_TYPE_CONTAINER
	}
	return provider.ResourceType_RESOURCE_TYPE_FILE
}

func (fs *localfs) retrieveArbitraryMetadata(ctx context.Context, fn string, mdKeys []string) (*provider.ArbitraryMetadata, error) {
	md, err := fs.getMetadata(ctx, fn)
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error listing metadata")
	}
	var mdKey, mdVal string
	metadata := map[string]string{}

	mdKeysMap := make(map[string]struct{})
	for _, k := range mdKeys {
		mdKeysMap[k] = struct{}{}
	}

	var returnAllKeys bool
	if _, ok := mdKeysMap["*"]; len(mdKeys) == 0 || ok {
		returnAllKeys = true
	}

	for md.Next() {
		err = md.Scan(&mdKey, &mdVal)
		if err != nil {
			return nil, errors.Wrap(err, "localfs: error scanning db rows")
		}
		if _, ok := mdKeysMap[mdKey]; returnAllKeys || ok {
			metadata[mdKey] = mdVal
		}
	}
	return &provider.ArbitraryMetadata{
		Metadata: metadata,
	}, nil
}

// GetPathByID returns the path pointed by the file id
// In this implementation the file id is in the form `fileid-url_encoded_path`
func (fs *localfs) GetPathByID(ctx context.Context, ref *provider.ResourceId) (string, error) {
	var layout string
	if !fs.conf.DisableHome {
		var err error
		layout, err = fs.GetHome(ctx)
		if err != nil {
			return "", err
		}
	}
	unescapedID, err := url.QueryUnescape(ref.OpaqueId)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(unescapedID, "fileid-"+layout), nil
}

func (fs *localfs) DenyGrant(ctx context.Context, ref *provider.Reference, g *provider.Grantee) error {
	return errtypes.NotSupported("localfs: deny grant not supported")
}

func (fs *localfs) AddGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	fn, err := fs.resolve(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "localfs: error resolving ref")
	}
	fn = fs.wrap(ctx, fn)

	role, err := grants.GetACLPerm(g.Permissions)
	if err != nil {
		return errors.Wrap(err, "localfs: unknown set permissions")
	}

	granteeType, err := grants.GetACLType(g.Grantee.Type)
	if err != nil {
		return errors.Wrap(err, "localfs: error getting grantee type")
	}
	var grantee string
	switch granteeType {
	case acl.TypeUser:
		grantee = fmt.Sprintf("%s:%s:%s@%s", granteeType, g.Grantee.GetUserId().OpaqueId, utils.UserTypeToString(g.Grantee.GetUserId().Type), g.Grantee.GetUserId().Idp)
	case acl.TypeGroup:
		grantee = fmt.Sprintf("%s::%s@%s", granteeType, g.Grantee.GetGroupId().OpaqueId, g.Grantee.GetGroupId().Idp)
	}

	err = fs.addToACLDB(ctx, fn, grantee, role)
	if err != nil {
		return errors.Wrap(err, "localfs: error adding entry to DB")
	}

	return fs.propagate(ctx, fn)
}

func (fs *localfs) ListGrants(ctx context.Context, ref *provider.Reference) ([]*provider.Grant, error) {
	fn, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error resolving ref")
	}
	fn = fs.wrap(ctx, fn)

	g, err := fs.getACLs(ctx, fn)
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error listing grants")
	}
	var granteeID, role string
	var grantList []*provider.Grant

	for g.Next() {
		err = g.Scan(&granteeID, &role)
		if err != nil {
			return nil, errors.Wrap(err, "localfs: error scanning db rows")
		}
		grantSplit := strings.Split(granteeID, ":")
		grantee := &provider.Grantee{Type: grants.GetGranteeType(grantSplit[0])}
		parts := strings.Split(grantSplit[2], "@")
		switch grantSplit[0] {
		case acl.TypeUser:
			grantee.Id = &provider.Grantee_UserId{UserId: &userpb.UserId{OpaqueId: parts[0], Idp: parts[1], Type: utils.UserTypeMap(grantSplit[1])}}
		case acl.TypeGroup:
			grantee.Id = &provider.Grantee_GroupId{GroupId: &grouppb.GroupId{OpaqueId: parts[0], Idp: parts[1]}}
		}
		permissions := grants.GetGrantPermissionSet(role)

		grantList = append(grantList, &provider.Grant{
			Grantee:     grantee,
			Permissions: permissions,
		})
	}
	return grantList, nil

}

func (fs *localfs) RemoveGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	fn, err := fs.resolve(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "localfs: error resolving ref")
	}
	fn = fs.wrap(ctx, fn)

	granteeType, err := grants.GetACLType(g.Grantee.Type)
	if err != nil {
		return errors.Wrap(err, "localfs: error getting grantee type")
	}
	var grantee string
	switch granteeType {
	case acl.TypeUser:
		grantee = fmt.Sprintf("%s:%s@%s", granteeType, g.Grantee.GetUserId().OpaqueId, g.Grantee.GetUserId().Idp)
	case acl.TypeGroup:
		grantee = fmt.Sprintf("%s:%s@%s", granteeType, g.Grantee.GetGroupId().OpaqueId, g.Grantee.GetGroupId().Idp)
	}

	err = fs.removeFromACLDB(ctx, fn, grantee)
	if err != nil {
		return errors.Wrap(err, "localfs: error removing from DB")
	}

	return fs.propagate(ctx, fn)
}

func (fs *localfs) UpdateGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	return fs.AddGrant(ctx, ref, g)
}

func (fs *localfs) CreateReference(ctx context.Context, path string, targetURI *url.URL) error {
	var fn string
	switch {
	case fs.isShareFolder(ctx, path):
		fn = fs.wrapReferences(ctx, path)
	case fs.isDataTransfersFolder(ctx, path):
		fn = fs.wrap(ctx, path)
	default:
		return errtypes.PermissionDenied("localfs: cannot create references outside the share folder and data transfers folder")
	}

	err := os.MkdirAll(fn, 0700)
	if err != nil {
		if os.IsNotExist(err) {
			return errtypes.NotFound(fn)
		}
		return errors.Wrap(err, "localfs: error creating dir "+fn)
	}

	if err = fs.addToReferencesDB(ctx, fn, targetURI.String()); err != nil {
		return errors.Wrap(err, "localfs: error adding entry to DB")
	}

	return fs.propagate(ctx, fn)
}

// CreateStorageSpace creates a storage space
func (fs *localfs) CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error) {
	return nil, errtypes.NotSupported("unimplemented: CreateStorageSpace")
}

func (fs *localfs) SetArbitraryMetadata(ctx context.Context, ref *provider.Reference, md *provider.ArbitraryMetadata) error {

	np, err := fs.resolve(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "localfs: error resolving ref")
	}

	if fs.isShareFolderRoot(ctx, np) {
		return errtypes.PermissionDenied("localfs: cannot set metadata for the virtual share folder")
	}

	if fs.isShareFolderChild(ctx, np) {
		np = fs.wrapReferences(ctx, np)
	} else {
		np = fs.wrap(ctx, np)
	}

	fi, err := os.Stat(np)
	if err != nil {
		if os.IsNotExist(err) {
			return errtypes.NotFound(fs.unwrap(ctx, np))
		}
		return errors.Wrap(err, "localfs: error stating "+np)
	}

	if md.Metadata != nil {
		if val, ok := md.Metadata["mtime"]; ok {
			if mtime, err := parseMTime(val); err == nil {
				// updating mtime also updates atime
				if err := os.Chtimes(np, mtime, mtime); err != nil {
					return errors.Wrap(err, "could not set mtime")
				}
			} else {
				return errors.Wrap(err, "could not parse mtime")
			}
			delete(md.Metadata, "mtime")
		}

		if _, ok := md.Metadata["etag"]; ok {
			etag := calcEtag(ctx, fi)
			if etag != md.Metadata["etag"] {
				err = fs.addToMetadataDB(ctx, np, "etag", etag)
				if err != nil {
					return errors.Wrap(err, "localfs: error adding entry to DB")
				}
			}
			delete(md.Metadata, "etag")
		}

		if _, ok := md.Metadata["favorite"]; ok {
			u, err := getUser(ctx)
			if err != nil {
				return err
			}
			if uid := u.GetId(); uid != nil {
				usr := fmt.Sprintf("u:%s@%s", uid.GetOpaqueId(), uid.GetIdp())
				if err = fs.addToFavoritesDB(ctx, np, usr); err != nil {
					return errors.Wrap(err, "localfs: error adding entry to DB")
				}
			} else {
				return errors.Wrap(errtypes.UserRequired("userrequired"), "user has no id")
			}
			delete(md.Metadata, "favorite")
		}
	}

	for k, v := range md.Metadata {
		err = fs.addToMetadataDB(ctx, np, k, v)
		if err != nil {
			return errors.Wrap(err, "localfs: error adding entry to DB")
		}
	}

	return fs.propagate(ctx, np)
}

func parseMTime(v string) (t time.Time, err error) {
	p := strings.SplitN(v, ".", 2)
	var sec, nsec int64
	if sec, err = strconv.ParseInt(p[0], 10, 64); err == nil {
		if len(p) > 1 {
			nsec, err = strconv.ParseInt(p[1], 10, 64)
		}
	}
	return time.Unix(sec, nsec), err
}

func (fs *localfs) UnsetArbitraryMetadata(ctx context.Context, ref *provider.Reference, keys []string) error {

	np, err := fs.resolve(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "localfs: error resolving ref")
	}

	if fs.isShareFolderRoot(ctx, np) {
		return errtypes.PermissionDenied("localfs: cannot set metadata for the virtual share folder")
	}

	if fs.isShareFolderChild(ctx, np) {
		np = fs.wrapReferences(ctx, np)
	} else {
		np = fs.wrap(ctx, np)
	}

	_, err = os.Stat(np)
	if err != nil {
		if os.IsNotExist(err) {
			return errtypes.NotFound(fs.unwrap(ctx, np))
		}
		return errors.Wrap(err, "localfs: error stating "+np)
	}

	for _, k := range keys {
		switch k {
		case "favorite":
			u, err := getUser(ctx)
			if err != nil {
				return err
			}
			if uid := u.GetId(); uid != nil {
				usr := fmt.Sprintf("u:%s@%s", uid.GetOpaqueId(), uid.GetIdp())
				if err = fs.removeFromFavoritesDB(ctx, np, usr); err != nil {
					return errors.Wrap(err, "localfs: error removing entry from DB")
				}
			} else {
				return errors.Wrap(errtypes.UserRequired("userrequired"), "user has no id")
			}
		case "etag":
			return errors.Wrap(errtypes.NotSupported("unsetting etag not supported"), "could not unset metadata")
		case "mtime":
			return errors.Wrap(errtypes.NotSupported("unsetting mtime not supported"), "could not unset metadata")
		default:
			err = fs.removeFromMetadataDB(ctx, np, k)
			if err != nil {
				return errors.Wrap(err, "localfs: error adding entry to DB")
			}
		}
	}

	return fs.propagate(ctx, np)
}

// GetLock returns an existing lock on the given reference
func (fs *localfs) GetLock(ctx context.Context, ref *provider.Reference) (*provider.Lock, error) {
	return nil, errtypes.NotSupported("unimplemented")
}

// SetLock puts a lock on the given reference
func (fs *localfs) SetLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	return errtypes.NotSupported("unimplemented")
}

// RefreshLock refreshes an existing lock on the given reference
func (fs *localfs) RefreshLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock, existingLockID string) error {
	return errtypes.NotSupported("unimplemented")
}

// Unlock removes an existing lock from the given reference
func (fs *localfs) Unlock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	return errtypes.NotSupported("unimplemented")
}

func (fs *localfs) GetHome(ctx context.Context) (string, error) {
	if fs.conf.DisableHome {
		return "", errtypes.NotSupported("local: get home not supported")
	}

	u, err := getUser(ctx)
	if err != nil {
		err = errors.Wrap(err, "local: wrap: no user in ctx and home is enabled")
		return "", err
	}
	relativeHome := templates.WithUser(u, fs.conf.UserLayout)

	return relativeHome, nil
}

func (fs *localfs) CreateHome(ctx context.Context) error {
	if fs.conf.DisableHome {
		return errtypes.NotSupported("localfs: create home not supported")
	}

	homePaths := []string{fs.wrap(ctx, "/"), fs.wrapRecycleBin(ctx, "/"), fs.wrapVersions(ctx, "/"), fs.wrapReferences(ctx, fs.conf.ShareFolder)}

	for _, v := range homePaths {
		if err := fs.createHomeInternal(ctx, v); err != nil {
			return errors.Wrap(err, "local: error creating home dir "+v)
		}
	}

	return nil
}

func (fs *localfs) createHomeInternal(ctx context.Context, fn string) error {
	_, err := os.Stat(fn)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "local: error stating:"+fn)
		}
	}
	err = os.MkdirAll(fn, 0700)
	if err != nil {
		return errors.Wrap(err, "local: error creating dir:"+fn)
	}
	return nil
}

func (fs *localfs) CreateDir(ctx context.Context, ref *provider.Reference) error {

	fn, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil
	}

	if fs.isShareFolder(ctx, fn) {
		return errtypes.PermissionDenied("localfs: cannot create folder under the share folder")
	}

	fn = fs.wrap(ctx, fn)
	if _, err := os.Stat(fn); err == nil {
		return errtypes.AlreadyExists(fn)
	}
	err = os.Mkdir(fn, 0700)
	if err != nil {
		if os.IsNotExist(err) {
			return errtypes.PreconditionFailed(fn)
		}
		return errors.Wrap(err, "localfs: error creating dir "+fn)
	}

	return fs.propagate(ctx, path.Dir(fn))
}

// TouchFile as defined in the storage.FS interface
func (fs *localfs) TouchFile(ctx context.Context, ref *provider.Reference, _ bool, _ string) error {
	return fmt.Errorf("unimplemented: TouchFile")
}

func (fs *localfs) Delete(ctx context.Context, ref *provider.Reference) error {
	fn, err := fs.resolve(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "localfs: error resolving ref")
	}

	if fs.isShareFolderRoot(ctx, fn) {
		return errtypes.PermissionDenied("localfs: cannot delete the virtual share folder")
	}

	var fp string
	if fs.isShareFolderChild(ctx, fn) {
		fp = fs.wrapReferences(ctx, fn)
	} else {
		fp = fs.wrap(ctx, fn)
	}

	_, err = os.Stat(fp)
	if err != nil {
		if os.IsNotExist(err) {
			return errtypes.NotFound(fn)
		}
		return errors.Wrap(err, "localfs: error stating "+fp)
	}

	key := fmt.Sprintf("%s.d%d", path.Base(fn), time.Now().UnixNano()/int64(time.Millisecond))
	if err := os.Rename(fp, fs.wrapRecycleBin(ctx, key)); err != nil {
		return errors.Wrap(err, "localfs: could not delete item")
	}

	err = fs.addToRecycledDB(ctx, key, fn)
	if err != nil {
		return errors.Wrap(err, "localfs: error adding entry to DB")
	}

	return fs.propagate(ctx, path.Dir(fp))
}

func (fs *localfs) Move(ctx context.Context, oldRef, newRef *provider.Reference) error {
	oldName, err := fs.resolve(ctx, oldRef)
	if err != nil {
		return errors.Wrap(err, "localfs: error resolving ref")
	}

	newName, err := fs.resolve(ctx, newRef)
	if err != nil {
		return errors.Wrap(err, "localfs: error resolving ref")
	}

	if fs.isShareFolder(ctx, oldName) || fs.isShareFolder(ctx, newName) {
		return fs.moveReferences(ctx, oldName, newName)
	}

	oldName = fs.wrap(ctx, oldName)
	newName = fs.wrap(ctx, newName)

	if err := os.Rename(oldName, newName); err != nil {
		return errors.Wrap(err, "localfs: error moving "+oldName+" to "+newName)
	}

	if err := fs.copyMD(oldName, newName); err != nil {
		return errors.Wrap(err, "localfs: error copying metadata")
	}

	if err := fs.propagate(ctx, newName); err != nil {
		return err
	}
	if err := fs.propagate(ctx, path.Dir(oldName)); err != nil {
		return err
	}

	return nil
}

func (fs *localfs) moveReferences(ctx context.Context, oldName, newName string) error {

	if fs.isShareFolderRoot(ctx, oldName) || fs.isShareFolderRoot(ctx, newName) {
		return errtypes.PermissionDenied("localfs: cannot move/rename the virtual share folder")
	}

	// only rename of the reference is allowed, hence having the same basedir
	bold, _ := path.Split(oldName)
	bnew, _ := path.Split(newName)

	if bold != bnew {
		return errtypes.PermissionDenied("localfs: cannot move references under the virtual share folder")
	}

	oldName = fs.wrapReferences(ctx, oldName)
	newName = fs.wrapReferences(ctx, newName)

	if err := os.Rename(oldName, newName); err != nil {
		return errors.Wrap(err, "localfs: error moving "+oldName+" to "+newName)
	}

	if err := fs.copyMD(oldName, newName); err != nil {
		return errors.Wrap(err, "localfs: error copying metadata")
	}

	if err := fs.propagate(ctx, newName); err != nil {
		return err
	}
	if err := fs.propagate(ctx, path.Dir(oldName)); err != nil {
		return err
	}

	return nil
}

func (fs *localfs) GetMD(ctx context.Context, ref *provider.Reference, mdKeys []string, fieldMask []string) (*provider.ResourceInfo, error) {
	fn, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error resolving ref")
	}

	if !fs.conf.DisableHome {
		if fs.isShareFolder(ctx, fn) {
			return fs.getMDShareFolder(ctx, fn, mdKeys)
		}
	}

	fn = fs.wrap(ctx, fn)
	md, err := os.Stat(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errtypes.NotFound(fn)
		}
		return nil, errors.Wrap(err, "localfs: error stating "+fn)
	}

	return fs.normalize(ctx, md, fn, mdKeys)
}

func (fs *localfs) getMDShareFolder(ctx context.Context, p string, mdKeys []string) (*provider.ResourceInfo, error) {

	fn := fs.wrapReferences(ctx, p)
	md, err := os.Stat(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errtypes.NotFound(fn)
		}
		return nil, errors.Wrap(err, "localfs: error stating "+fn)
	}

	if fs.isShareFolderRoot(ctx, p) {
		return fs.normalize(ctx, md, fn, mdKeys)
	}
	return fs.convertToFileReference(ctx, md, fn, mdKeys)
}

func (fs *localfs) ListFolder(ctx context.Context, ref *provider.Reference, mdKeys, fieldMask []string) ([]*provider.ResourceInfo, error) {
	fn, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error resolving ref")
	}

	if fn == "/" {
		homeFiles, err := fs.listFolder(ctx, fn, mdKeys)
		if err != nil {
			return nil, err
		}
		if !fs.conf.DisableHome {
			sharedReferences, err := fs.listShareFolderRoot(ctx, fn, mdKeys)
			if err != nil {
				return nil, err
			}
			homeFiles = append(homeFiles, sharedReferences...)
		}
		return homeFiles, nil
	}

	if fs.isShareFolderRoot(ctx, fn) {
		return fs.listShareFolderRoot(ctx, fn, mdKeys)
	}

	if fs.isShareFolderChild(ctx, fn) {
		return nil, errtypes.PermissionDenied("localfs: error listing folders inside the shared folder, only file references are stored inside")
	}

	return fs.listFolder(ctx, fn, mdKeys)
}

func (fs *localfs) listFolder(ctx context.Context, fn string, mdKeys []string) ([]*provider.ResourceInfo, error) {

	fn = fs.wrap(ctx, fn)

	mds, err := os.ReadDir(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errtypes.NotFound(fn)
		}
		return nil, errors.Wrap(err, "localfs: error listing "+fn)
	}

	finfos := []*provider.ResourceInfo{}
	for _, md := range mds {
		mdInfo, _ := md.Info()
		info, err := fs.normalize(ctx, mdInfo, path.Join(fn, md.Name()), mdKeys)
		if err == nil {
			finfos = append(finfos, info)
		}
	}
	return finfos, nil
}

func (fs *localfs) listShareFolderRoot(ctx context.Context, home string, mdKeys []string) ([]*provider.ResourceInfo, error) {

	fn := fs.wrapReferences(ctx, home)

	mds, err := os.ReadDir(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errtypes.NotFound(fn)
		}
		return nil, errors.Wrap(err, "localfs: error listing "+fn)
	}

	finfos := []*provider.ResourceInfo{}
	for _, md := range mds {
		var info *provider.ResourceInfo
		var err error
		if fs.isShareFolderRoot(ctx, path.Join("/", md.Name())) {
			mdInfo, _ := md.Info()
			info, err = fs.normalize(ctx, mdInfo, path.Join(fn, md.Name()), mdKeys)
		} else {
			mdInfo, _ := md.Info()
			info, err = fs.convertToFileReference(ctx, mdInfo, path.Join(fn, md.Name()), mdKeys)
		}
		if err == nil {
			finfos = append(finfos, info)
		}
	}
	return finfos, nil
}

func (fs *localfs) Download(ctx context.Context, ref *provider.Reference, openReaderfunc func(*provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	fn, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, nil, errors.Wrap(err, "localfs: error resolving ref")
	}

	if fs.isShareFolder(ctx, fn) {
		return nil, nil, errtypes.PermissionDenied("localfs: cannot download under the virtual share folder")
	}

	fn = fs.wrap(ctx, fn)
	md, err := os.Stat(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, errtypes.NotFound(fn)
		}
		return nil, nil, errors.Wrap(err, "localfs: error stating "+fn)
	}

	ri, err := fs.normalize(ctx, md, fn, []string{"size", "mimetype", "etag"})
	if err != nil {
		return nil, nil, err
	}

	if !openReaderfunc(ri) {
		return ri, nil, nil
	}

	r, err := os.Open(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, errtypes.NotFound(fn)
		}
		return nil, nil, errors.Wrap(err, "localfs: error reading "+fn)
	}
	return ri, r, nil
}

func (fs *localfs) archiveRevision(ctx context.Context, np string) error {

	versionsDir := fs.wrapVersions(ctx, fs.unwrap(ctx, np))
	if err := os.MkdirAll(versionsDir, 0700); err != nil {
		return errors.Wrap(err, "localfs: error creating file versions dir "+versionsDir)
	}

	vp := path.Join(versionsDir, fmt.Sprintf("v%d", time.Now().UnixNano()/int64(time.Millisecond)))
	if err := os.Rename(np, vp); err != nil {
		return errors.Wrap(err, "localfs: error renaming from "+np+" to "+vp)
	}

	return nil
}

func (fs *localfs) ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error) {
	np, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error resolving ref")
	}

	if fs.isShareFolder(ctx, np) {
		return nil, errtypes.PermissionDenied("localfs: cannot list revisions under the virtual share folder")
	}

	versionsDir := fs.wrapVersions(ctx, np)
	revisions := []*provider.FileVersion{}
	mds, err := os.ReadDir(versionsDir)
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error reading"+versionsDir)
	}

	for i := range mds {
		// versions resemble v12345678
		version := mds[i].Name()[1:]

		mtime, err := strconv.Atoi(version)
		if err != nil {
			continue
		}
		mdsInfo, _ := mds[i].Info()
		revisions = append(revisions, &provider.FileVersion{
			Key:   version,
			Size:  uint64(mdsInfo.Size()),
			Mtime: uint64(mtime),
			Etag:  calcEtag(ctx, mdsInfo),
		})
	}
	return revisions, nil
}

func (fs *localfs) DownloadRevision(ctx context.Context, ref *provider.Reference, revisionKey string, openReaderfunc func(*provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	np, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, nil, errors.Wrap(err, "localfs: error resolving ref")
	}

	if fs.isShareFolder(ctx, np) {
		return nil, nil, errtypes.PermissionDenied("localfs: cannot download revisions under the virtual share folder")
	}

	versionsDir := fs.wrapVersions(ctx, np)
	vp := path.Join(versionsDir, revisionKey)

	md, err := os.Stat(vp)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, errtypes.NotFound(vp)
		}
		return nil, nil, errors.Wrap(err, "localfs: error stating "+vp)
	}

	ri, err := fs.normalize(ctx, md, vp, []string{"size", "mimetype", "etag"})
	if err != nil {
		return nil, nil, err
	}

	if !openReaderfunc(ri) {
		return ri, nil, nil
	}

	r, err := os.Open(vp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "localfs: error reading "+vp)
	}

	return ri, r, nil
}

func (fs *localfs) RestoreRevision(ctx context.Context, ref *provider.Reference, revisionKey string) error {
	np, err := fs.resolve(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "localfs: error resolving ref")
	}

	if fs.isShareFolder(ctx, np) {
		return errtypes.PermissionDenied("localfs: cannot restore revisions under the virtual share folder")
	}

	versionsDir := fs.wrapVersions(ctx, np)
	vp := path.Join(versionsDir, revisionKey)
	np = fs.wrap(ctx, np)

	// check revision exists
	vs, err := os.Stat(vp)
	if err != nil {
		if os.IsNotExist(err) {
			return errtypes.NotFound(revisionKey)
		}
		return errors.Wrap(err, "localfs: error stating "+vp)
	}

	if !vs.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", vp)
	}

	if err := fs.archiveRevision(ctx, np); err != nil {
		return err
	}

	if err := os.Rename(vp, np); err != nil {
		return errors.Wrap(err, "localfs: error renaming from "+vp+" to "+np)
	}

	return fs.propagate(ctx, np)
}

func (fs *localfs) PurgeRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string) error {
	rp := fs.wrapRecycleBin(ctx, key)

	if err := os.Remove(rp); err != nil {
		return errors.Wrap(err, "localfs: error deleting recycle item")
	}
	return nil
}

func (fs *localfs) EmptyRecycle(ctx context.Context, ref *provider.Reference) error {
	rp := fs.wrapRecycleBin(ctx, "/")

	if err := os.RemoveAll(rp); err != nil {
		return errors.Wrap(err, "localfs: error deleting recycle files")
	}
	if err := fs.createHomeInternal(ctx, rp); err != nil {
		return errors.Wrap(err, "localfs: error deleting recycle files")
	}
	return nil
}

func (fs *localfs) convertToRecycleItem(ctx context.Context, rp string, md os.FileInfo) *provider.RecycleItem {
	// trashbin items have filename.txt.d12345678
	suffix := path.Ext(md.Name())
	if len(suffix) == 0 || !strings.HasPrefix(suffix, ".d") {
		return nil
	}

	trashtime := suffix[2:]
	ttime, err := strconv.Atoi(trashtime)
	if err != nil {
		return nil
	}

	filePath, err := fs.getRecycledEntry(ctx, md.Name())
	if err != nil {
		return nil
	}

	return &provider.RecycleItem{
		Type: getResourceType(md.IsDir()),
		Key:  md.Name(),
		Ref:  &provider.Reference{Path: filePath},
		Size: uint64(md.Size()),
		DeletionTime: &types.Timestamp{
			Seconds: uint64(ttime),
		},
	}
}

func (fs *localfs) ListRecycle(ctx context.Context, ref *provider.Reference, key, relativePath string) ([]*provider.RecycleItem, error) {

	rp := fs.wrapRecycleBin(ctx, "/")

	mds, err := os.ReadDir(rp)
	if err != nil {
		return nil, errors.Wrap(err, "localfs: error listing deleted files")
	}
	items := []*provider.RecycleItem{}
	for i := range mds {
		mdsInfo, _ := mds[i].Info()
		ri := fs.convertToRecycleItem(ctx, rp, mdsInfo)
		if ri != nil {
			items = append(items, ri)
		}
	}
	return items, nil
}

func (fs *localfs) RestoreRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string, restoreRef *provider.Reference) error {

	suffix := path.Ext(key)
	if len(suffix) == 0 || !strings.HasPrefix(suffix, ".d") {
		return errors.New("localfs: invalid trash item suffix")
	}

	filePath, err := fs.getRecycledEntry(ctx, key)
	if err != nil {
		return errors.Wrap(err, "localfs: invalid key")
	}

	var localRestorePath string
	switch {
	case restoreRef != nil && restoreRef.Path != "":
		localRestorePath = fs.wrap(ctx, restoreRef.Path)
	case fs.isShareFolder(ctx, filePath):
		localRestorePath = fs.wrapReferences(ctx, filePath)
	default:
		localRestorePath = fs.wrap(ctx, filePath)
	}

	if _, err = os.Stat(localRestorePath); err == nil {
		return errors.New("localfs: can't restore - file already exists at original path")
	}

	rp := fs.wrapRecycleBin(ctx, key)
	if _, err = os.Stat(rp); err != nil {
		if os.IsNotExist(err) {
			return errtypes.NotFound(key)
		}
		return errors.Wrap(err, "localfs: error stating "+rp)
	}

	if err := os.Rename(rp, localRestorePath); err != nil {
		return errors.Wrap(err, "ocfs: could not restore item")
	}

	err = fs.removeFromRecycledDB(ctx, key)
	if err != nil {
		return errors.Wrap(err, "localfs: error adding entry to DB")
	}

	return fs.propagate(ctx, localRestorePath)
}

func (fs *localfs) ListStorageSpaces(ctx context.Context, filter []*provider.ListStorageSpacesRequest_Filter, unrestricted bool) ([]*provider.StorageSpace, error) {
	return nil, errtypes.NotSupported("list storage spaces")
}

// UpdateStorageSpace updates a storage space
func (fs *localfs) UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error) {
	return nil, errtypes.NotSupported("update storage space")
}

// DeleteStorageSpace deletes a storage space
func (fs *localfs) DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) error {
	return errtypes.NotSupported("delete storage space")
}

func (fs *localfs) propagate(ctx context.Context, leafPath string) error {

	var root string
	if fs.isShareFolderChild(ctx, leafPath) || strings.HasSuffix(path.Clean(leafPath), fs.conf.ShareFolder) {
		root = fs.wrapReferences(ctx, "/")
	} else {
		root = fs.wrap(ctx, "/")
	}

	if !strings.HasPrefix(leafPath, root) {
		return errors.New("internal path: " + leafPath + " outside root: " + root)
	}

	fi, err := os.Stat(leafPath)
	if err != nil {
		return err
	}

	parts := strings.Split(strings.TrimPrefix(leafPath, root), "/")
	// root never ends in / so the split returns an empty first element, which we can skip
	// we do not need to chmod the last element because it is the leaf path (< and not <= comparison)
	for i := 1; i < len(parts); i++ {
		if err := os.Chtimes(root, fi.ModTime(), fi.ModTime()); err != nil {
			return err
		}
		root = path.Join(root, parts[i])
	}
	return nil
}
