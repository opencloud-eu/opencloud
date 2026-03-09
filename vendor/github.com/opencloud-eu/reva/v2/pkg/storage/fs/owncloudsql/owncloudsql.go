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

package owncloudsql

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"database/sql"
	"fmt"
	"hash/adler32"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/pkg/xattr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/opencloud-eu/reva/v2/internal/grpc/services/storageprovider"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/logger"
	"github.com/opencloud-eu/reva/v2/pkg/mime"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/owncloudsql/filecache"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/chunking"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
)

const (
	// Currently,extended file attributes have four separated
	// namespaces (user, trusted, security and system) followed by a dot.
	// A non root user can only manipulate the user. namespace, which is what
	// we will use to store ownCloud specific metadata. To prevent name
	// collisions with other apps We are going to introduce a sub namespace
	// "user.oc."
	ocPrefix string = "user.oc."

	mdPrefix     string = ocPrefix + "md."   // arbitrary metadata
	favPrefix    string = ocPrefix + "fav."  // favorite flag, per user
	etagPrefix   string = ocPrefix + "etag." // allow overriding a calculated etag with one from the extended attributes
	checksumsKey string = "http://owncloud.org/ns/checksums"
)

var defaultPermissions *provider.ResourcePermissions = &provider.ResourcePermissions{
	// no permissions
}
var ownerPermissions *provider.ResourcePermissions = &provider.ResourcePermissions{
	// all permissions
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
	DenyGrant:            true,
}

func init() {
	registry.Register("owncloudsql", New)
}

type config struct {
	DataDirectory            string `mapstructure:"datadirectory"`
	UploadInfoDir            string `mapstructure:"upload_info_dir"`
	DeprecatedShareDirectory string `mapstructure:"sharedirectory"`
	ShareFolder              string `mapstructure:"share_folder"`
	UserLayout               string `mapstructure:"user_layout"`
	EnableHome               bool   `mapstructure:"enable_home"`
	UserProviderEndpoint     string `mapstructure:"userprovidersvc"`
	DbUsername               string `mapstructure:"dbusername"`
	DbPassword               string `mapstructure:"dbpassword"`
	DbHost                   string `mapstructure:"dbhost"`
	DbPort                   int    `mapstructure:"dbport"`
	DbName                   string `mapstructure:"dbname"`
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

func (c *config) init(m map[string]interface{}) {
	if c.UserLayout == "" {
		c.UserLayout = "{{.Username}}"
	}
	if c.UploadInfoDir == "" {
		c.UploadInfoDir = "/var/tmp/reva/uploadinfo"
	}
	// fallback for old config
	if c.DeprecatedShareDirectory != "" {
		c.ShareFolder = c.DeprecatedShareDirectory
	}
	if c.ShareFolder == "" {
		c.ShareFolder = "/Shares"
	}
	// ensure share folder always starts with slash
	c.ShareFolder = filepath.Join("/", c.ShareFolder)

	c.UserProviderEndpoint = sharedconf.GetGatewaySVC(c.UserProviderEndpoint)
}

// New returns an implementation to of the storage.FS interface that talk to
// a local filesystem.
func New(m map[string]interface{}, _ events.Stream, _ *zerolog.Logger) (storage.FS, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}
	c.init(m)

	// c.DataDirectory should never end in / unless it is the root?
	c.DataDirectory = filepath.Clean(c.DataDirectory)

	// create datadir if it does not exist
	err = os.MkdirAll(c.DataDirectory, 0700)
	if err != nil {
		logger.New().Error().Err(err).
			Str("path", c.DataDirectory).
			Msg("could not create datadir")
	}

	err = os.MkdirAll(c.UploadInfoDir, 0700)
	if err != nil {
		logger.New().Error().Err(err).
			Str("path", c.UploadInfoDir).
			Msg("could not create uploadinfo dir")
	}

	dbSource := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", c.DbUsername, c.DbPassword, c.DbHost, c.DbPort, c.DbName)
	filecache, err := filecache.NewMysql(dbSource)
	if err != nil {
		return nil, err
	}

	return &owncloudsqlfs{
		c:            c,
		chunkHandler: chunking.NewChunkHandler(c.UploadInfoDir),
		filecache:    filecache,
	}, nil
}

type owncloudsqlfs struct {
	c            *config
	chunkHandler *chunking.ChunkHandler
	filecache    *filecache.Cache
}

func (fs *owncloudsqlfs) Shutdown(ctx context.Context) error {
	return nil
}

// owncloudsql stores files in the files subfolder
// the incoming path starts with /<username>, so we need to insert the files subfolder into the path
// and prefix the data directory
// TODO the path handed to a storage provider should not contain the username
func (fs *owncloudsqlfs) toInternalPath(ctx context.Context, sp string) (ip string) {
	if fs.c.EnableHome {
		u := ctxpkg.ContextMustGetUser(ctx)
		layout := templates.WithUser(u, fs.c.UserLayout)
		ip = filepath.Join(fs.c.DataDirectory, layout, "files", sp)
	} else {
		// trim all /
		sp = strings.Trim(sp, "/")
		// p = "" or
		// p = <username> or
		// p = <username>/foo/bar.txt
		segments := strings.SplitN(sp, "/", 2)

		if len(segments) == 1 && segments[0] == "" {
			ip = fs.c.DataDirectory
			return
		}

		// parts[0] contains the username or userid.
		u, err := fs.getUser(ctx, segments[0])
		if err != nil {
			// TODO return invalid internal path?
			return
		}
		layout := templates.WithUser(u, fs.c.UserLayout)

		if len(segments) == 1 {
			// parts = "<username>"
			ip = filepath.Join(fs.c.DataDirectory, layout, "files")
		} else {
			// parts = "<username>", "foo/bar.txt"
			ip = filepath.Join(fs.c.DataDirectory, layout, "files", segments[1])
		}

	}
	return
}

// owncloudsql stores versions in the files_versions subfolder
// the incoming path starts with /<username>, so we need to insert the files subfolder into the path
// and prefix the data directory
// TODO the path handed to a storage provider should not contain the username
func (fs *owncloudsqlfs) getVersionsPath(ctx context.Context, ip string) string {
	// ip = /path/to/data/<username>/files/foo/bar.txt
	// remove data dir
	if fs.c.DataDirectory != "/" {
		// fs.c.DataDirectory is a clean path, so it never ends in /
		ip = strings.TrimPrefix(ip, fs.c.DataDirectory)
	}
	// ip = /<username>/files/foo/bar.txt
	parts := strings.SplitN(ip, "/", 4)

	// parts[1] contains the username or userid.
	u, err := fs.getUser(ctx, parts[1])
	if err != nil {
		// TODO return invalid internal path?
		return ""
	}
	layout := templates.WithUser(u, fs.c.UserLayout)

	switch len(parts) {
	case 3:
		// parts = "", "<username>"
		return filepath.Join(fs.c.DataDirectory, layout, "files_versions")
	case 4:
		// parts = "", "<username>", "foo/bar.txt"
		return filepath.Join(fs.c.DataDirectory, layout, "files_versions", parts[3])
	default:
		return "" // TODO Must not happen?
	}

}

// owncloudsql stores trashed items in the files_trashbin subfolder of a users home
func (fs *owncloudsqlfs) getRecyclePath(ctx context.Context) (string, error) {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		err := errors.Wrap(errtypes.UserRequired("userrequired"), "error getting user from ctx")
		return "", err
	}
	layout := templates.WithUser(u, fs.c.UserLayout)
	return fs.getRecyclePathForUser(layout)
}

func (fs *owncloudsqlfs) getRecyclePathForUser(user string) (string, error) {
	return filepath.Join(fs.c.DataDirectory, user, "files_trashbin/files"), nil
}

func (fs *owncloudsqlfs) getVersionRecyclePath(ctx context.Context) (string, error) {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		err := errors.Wrap(errtypes.UserRequired("userrequired"), "error getting user from ctx")
		return "", err
	}
	layout := templates.WithUser(u, fs.c.UserLayout)
	return filepath.Join(fs.c.DataDirectory, layout, "files_trashbin/versions"), nil
}

func (fs *owncloudsqlfs) toDatabasePath(ip string) string {
	owner := fs.getOwner(ip)
	trim := filepath.Join(fs.c.DataDirectory, owner)
	p := strings.TrimPrefix(ip, trim)
	p = strings.TrimPrefix(p, "/")
	return p
}

func (fs *owncloudsqlfs) toResourcePath(ip, owner string) string {
	trim := filepath.Join(fs.c.DataDirectory, owner, "files")
	p := strings.TrimPrefix(ip, trim)
	p = strings.TrimPrefix(p, "/")
	// root directory
	if p == "" {
		p = "."
	}
	return p
}

func (fs *owncloudsqlfs) toStoragePath(ctx context.Context, ip string) (sp string) {
	if fs.c.EnableHome {
		u := ctxpkg.ContextMustGetUser(ctx)
		layout := templates.WithUser(u, fs.c.UserLayout)
		trim := filepath.Join(fs.c.DataDirectory, layout, "files")
		sp = strings.TrimPrefix(ip, trim)
		// root directory
		if sp == "" {
			sp = "/"
		}
	} else {
		// ip = /data/<username>/files/foo/bar.txt
		// remove data dir
		if fs.c.DataDirectory != "/" {
			// fs.c.DataDirectory is a clean path, so it never ends in /
			ip = strings.TrimPrefix(ip, fs.c.DataDirectory)
			// ip = /<username>/files/foo/bar.txt
		}

		segments := strings.SplitN(ip, "/", 4)
		// parts = "", "<username>", "files", "foo/bar.txt"
		switch len(segments) {
		case 1:
			sp = "/"
		case 2:
			sp = filepath.Join("/", segments[1])
		case 3:
			sp = filepath.Join("/", segments[1])
		default:
			sp = filepath.Join(segments[1], segments[3])
		}
	}
	log := appctx.GetLogger(ctx)
	log.Debug().Str("driver", "owncloudsql").Str("ipath", ip).Str("spath", sp).Msg("toStoragePath")
	return
}

// TODO the owner needs to come from a different place
func (fs *owncloudsqlfs) getOwner(ip string) string {
	ip = strings.TrimPrefix(ip, fs.c.DataDirectory)
	parts := strings.SplitN(ip, "/", 3)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// TODO cache user lookup
func (fs *owncloudsqlfs) getUser(ctx context.Context, usernameOrID string) (id *userpb.User, err error) {
	u := ctxpkg.ContextMustGetUser(ctx)
	// check if username matches and id is set
	if u.Username == usernameOrID && u.Id != nil && u.Id.OpaqueId != "" {
		return u, nil
	}
	// check if userid matches and username is set
	if u.Id != nil && u.Id.OpaqueId == usernameOrID && u.Username != "" {
		return u, nil
	}
	// look up at the userprovider

	// parts[0] contains the username or userid. use  user service to look up id
	c, err := pool.GetUserProviderServiceClient(fs.c.UserProviderEndpoint)
	if err != nil {
		appctx.GetLogger(ctx).
			Error().Err(err).
			Str("userprovidersvc", fs.c.UserProviderEndpoint).
			Str("usernameOrID", usernameOrID).
			Msg("could not get user provider client")
		return nil, err
	}
	res, err := c.GetUser(ctx, &userpb.GetUserRequest{
		UserId: &userpb.UserId{OpaqueId: usernameOrID},
	})
	if err != nil {
		appctx.GetLogger(ctx).
			Error().Err(err).
			Str("userprovidersvc", fs.c.UserProviderEndpoint).
			Str("usernameOrID", usernameOrID).
			Msg("could not get user")
		return nil, err
	}

	if res.Status.Code == rpc.Code_CODE_NOT_FOUND {
		appctx.GetLogger(ctx).
			Error().
			Str("userprovidersvc", fs.c.UserProviderEndpoint).
			Str("usernameOrID", usernameOrID).
			Interface("status", res.Status).
			Msg("user not found by id. Trying by name")

		var cres *userpb.GetUserByClaimResponse
		cres, err = c.GetUserByClaim(ctx, &userpb.GetUserByClaimRequest{
			Claim: "username",
			Value: usernameOrID,
		})
		if err != nil {
			appctx.GetLogger(ctx).
				Error().Err(err).
				Str("userprovidersvc", fs.c.UserProviderEndpoint).
				Str("usernameOrID", usernameOrID).
				Msg("could not get user by username")
			return nil, err
		}
		if cres.Status.Code == rpc.Code_CODE_NOT_FOUND {
			appctx.GetLogger(ctx).
				Error().
				Str("userprovidersvc", fs.c.UserProviderEndpoint).
				Str("usernameOrID", usernameOrID).
				Interface("status", cres.Status).
				Msg("user not found by username")
			return nil, fmt.Errorf("user not found")
		}
		res.User = cres.User
		res.Status = cres.Status
	}

	if res.Status.Code != rpc.Code_CODE_OK {
		appctx.GetLogger(ctx).
			Error().
			Str("userprovidersvc", fs.c.UserProviderEndpoint).
			Str("usernameOrID", usernameOrID).
			Interface("status", res.Status).
			Msg("user lookup failed")
		return nil, fmt.Errorf("user lookup failed")
	}
	return res.User, nil
}

// permissionSet returns the permission set for the current user
func (fs *owncloudsqlfs) permissionSet(ctx context.Context, owner *userpb.UserId) *provider.ResourcePermissions {
	if owner == nil {
		return &provider.ResourcePermissions{
			Stat: true,
		}
	}
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

func (fs *owncloudsqlfs) getStorage(ctx context.Context, ip string) (int, error) {
	return fs.filecache.GetNumericStorageID(ctx, "home::"+fs.getOwner(ip))
}

func (fs *owncloudsqlfs) getUserStorage(ctx context.Context, user string) (int, error) {
	id, err := fs.filecache.GetNumericStorageID(ctx, "home::"+user)
	if err != nil {
		id, err = fs.filecache.CreateStorage(ctx, "home::"+user)
	}
	return id, err
}

func (fs *owncloudsqlfs) convertToResourceInfo(ctx context.Context, entry *filecache.File, ip string, mdKeys []string) (*provider.ResourceInfo, error) {
	mdKeysMap := make(map[string]struct{})
	for _, k := range mdKeys {
		mdKeysMap[k] = struct{}{}
	}

	var returnAllKeys bool
	if _, ok := mdKeysMap["*"]; len(mdKeys) == 0 || ok {
		returnAllKeys = true
	}
	owner := fs.getOwner(ip)
	path := fs.toResourcePath(ip, owner)
	isDir := entry.MimeTypeString == "httpd/unix-directory"
	ri := &provider.ResourceInfo{
		Id: &provider.ResourceId{
			// return ownclouds numeric storage id as the space id!
			SpaceId: strconv.Itoa(entry.Storage), OpaqueId: strconv.Itoa(entry.ID),
		},
		Path:     filepath.Base(path), // we currently only return the name, decomposedfs returns the path if the request was path based. is that even still possible?
		Type:     getResourceType(isDir),
		Etag:     entry.Etag,
		MimeType: entry.MimeTypeString,
		Size:     uint64(entry.Size),
		Mtime: &types.Timestamp{
			Seconds: uint64(entry.MTime),
		},
		ArbitraryMetadata: &provider.ArbitraryMetadata{
			Metadata: map[string]string{}, // TODO aduffeck: which metadata needs to go in here?
		},
	}
	// omit parentid for root
	if path != "." {
		ri.Name = entry.Name
		ri.ParentId = &provider.ResourceId{
			// return ownclouds numeric storage id as the space id!
			SpaceId: strconv.Itoa(entry.Storage), OpaqueId: strconv.Itoa(entry.Parent),
		}
	}

	if owner, err := fs.getUser(ctx, owner); err == nil {
		ri.Owner = owner.Id
	} else {
		appctx.GetLogger(ctx).Error().Err(err).Msg("error getting owner")
	}

	ri.PermissionSet = fs.permissionSet(ctx, ri.Owner)

	// checksums
	if !isDir {
		if _, checksumRequested := mdKeysMap[checksumsKey]; returnAllKeys || checksumRequested {
			// TODO which checksum was requested? sha1 adler32 or md5? for now hardcode sha1?
			readChecksumIntoResourceChecksum(ctx, entry.Checksum, storageprovider.XSSHA1, ri)
			readChecksumIntoOpaque(ctx, entry.Checksum, storageprovider.XSMD5, ri)
			readChecksumIntoOpaque(ctx, ip, storageprovider.XSAdler32, ri)
		}
	}

	return ri, nil
}

// GetPathByID returns the storage relative path for the file id, without the internal namespace
func (fs *owncloudsqlfs) GetPathByID(ctx context.Context, id *provider.ResourceId) (string, error) {
	ip, err := fs.resolve(ctx, &provider.Reference{ResourceId: id})
	if err != nil {
		return "", err
	}

	// check permissions
	if perm, err := fs.readPermissions(ctx, ip); err == nil {
		if !perm.GetPath {
			return "", errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return "", errtypes.NotFound(fs.toStoragePath(ctx, ip))
		}
		return "", errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	return fs.toStoragePath(ctx, ip), nil
}

// resolve takes in a request path or request id and converts it to an internal path.
func (fs *owncloudsqlfs) resolve(ctx context.Context, ref *provider.Reference) (string, error) {

	if ref.GetResourceId() != nil {
		p, err := fs.filecache.Path(ctx, ref.GetResourceId().OpaqueId)
		if err != nil {
			return "", err
		}
		p = strings.TrimPrefix(p, "files/")
		if !fs.c.EnableHome {
			owner, err := fs.filecache.GetStorageOwnerByFileID(ctx, ref.GetResourceId().OpaqueId)
			if err != nil {
				return "", err
			}
			p = filepath.Join(owner, p)
		}
		if ref.GetPath() != "" {
			p = filepath.Join(p, ref.GetPath())
		}
		return fs.toInternalPath(ctx, p), nil
	}

	if ref.GetPath() != "" {
		return fs.toInternalPath(ctx, ref.GetPath()), nil
	}

	// reference is invalid
	return "", fmt.Errorf("invalid reference %+v", ref)
}

func (fs *owncloudsqlfs) DenyGrant(ctx context.Context, ref *provider.Reference, g *provider.Grantee) error {
	return errtypes.NotSupported("owncloudsqlfs: deny grant not supported")
}

func (fs *owncloudsqlfs) AddGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	return errtypes.NotSupported("owncloudsqlfs: add grant not supported")
}

func (fs *owncloudsqlfs) readPermissions(ctx context.Context, ip string) (p *provider.ResourcePermissions, err error) {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		appctx.GetLogger(ctx).Debug().Str("ipath", ip).Msg("no user in context, returning default permissions")
		return defaultPermissions, nil
	}
	// check if the current user is the owner
	owner := fs.getOwner(ip)
	if owner == u.Username {
		appctx.GetLogger(ctx).Debug().Str("ipath", ip).Msg("user is owner, returning owner permissions")
		return ownerPermissions, nil
	}

	// otherwise this is a share
	ownerStorageID, err := fs.filecache.GetNumericStorageID(ctx, "home::"+owner)
	if err != nil {
		return nil, err
	}
	entry, err := fs.filecache.Get(ctx, ownerStorageID, fs.toDatabasePath(ip))
	if err != nil {
		return nil, err
	}
	perms, err := conversions.NewPermissions(entry.Permissions)
	if err != nil {
		return nil, err
	}
	return conversions.RoleFromOCSPermissions(perms, nil).CS3ResourcePermissions(), nil
}

// The os not exists error is buried inside the xattr error,
// so we cannot just use os.IsNotExists().
func isNotFound(err error) bool {
	if xerr, ok := err.(*xattr.Error); ok {
		if serr, ok2 := xerr.Err.(syscall.Errno); ok2 {
			return serr == syscall.ENOENT
		}
	}
	return false
}

func (fs *owncloudsqlfs) ListGrants(ctx context.Context, ref *provider.Reference) (grants []*provider.Grant, err error) {
	return []*provider.Grant{}, nil // nop
}

func (fs *owncloudsqlfs) RemoveGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) (err error) {
	return nil // nop
}

func (fs *owncloudsqlfs) UpdateGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	return nil // nop
}

func (fs *owncloudsqlfs) CreateHome(ctx context.Context) error {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		err := errors.Wrap(errtypes.UserRequired("userrequired"), "error getting user from ctx")
		return err
	}
	return fs.createHomeForUser(ctx, templates.WithUser(u, fs.c.UserLayout))
}

func (fs *owncloudsqlfs) createHomeForUser(ctx context.Context, user string) error {
	homePaths := []string{
		filepath.Join(fs.c.DataDirectory, user),
		filepath.Join(fs.c.DataDirectory, user, "files"),
		filepath.Join(fs.c.DataDirectory, user, "files_trashbin"),
		filepath.Join(fs.c.DataDirectory, user, "files_trashbin/files"),
		filepath.Join(fs.c.DataDirectory, user, "files_trashbin/versions"),
		filepath.Join(fs.c.DataDirectory, user, "uploads"),
	}

	storageID, err := fs.getUserStorage(ctx, user)
	if err != nil {
		return err
	}
	for _, v := range homePaths {
		if err := os.MkdirAll(v, 0755); err != nil {
			return errors.Wrap(err, "owncloudsql: error creating home path: "+v)
		}

		fi, err := os.Stat(v)
		if err != nil {
			return err
		}
		data := map[string]interface{}{
			"path":        fs.toDatabasePath(v),
			"etag":        calcEtag(ctx, fi),
			"mimetype":    "httpd/unix-directory",
			"permissions": 31, // 1: READ, 2: UPDATE, 4: CREATE, 8: DELETE, 16: SHARE
		}

		allowEmptyParent := v == filepath.Join(fs.c.DataDirectory, user) // the root doesn't have a parent
		_, err = fs.filecache.InsertOrUpdate(ctx, storageID, data, allowEmptyParent)
		if err != nil {
			return err
		}
	}
	return nil
}

// If home is enabled, the relative home is always the empty string
func (fs *owncloudsqlfs) GetHome(ctx context.Context) (string, error) {
	if !fs.c.EnableHome {
		return "", errtypes.NotSupported("owncloudsql: get home not supported")
	}
	return "", nil
}

func (fs *owncloudsqlfs) CreateDir(ctx context.Context, ref *provider.Reference) (err error) {

	ip, err := fs.resolve(ctx, ref)
	if err != nil {
		return err
	}

	// check permissions of parent dir
	if perm, err := fs.readPermissions(ctx, filepath.Dir(ip)); err == nil {
		if !perm.CreateContainer {
			return errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return errtypes.PreconditionFailed(ref.Path)
		}
		return errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	if err = os.Mkdir(ip, 0700); err != nil {
		if os.IsNotExist(err) {
			return errtypes.PreconditionFailed(ref.Path)
		}
		if os.IsExist(err) {
			return errtypes.AlreadyExists(ref.Path)
		}
		return errors.Wrap(err, "owncloudsql: error creating dir "+fs.toStoragePath(ctx, filepath.Dir(ip)))
	}

	fi, err := os.Stat(ip)
	if err != nil {
		return err
	}
	mtime := time.Now().Unix()

	permissions := 31 // 1: READ, 2: UPDATE, 4: CREATE, 8: DELETE, 16: SHARE
	if perm, err := fs.readPermissions(ctx, filepath.Dir(ip)); err == nil {
		permissions = int(conversions.RoleFromResourcePermissions(perm, false).OCSPermissions()) // inherit permissions of parent
	}
	data := map[string]interface{}{
		"path":          fs.toDatabasePath(ip),
		"etag":          calcEtag(ctx, fi),
		"mimetype":      "httpd/unix-directory",
		"permissions":   permissions,
		"mtime":         mtime,
		"storage_mtime": mtime,
	}
	storageID, err := fs.getStorage(ctx, ip)
	if err != nil {
		return err
	}
	_, err = fs.filecache.InsertOrUpdate(ctx, storageID, data, false)
	if err != nil {
		if err != nil {
			return err
		}
	}

	return fs.propagate(ctx, filepath.Dir(ip))
}

// TouchFile as defined in the storage.FS interface
func (fs *owncloudsqlfs) TouchFile(ctx context.Context, ref *provider.Reference, markprocessing bool, mtime string) error {
	ip, err := fs.resolve(ctx, ref)
	if err != nil {
		return err
	}

	// check permissions of parent dir
	parentPerms, err := fs.readPermissions(ctx, filepath.Dir(ip))
	if err == nil {
		if !parentPerms.InitiateFileUpload {
			return errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return errtypes.NotFound(ref.Path)
		}
		return errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	_, err = os.Create(ip)
	if err != nil {
		if os.IsNotExist(err) {
			return errtypes.NotFound(ref.Path)
		}
		// FIXME we also need already exists error, webdav expects 405 MethodNotAllowed
		return errors.Wrap(err, "owncloudsql: error creating file "+fs.toStoragePath(ctx, filepath.Dir(ip)))
	}

	if err = os.Chmod(ip, 0700); err != nil {
		return errors.Wrap(err, "owncloudsql: error setting file permissions on "+fs.toStoragePath(ctx, filepath.Dir(ip)))
	}

	fi, err := os.Stat(ip)
	if err != nil {
		return err
	}
	storageMtime := time.Now().Unix()
	mt := storageMtime
	if mtime != "" {
		t, err := strconv.Atoi(mtime)
		if err != nil {
			log.Info().
				Str("owncloudsql", ip).
				Msg("error mtime conversion. mtine set to system time")
		}
		mt = time.Unix(int64(t), 0).Unix()
	}

	data := map[string]interface{}{
		"path":          fs.toDatabasePath(ip),
		"etag":          calcEtag(ctx, fi),
		"mimetype":      mime.Detect(false, ip),
		"permissions":   int(conversions.RoleFromResourcePermissions(parentPerms, false).OCSPermissions()), // inherit permissions of parent
		"mtime":         mt,
		"storage_mtime": storageMtime,
	}
	storageID, err := fs.getStorage(ctx, ip)
	if err != nil {
		return err
	}
	_, err = fs.filecache.InsertOrUpdate(ctx, storageID, data, false)
	if err != nil {
		return err
	}

	return fs.propagate(ctx, filepath.Dir(ip))
}

func (fs *owncloudsqlfs) CreateReference(ctx context.Context, sp string, targetURI *url.URL) error {
	return errtypes.NotSupported("owncloudsql: operation not supported")
}

func (fs *owncloudsqlfs) setMtime(ctx context.Context, ip string, mtime string) error {
	log := appctx.GetLogger(ctx)
	if mt, err := parseMTime(mtime); err == nil {
		// updating mtime also updates atime
		if err := os.Chtimes(ip, mt, mt); err != nil {
			log.Error().Err(err).
				Str("ipath", ip).
				Time("mtime", mt).
				Msg("could not set mtime")
			return errors.Wrap(err, "could not set mtime")
		}
	} else {
		log.Error().Err(err).
			Str("ipath", ip).
			Str("mtime", mtime).
			Msg("could not parse mtime")
		return errors.Wrap(err, "could not parse mtime")
	}
	return nil
}
func (fs *owncloudsqlfs) SetArbitraryMetadata(ctx context.Context, ref *provider.Reference, md *provider.ArbitraryMetadata) (err error) {
	log := appctx.GetLogger(ctx)

	var ip string
	if ip, err = fs.resolve(ctx, ref); err != nil {
		return errors.Wrap(err, "owncloudsql: error resolving reference")
	}

	// check permissions
	if perm, err := fs.readPermissions(ctx, ip); err == nil {
		if !perm.InitiateFileUpload { // TODO add dedicated permission?
			return errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return errtypes.NotFound(fs.toStoragePath(ctx, filepath.Dir(ip)))
		}
		return errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	var fi os.FileInfo
	fi, err = os.Stat(ip)
	if err != nil {
		if os.IsNotExist(err) {
			return errtypes.NotFound(fs.toStoragePath(ctx, ip))
		}
		return errors.Wrap(err, "owncloudsql: error stating "+ip)
	}

	errs := []error{}

	if md.Metadata != nil {
		if val, ok := md.Metadata["mtime"]; ok {
			err := fs.setMtime(ctx, ip, val)
			if err != nil {
				errs = append(errs, errors.Wrap(err, "could not set mtime"))
			}
			// remove from metadata
			delete(md.Metadata, "mtime")
		}
		// TODO(jfd) special handling for atime?
		// TODO(jfd) allow setting birth time (btime)?
		// TODO(jfd) any other metadata that is interesting? fileid?
		if val, ok := md.Metadata["etag"]; ok {
			etag := calcEtag(ctx, fi)
			val = fmt.Sprintf("\"%s\"", strings.Trim(val, "\""))
			if etag == val {
				log.Debug().
					Str("ipath", ip).
					Str("etag", val).
					Msg("ignoring request to update identical etag")
			} else
			// etag is only valid until the calculated etag changes
			// TODO(jfd) cleanup in a batch job
			if err := xattr.Set(ip, etagPrefix+etag, []byte(val)); err != nil {
				log.Error().Err(err).
					Str("ipath", ip).
					Str("calcetag", etag).
					Str("etag", val).
					Msg("could not set etag")
				errs = append(errs, errors.Wrap(err, "could not set etag"))
			}
			delete(md.Metadata, "etag")
		}
		if val, ok := md.Metadata["http://owncloud.org/ns/favorite"]; ok {
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
			if u, ok := ctxpkg.ContextGetUser(ctx); ok {
				// the favorite flag is specific to the user, so we need to incorporate the userid
				if uid := u.GetId(); uid != nil {
					fa := fmt.Sprintf("%s%s@%s", favPrefix, uid.GetOpaqueId(), uid.GetIdp())
					if err := xattr.Set(ip, fa, []byte(val)); err != nil {
						log.Error().Err(err).
							Str("ipath", ip).
							Interface("user", u).
							Str("key", fa).
							Msg("could not set favorite flag")
						errs = append(errs, errors.Wrap(err, "could not set favorite flag"))
					}
				} else {
					log.Error().
						Str("ipath", ip).
						Interface("user", u).
						Msg("user has no id")
					errs = append(errs, errors.Wrap(errtypes.UserRequired("userrequired"), "user has no id"))
				}
			} else {
				log.Error().
					Str("ipath", ip).
					Interface("user", u).
					Msg("error getting user from ctx")
				errs = append(errs, errors.Wrap(errtypes.UserRequired("userrequired"), "error getting user from ctx"))
			}
			// remove from metadata
			delete(md.Metadata, "http://owncloud.org/ns/favorite")
		}
	}
	for k, v := range md.Metadata {
		if err := xattr.Set(ip, mdPrefix+k, []byte(v)); err != nil {
			log.Error().Err(err).
				Str("ipath", ip).
				Str("key", k).
				Str("val", v).
				Msg("could not set metadata")
			errs = append(errs, errors.Wrap(err, "could not set metadata"))
		}
	}
	switch len(errs) {
	case 0:
		return fs.propagate(ctx, ip)
	case 1:
		return errs[0]
	default:
		// TODO how to return multiple errors?
		return errors.New("multiple errors occurred, see log for details")
	}
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

func (fs *owncloudsqlfs) UnsetArbitraryMetadata(ctx context.Context, ref *provider.Reference, keys []string) (err error) {
	log := appctx.GetLogger(ctx)

	var ip string
	if ip, err = fs.resolve(ctx, ref); err != nil {
		return errors.Wrap(err, "owncloudsql: error resolving reference")
	}

	// check permissions
	if perm, err := fs.readPermissions(ctx, ip); err == nil {
		if !perm.InitiateFileUpload { // TODO add dedicated permission?
			return errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return errtypes.NotFound(fs.toStoragePath(ctx, ip))
		}
		return errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	_, err = os.Stat(ip)
	if err != nil {
		if os.IsNotExist(err) {
			return errtypes.NotFound(fs.toStoragePath(ctx, ip))
		}
		return errors.Wrap(err, "owncloudsql: error stating "+ip)
	}

	errs := []error{}
	for _, k := range keys {
		switch k {
		case "http://owncloud.org/ns/favorite":
			if u, ok := ctxpkg.ContextGetUser(ctx); ok {
				// the favorite flag is specific to the user, so we need to incorporate the userid
				if uid := u.GetId(); uid != nil {
					fa := fmt.Sprintf("%s%s@%s", favPrefix, uid.GetOpaqueId(), uid.GetIdp())
					if err := xattr.Remove(ip, fa); err != nil {
						log.Error().Err(err).
							Str("ipath", ip).
							Interface("user", u).
							Str("key", fa).
							Msg("could not unset favorite flag")
						errs = append(errs, errors.Wrap(err, "could not unset favorite flag"))
					}
				} else {
					log.Error().
						Str("ipath", ip).
						Interface("user", u).
						Msg("user has no id")
					errs = append(errs, errors.Wrap(errtypes.UserRequired("userrequired"), "user has no id"))
				}
			} else {
				log.Error().
					Str("ipath", ip).
					Interface("user", u).
					Msg("error getting user from ctx")
				errs = append(errs, errors.Wrap(errtypes.UserRequired("userrequired"), "error getting user from ctx"))
			}
		default:
			if err = xattr.Remove(ip, mdPrefix+k); err != nil {
				// a non-existing attribute will return an error, which we can ignore
				// (using string compare because the error type is syscall.Errno and not wrapped/recognizable)
				if e, ok := err.(*xattr.Error); !ok || (e.Err.Error() != "no data available" &&
					// darwin
					e.Err.Error() != "attribute not found") {
					log.Error().Err(err).
						Str("ipath", ip).
						Str("key", k).
						Msg("could not unset metadata")
					errs = append(errs, errors.Wrap(err, "could not unset metadata"))
				}
			}
		}
	}

	switch len(errs) {
	case 0:
		return fs.propagate(ctx, ip)
	case 1:
		return errs[0]
	default:
		// TODO how to return multiple errors?
		return errors.New("multiple errors occurred, see log for details")
	}
}

// GetLock returns an existing lock on the given reference
func (fs *owncloudsqlfs) GetLock(ctx context.Context, ref *provider.Reference) (*provider.Lock, error) {
	return nil, errtypes.NotSupported("unimplemented")
}

// SetLock puts a lock on the given reference
func (fs *owncloudsqlfs) SetLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	return errtypes.NotSupported("unimplemented")
}

// RefreshLock refreshes an existing lock on the given reference
func (fs *owncloudsqlfs) RefreshLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock, existingLockID string) error {
	return errtypes.NotSupported("unimplemented")
}

// Unlock removes an existing lock from the given reference
func (fs *owncloudsqlfs) Unlock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	return errtypes.NotSupported("unimplemented")
}

// Delete is actually only a move to trash
//
// This is a first optimistic approach.
// When a file has versions and we want to delete the file it could happen that
// the service crashes before all moves are finished.
// That would result in invalid state like the main files was moved but the
// versions were not.
// We will live with that compromise since this storage driver will be
// deprecated soon.
func (fs *owncloudsqlfs) Delete(ctx context.Context, ref *provider.Reference) (err error) {
	var ip string
	if ip, err = fs.resolve(ctx, ref); err != nil {
		return errors.Wrap(err, "owncloudsql: error resolving reference")
	}

	// check permissions
	if perm, err := fs.readPermissions(ctx, ip); err == nil {
		if !perm.Delete {
			return errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return errtypes.NotFound(fs.toStoragePath(ctx, filepath.Dir(ip)))
		}
		return errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	_, err = os.Stat(ip)
	if err != nil {
		if os.IsNotExist(err) {
			return errtypes.NotFound(fs.toStoragePath(ctx, ip))
		}
		return errors.Wrap(err, "owncloudsql: error stating "+ip)
	}

	// Delete file into the owner's trash, not the user's (in case of shares)
	rp, err := fs.getRecyclePathForUser(fs.getOwner(ip))
	if err != nil {
		return errors.Wrap(err, "owncloudsql: error resolving recycle path")
	}

	if err := os.MkdirAll(rp, 0700); err != nil {
		return errors.Wrap(err, "owncloudsql: error creating trashbin dir "+rp)
	}

	// ip is the path on disk ... we need only the path relative to root
	origin := filepath.Dir(fs.toStoragePath(ctx, ip))

	err = fs.trash(ctx, ip, rp, origin)
	if err != nil {
		return errors.Wrapf(err, "owncloudsql: error deleting file %s", ip)
	}
	return nil
}

func (fs *owncloudsqlfs) trash(ctx context.Context, ip string, rp string, origin string) error {
	// move to trash location
	dtime := time.Now().Unix()
	tgt := filepath.Join(rp, fmt.Sprintf("%s.d%d", filepath.Base(ip), dtime))
	if err := os.Rename(ip, tgt); err != nil {
		if os.IsExist(err) {
			// timestamp collision, try again with higher value:
			dtime++
			tgt := filepath.Join(rp, fmt.Sprintf("%s.d%d", filepath.Base(ip), dtime))
			if err := os.Rename(ip, tgt); err != nil {
				return errors.Wrap(err, "owncloudsql: could not move item to trash")
			}
		}
	}

	storage, err := fs.getStorage(ctx, ip)
	if err != nil {
		return err
	}

	tryDelete := func() error {
		return fs.filecache.Delete(ctx, storage, fs.getOwner(ip), fs.toDatabasePath(ip), fs.toDatabasePath(tgt))
	}
	err = tryDelete()
	if err != nil {
		err = fs.createHomeForUser(ctx, fs.getOwner(ip)) // Try setting up the owner's home (incl. trash) to fix the problem
		if err != nil {
			return err
		}
		err = tryDelete()
		if err != nil {
			return err
		}
	}

	err = fs.trashVersions(ctx, ip, origin, dtime)
	if err != nil {
		return errors.Wrapf(err, "owncloudsql: error deleting versions of file %s", ip)
	}

	return fs.propagate(ctx, filepath.Dir(ip))
}

func (fs *owncloudsqlfs) trashVersions(ctx context.Context, ip string, origin string, dtime int64) error {
	vp := fs.getVersionsPath(ctx, ip)
	vrp, err := fs.getVersionRecyclePath(ctx)
	if err != nil {
		return errors.Wrap(err, "error resolving versions recycle path")
	}

	if err := os.MkdirAll(vrp, 0700); err != nil {
		return errors.Wrap(err, "owncloudsql: error creating trashbin dir "+vrp)
	}

	// Ignore error since the only possible error is malformed pattern.
	versions, _ := filepath.Glob(vp + ".v*")
	storage, err := fs.getStorage(ctx, ip)
	if err != nil {
		return err
	}
	for _, v := range versions {
		tgt := filepath.Join(vrp, fmt.Sprintf("%s.d%d", filepath.Base(v), dtime))
		if err := os.Rename(v, tgt); err != nil {
			if os.IsExist(err) {
				// timestamp collision, try again with higher value:
				dtime++
				tgt := filepath.Join(vrp, fmt.Sprintf("%s.d%d", filepath.Base(ip), dtime))
				if err := os.Rename(ip, tgt); err != nil {
					return errors.Wrap(err, "owncloudsql: could not move item to trash")
				}
			}
		}
		if err != nil {
			return errors.Wrap(err, "owncloudsql: error deleting file "+v)
		}
		err = fs.filecache.Move(ctx, storage, fs.toDatabasePath(v), fs.toDatabasePath(tgt))
		if err != nil {
			return errors.Wrap(err, "owncloudsql: error deleting file "+v)
		}
	}
	return nil
}

func (fs *owncloudsqlfs) Move(ctx context.Context, oldRef, newRef *provider.Reference) (err error) {
	var oldIP string
	if oldIP, err = fs.resolve(ctx, oldRef); err != nil {
		return errors.Wrap(err, "owncloudsql: error resolving reference")
	}

	// check permissions
	if perm, err := fs.readPermissions(ctx, oldIP); err == nil {
		if !perm.Move { // TODO add dedicated permission?
			return errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return errtypes.NotFound(fs.toStoragePath(ctx, filepath.Dir(oldIP)))
		}
		return errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	var newIP string
	if newIP, err = fs.resolve(ctx, newRef); err != nil {
		return errors.Wrap(err, "owncloudsql: error resolving reference")
	}

	// TODO check target permissions ... if it exists
	storage, err := fs.getStorage(ctx, oldIP)
	if err != nil {
		return err
	}
	err = fs.filecache.Move(ctx, storage, fs.toDatabasePath(oldIP), fs.toDatabasePath(newIP))
	if err != nil {
		return err
	}
	if err = os.Rename(oldIP, newIP); err != nil {
		return errors.Wrap(err, "owncloudsql: error moving "+oldIP+" to "+newIP)
	}

	if err := fs.propagate(ctx, newIP); err != nil {
		return err
	}
	if filepath.Dir(newIP) != filepath.Dir(oldIP) {
		if err := fs.propagate(ctx, filepath.Dir(oldIP)); err != nil {
			return err
		}
	}
	return nil
}

func (fs *owncloudsqlfs) GetMD(ctx context.Context, ref *provider.Reference, mdKeys []string, fieldMask []string) (*provider.ResourceInfo, error) {
	ip, err := fs.resolve(ctx, ref)
	if err != nil {
		// TODO return correct errtype
		if _, ok := err.(errtypes.IsNotFound); ok {
			return nil, err
		}
		return nil, errors.Wrap(err, "owncloudsql: error resolving reference")
	}
	p := fs.toStoragePath(ctx, ip)

	// If GetMD is called for a path shared with the user then the path is
	// already wrapped. (fs.resolve wraps the path)
	if strings.HasPrefix(p, fs.c.DataDirectory) {
		ip = p
	}

	// check permissions
	if perm, err := fs.readPermissions(ctx, ip); err == nil {
		if !perm.Stat {
			return nil, errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return nil, errtypes.NotFound(fs.toStoragePath(ctx, filepath.Dir(ip)))
		}
		return nil, errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	ownerStorageID, err := fs.filecache.GetNumericStorageID(ctx, "home::"+fs.getOwner(ip))
	if err != nil {
		return nil, err
	}
	entry, err := fs.filecache.Get(ctx, ownerStorageID, fs.toDatabasePath(ip))
	switch {
	case err == sql.ErrNoRows:
		return nil, errtypes.NotFound(fs.toStoragePath(ctx, filepath.Dir(ip)))
	case err != nil:
		return nil, err
	}

	return fs.convertToResourceInfo(ctx, entry, ip, mdKeys)
}

func (fs *owncloudsqlfs) ListFolder(ctx context.Context, ref *provider.Reference, mdKeys, fieldMask []string) ([]*provider.ResourceInfo, error) {
	log := appctx.GetLogger(ctx)

	ip, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "owncloudsql: error resolving reference")
	}
	sp := fs.toStoragePath(ctx, ip)

	if fs.c.EnableHome {
		log.Debug().Msg("home enabled")
		if strings.HasPrefix(sp, "/") {
			// permissions checked in listWithHome
			return fs.listWithHome(ctx, "/", sp, mdKeys)
		}
	}

	log.Debug().Msg("list with nominal home")
	// permissions checked in listWithNominalHome
	return fs.listWithNominalHome(ctx, sp, mdKeys)
}

func (fs *owncloudsqlfs) listWithNominalHome(ctx context.Context, ip string, mdKeys []string) ([]*provider.ResourceInfo, error) {

	// If a user wants to list a folder shared with him the path will already
	// be wrapped with the files directory path of the share owner.
	// In that case we don't want to wrap the path again.
	if !strings.HasPrefix(ip, fs.c.DataDirectory) {
		ip = fs.toInternalPath(ctx, ip)
	}

	// check permissions
	if perm, err := fs.readPermissions(ctx, ip); err == nil {
		if !perm.ListContainer {
			return nil, errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return nil, errtypes.NotFound(fs.toStoragePath(ctx, filepath.Dir(ip)))
		}
		return nil, errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	storage, err := fs.getStorage(ctx, ip)
	if err != nil {
		return nil, err
	}
	entries, err := fs.filecache.List(ctx, storage, fs.toDatabasePath(ip)+"/")
	if err != nil {
		return nil, errors.Wrapf(err, "owncloudsql: error listing %s", ip)
	}
	owner := fs.getOwner(ip)
	finfos := []*provider.ResourceInfo{}
	for _, entry := range entries {
		cp := filepath.Join(fs.c.DataDirectory, owner, entry.Path)
		m, err := fs.convertToResourceInfo(ctx, entry, cp, mdKeys)
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("path", cp).Msg("error converting to a resource info")
		}
		finfos = append(finfos, m)
	}
	return finfos, nil
}

func (fs *owncloudsqlfs) listWithHome(ctx context.Context, home, p string, mdKeys []string) ([]*provider.ResourceInfo, error) {
	log := appctx.GetLogger(ctx)
	if p == home {
		log.Debug().Msg("listing home")
		return fs.listHome(ctx, home, mdKeys)
	}

	log.Debug().Msg("listing nominal home")
	return fs.listWithNominalHome(ctx, p, mdKeys)
}

func (fs *owncloudsqlfs) listHome(ctx context.Context, home string, mdKeys []string) ([]*provider.ResourceInfo, error) {
	// list files
	ip := fs.toInternalPath(ctx, home)

	// check permissions
	if perm, err := fs.readPermissions(ctx, ip); err == nil {
		if !perm.ListContainer {
			return nil, errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return nil, errtypes.NotFound(fs.toStoragePath(ctx, filepath.Dir(ip)))
		}
		return nil, errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	storage, err := fs.getStorage(ctx, ip)
	if err != nil {
		return nil, err
	}
	entries, err := fs.filecache.List(ctx, storage, fs.toDatabasePath(ip)+"/")
	if err != nil {
		return nil, errors.Wrapf(err, "owncloudsql: error listing %s", ip)
	}
	owner := fs.getOwner(ip)
	finfos := []*provider.ResourceInfo{}
	for _, entry := range entries {
		cp := filepath.Join(fs.c.DataDirectory, owner, entry.Path)
		m, err := fs.convertToResourceInfo(ctx, entry, cp, mdKeys)
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("path", cp).Msg("error converting to a resource info")
		}
		finfos = append(finfos, m)
	}
	return finfos, nil
}

func (fs *owncloudsqlfs) archiveRevision(ctx context.Context, vbp string, ip string) error {
	// move existing file to versions dir
	vp := fmt.Sprintf("%s.v%d", vbp, time.Now().Unix())
	if err := os.MkdirAll(filepath.Dir(vp), 0700); err != nil {
		return errors.Wrap(err, "owncloudsql: error creating versions dir "+vp)
	}

	// TODO(jfd): make sure rename is atomic, missing fsync ...
	if err := os.Rename(ip, vp); err != nil {
		return errors.Wrap(err, "owncloudsql: error renaming from "+ip+" to "+vp)
	}

	storage, err := fs.getStorage(ctx, ip)
	if err != nil {
		return err
	}

	vdp := fs.toDatabasePath(vp)
	basePath := strings.TrimSuffix(vp, vdp)
	parts := strings.Split(filepath.Dir(vdp), "/")
	walkPath := ""
	for i := 0; i < len(parts); i++ {
		walkPath = filepath.Join(walkPath, parts[i])
		_, err := fs.filecache.Get(ctx, storage, walkPath)
		if err == nil {
			continue
		}

		fi, err := os.Stat(filepath.Join(basePath, walkPath))
		if err != nil {
			return errors.Wrap(err, "could not stat parent version directory")
		}
		data := map[string]interface{}{
			"path":        walkPath,
			"mimetype":    "httpd/unix-directory",
			"etag":        calcEtag(ctx, fi),
			"permissions": 31, // 1: READ, 2: UPDATE, 4: CREATE, 8: DELETE, 16: SHARE
		}

		_, err = fs.filecache.InsertOrUpdate(ctx, storage, data, false)
		if err != nil {
			return errors.Wrap(err, "could not create parent version directory")
		}
	}
	_, err = fs.filecache.Copy(ctx, storage, fs.toDatabasePath(ip), vdp)
	return err
}

func (fs *owncloudsqlfs) Download(ctx context.Context, ref *provider.Reference, openReaderfunc func(*provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	ip, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, nil, errors.Wrap(err, "owncloudsql: error resolving reference")
	}
	p := fs.toStoragePath(ctx, ip)

	// If Download is called for a path shared with the user then the path is
	// already wrapped. (fs.resolve wraps the path)
	if strings.HasPrefix(p, fs.c.DataDirectory) {
		ip = p
	}

	// check permissions
	if perm, err := fs.readPermissions(ctx, ip); err == nil {
		if !perm.InitiateFileDownload {
			return nil, nil, errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return nil, nil, errtypes.NotFound(fs.toStoragePath(ctx, filepath.Dir(ip)))
		}
		return nil, nil, errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	ownerStorageID, err := fs.filecache.GetNumericStorageID(ctx, "home::"+fs.getOwner(ip))
	if err != nil {
		return nil, nil, err
	}
	entry, err := fs.filecache.Get(ctx, ownerStorageID, fs.toDatabasePath(ip))
	switch {
	case err == sql.ErrNoRows:
		return nil, nil, errtypes.NotFound(fs.toStoragePath(ctx, filepath.Dir(ip)))
	case err != nil:
		return nil, nil, err
	}

	md, err := fs.convertToResourceInfo(ctx, entry, ip, nil)
	if err != nil {
		return nil, nil, err
	}

	if !openReaderfunc(md) {
		return md, nil, nil
	}

	r, err := os.Open(ip)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, errtypes.NotFound(fs.toStoragePath(ctx, ip))
		}
		return nil, nil, errors.Wrap(err, "owncloudsql: error reading "+ip)
	}
	return md, r, nil
}

func (fs *owncloudsqlfs) ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error) {
	ip, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "owncloudsql: error resolving reference")
	}

	// check permissions
	if perm, err := fs.readPermissions(ctx, ip); err == nil {
		if !perm.ListFileVersions {
			return nil, errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return nil, errtypes.NotFound(fs.toStoragePath(ctx, filepath.Dir(ip)))
		}
		return nil, errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	vp := fs.getVersionsPath(ctx, ip)
	bn := filepath.Base(ip)
	storageID, err := fs.getStorage(ctx, ip)
	if err != nil {
		return nil, err
	}
	entries, err := fs.filecache.List(ctx, storageID, filepath.Dir(fs.toDatabasePath(vp))+"/")
	if err != nil {
		return nil, err
	}
	revisions := []*provider.FileVersion{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name, bn) {
			// versions have filename.ext.v12345678
			version := entry.Name[len(bn)+2:] // truncate "<base filename>.v" to get version mtime
			mtime, err := strconv.Atoi(version)
			if err != nil {
				log := appctx.GetLogger(ctx)
				log.Error().Err(err).Str("path", entry.Name).Msg("invalid version mtime")
				return nil, err
			}
			revisions = append(revisions, &provider.FileVersion{
				Key:   version,
				Size:  uint64(entry.Size),
				Mtime: uint64(mtime),
				Etag:  entry.Etag,
			})
		}
	}

	return revisions, nil
}

func (fs *owncloudsqlfs) DownloadRevision(ctx context.Context, ref *provider.Reference, revisionKey string, openReaderfunc func(*provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	return nil, nil, errtypes.NotSupported("download revision")
}

func (fs *owncloudsqlfs) RestoreRevision(ctx context.Context, ref *provider.Reference, revisionKey string) error {
	ip, err := fs.resolve(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "owncloudsql: error resolving reference")
	}

	// check permissions
	if perm, err := fs.readPermissions(ctx, ip); err == nil {
		if !perm.RestoreFileVersion {
			return errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return errtypes.NotFound(fs.toStoragePath(ctx, filepath.Dir(ip)))
		}
		return errors.Wrap(err, "owncloudsql: error reading permissions")
	}

	vp := fs.getVersionsPath(ctx, ip)
	rp := vp + ".v" + revisionKey

	// check revision exists
	rs, err := os.Stat(rp)
	if err != nil {
		return err
	}

	if !rs.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", rp)
	}

	source, err := os.Open(rp)
	if err != nil {
		return err
	}
	defer source.Close()

	// destination should be available, otherwise we could not have navigated to its revisions
	if err := fs.archiveRevision(ctx, fs.getVersionsPath(ctx, ip), ip); err != nil {
		return err
	}

	destination, err := os.Create(ip)
	if err != nil {
		// TODO(jfd) bring back revision in case sth goes wrong?
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	sha1h, md5h, adler32h, err := fs.HashFile(ip)
	if err != nil {
		log.Err(err).Msg("owncloudsql: could not open file for checksumming")
	}
	fi, err := os.Stat(ip)
	if err != nil {
		return err
	}
	mtime := time.Now().Unix()
	data := map[string]interface{}{
		"path":          fs.toDatabasePath(ip),
		"checksum":      fmt.Sprintf("SHA1:%032x MD5:%032x ADLER32:%032x", sha1h, md5h, adler32h),
		"etag":          calcEtag(ctx, fi),
		"size":          fi.Size(),
		"mimetype":      mime.Detect(false, ip),
		"mtime":         mtime,
		"storage_mtime": mtime,
	}
	storageID, err := fs.getStorage(ctx, ip)
	if err != nil {
		return err
	}
	_, err = fs.filecache.InsertOrUpdate(ctx, storageID, data, false)
	if err != nil {
		return err
	}

	// TODO(jfd) bring back revision in case sth goes wrong?
	return fs.propagate(ctx, ip)
}

func (fs *owncloudsqlfs) PurgeRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string) error {
	rp, err := fs.getRecyclePath(ctx)
	if err != nil {
		return errors.Wrap(err, "owncloudsql: error resolving recycle path")
	}
	vp := filepath.Join(filepath.Dir(rp), "versions")
	ip := filepath.Join(rp, filepath.Clean(key))
	// TODO check permission?

	// check permissions
	/* are they stored in the trash?
	if perm, err := fs.readPermissions(ctx, ip); err == nil {
		if !perm.ListContainer {
			return nil, errtypes.PermissionDenied("")
		}
	} else {
		if isNotFound(err) {
			return nil, errtypes.NotFound(fs.unwrap(ctx, filepath.Dir(ip)))
		}
		return nil, errors.Wrap(err, "owncloudsql: error reading permissions")
	}
	*/

	err = os.RemoveAll(ip)
	if err != nil {
		return errors.Wrap(err, "owncloudsql: error deleting recycle item")
	}
	base, ttime, err := splitTrashKey(key)
	if err != nil {
		return err
	}
	err = fs.filecache.PurgeRecycleItem(ctx, ctxpkg.ContextMustGetUser(ctx).Username, base, ttime, false)
	if err != nil {
		return err
	}

	versionsGlob := filepath.Join(vp, base+".v*.d"+strconv.Itoa(ttime))
	versionFiles, err := filepath.Glob(versionsGlob)
	if err != nil {
		return errors.Wrap(err, "owncloudsql: error listing recycle item versions")
	}
	storageID, err := fs.getStorage(ctx, ip)
	if err != nil {
		return err
	}
	for _, versionFile := range versionFiles {
		err = os.Remove(versionFile)
		if err != nil {
			return errors.Wrap(err, "owncloudsql: error deleting recycle item versions")
		}
		err = fs.filecache.Purge(ctx, storageID, fs.toDatabasePath(versionFile))
		if err != nil {
			return err
		}
	}

	// TODO delete keyfiles, keys, share-keys
	return nil
}

func (fs *owncloudsqlfs) EmptyRecycle(ctx context.Context, ref *provider.Reference) error {
	// TODO check permission? on what? user must be the owner
	rp, err := fs.getRecyclePath(ctx)
	if err != nil {
		return errors.Wrap(err, "owncloudsql: error resolving recycle path")
	}
	err = os.RemoveAll(rp)
	if err != nil {
		return errors.Wrap(err, "owncloudsql: error deleting recycle files")
	}
	err = os.RemoveAll(filepath.Join(filepath.Dir(rp), "versions"))
	if err != nil {
		return errors.Wrap(err, "owncloudsql: error deleting recycle files versions")
	}

	u := ctxpkg.ContextMustGetUser(ctx)
	err = fs.filecache.EmptyRecycle(ctx, u.Username)
	if err != nil {
		return errors.Wrap(err, "owncloudsql: error deleting recycle items from the database")
	}

	// TODO delete keyfiles, keys, share-keys ... or just everything?
	return nil
}

func splitTrashKey(key string) (string, int, error) {
	// trashbin items have filename.ext.d12345678
	suffix := filepath.Ext(key)
	if len(suffix) == 0 || !strings.HasPrefix(suffix, ".d") {
		return "", -1, fmt.Errorf("invalid suffix")
	}
	trashtime := suffix[2:] // truncate "d" to get trashbin time
	ttime, err := strconv.Atoi(trashtime)
	if err != nil {
		return "", -1, fmt.Errorf("invalid suffix")
	}
	return strings.TrimSuffix(filepath.Base(key), suffix), ttime, nil
}

func (fs *owncloudsqlfs) convertToRecycleItem(ctx context.Context, md os.FileInfo) *provider.RecycleItem {
	base, ttime, err := splitTrashKey(md.Name())
	if err != nil {
		log := appctx.GetLogger(ctx)
		log.Error().Str("path", md.Name()).Msg("invalid trash item key")
	}

	u := ctxpkg.ContextMustGetUser(ctx)
	item, err := fs.filecache.GetRecycleItem(ctx, u.Username, base, ttime)
	if err != nil {
		log := appctx.GetLogger(ctx)
		log.Error().Err(err).Str("path", md.Name()).Msg("could not get trash item")
		return nil
	}

	// ownCloud 10 stores the parent dir of the deleted item as the location in the oc_files_trashbin table
	// we use extended attributes for original location, but also only the parent location, which is why
	// we need to join and trim the path when listing it
	originalPath := filepath.Join(item.Path, base)

	return &provider.RecycleItem{
		Type: getResourceType(md.IsDir()),
		Key:  md.Name(),
		// TODO do we need to prefix the path? it should be relative to this storage root, right?
		Ref:  &provider.Reference{Path: originalPath},
		Size: uint64(md.Size()),
		DeletionTime: &types.Timestamp{
			Seconds: uint64(ttime),
			// no nanos available
		},
	}
}

func (fs *owncloudsqlfs) ListRecycle(ctx context.Context, ref *provider.Reference, key, relativePath string) ([]*provider.RecycleItem, error) {
	// TODO check permission? on what? user must be the owner?
	rp, err := fs.getRecyclePath(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "owncloudsql: error resolving recycle path")
	}

	// list files folder
	mds, err := os.ReadDir(rp)
	if err != nil {
		log := appctx.GetLogger(ctx)
		log.Debug().Err(err).Str("path", rp).Msg("trash not readable")
		// TODO jfd only ignore not found errors
		return []*provider.RecycleItem{}, nil
	}
	// TODO (jfd) limit and offset
	items := []*provider.RecycleItem{}
	for i := range mds {
		mdsInfo, _ := mds[i].Info()
		ri := fs.convertToRecycleItem(ctx, mdsInfo)
		if ri != nil {
			items = append(items, ri)
		}

	}
	return items, nil
}

func (fs *owncloudsqlfs) RestoreRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string, restoreRef *provider.Reference) error {
	log := appctx.GetLogger(ctx)

	base, ttime, err := splitTrashKey(key)
	if err != nil {
		log.Error().Str("path", key).Msg("invalid trash item key")
		return fmt.Errorf("invalid trash item suffix")
	}

	recyclePath, err := fs.getRecyclePath(ctx)
	if err != nil {
		return errors.Wrap(err, "owncloudsql: error resolving recycle path")
	}
	src := filepath.Join(recyclePath, filepath.Clean(key))

	if restoreRef.Path == "" {
		u := ctxpkg.ContextMustGetUser(ctx)
		item, err := fs.filecache.GetRecycleItem(ctx, u.Username, base, ttime)
		if err != nil {
			log := appctx.GetLogger(ctx)
			log.Error().Err(err).Str("path", key).Msg("could not get trash item")
			return nil
		}
		restoreRef.Path = filepath.Join(item.Path, item.Name)
	}

	tgt := fs.toInternalPath(ctx, restoreRef.Path)
	// move back to original location
	if err := os.Rename(src, tgt); err != nil {
		log.Error().Err(err).Str("key", key).Str("restorePath", restoreRef.Path).Str("src", src).Str("tgt", tgt).Msg("could not restore item")
		return errors.Wrap(err, "owncloudsql: could not restore item")
	}

	storage, err := fs.getStorage(ctx, src)
	if err != nil {
		return err
	}
	err = fs.filecache.Move(ctx, storage, fs.toDatabasePath(src), fs.toDatabasePath(tgt))
	if err != nil {
		return err
	}
	err = fs.filecache.DeleteRecycleItem(ctx, ctxpkg.ContextMustGetUser(ctx).Username, base, ttime)
	if err != nil {
		return err
	}
	err = fs.RestoreRecycleItemVersions(ctx, key, tgt)
	if err != nil {
		return err
	}

	return fs.propagate(ctx, tgt)
}

func (fs *owncloudsqlfs) RestoreRecycleItemVersions(ctx context.Context, key, target string) error {
	base, ttime, err := splitTrashKey(key)
	if err != nil {
		return fmt.Errorf("invalid trash item suffix")
	}
	storage, err := fs.getStorage(ctx, target)
	if err != nil {
		return err
	}

	recyclePath, err := fs.getRecyclePath(ctx)
	if err != nil {
		return errors.Wrap(err, "owncloudsql: error resolving recycle path")
	}
	versionsRecyclePath := filepath.Join(filepath.Dir(recyclePath), "versions")

	// Restore versions
	deleteSuffix := ".d" + strconv.Itoa(ttime)
	versionsGlob := filepath.Join(versionsRecyclePath, base+".v*"+deleteSuffix)
	versionFiles, err := filepath.Glob(versionsGlob)
	versionsRoot := filepath.Dir(fs.getVersionsPath(ctx, target))

	if err != nil {
		return errors.Wrap(err, "owncloudsql: error listing recycle item versions")
	}
	for _, versionFile := range versionFiles {
		versionBase := strings.TrimSuffix(filepath.Base(versionFile), deleteSuffix)
		versionsRestorePath := filepath.Join(versionsRoot, versionBase)
		if err = os.Rename(versionFile, versionsRestorePath); err != nil {
			return errors.Wrap(err, "owncloudsql: could not restore version file")
		}
		err = fs.filecache.Move(ctx, storage, fs.toDatabasePath(versionFile), fs.toDatabasePath(versionsRestorePath))
		if err != nil {
			return err
		}
	}
	return nil
}

func (fs *owncloudsqlfs) propagate(ctx context.Context, leafPath string) error {
	var root string
	if fs.c.EnableHome {
		root = filepath.Clean(fs.toInternalPath(ctx, "/"))
	} else {
		owner := fs.getOwner(leafPath)
		root = filepath.Clean(fs.toInternalPath(ctx, owner))
	}
	versionsRoot := filepath.Join(filepath.Dir(root), "files_versions")
	if !strings.HasPrefix(leafPath, root) {
		err := errors.New("internal path outside root")
		appctx.GetLogger(ctx).Error().
			Err(err).
			Str("leafPath", leafPath).
			Str("root", root).
			Msg("could not propagate change")
		return err
	}

	fi, err := os.Stat(leafPath)
	if err != nil {
		appctx.GetLogger(ctx).Error().
			Err(err).
			Str("leafPath", leafPath).
			Str("root", root).
			Msg("could not propagate change")
		return err
	}

	storageID, err := fs.getStorage(ctx, leafPath)
	if err != nil {
		return err
	}

	currentPath := filepath.Clean(leafPath)
	for currentPath != root && currentPath != versionsRoot {
		appctx.GetLogger(ctx).Debug().
			Str("leafPath", leafPath).
			Str("currentPath", currentPath).
			Msg("propagating change")
		parentFi, err := os.Stat(currentPath)
		if err != nil {
			return err
		}
		if fi.ModTime().UnixNano() > parentFi.ModTime().UnixNano() {
			if err := os.Chtimes(currentPath, fi.ModTime(), fi.ModTime()); err != nil {
				appctx.GetLogger(ctx).Error().
					Err(err).
					Str("leafPath", leafPath).
					Str("currentPath", currentPath).
					Msg("could not propagate change")
				return err
			}
		}
		fi, err = os.Stat(currentPath)
		if err != nil {
			return err
		}
		etag := calcEtag(ctx, fi)
		if err := fs.filecache.SetEtag(ctx, storageID, fs.toDatabasePath(currentPath), etag); err != nil {
			appctx.GetLogger(ctx).Error().
				Err(err).
				Str("leafPath", leafPath).
				Str("currentPath", currentPath).
				Msg("could not set etag")
			return err
		}

		currentPath = filepath.Dir(currentPath)
	}
	return nil
}

func (fs *owncloudsqlfs) HashFile(path string) (string, string, string, error) {
	sha1h := sha1.New()
	md5h := md5.New()
	adler32h := adler32.New()
	{
		f, err := os.Open(path)
		if err != nil {
			return "", "", "", errors.Wrap(err, "owncloudsql: could not copy bytes for checksumming")
		}
		defer f.Close()

		r1 := io.TeeReader(f, sha1h)
		r2 := io.TeeReader(r1, md5h)

		if _, err := io.Copy(adler32h, r2); err != nil {
			return "", "", "", errors.Wrap(err, "owncloudsql: could not copy bytes for checksumming")
		}

		return string(sha1h.Sum(nil)), string(md5h.Sum(nil)), string(adler32h.Sum(nil)), nil
	}
}

func readChecksumIntoResourceChecksum(ctx context.Context, checksums, algo string, ri *provider.ResourceInfo) {
	re := regexp.MustCompile(strings.ToUpper(algo) + `:(.*)`)
	matches := re.FindStringSubmatch(checksums)
	if len(matches) < 2 {
		appctx.GetLogger(ctx).
			Debug().
			Str("nodepath", checksums).
			Str("algorithm", algo).
			Msg("checksum not set")
	} else {
		ri.Checksum = &provider.ResourceChecksum{
			Type: storageprovider.PKG2GRPCXS(algo),
			Sum:  matches[1],
		}
	}
}

func readChecksumIntoOpaque(ctx context.Context, checksums, algo string, ri *provider.ResourceInfo) {
	re := regexp.MustCompile(strings.ToUpper(algo) + `:(.*)`)
	matches := re.FindStringSubmatch(checksums)
	if len(matches) < 2 {
		appctx.GetLogger(ctx).
			Debug().
			Str("nodepath", checksums).
			Str("algorithm", algo).
			Msg("checksum not set")
	} else {
		if ri.Opaque == nil {
			ri.Opaque = &types.Opaque{
				Map: map[string]*types.OpaqueEntry{},
			}
		}
		ri.Opaque.Map[algo] = &types.OpaqueEntry{
			Decoder: "plain",
			Value:   []byte(matches[1]),
		}
	}
}

func getResourceType(isDir bool) provider.ResourceType {
	if isDir {
		return provider.ResourceType_RESOURCE_TYPE_CONTAINER
	}
	return provider.ResourceType_RESOURCE_TYPE_FILE
}

// TODO propagate etag and mtime or append event to history? propagate on disk ...
// - but propagation is a separate task. only if upload was successful ...
