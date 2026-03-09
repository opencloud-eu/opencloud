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

package eosfs

import (
	"context"
	"database/sql"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bluele/gcache"
	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/jellydator/ttlcache/v2"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/eosclient"
	"github.com/opencloud-eu/reva/v2/pkg/eosclient/eosbinary"
	"github.com/opencloud-eu/reva/v2/pkg/eosclient/eosgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/mime"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/acl"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/chunking"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/grants"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
)

const (
	refTargetAttrKey = "reva.target"
)

const (
	// SystemAttr is the system extended attribute.
	SystemAttr eosclient.AttrType = iota
	// UserAttr is the user extended attribute.
	UserAttr
)

// LockPayloadKey is the key in the xattr for lock payload
const LockPayloadKey = "reva.lock.payload"

// LockExpirationKey is the key in the xattr for lock expiration
const LockExpirationKey = "reva.lock.expiration"

// LockTypeKey is the key in the xattr for lock payload
const LockTypeKey = "reva.lock.type"

var hiddenReg = regexp.MustCompile(`\.sys\..#.`)

func (c *Config) init() {
	c.Namespace = path.Clean(c.Namespace)
	if !strings.HasPrefix(c.Namespace, "/") {
		c.Namespace = "/"
	}

	if c.ShadowNamespace == "" {
		c.ShadowNamespace = path.Join(c.Namespace, ".shadow")
	}

	// Quota node defaults to namespace if empty
	if c.QuotaNode == "" {
		c.QuotaNode = c.Namespace
	}

	if c.DefaultQuotaBytes == 0 {
		c.DefaultQuotaBytes = 1000000000000 // 1 TB
	}
	if c.DefaultQuotaFiles == 0 {
		c.DefaultQuotaFiles = 1000000 // 1 Million
	}

	if c.ShareFolder == "" {
		c.ShareFolder = "/MyShares"
	}
	// ensure share folder always starts with slash
	c.ShareFolder = path.Join("/", c.ShareFolder)

	if c.EosBinary == "" {
		c.EosBinary = "/usr/bin/eos"
	}

	if c.XrdcopyBinary == "" {
		c.XrdcopyBinary = "/opt/eos/xrootd/bin/xrdcopy"
	}

	if c.MasterURL == "" {
		c.MasterURL = "root://eos-example.org"
	}

	if c.SlaveURL == "" {
		c.SlaveURL = c.MasterURL
	}

	if c.CacheDirectory == "" {
		c.CacheDirectory = os.TempDir()
	}

	if c.UserLayout == "" {
		c.UserLayout = "{{.Username}}" // TODO set better layout
	}

	if c.UserIDCacheSize == 0 {
		c.UserIDCacheSize = 1000000
	}

	if c.UserIDCacheWarmupDepth == 0 {
		c.UserIDCacheWarmupDepth = 2
	}

	if c.TokenExpiry == 0 {
		c.TokenExpiry = 3600
	}

	c.GatewaySvc = sharedconf.GetGatewaySVC(c.GatewaySvc)
}

type eosfs struct {
	c              eosclient.EOSClient
	conf           *Config
	chunkHandler   *chunking.ChunkHandler
	spacesDB       *sql.DB
	singleUserAuth eosclient.Authorization
	userIDCache    *ttlcache.Cache
	tokenCache     gcache.Cache
	spacesCache    gcache.Cache
}

// NewEOSFS returns a storage.FS interface implementation that connects to an EOS instance
func NewEOSFS(c *Config) (storage.FS, error) {
	c.init()

	// bail out if keytab is not found.
	if c.UseKeytab {
		if _, err := os.Stat(c.Keytab); err != nil {
			err = errors.Wrapf(err, "eosfs: keytab not accessible at location: %s", err)
			return nil, err
		}
	}

	var eosClient eosclient.EOSClient
	var err error
	if c.UseGRPC {
		eosClientOpts := &eosgrpc.Options{
			XrdcopyBinary:      c.XrdcopyBinary,
			URL:                c.MasterURL,
			GrpcURI:            c.GrpcURI,
			CacheDirectory:     c.CacheDirectory,
			UseKeytab:          c.UseKeytab,
			Keytab:             c.Keytab,
			Authkey:            c.GRPCAuthkey,
			SecProtocol:        c.SecProtocol,
			VersionInvariant:   c.VersionInvariant,
			ReadUsesLocalTemp:  c.ReadUsesLocalTemp,
			WriteUsesLocalTemp: c.WriteUsesLocalTemp,
		}
		eosHTTPOpts := &eosgrpc.HTTPOptions{
			BaseURL:             c.MasterURL,
			MaxIdleConns:        c.MaxIdleConns,
			MaxConnsPerHost:     c.MaxConnsPerHost,
			MaxIdleConnsPerHost: c.MaxIdleConnsPerHost,
			IdleConnTimeout:     c.IdleConnTimeout,
			ClientCertFile:      c.ClientCertFile,
			ClientKeyFile:       c.ClientKeyFile,
			ClientCADirs:        c.ClientCADirs,
			ClientCAFiles:       c.ClientCAFiles,
		}
		eosClient, err = eosgrpc.New(eosClientOpts, eosHTTPOpts)
	} else {
		eosClientOpts := &eosbinary.Options{
			XrdcopyBinary:       c.XrdcopyBinary,
			URL:                 c.MasterURL,
			EosBinary:           c.EosBinary,
			CacheDirectory:      c.CacheDirectory,
			ForceSingleUserMode: c.ForceSingleUserMode,
			SingleUsername:      c.SingleUsername,
			UseKeytab:           c.UseKeytab,
			Keytab:              c.Keytab,
			SecProtocol:         c.SecProtocol,
			VersionInvariant:    c.VersionInvariant,
			TokenExpiry:         c.TokenExpiry,
		}
		eosClient, err = eosbinary.New(eosClientOpts)
	}

	if err != nil {
		return nil, errors.Wrap(err, "error initializing eosclient")
	}

	var db *sql.DB
	if c.SpacesConfig.Enabled {
		db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", c.SpacesConfig.DbUsername, c.SpacesConfig.DbPassword, c.SpacesConfig.DbHost, c.SpacesConfig.DbPort, c.SpacesConfig.DbName))
		if err != nil {
			return nil, err
		}
	}

	eosfs := &eosfs{
		c:            eosClient,
		conf:         c,
		spacesDB:     db,
		chunkHandler: chunking.NewChunkHandler(c.CacheDirectory),
		userIDCache:  ttlcache.NewCache(),
		tokenCache:   gcache.New(c.UserIDCacheSize).LFU().Build(),
		spacesCache:  gcache.New(c.UserIDCacheSize).LFU().Build(),
	}

	eosfs.userIDCache.SetCacheSizeLimit(c.UserIDCacheSize)
	eosfs.userIDCache.SetExpirationReasonCallback(func(key string, reason ttlcache.EvictionReason, value interface{}) {
		// We only set those keys with TTL which we weren't able to retrieve the last time
		// For those keys, try to contact the userprovider service again when they expire
		if reason == ttlcache.Expired {
			_, _ = eosfs.getUserIDGateway(context.Background(), key)
		}
	})

	go eosfs.userIDcacheWarmup()

	return eosfs, nil
}

func (fs *eosfs) userIDcacheWarmup() {
	if !fs.conf.EnableHome {
		time.Sleep(2 * time.Second)
		ctx := context.Background()
		paths := []string{fs.wrap(ctx, "/")}
		auth, _ := fs.getRootAuth(ctx)

		for i := 0; i < fs.conf.UserIDCacheWarmupDepth; i++ {
			var newPaths []string
			for _, fn := range paths {
				if eosFileInfos, err := fs.c.List(ctx, auth, fn); err == nil {
					for _, f := range eosFileInfos {
						_, _ = fs.getUserIDGateway(ctx, strconv.FormatUint(f.UID, 10))
						newPaths = append(newPaths, f.File)
					}
				}
			}
			paths = newPaths
		}
	}
}

func (fs *eosfs) Shutdown(ctx context.Context) error {
	// TODO(labkode): in a grpc implementation we can close connections.
	return nil
}

func getUser(ctx context.Context) (*userpb.User, error) {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		err := errors.Wrap(errtypes.UserRequired(""), "eosfs: error getting user from ctx")
		return nil, err
	}
	return u, nil
}

func (fs *eosfs) getLayout(ctx context.Context) (layout string) {
	if fs.conf.EnableHome {
		u, err := getUser(ctx)
		if err != nil {
			panic(err)
		}
		layout = templates.WithUser(u, fs.conf.UserLayout)
	}
	return
}

func (fs *eosfs) getInternalHome(ctx context.Context) (string, error) {
	if !fs.conf.EnableHome {
		return "", errtypes.NotSupported("eos: get home not supported")
	}

	u, err := getUser(ctx)
	if err != nil {
		err = errors.Wrap(err, "eosfs: wrap: no user in ctx and home is enabled")
		return "", err
	}

	relativeHome := templates.WithUser(u, fs.conf.UserLayout)
	return relativeHome, nil
}

func (fs *eosfs) wrapShadow(ctx context.Context, fn string) (internal string) {
	if fs.conf.EnableHome {
		layout, err := fs.getInternalHome(ctx)
		if err != nil {
			panic(err)
		}
		internal = path.Join(fs.conf.ShadowNamespace, layout, fn)
	} else {
		internal = path.Join(fs.conf.ShadowNamespace, fn)
	}
	return
}

func (fs *eosfs) wrap(ctx context.Context, fn string) (internal string) {
	fn = strings.TrimPrefix(fn, fs.conf.MountPath)
	if fs.conf.EnableHome {
		layout, err := fs.getInternalHome(ctx)
		if err != nil {
			panic(err)
		}
		internal = path.Join(fs.conf.Namespace, layout, fn)
	} else {
		internal = path.Join(fs.conf.Namespace, fn)
	}
	log := appctx.GetLogger(ctx)
	log.Debug().Msg("eosfs: wrap external=" + fn + " internal=" + internal)
	return
}

func (fs *eosfs) unwrap(ctx context.Context, internal string) (string, error) {
	log := appctx.GetLogger(ctx)
	layout := fs.getLayout(ctx)
	ns, err := fs.getNsMatch(internal, []string{fs.conf.Namespace, fs.conf.ShadowNamespace})
	if err != nil {
		return "", err
	}
	external, err := fs.unwrapInternal(ctx, ns, internal, layout)
	if err != nil {
		return "", err
	}
	log.Debug().Msgf("eosfs: unwrap: internal=%s external=%s", internal, external)
	return external, nil
}

func (fs *eosfs) getNsMatch(internal string, nss []string) (string, error) {
	var match string

	for _, ns := range nss {
		if strings.HasPrefix(internal, ns) && len(ns) > len(match) {
			match = ns
		}
	}

	if match == "" {
		return "", errtypes.NotFound(fmt.Sprintf("eosfs: path is outside namespaces: path=%s namespaces=%+v", internal, nss))
	}

	return match, nil
}

func (fs *eosfs) unwrapInternal(ctx context.Context, ns, np, layout string) (string, error) {
	trim := path.Join(ns, layout)

	if !strings.HasPrefix(np, trim) {
		return "", errtypes.NotFound(fmt.Sprintf("eosfs: path is outside the directory of the logged-in user: internal=%s trim=%s namespace=%+v", np, trim, ns))
	}

	external := strings.TrimPrefix(np, trim)

	if external == "" {
		external = "/"
	}

	return external, nil
}

func (fs *eosfs) resolveRefForbidShareFolder(ctx context.Context, ref *provider.Reference) (string, eosclient.Authorization, error) {
	p, err := fs.resolve(ctx, ref)
	if err != nil {
		return "", eosclient.Authorization{}, errors.Wrap(err, "eosfs: error resolving reference")
	}
	if fs.isShareFolder(ctx, p) {
		return "", eosclient.Authorization{}, errtypes.PermissionDenied("eosfs: cannot perform operation under the virtual share folder")
	}
	fn := fs.wrap(ctx, p)

	u, err := getUser(ctx)
	if err != nil {
		return "", eosclient.Authorization{}, errors.Wrap(err, "eosfs: no user in ctx")
	}
	auth, err := fs.getUserAuth(ctx, u, fn)
	if err != nil {
		return "", eosclient.Authorization{}, err
	}

	return fn, auth, nil
}

func (fs *eosfs) resolveRefAndGetAuth(ctx context.Context, ref *provider.Reference) (string, eosclient.Authorization, error) {
	p, err := fs.resolve(ctx, ref)
	if err != nil {
		return "", eosclient.Authorization{}, errors.Wrap(err, "eosfs: error resolving reference")
	}
	fn := fs.wrap(ctx, p)

	u, err := getUser(ctx)
	if err != nil {
		return "", eosclient.Authorization{}, errors.Wrap(err, "eosfs: no user in ctx")
	}
	auth, err := fs.getUserAuth(ctx, u, fn)
	if err != nil {
		return "", eosclient.Authorization{}, err
	}

	return fn, auth, nil
}

// resolve takes in a request path or request id and returns the unwrapped path.
func (fs *eosfs) resolve(ctx context.Context, ref *provider.Reference) (string, error) {
	if ref.ResourceId != nil {
		p, err := fs.getPath(ctx, ref.ResourceId)
		if err != nil {
			return "", err
		}
		p = path.Join(p, ref.Path)
		return p, nil
	}
	if ref.Path != "" {
		return ref.Path, nil
	}

	// reference is invalid
	return "", fmt.Errorf("invalid reference %+v. at least resource_id or path must be set", ref)
}

func (fs *eosfs) getPath(ctx context.Context, id *provider.ResourceId) (string, error) {
	fid, err := strconv.ParseUint(id.OpaqueId, 10, 64)
	if err != nil {
		return "", fmt.Errorf("error converting string to int for eos fileid: %s", id.OpaqueId)
	}

	auth, err := fs.getRootAuth(ctx)
	if err != nil {
		return "", err
	}

	eosFileInfo, err := fs.c.GetFileInfoByInode(ctx, auth, fid)
	if err != nil {
		return "", errors.Wrap(err, "eosfs: error getting file info by inode")
	}

	return fs.unwrap(ctx, eosFileInfo.File)
}

func (fs *eosfs) isShareFolder(ctx context.Context, p string) bool {
	return strings.HasPrefix(p, fs.conf.ShareFolder)
}

func (fs *eosfs) isShareFolderRoot(ctx context.Context, p string) bool {
	return path.Clean(p) == fs.conf.ShareFolder
}

func (fs *eosfs) isShareFolderChild(ctx context.Context, p string) bool {
	p = path.Clean(p)
	vals := strings.Split(p, fs.conf.ShareFolder+"/")
	return len(vals) > 1 && vals[1] != ""
}

func (fs *eosfs) GetPathByID(ctx context.Context, id *provider.ResourceId) (string, error) {
	fid, err := strconv.ParseUint(id.OpaqueId, 10, 64)
	if err != nil {
		return "", errors.Wrap(err, "eosfs: error parsing fileid string")
	}

	u, err := getUser(ctx)
	if err != nil {
		return "", errors.Wrap(err, "eosfs: no user in ctx")
	}
	if u.Id.Type == userpb.UserType_USER_TYPE_LIGHTWEIGHT || u.Id.Type == userpb.UserType_USER_TYPE_FEDERATED {
		auth, err := fs.getRootAuth(ctx)
		if err != nil {
			return "", err
		}
		eosFileInfo, err := fs.c.GetFileInfoByInode(ctx, auth, fid)
		if err != nil {
			return "", errors.Wrap(err, "eosfs: error getting file info by inode")
		}
		if perm := fs.permissionSet(ctx, eosFileInfo, nil); perm.GetPath {
			return fs.unwrap(ctx, eosFileInfo.File)
		}
		return "", errtypes.PermissionDenied("eosfs: getting path for id not allowed")
	}

	auth, err := fs.getUserAuth(ctx, u, "")
	if err != nil {
		return "", err
	}

	eosFileInfo, err := fs.c.GetFileInfoByInode(ctx, auth, fid)
	if err != nil {
		return "", errors.Wrap(err, "eosfs: error getting file info by inode")
	}

	p, err := fs.unwrap(ctx, eosFileInfo.File)
	if err != nil {
		return "", err
	}
	return path.Join(fs.conf.MountPath, p), nil
}

func (fs *eosfs) SetArbitraryMetadata(ctx context.Context, ref *provider.Reference, md *provider.ArbitraryMetadata) error {
	if len(md.Metadata) == 0 {
		return errtypes.BadRequest("eosfs: no metadata set")
	}

	fn, auth, err := fs.resolveRefAndGetAuth(ctx, ref)
	if err != nil {
		return err
	}

	for k, v := range md.Metadata {
		if k == "" || v == "" {
			return errtypes.BadRequest(fmt.Sprintf("eosfs: key or value is empty: key:%s, value:%s", k, v))
		}

		// do not allow to set a lock key attr
		if k == LockPayloadKey || k == LockExpirationKey || k == LockTypeKey {
			return errtypes.BadRequest(fmt.Sprintf("eosfs: key %s not allowed", k))
		}

		attr := &eosclient.Attribute{
			Type: UserAttr,
			Key:  k,
			Val:  v,
		}

		// TODO(labkode): SetArbitraryMetadata does not have semantics for recursivity.
		// We set it to false
		err := fs.c.SetAttr(ctx, auth, attr, false, false, fn)
		if err != nil {
			return errors.Wrap(err, "eosfs: error setting xattr in eos driver")
		}

	}
	return nil
}

func (fs *eosfs) UnsetArbitraryMetadata(ctx context.Context, ref *provider.Reference, keys []string) error {
	if len(keys) == 0 {
		return errtypes.BadRequest("eosfs: no keys set")
	}

	fn, auth, err := fs.resolveRefAndGetAuth(ctx, ref)
	if err != nil {
		return err
	}

	for _, k := range keys {
		if k == "" {
			return errtypes.BadRequest("eosfs: key is empty")
		}

		attr := &eosclient.Attribute{
			Type: UserAttr,
			Key:  k,
		}

		err := fs.c.UnsetAttr(ctx, auth, attr, false, fn)
		if err != nil {
			return errors.Wrap(err, "eosfs: error unsetting xattr in eos driver")
		}

	}
	return nil
}

func (fs *eosfs) getLockExpiration(ctx context.Context, auth eosclient.Authorization, path string) (*types.Timestamp, bool, error) {
	expiration, err := fs.c.GetAttr(ctx, auth, "sys."+LockExpirationKey, path)
	if err != nil {
		// since the expiration is optional, if we do not find it in the attr
		// just return a nil value, without reporting the error
		if _, ok := err.(errtypes.NotFound); ok {
			return nil, true, nil
		}
		return nil, false, err
	}
	// the expiration value should be unix time encoded
	unixTime, err := strconv.ParseInt(expiration.Val, 10, 64)
	if err != nil {
		return nil, false, errors.Wrap(err, "eosfs: error converting unix time")
	}
	t := time.Unix(unixTime, 0)
	timestamp := &types.Timestamp{
		Seconds: uint64(unixTime),
	}
	return timestamp, t.After(time.Now()), nil
}

func (fs *eosfs) getLockContent(ctx context.Context, auth eosclient.Authorization, path string, expiration *types.Timestamp) (*provider.Lock, error) {
	t, err := fs.c.GetAttr(ctx, auth, "sys."+LockTypeKey, path)
	if err != nil {
		return nil, err
	}
	lockType, err := strconv.ParseInt(t.Val, 10, 32)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: error decoding lock type")
	}

	d, err := fs.c.GetAttr(ctx, auth, "sys."+LockPayloadKey, path)
	if err != nil {
		return nil, err
	}

	data, err := b64.StdEncoding.DecodeString(d.Val)
	if err != nil {
		return nil, err
	}
	l := new(provider.Lock)
	err = json.Unmarshal(data, l)
	if err != nil {
		return nil, err
	}

	l.Type = provider.LockType(lockType)
	l.Expiration = expiration

	return l, nil

}

func (fs *eosfs) removeLockAttrs(ctx context.Context, auth eosclient.Authorization, path string) error {
	err := fs.c.UnsetAttr(ctx, auth, &eosclient.Attribute{
		Type: SystemAttr,
		Key:  LockExpirationKey,
	}, false, path)
	if err != nil {
		// as the expiration time in the lock is optional
		// we will discard the error if the attr is not set
		if !errors.Is(err, eosclient.AttrNotExistsError) {
			return errors.Wrap(err, "eosfs: error unsetting the lock expiration")
		}
	}

	err = fs.c.UnsetAttr(ctx, auth, &eosclient.Attribute{
		Type: SystemAttr,
		Key:  LockTypeKey,
	}, false, path)
	if err != nil {
		return errors.Wrap(err, "eosfs: error unsetting the lock type")
	}

	err = fs.c.UnsetAttr(ctx, auth, &eosclient.Attribute{
		Type: SystemAttr,
		Key:  LockPayloadKey,
	}, false, path)
	if err != nil {
		return errors.Wrap(err, "eosfs: error unsetting the lock payload")
	}

	return nil
}

func (fs *eosfs) getLock(ctx context.Context, auth eosclient.Authorization, user *userpb.User, path string, ref *provider.Reference) (*provider.Lock, error) {
	// the cs3apis require to have the read permission on the resource
	// to get the eventual lock.
	has, err := fs.userHasReadAccess(ctx, user, ref)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: error checking read access to resource")
	}
	if !has {
		return nil, errtypes.BadRequest("user has not read access on resource")
	}

	expiration, valid, err := fs.getLockExpiration(ctx, auth, path)
	if err != nil {
		return nil, err
	}

	if !valid {
		// the previous lock expired
		if err := fs.removeLockAttrs(ctx, auth, path); err != nil {
			return nil, err
		}
		return nil, errtypes.NotFound("lock not found for ref")
	}

	l, err := fs.getLockContent(ctx, auth, path, expiration)
	if err != nil {
		if !errors.Is(err, eosclient.AttrNotExistsError) {
			return nil, errtypes.NotFound("lock not found for ref")
		}
	}
	return l, nil
}

// GetLock returns an existing lock on the given reference
func (fs *eosfs) GetLock(ctx context.Context, ref *provider.Reference) (*provider.Lock, error) {
	path, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: error resolving reference")
	}
	path = fs.wrap(ctx, path)

	user, err := getUser(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: no user in ctx")
	}
	auth, err := fs.getUserAuth(ctx, user, path)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: error getting uid and gid for user")
	}

	return fs.getLock(ctx, auth, user, path, ref)
}

func (fs *eosfs) setLock(ctx context.Context, lock *provider.Lock, path string, check bool) error {
	auth, err := fs.getRootAuth(ctx)
	if err != nil {
		return err
	}

	encodedLock, err := encodeLock(lock)
	if err != nil {
		return errors.Wrap(err, "eosfs: error encoding lock")
	}

	if lock.Expiration != nil {
		// set expiration
		err = fs.c.SetAttr(ctx, auth, &eosclient.Attribute{
			Type: SystemAttr,
			Key:  LockExpirationKey,
			Val:  strconv.FormatUint(lock.Expiration.Seconds, 10),
		}, check, false, path)
		switch {
		case errors.Is(err, eosclient.AttrAlreadyExistsError):
			return errtypes.BadRequest("lock already set")
		case err != nil:
			return err
		}
	}

	// set lock type
	err = fs.c.SetAttr(ctx, auth, &eosclient.Attribute{
		Type: SystemAttr,
		Key:  LockTypeKey,
		Val:  strconv.FormatUint(uint64(lock.Type), 10),
	}, false, false, path)
	if err != nil {
		return errors.Wrap(err, "eosfs: error setting lock type")
	}

	// set payload
	err = fs.c.SetAttr(ctx, auth, &eosclient.Attribute{
		Type: SystemAttr,
		Key:  LockPayloadKey,
		Val:  encodedLock,
	}, false, false, path)
	if err != nil {
		return errors.Wrap(err, "eosfs: error setting lock payload")
	}
	return nil
}

// SetLock puts a lock on the given reference
func (fs *eosfs) SetLock(ctx context.Context, ref *provider.Reference, l *provider.Lock) error {
	if l.Type == provider.LockType_LOCK_TYPE_SHARED {
		return errtypes.NotSupported("shared lock not yet implemented")
	}

	path, err := fs.resolve(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "eosfs: error resolving reference")
	}
	path = fs.wrap(ctx, path)

	user, err := getUser(ctx)
	if err != nil {
		return errors.Wrap(err, "eosfs: no user in ctx")
	}
	auth, err := fs.getUserAuth(ctx, user, path)
	if err != nil {
		return errors.Wrap(err, "eosfs: error getting uid and gid for user")
	}

	_, err = fs.getLock(ctx, auth, user, path, ref)
	if err != nil {
		// if the err is NotFound it is fine, otherwise we have to return
		if _, ok := err.(errtypes.NotFound); !ok {
			return err
		}
	}
	if err == nil {
		// the resource is already locked
		return errtypes.BadRequest("resource already locked")
	}

	// the cs3apis require to have the write permission on the resource
	// to set a lock. because in eos we can set attrs even if the user does
	// not have the write permission, we need to check if the user that made
	// the request has it
	has, err := fs.userHasWriteAccess(ctx, user, ref)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("eosfs: cannot check if user %s has write access on resource", user.Username))
	}
	if !has {
		return errtypes.PermissionDenied(fmt.Sprintf("user %s has not write access on resource", user.Username))
	}

	// the user in the lock could differ from the user in the context
	// in that case, also the user in the lock MUST have the write permission
	if l.User != nil && !utils.UserEqual(user.Id, l.User) {
		has, err := fs.userIDHasWriteAccess(ctx, l.User, ref)
		if err != nil {
			return errors.Wrap(err, "eosfs: cannot check if user has write access on resource")
		}
		if !has {
			return errtypes.PermissionDenied(fmt.Sprintf("user %s has not write access on resource", user.Username))
		}
	}

	return fs.setLock(ctx, l, path, true)
}

func (fs *eosfs) getUserFromID(ctx context.Context, userID *userpb.UserId) (*userpb.User, error) {
	client, err := pool.GetGatewayServiceClient(fs.conf.GatewaySvc)
	if err != nil {
		return nil, err
	}
	res, err := client.GetUser(ctx, &userpb.GetUserRequest{
		UserId: userID,
	})

	if err != nil {
		return nil, err
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return nil, errtypes.InternalError(res.Status.Message)
	}
	return res.User, nil
}

func (fs *eosfs) userHasWriteAccess(ctx context.Context, user *userpb.User, ref *provider.Reference) (bool, error) {
	ctx = ctxpkg.ContextSetUser(ctx, user)
	resInfo, err := fs.GetMD(ctx, ref, nil, nil)
	if err != nil {
		return false, err
	}
	return resInfo.PermissionSet.InitiateFileUpload, nil
}

func (fs *eosfs) userIDHasWriteAccess(ctx context.Context, userID *userpb.UserId, ref *provider.Reference) (bool, error) {
	user, err := fs.getUserFromID(ctx, userID)
	if err != nil {
		return false, nil
	}
	return fs.userHasWriteAccess(ctx, user, ref)
}

func (fs *eosfs) userHasReadAccess(ctx context.Context, user *userpb.User, ref *provider.Reference) (bool, error) {
	ctx = ctxpkg.ContextSetUser(ctx, user)
	resInfo, err := fs.GetMD(ctx, ref, nil, nil)
	if err != nil {
		return false, err
	}
	return resInfo.PermissionSet.InitiateFileDownload, nil
}

func encodeLock(l *provider.Lock) (string, error) {
	data, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	return b64.StdEncoding.EncodeToString(data), nil
}

// RefreshLock refreshes an existing lock on the given reference
// TODO: use existingLockId. See https://github.com/cs3org/reva/pull/3286
func (fs *eosfs) RefreshLock(ctx context.Context, ref *provider.Reference, newLock *provider.Lock, _ string) error {
	// TODO (gdelmont): check if the new lock is already expired?

	if newLock.Type == provider.LockType_LOCK_TYPE_SHARED {
		return errtypes.NotSupported("shared lock not yet implemented")
	}

	oldLock, err := fs.GetLock(ctx, ref)
	if err != nil {
		switch err.(type) {
		case errtypes.NotFound:
			// the lock does not exist
			return errtypes.BadRequest("file was not locked")
		default:
			return err
		}
	}

	user, err := getUser(ctx)
	if err != nil {
		return errors.Wrap(err, "eosfs: error getting user")
	}

	// check if the holder is the same of the new lock
	if !sameHolder(oldLock, newLock) {
		return errtypes.BadRequest("caller does not hold the lock")
	}

	path, err := fs.resolve(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "eosfs: error resolving reference")
	}
	path = fs.wrap(ctx, path)

	// the cs3apis require to have the write permission on the resource
	// to set a lock
	has, err := fs.userHasWriteAccess(ctx, user, ref)
	if err != nil {
		return errors.Wrap(err, "eosfs: cannot check if user has write access on resource")
	}
	if !has {
		return errtypes.PermissionDenied(fmt.Sprintf("user %s has not write access on resource", user.Username))
	}

	return fs.setLock(ctx, newLock, path, false)
}

func sameHolder(l1, l2 *provider.Lock) bool {
	same := true
	if l1.User != nil || l2.User != nil {
		same = utils.UserEqual(l1.User, l2.User)
	}
	if l1.AppName != "" || l2.AppName != "" {
		same = l1.AppName == l2.AppName
	}
	return same
}

// Unlock removes an existing lock from the given reference
func (fs *eosfs) Unlock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	if lock.Type == provider.LockType_LOCK_TYPE_SHARED {
		return errtypes.NotSupported("shared lock not yet implemented")
	}

	oldLock, err := fs.GetLock(ctx, ref)
	if err != nil {
		switch err.(type) {
		case errtypes.NotFound:
			// the lock does not exist
			return errtypes.BadRequest("file was not locked")
		default:
			return err
		}
	}

	// check if the lock id of the lock corresponds to the stored lock
	if oldLock.LockId != lock.LockId {
		return errtypes.BadRequest("lock id does not match")
	}

	if !sameHolder(oldLock, lock) {
		return errtypes.BadRequest("caller does not hold the lock")
	}

	user, err := getUser(ctx)
	if err != nil {
		return errors.Wrap(err, "eosfs: error getting user")
	}

	// the cs3apis require to have the write permission on the resource
	// to remove the lock
	has, err := fs.userHasWriteAccess(ctx, user, ref)
	if err != nil {
		return errors.Wrap(err, "eosfs: cannot check if user has write access on resource")
	}
	if !has {
		return errtypes.PermissionDenied(fmt.Sprintf("user %s has not write access on resource", user.Username))
	}

	path, err := fs.resolve(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "eosfs: error resolving reference")
	}
	path = fs.wrap(ctx, path)

	auth, err := fs.getRootAuth(ctx)
	if err != nil {
		return errors.Wrap(err, "eosfs: error getting uid and gid for user")
	}
	return fs.removeLockAttrs(ctx, auth, path)
}

func (fs *eosfs) AddGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	fn, auth, err := fs.resolveRefAndGetAuth(ctx, ref)
	if err != nil {
		return err
	}

	rootAuth, err := fs.getRootAuth(ctx)
	if err != nil {
		return err
	}

	// position where put the ACL
	position := eosclient.StartPosition

	eosACL, err := fs.getEosACL(ctx, g)
	if err != nil {
		return err
	}

	err = fs.c.AddACL(ctx, auth, rootAuth, fn, position, eosACL)
	if err != nil {
		return errors.Wrap(err, "eosfs: error adding acl")
	}
	return nil

}

func (fs *eosfs) DenyGrant(ctx context.Context, ref *provider.Reference, g *provider.Grantee) error {
	fn, auth, err := fs.resolveRefAndGetAuth(ctx, ref)
	if err != nil {
		return err
	}

	position := eosclient.EndPosition

	rootAuth, err := fs.getRootAuth(ctx)
	if err != nil {
		return err
	}

	// empty permissions => deny
	grant := &provider.Grant{
		Grantee:     g,
		Permissions: &provider.ResourcePermissions{},
	}

	eosACL, err := fs.getEosACL(ctx, grant)
	if err != nil {
		return err
	}

	err = fs.c.AddACL(ctx, auth, rootAuth, fn, position, eosACL)
	if err != nil {
		return errors.Wrap(err, "eosfs: error adding acl")
	}
	return nil
}

func (fs *eosfs) getEosACL(ctx context.Context, g *provider.Grant) (*acl.Entry, error) {
	permissions, err := grants.GetACLPerm(g.Permissions)
	if err != nil {
		return nil, err
	}
	t, err := grants.GetACLType(g.Grantee.Type)
	if err != nil {
		return nil, err
	}

	var qualifier string
	if t == acl.TypeUser {
		// if the grantee is a lightweight account, we need to set it accordingly
		if g.Grantee.GetUserId().Type == userpb.UserType_USER_TYPE_LIGHTWEIGHT ||
			g.Grantee.GetUserId().Type == userpb.UserType_USER_TYPE_FEDERATED {
			t = acl.TypeLightweight
			qualifier = g.Grantee.GetUserId().OpaqueId
		} else {
			// since EOS Citrine ACLs are stored with uid, we need to convert username to
			// uid only for users.
			auth, err := fs.getUIDGateway(ctx, g.Grantee.GetUserId())
			if err != nil {
				return nil, err
			}
			qualifier = auth.Role.UID
		}
	} else {
		qualifier = g.Grantee.GetGroupId().OpaqueId
	}

	eosACL := &acl.Entry{
		Qualifier:   qualifier,
		Permissions: permissions,
		Type:        t,
	}
	return eosACL, nil
}

func (fs *eosfs) RemoveGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	eosACLType, err := grants.GetACLType(g.Grantee.Type)
	if err != nil {
		return err
	}

	var recipient string
	if eosACLType == acl.TypeUser {
		// if the grantee is a lightweight account, we need to set it accordingly
		if g.Grantee.GetUserId().Type == userpb.UserType_USER_TYPE_LIGHTWEIGHT ||
			g.Grantee.GetUserId().Type == userpb.UserType_USER_TYPE_FEDERATED {
			eosACLType = acl.TypeLightweight
			recipient = g.Grantee.GetUserId().OpaqueId
		} else {
			// since EOS Citrine ACLs are stored with uid, we need to convert username to uid
			auth, err := fs.getUIDGateway(ctx, g.Grantee.GetUserId())
			if err != nil {
				return err
			}
			recipient = auth.Role.UID
		}
	} else {
		recipient = g.Grantee.GetGroupId().OpaqueId
	}

	eosACL := &acl.Entry{
		Qualifier: recipient,
		Type:      eosACLType,
	}

	fn, auth, err := fs.resolveRefAndGetAuth(ctx, ref)
	if err != nil {
		return err
	}

	rootAuth, err := fs.getRootAuth(ctx)
	if err != nil {
		return err
	}

	err = fs.c.RemoveACL(ctx, auth, rootAuth, fn, eosACL)
	if err != nil {
		return errors.Wrap(err, "eosfs: error removing acl")
	}
	return nil
}

func (fs *eosfs) UpdateGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	return fs.AddGrant(ctx, ref, g)
}

func (fs *eosfs) ListGrants(ctx context.Context, ref *provider.Reference) ([]*provider.Grant, error) {
	fn, auth, err := fs.resolveRefAndGetAuth(ctx, ref)
	if err != nil {
		return nil, err
	}

	acls, err := fs.c.ListACLs(ctx, auth, fn)
	if err != nil {
		return nil, err
	}

	grantList := []*provider.Grant{}
	for _, a := range acls {
		var grantee *provider.Grantee
		switch a.Type {
		case acl.TypeUser:
			// EOS Citrine ACLs are stored with uid for users.
			// This needs to be resolved to the user opaque ID.
			qualifier, err := fs.getUserIDGateway(ctx, a.Qualifier)
			if err != nil {
				return nil, err
			}
			grantee = &provider.Grantee{
				Id:   &provider.Grantee_UserId{UserId: qualifier},
				Type: grants.GetGranteeType(a.Type),
			}
		case acl.TypeLightweight:
			a.Type = acl.TypeUser
			grantee = &provider.Grantee{
				Id:   &provider.Grantee_UserId{UserId: &userpb.UserId{OpaqueId: a.Qualifier}},
				Type: grants.GetGranteeType(a.Type),
			}
		default:
			grantee = &provider.Grantee{
				Id:   &provider.Grantee_GroupId{GroupId: &grouppb.GroupId{OpaqueId: a.Qualifier}},
				Type: grants.GetGranteeType(a.Type),
			}
		}

		grantList = append(grantList, &provider.Grant{
			Grantee:     grantee,
			Permissions: grants.GetGrantPermissionSet(a.Permissions),
		})
	}

	return grantList, nil
}

func (fs *eosfs) GetMD(ctx context.Context, ref *provider.Reference, mdKeys []string, fieldMask []string) (*provider.ResourceInfo, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Msg("eosfs: get md for ref:" + ref.String())

	u, err := getUser(ctx)
	if err != nil {
		return nil, err
	}

	fn := ""
	p := ref.Path

	if u.Id.Type == userpb.UserType_USER_TYPE_LIGHTWEIGHT ||
		u.Id.Type == userpb.UserType_USER_TYPE_FEDERATED {
		p, err := fs.resolve(ctx, ref)
		if err != nil {
			return nil, errors.Wrap(err, "eosfs: error resolving reference")
		}

		fn = fs.wrap(ctx, p)
	}

	auth, err := fs.getUserAuth(ctx, u, fn)
	if err != nil {
		return nil, err
	}

	// We handle the case when resource ID is set to avoid making duplicate calls to EOS.
	// In the previous workflow, we would have called the resolve() method which would return
	// the path and then we'll stat the path.
	if ref.ResourceId != nil {
		fid, err := strconv.ParseUint(ref.ResourceId.OpaqueId, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error converting string to int for eos fileid: %s", ref.ResourceId.OpaqueId)
		}

		eosFileInfo, err := fs.c.GetFileInfoByInode(ctx, auth, fid)
		if err != nil {
			return nil, err
		}

		// If it's not a relative reference, return now, else we need to append the path
		if !utils.IsRelativeReference(ref) {
			return fs.convertToResourceInfo(ctx, eosFileInfo)
		}

		parent, err := fs.unwrap(ctx, eosFileInfo.File)
		if err != nil {
			return nil, err
		}

		p = path.Join(parent, p)
	}

	// if path is home we need to add in the response any shadow folder in the shadow homedirectory.
	if fs.conf.EnableHome {
		if fs.isShareFolder(ctx, p) {
			return fs.getMDShareFolder(ctx, p, mdKeys)
		}
	}

	fn = fs.wrap(ctx, p)
	eosFileInfo, err := fs.c.GetFileInfoByPath(ctx, auth, fn)
	if err != nil {
		return nil, err
	}

	return fs.convertToResourceInfo(ctx, eosFileInfo)
}

func (fs *eosfs) getMDShareFolder(ctx context.Context, p string, mdKeys []string) (*provider.ResourceInfo, error) {
	fn := fs.wrapShadow(ctx, p)

	u, err := getUser(ctx)
	if err != nil {
		return nil, err
	}

	// lightweight accounts don't have share folders, so we're passing an empty string as path
	auth, err := fs.getUserAuth(ctx, u, "")
	if err != nil {
		return nil, err
	}

	eosFileInfo, err := fs.c.GetFileInfoByPath(ctx, auth, fn)
	if err != nil {
		return nil, err
	}

	if fs.isShareFolderRoot(ctx, p) {
		return fs.convertToResourceInfo(ctx, eosFileInfo)
	}
	return fs.convertToFileReference(ctx, eosFileInfo)
}

func (fs *eosfs) ListFolder(ctx context.Context, ref *provider.Reference, mdKeys, fieldMask []string) ([]*provider.ResourceInfo, error) {
	p, err := fs.resolve(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: error resolving reference")
	}

	// if path is home we need to add in the response any shadow folder in the shadow homedirectory.
	if fs.conf.EnableHome {
		return fs.listWithHome(ctx, p)
	}

	return fs.listWithNominalHome(ctx, p)
}

func (fs *eosfs) listWithNominalHome(ctx context.Context, p string) (finfos []*provider.ResourceInfo, err error) {
	log := appctx.GetLogger(ctx)
	fn := fs.wrap(ctx, p)

	u, err := getUser(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: no user in ctx")
	}
	auth, err := fs.getUserAuth(ctx, u, fn)
	if err != nil {
		return nil, err
	}

	eosFileInfos, err := fs.c.List(ctx, auth, fn)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: error listing")
	}

	for _, eosFileInfo := range eosFileInfos {
		// filter out sys files
		if !fs.conf.ShowHiddenSysFiles {
			base := path.Base(eosFileInfo.File)
			if hiddenReg.MatchString(base) {
				log.Debug().Msgf("eosfs: path is filtered because is considered hidden: path=%s hiddenReg=%s", base, hiddenReg)
				continue
			}
		}

		// Remove the hidden folders in the topmost directory
		if finfo, err := fs.convertToResourceInfo(ctx, eosFileInfo); err == nil && finfo.Path != "/" && !strings.HasPrefix(finfo.Path, "/.") {
			finfos = append(finfos, finfo)
		}
	}

	return finfos, nil
}

func (fs *eosfs) listWithHome(ctx context.Context, p string) ([]*provider.ResourceInfo, error) {
	if p == "/" {
		return fs.listHome(ctx)
	}

	if fs.isShareFolderRoot(ctx, p) {
		return fs.listShareFolderRoot(ctx, p)
	}

	if fs.isShareFolderChild(ctx, p) {
		return nil, errtypes.PermissionDenied("eosfs: error listing folders inside the shared folder, only file references are stored inside")
	}

	// path points to a resource in the nominal home
	return fs.listWithNominalHome(ctx, p)
}

func (fs *eosfs) listHome(ctx context.Context) ([]*provider.ResourceInfo, error) {
	fns := []string{fs.wrap(ctx, "/"), fs.wrapShadow(ctx, "/")}

	u, err := getUser(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: no user in ctx")
	}
	// lightweight accounts don't have home folders, so we're passing an empty string as path
	auth, err := fs.getUserAuth(ctx, u, "")
	if err != nil {
		return nil, err
	}

	finfos := []*provider.ResourceInfo{}
	for _, fn := range fns {
		eosFileInfos, err := fs.c.List(ctx, auth, fn)
		if err != nil {
			return nil, errors.Wrap(err, "eosfs: error listing")
		}

		for _, eosFileInfo := range eosFileInfos {
			// filter out sys files
			if !fs.conf.ShowHiddenSysFiles {
				base := path.Base(eosFileInfo.File)
				if hiddenReg.MatchString(base) {
					continue
				}
			}

			if finfo, err := fs.convertToResourceInfo(ctx, eosFileInfo); err == nil && finfo.Path != "/" && !strings.HasPrefix(finfo.Path, "/.") {
				finfos = append(finfos, finfo)
			}
		}

	}
	return finfos, nil
}

func (fs *eosfs) listShareFolderRoot(ctx context.Context, p string) (finfos []*provider.ResourceInfo, err error) {
	fn := fs.wrapShadow(ctx, p)

	u, err := getUser(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: no user in ctx")
	}
	// lightweight accounts don't have share folders, so we're passing an empty string as path
	auth, err := fs.getUserAuth(ctx, u, "")
	if err != nil {
		return nil, err
	}

	eosFileInfos, err := fs.c.List(ctx, auth, fn)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: error listing")
	}

	for _, eosFileInfo := range eosFileInfos {
		// filter out sys files
		if !fs.conf.ShowHiddenSysFiles {
			base := path.Base(eosFileInfo.File)
			if hiddenReg.MatchString(base) {
				continue
			}
		}

		if finfo, err := fs.convertToFileReference(ctx, eosFileInfo); err == nil {
			finfos = append(finfos, finfo)
		}
	}

	return finfos, nil
}

func (fs *eosfs) GetQuota(ctx context.Context, ref *provider.Reference) (uint64, uint64, uint64, error) {
	// Check if the quota request is for the user's home or a project space
	u, err := getUser(ctx)
	if err != nil {
		return 0, 0, 0, errors.Wrap(err, "eosfs: no user in ctx")
	}

	// If the quota request is for a resource different than the user home,
	// we impersonate the owner in that case
	uid := strconv.FormatInt(u.UidNumber, 10)
	if ref.ResourceId != nil {
		fid, err := strconv.ParseUint(ref.ResourceId.OpaqueId, 10, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("error converting string to int for eos fileid: %s", ref.ResourceId.OpaqueId)
		}

		// lightweight accounts don't have quota nodes, so we're passing an empty string as path
		auth, err := fs.getUserAuth(ctx, u, "")
		if err != nil {
			return 0, 0, 0, errors.Wrap(err, "eosfs: error getting uid and gid for user")
		}
		eosFileInfo, err := fs.c.GetFileInfoByInode(ctx, auth, fid)
		if err != nil {
			return 0, 0, 0, err
		}
		uid = strconv.FormatUint(eosFileInfo.UID, 10)
	}

	rootAuth, err := fs.getRootAuth(ctx)
	if err != nil {
		return 0, 0, 0, err
	}

	qi, err := fs.c.GetQuota(ctx, uid, rootAuth, fs.conf.QuotaNode)
	if err != nil {
		err := errors.Wrap(err, "eosfs: error getting quota")
		return 0, 0, 0, err
	}

	remaining := qi.AvailableBytes - qi.UsedBytes

	return qi.AvailableBytes, qi.UsedBytes, remaining, nil
}

func (fs *eosfs) GetHome(ctx context.Context) (string, error) {
	if !fs.conf.EnableHome {
		return "", errtypes.NotSupported("eosfs: get home not supported")
	}

	// eos drive for homes assumes root(/) points to the user home.
	return "/", nil
}

func (fs *eosfs) createShadowHome(ctx context.Context, home string) error {
	u, err := getUser(ctx)
	if err != nil {
		return errors.Wrap(err, "eosfs: no user in ctx")
	}
	rootAuth, err := fs.getRootAuth(ctx)
	if err != nil {
		return nil
	}
	shadowFolders := []string{fs.conf.ShareFolder}

	for _, sf := range shadowFolders {
		fn := path.Join(home, sf)
		_, err = fs.c.GetFileInfoByPath(ctx, rootAuth, fn)
		if err != nil {
			if _, ok := err.(errtypes.IsNotFound); !ok {
				return errors.Wrap(err, "eosfs: error verifying if shadow directory exists")
			}
			err = fs.createUserDir(ctx, u, fn, false)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (fs *eosfs) createNominalHome(ctx context.Context, home string) error {
	u, err := getUser(ctx)
	if err != nil {
		return errors.Wrap(err, "eosfs: no user in ctx")
	}
	auth, err := fs.getUserAuth(ctx, u, "")
	if err != nil {
		return err
	}

	rootAuth, err := fs.getRootAuth(ctx)
	if err != nil {
		return nil
	}

	_, err = fs.c.GetFileInfoByPath(ctx, rootAuth, home)
	if err == nil { // home already exists
		return nil
	}

	if _, ok := err.(errtypes.IsNotFound); !ok {
		return errors.Wrap(err, "eosfs: error verifying if user home directory exists")
	}

	err = fs.createUserDir(ctx, u, home, false)
	if err != nil {
		err := errors.Wrap(err, "eosfs: error creating user dir")
		return err
	}

	// set quota for user
	quotaInfo := &eosclient.SetQuotaInfo{
		Username:  u.Username,
		UID:       auth.Role.UID,
		GID:       auth.Role.GID,
		MaxBytes:  fs.conf.DefaultQuotaBytes,
		MaxFiles:  fs.conf.DefaultQuotaFiles,
		QuotaNode: fs.conf.QuotaNode,
	}

	err = fs.c.SetQuota(ctx, rootAuth, quotaInfo)
	if err != nil {
		err := errors.Wrap(err, "eosfs: error setting quota")
		return err
	}

	return err
}

func (fs *eosfs) CreateHome(ctx context.Context) error {
	if !fs.conf.EnableHome {
		return errtypes.NotSupported("eosfs: create home not supported")
	}

	if err := fs.createNominalHome(ctx, fs.wrap(ctx, "/")); err != nil {
		return errors.Wrap(err, "eosfs: error creating nominal home")
	}

	if err := fs.createShadowHome(ctx, fs.wrapShadow(ctx, "/")); err != nil {
		return errors.Wrap(err, "eosfs: error creating shadow home")
	}

	return nil
}

func (fs *eosfs) createUserDir(ctx context.Context, u *userpb.User, path string, recursiveAttr bool) error {
	rootAuth, err := fs.getRootAuth(ctx)
	if err != nil {
		return nil
	}

	chownAuth, err := fs.getUserAuth(ctx, u, "")
	if err != nil {
		return err
	}

	err = fs.c.CreateDir(ctx, rootAuth, path)
	if err != nil {
		// EOS will return success on mkdir over an existing directory.
		return errors.Wrap(err, "eosfs: error creating dir")
	}

	err = fs.c.Chown(ctx, rootAuth, chownAuth, path)
	if err != nil {
		return errors.Wrap(err, "eosfs: error chowning directory")
	}

	err = fs.c.Chmod(ctx, rootAuth, "2770", path)
	if err != nil {
		return errors.Wrap(err, "eosfs: error chmoding directory")
	}

	attrs := []*eosclient.Attribute{
		{
			Type: SystemAttr,
			Key:  "mask",
			Val:  "700",
		},
		{
			Type: SystemAttr,
			Key:  "allow.oc.sync",
			Val:  "1",
		},
		{
			Type: SystemAttr,
			Key:  "mtime.propagation",
			Val:  "1",
		},
		{
			Type: SystemAttr,
			Key:  "forced.atomic",
			Val:  "1",
		},
	}

	for _, attr := range attrs {
		err = fs.c.SetAttr(ctx, rootAuth, attr, false, recursiveAttr, path)
		if err != nil {
			return errors.Wrap(err, "eosfs: error setting attribute")
		}
	}

	return nil
}

func (fs *eosfs) CreateDir(ctx context.Context, ref *provider.Reference) error {
	log := appctx.GetLogger(ctx)
	p, err := fs.resolve(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "eosfs: error resolving reference")
	}
	if fs.isShareFolder(ctx, p) {
		return errtypes.PermissionDenied("eosfs: cannot perform operation under the virtual share folder")
	}
	fn := fs.wrap(ctx, p)

	u, err := getUser(ctx)
	if err != nil {
		return errors.Wrap(err, "eosfs: no user in ctx")
	}

	// We need the auth corresponding to the parent directory
	// as the file might not exist at the moment
	auth, err := fs.getUserAuth(ctx, u, path.Dir(fn))
	if err != nil {
		return err
	}

	log.Info().Msgf("eosfs: createdir: path=%s", fn)
	return fs.c.CreateDir(ctx, auth, fn)
}

// TouchFile as defined in the storage.FS interface
func (fs *eosfs) TouchFile(ctx context.Context, ref *provider.Reference, _ bool, _ string) error {
	log := appctx.GetLogger(ctx)

	fn, auth, err := fs.resolveRefAndGetAuth(ctx, ref)
	if err != nil {
		return err
	}
	log.Info().Msgf("eosfs: touch file: path=%s", fn)

	return fs.c.Touch(ctx, auth, fn)
}

func (fs *eosfs) CreateReference(ctx context.Context, p string, targetURI *url.URL) error {
	// TODO(labkode): for the time being we only allow creating references
	// in the virtual share folder to not pollute the nominal user tree.
	if !fs.isShareFolder(ctx, p) {
		return errtypes.PermissionDenied("eosfs: cannot create references outside the share folder: share_folder=" + fs.conf.ShareFolder + " path=" + p)
	}
	u, err := getUser(ctx)
	if err != nil {
		return errors.Wrap(err, "eosfs: no user in ctx")
	}

	fn := fs.wrapShadow(ctx, p)

	// TODO(labkode): with the grpc plugin we can create a file touching with xattrs.
	// Current mechanism is: touch to hidden dir, set xattr, rename.
	dir, base := path.Split(fn)
	tmp := path.Join(dir, fmt.Sprintf(".sys.reva#.%s", base))
	rootAuth, err := fs.getRootAuth(ctx)
	if err != nil {
		return nil
	}

	if err := fs.createUserDir(ctx, u, tmp, false); err != nil {
		err = errors.Wrapf(err, "eosfs: error creating temporary ref file")
		return err
	}

	// set xattr on ref
	attr := &eosclient.Attribute{
		Type: UserAttr,
		Key:  refTargetAttrKey,
		Val:  targetURI.String(),
	}

	if err := fs.c.SetAttr(ctx, rootAuth, attr, false, false, tmp); err != nil {
		err = errors.Wrapf(err, "eosfs: error setting reva.ref attr on file: %q", tmp)
		return err
	}

	// rename to have the file visible in user space.
	if err := fs.c.Rename(ctx, rootAuth, tmp, fn); err != nil {
		err = errors.Wrapf(err, "eosfs: error renaming from: %q to %q", tmp, fn)
		return err
	}

	return nil
}

func (fs *eosfs) Delete(ctx context.Context, ref *provider.Reference) error {
	p, err := fs.resolve(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "eosfs: error resolving reference")
	}

	if fs.isShareFolder(ctx, p) {
		return fs.deleteShadow(ctx, p)
	}

	fn := fs.wrap(ctx, p)

	u, err := getUser(ctx)
	if err != nil {
		return errors.Wrap(err, "eosfs: no user in ctx")
	}
	auth, err := fs.getUserAuth(ctx, u, fn)
	if err != nil {
		return err
	}

	return fs.c.Remove(ctx, auth, fn, false)
}

func (fs *eosfs) deleteShadow(ctx context.Context, p string) error {
	if fs.isShareFolderRoot(ctx, p) {
		return errtypes.PermissionDenied("eosfs: cannot delete the virtual share folder")
	}

	if fs.isShareFolderChild(ctx, p) {
		fn := fs.wrapShadow(ctx, p)

		// in order to remove the folder or the file without
		// moving it to the recycle bin, we should take
		// the privileges of the root
		auth, err := fs.getRootAuth(ctx)
		if err != nil {
			return err
		}

		return fs.c.Remove(ctx, auth, fn, true)
	}

	return errors.New("eosfs: shadow delete of share folder that is neither root nor child. path=" + p)
}

func (fs *eosfs) Move(ctx context.Context, oldRef, newRef *provider.Reference) error {
	oldPath, err := fs.resolve(ctx, oldRef)
	if err != nil {
		return errors.Wrap(err, "eosfs: error resolving reference")
	}

	newPath, err := fs.resolve(ctx, newRef)
	if err != nil {
		return errors.Wrap(err, "eosfs: error resolving reference")
	}

	if fs.isShareFolder(ctx, oldPath) || fs.isShareFolder(ctx, newPath) {
		return fs.moveShadow(ctx, oldPath, newPath)
	}

	oldFn := fs.wrap(ctx, oldPath)
	newFn := fs.wrap(ctx, newPath)

	u, err := getUser(ctx)
	if err != nil {
		return errors.Wrap(err, "eosfs: no user in ctx")
	}
	auth, err := fs.getUserAuth(ctx, u, oldFn)
	if err != nil {
		return err
	}

	return fs.c.Rename(ctx, auth, oldFn, newFn)
}

func (fs *eosfs) moveShadow(ctx context.Context, oldPath, newPath string) error {
	if fs.isShareFolderRoot(ctx, oldPath) || fs.isShareFolderRoot(ctx, newPath) {
		return errtypes.PermissionDenied("eosfs: cannot move/rename the virtual share folder")
	}

	// only rename of the reference is allowed, hence having the same basedir
	bold, _ := path.Split(oldPath)
	bnew, _ := path.Split(newPath)

	if bold != bnew {
		return errtypes.PermissionDenied("eosfs: cannot move references under the virtual share folder")
	}

	oldfn := fs.wrapShadow(ctx, oldPath)
	newfn := fs.wrapShadow(ctx, newPath)

	u, err := getUser(ctx)
	if err != nil {
		return errors.Wrap(err, "eosfs: no user in ctx")
	}
	auth, err := fs.getUserAuth(ctx, u, "")
	if err != nil {
		return err
	}

	return fs.c.Rename(ctx, auth, oldfn, newfn)
}

func (fs *eosfs) Download(ctx context.Context, ref *provider.Reference, openReaderfunc func(*provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	fn, auth, err := fs.resolveRefForbidShareFolder(ctx, ref)
	if err != nil {
		return nil, nil, err
	}

	md, err := fs.GetMD(ctx, ref, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	if !openReaderfunc(md) {
		return md, nil, nil
	}

	reader, err := fs.c.Read(ctx, auth, fn)
	return md, reader, err
}

func (fs *eosfs) ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error) {
	var auth eosclient.Authorization
	var fn string
	var err error

	if !fs.conf.EnableHome && fs.conf.ImpersonateOwnerforRevisions {
		// We need to access the revisions for a non-home reference.
		// We'll get the owner of the particular resource and impersonate them
		// if we have access to it.
		md, err := fs.GetMD(ctx, ref, nil, nil)
		if err != nil {
			return nil, err
		}
		fn = fs.wrap(ctx, md.Path)

		if md.PermissionSet.ListFileVersions {
			auth, err = fs.getUIDGateway(ctx, md.Owner)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errtypes.PermissionDenied("eosfs: user doesn't have permissions to list revisions")
		}
	} else {
		fn, auth, err = fs.resolveRefForbidShareFolder(ctx, ref)
		if err != nil {
			return nil, err
		}
	}

	eosRevisions, err := fs.c.ListVersions(ctx, auth, fn)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: error listing versions")
	}
	revisions := []*provider.FileVersion{}
	for _, eosRev := range eosRevisions {
		if rev, err := fs.convertToRevision(ctx, eosRev); err == nil {
			revisions = append(revisions, rev)
		}
	}
	return revisions, nil
}

func (fs *eosfs) DownloadRevision(ctx context.Context, ref *provider.Reference, revisionKey string, openReaderfunc func(*provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	var auth eosclient.Authorization
	var fn string
	var err error

	md, err := fs.GetMD(ctx, ref, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	if !fs.conf.EnableHome && fs.conf.ImpersonateOwnerforRevisions {
		// We need to access the revisions for a non-home reference.
		// We'll get the owner of the particular resource and impersonate them
		// if we have access to it.

		fn = fs.wrap(ctx, md.Path)
		if md.PermissionSet.InitiateFileDownload {
			auth, err = fs.getUIDGateway(ctx, md.Owner)
			if err != nil {
				return nil, nil, err
			}
		} else {
			return nil, nil, errtypes.PermissionDenied("eosfs: user doesn't have permissions to download revisions")
		}
	} else {
		fn, auth, err = fs.resolveRefForbidShareFolder(ctx, ref)
		if err != nil {
			return nil, nil, err
		}
	}

	if !openReaderfunc(md) {
		return md, nil, nil
	}

	reader, err := fs.c.ReadVersion(ctx, auth, fn, revisionKey)
	return md, reader, err
}

func (fs *eosfs) RestoreRevision(ctx context.Context, ref *provider.Reference, revisionKey string) error {
	var auth eosclient.Authorization
	var fn string
	var err error

	if !fs.conf.EnableHome && fs.conf.ImpersonateOwnerforRevisions {
		// We need to access the revisions for a non-home reference.
		// We'll get the owner of the particular resource and impersonate them
		// if we have access to it.
		md, err := fs.GetMD(ctx, ref, nil, nil)
		if err != nil {
			return err
		}
		fn = fs.wrap(ctx, md.Path)

		if md.PermissionSet.RestoreFileVersion {
			auth, err = fs.getUIDGateway(ctx, md.Owner)
			if err != nil {
				return err
			}
		} else {
			return errtypes.PermissionDenied("eosfs: user doesn't have permissions to restore revisions")
		}
	} else {
		fn, auth, err = fs.resolveRefForbidShareFolder(ctx, ref)
		if err != nil {
			return err
		}
	}

	return fs.c.RollbackToVersion(ctx, auth, fn, revisionKey)
}

func (fs *eosfs) PurgeRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string) error {
	return errtypes.NotSupported("eosfs: operation not supported")
}

func (fs *eosfs) EmptyRecycle(ctx context.Context, ref *provider.Reference) error {
	u, err := getUser(ctx)
	if err != nil {
		return errors.Wrap(err, "eosfs: no user in ctx")
	}
	auth, err := fs.getUserAuth(ctx, u, "")
	if err != nil {
		return err
	}

	return fs.c.PurgeDeletedEntries(ctx, auth)
}

func (fs *eosfs) ListRecycle(ctx context.Context, ref *provider.Reference, key, relativePath string) ([]*provider.RecycleItem, error) {
	var auth eosclient.Authorization

	if !fs.conf.EnableHome && fs.conf.AllowPathRecycleOperations && ref.Path != "/" {
		// We need to access the recycle bin for a non-home reference.
		// We'll get the owner of the particular resource and impersonate them
		// if we have access to it.
		md, err := fs.GetMD(ctx, &provider.Reference{Path: ref.Path}, nil, nil)
		if err != nil {
			return nil, err
		}
		if md.PermissionSet.ListRecycle {
			auth, err = fs.getUIDGateway(ctx, md.Owner)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errtypes.PermissionDenied("eosfs: user doesn't have permissions to restore recycled items")
		}
	} else {
		// We just act on the logged-in user's recycle bin
		u, err := getUser(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "eosfs: no user in ctx")
		}
		auth, err = fs.getUserAuth(ctx, u, "")
		if err != nil {
			return nil, err
		}
	}

	eosDeletedEntries, err := fs.c.ListDeletedEntries(ctx, auth)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: error listing deleted entries")
	}
	recycleEntries := []*provider.RecycleItem{}
	for _, entry := range eosDeletedEntries {
		if !fs.conf.ShowHiddenSysFiles {
			base := path.Base(entry.RestorePath)
			if hiddenReg.MatchString(base) {
				continue
			}

		}
		if recycleItem, err := fs.convertToRecycleItem(ctx, entry); err == nil {
			recycleEntries = append(recycleEntries, recycleItem)
		}
	}
	return recycleEntries, nil
}

func (fs *eosfs) RestoreRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string, restoreRef *provider.Reference) error {
	var auth eosclient.Authorization

	if !fs.conf.EnableHome && fs.conf.AllowPathRecycleOperations && ref.Path != "/" {
		// We need to access the recycle bin for a non-home reference.
		// We'll get the owner of the particular resource and impersonate them
		// if we have access to it.
		md, err := fs.GetMD(ctx, &provider.Reference{Path: ref.Path}, nil, nil)
		if err != nil {
			return err
		}
		if md.PermissionSet.RestoreRecycleItem {
			auth, err = fs.getUIDGateway(ctx, md.Owner)
			if err != nil {
				return err
			}
		} else {
			return errtypes.PermissionDenied("eosfs: user doesn't have permissions to restore recycled items")
		}
	} else {
		// We just act on the logged-in user's recycle bin
		u, err := getUser(ctx)
		if err != nil {
			return errors.Wrap(err, "eosfs: no user in ctx")
		}
		auth, err = fs.getUserAuth(ctx, u, "")
		if err != nil {
			return err
		}
	}

	return fs.c.RestoreDeletedEntry(ctx, auth, key)
}

func (fs *eosfs) convertToRecycleItem(ctx context.Context, eosDeletedItem *eosclient.DeletedEntry) (*provider.RecycleItem, error) {
	path, err := fs.unwrap(ctx, eosDeletedItem.RestorePath)
	if err != nil {
		return nil, err
	}

	recycleItem := &provider.RecycleItem{
		Ref:          &provider.Reference{Path: path},
		Key:          eosDeletedItem.RestoreKey,
		Size:         eosDeletedItem.Size,
		DeletionTime: &types.Timestamp{Seconds: eosDeletedItem.DeletionMTime},
	}
	if eosDeletedItem.IsDir {
		recycleItem.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER
	} else {
		// TODO(labkode): if eos returns more types oin the future we need to map them.
		recycleItem.Type = provider.ResourceType_RESOURCE_TYPE_FILE
	}
	return recycleItem, nil
}

func (fs *eosfs) convertToRevision(ctx context.Context, eosFileInfo *eosclient.FileInfo) (*provider.FileVersion, error) {
	md, err := fs.convertToResourceInfo(ctx, eosFileInfo)
	if err != nil {
		return nil, err
	}
	revision := &provider.FileVersion{
		Key:   path.Base(md.Path),
		Size:  md.Size,
		Mtime: md.Mtime.Seconds, // TODO do we need nanos here?
		Etag:  md.Etag,
	}
	return revision, nil
}

func (fs *eosfs) convertToResourceInfo(ctx context.Context, eosFileInfo *eosclient.FileInfo) (*provider.ResourceInfo, error) {
	return fs.convert(ctx, eosFileInfo)
}

func (fs *eosfs) convertToFileReference(ctx context.Context, eosFileInfo *eosclient.FileInfo) (*provider.ResourceInfo, error) {
	info, err := fs.convert(ctx, eosFileInfo)
	if err != nil {
		return nil, err
	}
	info.Type = provider.ResourceType_RESOURCE_TYPE_REFERENCE
	val, ok := eosFileInfo.Attrs["reva.target"]
	if !ok || val == "" {
		return nil, errtypes.InternalError("eosfs: reference does not contain target: target=" + val + " file=" + eosFileInfo.File)
	}
	info.Target = val
	return info, nil
}

// permissionSet returns the permission set for the current user
func (fs *eosfs) permissionSet(ctx context.Context, eosFileInfo *eosclient.FileInfo, owner *userpb.UserId) *provider.ResourcePermissions {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok || u.Id == nil {
		return &provider.ResourcePermissions{
			// no permissions
		}
	}

	if owner != nil && u.Id.OpaqueId == owner.OpaqueId && u.Id.Idp == owner.Idp {
		// The logged-in user is the owner but we may be impersonating them
		// on behalf of a public share accessor.

		// NOTE: This will grant the user full access when the opaque is nil
		// it is likely that this can be used for attacks
		if u.Opaque != nil {
			// FIXME: "editor" and "viewer" are not sufficient anymore, they could have different permissions
			// The role names should not be hardcoded any more as they will come from config in the future
			if publicShare, ok := u.Opaque.Map["public-share-role"]; ok {
				if string(publicShare.Value) == "editor" {
					return conversions.NewEditorRole().CS3ResourcePermissions()
				} else if string(publicShare.Value) == "uploader" {
					return conversions.NewUploaderRole().CS3ResourcePermissions()
				}
				// Default to viewer role
				return conversions.NewViewerRole().CS3ResourcePermissions()
			}
		}

		// owner has all permissions
		return conversions.NewManagerRole().CS3ResourcePermissions()
	}

	auth, err := fs.getUserAuth(ctx, u, eosFileInfo.File)
	if err != nil {
		return &provider.ResourcePermissions{
			// no permissions
		}
	}

	if eosFileInfo.SysACL == nil {
		return &provider.ResourcePermissions{
			// no permissions
		}
	}
	var perm provider.ResourcePermissions

	for _, e := range eosFileInfo.SysACL.Entries {
		var userInGroup bool
		if e.Type == acl.TypeGroup {
			for _, g := range u.Groups {
				if e.Qualifier == g {
					userInGroup = true
					break
				}
			}
		}

		if (e.Type == acl.TypeUser && e.Qualifier == auth.Role.UID) || (e.Type == acl.TypeLightweight && e.Qualifier == u.Id.OpaqueId) || userInGroup {
			mergePermissions(&perm, grants.GetGrantPermissionSet(e.Permissions))
		}
	}

	return &perm
}

func mergePermissions(l *provider.ResourcePermissions, r *provider.ResourcePermissions) {
	l.AddGrant = l.AddGrant || r.AddGrant
	l.CreateContainer = l.CreateContainer || r.CreateContainer
	l.Delete = l.Delete || r.Delete
	l.GetPath = l.GetPath || r.GetPath
	l.GetQuota = l.GetQuota || r.GetQuota
	l.InitiateFileDownload = l.InitiateFileDownload || r.InitiateFileDownload
	l.InitiateFileUpload = l.InitiateFileUpload || r.InitiateFileUpload
	l.ListContainer = l.ListContainer || r.ListContainer
	l.ListFileVersions = l.ListFileVersions || r.ListFileVersions
	l.ListGrants = l.ListGrants || r.ListGrants
	l.ListRecycle = l.ListRecycle || r.ListRecycle
	l.Move = l.Move || r.Move
	l.PurgeRecycle = l.PurgeRecycle || r.PurgeRecycle
	l.RemoveGrant = l.RemoveGrant || r.RemoveGrant
	l.RestoreFileVersion = l.RestoreFileVersion || r.RestoreFileVersion
	l.RestoreRecycleItem = l.RestoreRecycleItem || r.RestoreRecycleItem
	l.Stat = l.Stat || r.Stat
	l.UpdateGrant = l.UpdateGrant || r.UpdateGrant
	l.DenyGrant = l.DenyGrant || r.DenyGrant
}

func (fs *eosfs) convert(ctx context.Context, eosFileInfo *eosclient.FileInfo) (*provider.ResourceInfo, error) {
	path, err := fs.unwrap(ctx, eosFileInfo.File)
	if err != nil {
		return nil, err
	}
	path = filepath.Join(fs.conf.MountPath, path)

	size := eosFileInfo.Size
	if eosFileInfo.IsDir {
		size = eosFileInfo.TreeSize
	}

	owner, err := fs.getUserIDGateway(ctx, strconv.FormatUint(eosFileInfo.UID, 10))
	if err != nil {
		sublog := appctx.GetLogger(ctx).With().Logger()
		sublog.Warn().Uint64("uid", eosFileInfo.UID).Msg("could not lookup userid, leaving empty")
	}

	var xs provider.ResourceChecksum
	if eosFileInfo.XS != nil {
		xs.Sum = eosFileInfo.XS.XSSum
		switch eosFileInfo.XS.XSType {
		case "adler":
			xs.Type = provider.ResourceChecksumType_RESOURCE_CHECKSUM_TYPE_ADLER32
		default:
			xs.Type = provider.ResourceChecksumType_RESOURCE_CHECKSUM_TYPE_INVALID
		}
	}

	// filter 'sys' attrs and the reserved lock
	filteredAttrs := make(map[string]string)
	for k, v := range eosFileInfo.Attrs {
		if !strings.HasPrefix(k, "sys") {
			filteredAttrs[k] = v
		}
	}

	info := &provider.ResourceInfo{
		Id:            &provider.ResourceId{OpaqueId: fmt.Sprintf("%d", eosFileInfo.Inode)},
		Path:          path,
		Owner:         owner,
		Etag:          fmt.Sprintf("\"%s\"", strings.Trim(eosFileInfo.ETag, "\"")),
		MimeType:      mime.Detect(eosFileInfo.IsDir, path),
		Size:          size,
		ParentId:      &provider.ResourceId{OpaqueId: fmt.Sprintf("%d", eosFileInfo.FID)},
		PermissionSet: fs.permissionSet(ctx, eosFileInfo, owner),
		Checksum:      &xs,
		Type:          getResourceType(eosFileInfo.IsDir),
		Mtime: &types.Timestamp{
			Seconds: eosFileInfo.MTimeSec,
			Nanos:   eosFileInfo.MTimeNanos,
		},
		Opaque: &types.Opaque{
			Map: map[string]*types.OpaqueEntry{
				"eos": {
					Decoder: "json",
					Value:   fs.getEosMetadata(eosFileInfo),
				},
			},
		},
		ArbitraryMetadata: &provider.ArbitraryMetadata{
			Metadata: filteredAttrs,
		},
	}

	if eosFileInfo.IsDir {
		info.Opaque.Map["disable_tus"] = &types.OpaqueEntry{
			Decoder: "plain",
			Value:   []byte("true"),
		}
	}

	return info, nil
}

func getResourceType(isDir bool) provider.ResourceType {
	if isDir {
		return provider.ResourceType_RESOURCE_TYPE_CONTAINER
	}
	return provider.ResourceType_RESOURCE_TYPE_FILE
}

func (fs *eosfs) extractUIDAndGID(u *userpb.User) (eosclient.Authorization, error) {
	if u.UidNumber == 0 {
		return eosclient.Authorization{}, errors.New("eosfs: uid missing for user")
	}
	if u.GidNumber == 0 {
		return eosclient.Authorization{}, errors.New("eosfs: gid missing for user")
	}
	return eosclient.Authorization{Role: eosclient.Role{UID: strconv.FormatInt(u.UidNumber, 10), GID: strconv.FormatInt(u.GidNumber, 10)}}, nil
}

func (fs *eosfs) getUIDGateway(ctx context.Context, u *userpb.UserId) (eosclient.Authorization, error) {
	log := appctx.GetLogger(ctx)
	if userIDInterface, err := fs.userIDCache.Get(u.OpaqueId); err == nil {
		log.Debug().Msg("eosfs: found cached user " + u.OpaqueId)
		return fs.extractUIDAndGID(userIDInterface.(*userpb.User))
	}

	client, err := pool.GetGatewayServiceClient(fs.conf.GatewaySvc)
	if err != nil {
		return eosclient.Authorization{}, errors.Wrap(err, "eosfs: error getting gateway grpc client")
	}
	getUserResp, err := client.GetUser(ctx, &userpb.GetUserRequest{
		UserId:                 u,
		SkipFetchingUserGroups: true,
	})
	if err != nil {
		_ = fs.userIDCache.SetWithTTL(u.OpaqueId, &userpb.User{}, 12*time.Hour)
		return eosclient.Authorization{}, errors.Wrap(err, "eosfs: error getting user")
	}
	if getUserResp.Status.Code != rpc.Code_CODE_OK {
		_ = fs.userIDCache.SetWithTTL(u.OpaqueId, &userpb.User{}, 12*time.Hour)
		return eosclient.Authorization{}, status.NewErrorFromCode(getUserResp.Status.Code, "eosfs")
	}

	_ = fs.userIDCache.Set(u.OpaqueId, getUserResp.User)
	return fs.extractUIDAndGID(getUserResp.User)
}

func (fs *eosfs) getUserIDGateway(ctx context.Context, uid string) (*userpb.UserId, error) {
	log := appctx.GetLogger(ctx)
	// Handle the case of root
	if uid == "0" {
		return nil, errtypes.BadRequest("eosfs: cannot return root user")
	}

	if userIDInterface, err := fs.userIDCache.Get(uid); err == nil {
		log.Debug().Msg("eosfs: found cached uid " + uid)
		return userIDInterface.(*userpb.UserId), nil
	}

	log.Debug().Msg("eosfs: retrieving user from gateway for uid " + uid)
	client, err := pool.GetGatewayServiceClient(fs.conf.GatewaySvc)
	if err != nil {
		return nil, errors.Wrap(err, "eosfs: error getting gateway grpc client")
	}
	getUserResp, err := client.GetUserByClaim(ctx, &userpb.GetUserByClaimRequest{
		Claim:                  "uid",
		Value:                  uid,
		SkipFetchingUserGroups: true,
	})
	if err != nil {
		// Insert an empty object in the cache so that we don't make another call
		// for a specific amount of time
		_ = fs.userIDCache.SetWithTTL(uid, &userpb.UserId{}, 12*time.Hour)
		return nil, errors.Wrap(err, "eosfs: error getting user")
	}
	if getUserResp.Status.Code != rpc.Code_CODE_OK {
		// Insert an empty object in the cache so that we don't make another call
		// for a specific amount of time
		_ = fs.userIDCache.SetWithTTL(uid, &userpb.UserId{}, 12*time.Hour)
		return nil, status.NewErrorFromCode(getUserResp.Status.Code, "eosfs")
	}

	_ = fs.userIDCache.Set(uid, getUserResp.User.Id)
	return getUserResp.User.Id, nil
}

func (fs *eosfs) getUserAuth(ctx context.Context, u *userpb.User, fn string) (eosclient.Authorization, error) {
	if fs.conf.ForceSingleUserMode {
		if fs.singleUserAuth.Role.UID != "" && fs.singleUserAuth.Role.GID != "" {
			return fs.singleUserAuth, nil
		}
		var err error
		fs.singleUserAuth, err = fs.getUIDGateway(ctx, &userpb.UserId{OpaqueId: fs.conf.SingleUsername})
		return fs.singleUserAuth, err
	}

	if u.Id.Type == userpb.UserType_USER_TYPE_LIGHTWEIGHT ||
		u.Id.Type == userpb.UserType_USER_TYPE_FEDERATED {
		return fs.getEOSToken(ctx, u, fn)
	}

	return fs.extractUIDAndGID(u)
}

func (fs *eosfs) getEOSToken(ctx context.Context, u *userpb.User, fn string) (eosclient.Authorization, error) {
	if fn == "" {
		return eosclient.Authorization{}, errtypes.BadRequest("eosfs: path cannot be empty")
	}

	rootAuth, err := fs.getRootAuth(ctx)
	if err != nil {
		return eosclient.Authorization{}, err
	}
	info, err := fs.c.GetFileInfoByPath(ctx, rootAuth, fn)
	if err != nil {
		return eosclient.Authorization{}, err
	}
	auth := eosclient.Authorization{
		Role: eosclient.Role{
			UID: strconv.FormatUint(info.UID, 10),
			GID: strconv.FormatUint(info.GID, 10),
		},
	}

	perm := "rwx"
	for _, e := range info.SysACL.Entries {
		if e.Type == acl.TypeLightweight && e.Qualifier == u.Id.OpaqueId {
			perm = e.Permissions
			break
		}
	}

	p := path.Clean(fn)
	for p != "." && p != fs.conf.Namespace {
		key := p + "!" + perm
		if tknIf, err := fs.tokenCache.Get(key); err == nil {
			return eosclient.Authorization{Token: tknIf.(string)}, nil
		}
		p = path.Dir(p)
	}

	if info.IsDir {
		// EOS expects directories to have a trailing slash when generating tokens
		fn = path.Clean(fn) + "/"
	}
	tkn, err := fs.c.GenerateToken(ctx, auth, fn, &acl.Entry{Permissions: perm})
	if err != nil {
		return eosclient.Authorization{}, err
	}

	key := path.Clean(fn) + "!" + perm
	_ = fs.tokenCache.SetWithExpire(key, tkn, time.Second*time.Duration(fs.conf.TokenExpiry))

	return eosclient.Authorization{Token: tkn}, nil
}

func (fs *eosfs) getRootAuth(ctx context.Context) (eosclient.Authorization, error) {
	if fs.conf.ForceSingleUserMode {
		if fs.singleUserAuth.Role.UID != "" && fs.singleUserAuth.Role.GID != "" {
			return fs.singleUserAuth, nil
		}
		var err error
		fs.singleUserAuth, err = fs.getUIDGateway(ctx, &userpb.UserId{OpaqueId: fs.conf.SingleUsername})
		return fs.singleUserAuth, err
	}
	return eosclient.Authorization{Role: eosclient.Role{UID: "0", GID: "0"}}, nil
}

type eosSysMetadata struct {
	TreeSize  uint64 `json:"tree_size"`
	TreeCount uint64 `json:"tree_count"`
	File      string `json:"file"`
	Instance  string `json:"instance"`
}

func (fs *eosfs) getEosMetadata(finfo *eosclient.FileInfo) []byte {
	sys := &eosSysMetadata{
		File:     finfo.File,
		Instance: finfo.Instance,
	}

	if finfo.IsDir {
		sys.TreeCount = finfo.TreeCount
		sys.TreeSize = finfo.TreeSize
	}

	v, _ := json.Marshal(sys)
	return v
}

/*
	Merge shadow on requests for /home ?

	No - GetHome(ctx context.Context) (string, error)
	No -CreateHome(ctx context.Context) error
	No - CreateDir(ctx context.Context, fn string) error
	No -Delete(ctx context.Context, ref *provider.Reference) error
	No -Move(ctx context.Context, oldRef, newRef *provider.Reference) error
	No -GetMD(ctx context.Context, ref *provider.Reference) (*provider.ResourceInfo, error)
	Yes -ListFolder(ctx context.Context, ref *provider.Reference) ([]*provider.ResourceInfo, error)
	No -Upload(ctx context.Context, ref *provider.Reference, r io.ReadCloser) error
	No -Download(ctx context.Context, ref *provider.Reference) (io.ReadCloser, error)
	No -ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error)
	No -DownloadRevision(ctx context.Context, ref *provider.Reference, key string) (io.ReadCloser, error)
	No -RestoreRevision(ctx context.Context, ref *provider.Reference, key string) error
	No ListRecycle(ctx context.Context) ([]*provider.RecycleItem, error)
	No RestoreRecycleItem(ctx context.Context, key string) error
	No PurgeRecycleItem(ctx context.Context, key string) error
	No EmptyRecycle(ctx context.Context) error
	? GetPathByID(ctx context.Context, id *provider.Reference) (string, error)
	No AddGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error
	No RemoveGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error
	No UpdateGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error
	No ListGrants(ctx context.Context, ref *provider.Reference) ([]*provider.Grant, error)
	No GetQuota(ctx context.Context) (int, int, error)
	No CreateReference(ctx context.Context, path string, targetURI *url.URL) error
	No Shutdown(ctx context.Context) error
	No SetArbitraryMetadata(ctx context.Context, ref *provider.Reference, md *provider.ArbitraryMetadata) error
	No UnsetArbitraryMetadata(ctx context.Context, ref *provider.Reference, keys []string) error
*/

/*
	Merge shadow on requests for /home/MyShares ?

	No - GetHome(ctx context.Context) (string, error)
	No -CreateHome(ctx context.Context) error
	No - CreateDir(ctx context.Context, fn string) error
	Maybe -Delete(ctx context.Context, ref *provider.Reference) error
	No -Move(ctx context.Context, oldRef, newRef *provider.Reference) error
	Yes -GetMD(ctx context.Context, ref *provider.Reference) (*provider.ResourceInfo, error)
	Yes -ListFolder(ctx context.Context, ref *provider.Reference) ([]*provider.ResourceInfo, error)
	No -Upload(ctx context.Context, ref *provider.Reference, r io.ReadCloser) error
	No -Download(ctx context.Context, ref *provider.Reference) (io.ReadCloser, error)
	No -ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error)
	No -DownloadRevision(ctx context.Context, ref *provider.Reference, key string) (io.ReadCloser, error)
	No -RestoreRevision(ctx context.Context, ref *provider.Reference, key string) error
	No ListRecycle(ctx context.Context) ([]*provider.RecycleItem, error)
	No RestoreRecycleItem(ctx context.Context, key string) error
	No PurgeRecycleItem(ctx context.Context, key string) error
	No EmptyRecycle(ctx context.Context) error
	?  GetPathByID(ctx context.Context, id *provider.Reference) (string, error)
	No AddGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error
	No RemoveGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error
	No UpdateGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error
	No ListGrants(ctx context.Context, ref *provider.Reference) ([]*provider.Grant, error)
	No GetQuota(ctx context.Context) (int, int, error)
	No CreateReference(ctx context.Context, path string, targetURI *url.URL) error
	No Shutdown(ctx context.Context) error
	No SetArbitraryMetadata(ctx context.Context, ref *provider.Reference, md *provider.ArbitraryMetadata) error
	No UnsetArbitraryMetadata(ctx context.Context, ref *provider.Reference, keys []string) error
*/

/*
	Merge shadow on requests for /home/MyShares/file-reference ?

	No - GetHome(ctx context.Context) (string, error)
	No -CreateHome(ctx context.Context) error
	No - CreateDir(ctx context.Context, fn string) error
	Maybe -Delete(ctx context.Context, ref *provider.Reference) error
	Yes -Move(ctx context.Context, oldRef, newRef *provider.Reference) error
	Yes -GetMD(ctx context.Context, ref *provider.Reference) (*provider.ResourceInfo, error)
	No -ListFolder(ctx context.Context, ref *provider.Reference) ([]*provider.ResourceInfo, error)
	No -Upload(ctx context.Context, ref *provider.Reference, r io.ReadCloser) error
	No -Download(ctx context.Context, ref *provider.Reference) (io.ReadCloser, error)
	No -ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error)
	No -DownloadRevision(ctx context.Context, ref *provider.Reference, key string) (io.ReadCloser, error)
	No -RestoreRevision(ctx context.Context, ref *provider.Reference, key string) error
	No ListRecycle(ctx context.Context) ([]*provider.RecycleItem, error)
	No RestoreRecycleItem(ctx context.Context, key string) error
	No PurgeRecycleItem(ctx context.Context, key string) error
	No EmptyRecycle(ctx context.Context) error
	?  GetPathByID(ctx context.Context, id *provider.Reference) (string, error)
	No AddGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error
	No RemoveGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error
	No UpdateGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error
	No ListGrants(ctx context.Context, ref *provider.Reference) ([]*provider.Grant, error)
	No GetQuota(ctx context.Context) (int, int, error)
	No CreateReference(ctx context.Context, path string, targetURI *url.URL) error
	No Shutdown(ctx context.Context) error
	Maybe SetArbitraryMetadata(ctx context.Context, ref *provider.Reference, md *provider.ArbitraryMetadata) error
	Maybe UnsetArbitraryMetadata(ctx context.Context, ref *provider.Reference, keys []string) error
*/
