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

package eosgrpc

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"

	"github.com/opencloud-eu/reva/v3/pkg/appctx"

	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v3/pkg/eosclient"
	erpc "github.com/opencloud-eu/reva/v3/pkg/eosclient/eosgrpc/eos_grpc"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/logger"
	"github.com/opencloud-eu/reva/v3/pkg/storage/utils/acl"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	versionPrefix = ".sys.v#."
	// lwShareAttrKey = "reva.lwshare"
)

const (
	// SystemAttr is the system extended attribute.
	SystemAttr eosclient.AttrType = iota
	// UserAttr is the user extended attribute.
	UserAttr
)

// Options to configure the Client.
type Options struct {

	// UseKeyTabAuth changes will authenticate requests by using an EOS keytab.
	UseKeytab bool

	// Whether to maintain the same inode across various versions of a file.
	// Requires extra metadata operations if set to true
	VersionInvariant bool

	// Set to true to use the local disk as a buffer for chunk
	// reads from EOS. Default is false, i.e. pure streaming
	ReadUsesLocalTemp bool

	// Set to true to use the local disk as a buffer for chunk
	// writes to EOS. Default is false, i.e. pure streaming
	// Beware: in pure streaming mode the FST must support
	// the HTTP chunked encoding
	WriteUsesLocalTemp bool

	// Location of the xrdcopy binary.
	// Default is /opt/eos/xrootd/bin/xrdcopy.
	XrdcopyBinary string

	// URL of the EOS MGM.
	// Default is root://eos-example.org
	URL string

	// URI of the EOS MGM grpc server
	GrpcURI string

	// Location on the local fs where to store reads.
	// Defaults to os.TempDir()
	CacheDirectory string

	// Keytab is the location of the EOS keytab file.
	Keytab string

	// Authkey is the key that authorizes this client to connect to the GRPC service
	// It's unclear whether this will be the final solution
	Authkey string

	// SecProtocol is the comma separated list of security protocols used by xrootd.
	// For example: "sss, unix"
	SecProtocol string
}

func (opt *Options) init() {

	if opt.XrdcopyBinary == "" {
		opt.XrdcopyBinary = "/opt/eos/xrootd/bin/xrdcopy"
	}

	if opt.URL == "" {
		opt.URL = "root://eos-example.org"
	}

	if opt.CacheDirectory == "" {
		opt.CacheDirectory = os.TempDir()
	}

}

// Client performs actions against a EOS management node (MGM)
// using the EOS GRPC interface.
type Client struct {
	opt    *Options
	httpcl *EOSHTTPClient
	cl     erpc.EosClient
}

// Create and connect a grpc eos Client
func newgrpc(ctx context.Context, opt *Options) (erpc.EosClient, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("Setting up GRPC towards ", "'"+opt.GrpcURI+"'").Msg("")

	conn, err := grpc.NewClient(opt.GrpcURI, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Warn().Str("Error connecting to ", "'"+opt.GrpcURI+"' ").Str("err", err.Error()).Msg("")
	}

	log.Debug().Str("Going to ping ", "'"+opt.GrpcURI+"' ").Msg("")
	ecl := erpc.NewEosClient(conn)
	// If we can't ping... just print warnings. In the case EOS is down, grpc will take care of
	// connecting later
	prq := new(erpc.PingRequest)
	prq.Authkey = opt.Authkey
	prq.Message = []byte("hi this is a ping from reva")
	prep, err := ecl.Ping(ctx, prq)
	if err != nil {
		log.Warn().Str("Could not ping to ", "'"+opt.GrpcURI+"' ").Str("err", err.Error()).Msg("")
	}

	if prep == nil {
		log.Warn().Str("Could not ping to ", "'"+opt.GrpcURI+"' ").Str("nil response", "").Msg("")
	}

	return ecl, nil
}

// New creates a new client with the given options.
func New(opt *Options, httpOpts *HTTPOptions) (*Client, error) {
	tlog := logger.New().With().Int("pid", os.Getpid()).Logger()
	tlog.Debug().Str("Creating new eosgrpc client. opt: ", "'"+fmt.Sprintf("%#v", opt)+"' ").Msg("")

	opt.init()
	httpcl, err := NewEOSHTTPClient(httpOpts)
	if err != nil {
		return nil, err
	}

	tctx := appctx.WithLogger(context.Background(), &tlog)
	cl, err := newgrpc(tctx, opt)
	if err != nil {
		return nil, err
	}

	return &Client{
		opt:    opt,
		httpcl: httpcl,
		cl:     cl,
	}, nil
}

// If the error is not nil, take that
// If there is an error coming from EOS, erturn a descriptive error
func (c *Client) getRespError(rsp *erpc.NSResponse, err error) error {
	if err != nil {
		return err
	}
	if rsp == nil || rsp.Error == nil || rsp.Error.Code == 0 {
		return nil
	}

	return errtypes.InternalError("Err from EOS: " + fmt.Sprintf("%#v", rsp.Error))
}

// Common code to create and initialize a NSRequest
func (c *Client) initNSRequest(ctx context.Context, auth eosclient.Authorization) (*erpc.NSRequest, error) {
	// Stuff filename, uid, gid into the MDRequest type

	log := appctx.GetLogger(ctx)
	log.Debug().Str("(uid,gid)", "("+auth.Role.UID+","+auth.Role.GID+")").Msg("New grpcNS req")

	rq := new(erpc.NSRequest)
	rq.Role = new(erpc.RoleId)

	uidInt, err := strconv.ParseUint(auth.Role.UID, 10, 64)
	if err != nil {
		return nil, err
	}
	gidInt, err := strconv.ParseUint(auth.Role.GID, 10, 64)
	if err != nil {
		return nil, err
	}
	rq.Role.Uid = uidInt
	rq.Role.Gid = gidInt
	rq.Authkey = c.opt.Authkey

	return rq, nil
}

// Common code to create and initialize a NSRequest
func (c *Client) initMDRequest(ctx context.Context, auth eosclient.Authorization) (*erpc.MDRequest, error) {
	// Stuff filename, uid, gid into the MDRequest type

	log := appctx.GetLogger(ctx)
	log.Debug().Str("(uid,gid)", "("+auth.Role.UID+","+auth.Role.GID+")").Msg("New grpcMD req")

	mdrq := new(erpc.MDRequest)
	mdrq.Role = new(erpc.RoleId)

	uidInt, err := strconv.ParseUint(auth.Role.UID, 10, 64)
	if err != nil {
		return nil, err
	}
	gidInt, err := strconv.ParseUint(auth.Role.GID, 10, 64)
	if err != nil {
		return nil, err
	}
	mdrq.Role.Uid = uidInt
	mdrq.Role.Gid = gidInt

	mdrq.Authkey = c.opt.Authkey

	return mdrq, nil
}

// AddACL adds an new acl to EOS with the given aclType.
func (c *Client) AddACL(ctx context.Context, auth, rootAuth eosclient.Authorization, path string, pos uint, a *acl.Entry) error {

	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "AddACL").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	// Init a new NSRequest
	rq, err := c.initNSRequest(ctx, rootAuth)
	if err != nil {
		return err
	}

	// workaround to be root
	// TODO: removed once fixed in eos grpc
	rq.Role.Gid = 1

	msg := new(erpc.NSRequest_AclRequest)
	msg.Cmd = erpc.NSRequest_AclRequest_ACL_COMMAND(erpc.NSRequest_AclRequest_ACL_COMMAND_value["MODIFY"])
	msg.Type = erpc.NSRequest_AclRequest_ACL_TYPE(erpc.NSRequest_AclRequest_ACL_TYPE_value["SYS_ACL"])
	msg.Recursive = true
	msg.Rule = a.CitrineSerialize()

	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(path)

	rq.Command = &erpc.NSRequest_Acl{Acl: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(context.Background(), rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "AddACL").Str("path", path).Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.NotFound(fmt.Sprintf("Path: %s", path))
	}

	log.Debug().Str("func", "AddACL").Str("path", path).Str("resp:", fmt.Sprintf("%#v", resp)).Msg("grpc response")

	return err

}

// RemoveACL removes the acl from EOS.
func (c *Client) RemoveACL(ctx context.Context, auth, rootAuth eosclient.Authorization, path string, a *acl.Entry) error {

	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "RemoveACL").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	acls, err := c.getACLForPath(ctx, auth, path)
	if err != nil {
		return err
	}

	acls.DeleteEntry(a.Type, a.Qualifier)
	sysACL := acls.Serialize()

	// Init a new NSRequest
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_AclRequest)
	msg.Cmd = erpc.NSRequest_AclRequest_ACL_COMMAND(erpc.NSRequest_AclRequest_ACL_COMMAND_value["MODIFY"])
	msg.Type = erpc.NSRequest_AclRequest_ACL_TYPE(erpc.NSRequest_AclRequest_ACL_TYPE_value["SYS_ACL"])
	msg.Recursive = true
	msg.Rule = sysACL

	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(path)

	rq.Command = &erpc.NSRequest_Acl{Acl: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(context.Background(), rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "RemoveACL").Str("path", path).Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.NotFound(fmt.Sprintf("Path: %s", path))
	}

	log.Debug().Str("func", "RemoveACL").Str("path", path).Str("resp:", fmt.Sprintf("%#v", resp)).Msg("grpc response")

	return err

}

// UpdateACL updates the EOS acl.
func (c *Client) UpdateACL(ctx context.Context, auth, rootAuth eosclient.Authorization, path string, position uint, a *acl.Entry) error {
	return c.AddACL(ctx, auth, rootAuth, path, position, a)
}

// GetACL for a file
func (c *Client) GetACL(ctx context.Context, auth eosclient.Authorization, path, aclType, target string) (*acl.Entry, error) {

	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "GetACL").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	acls, err := c.ListACLs(ctx, auth, path)
	if err != nil {
		return nil, err
	}
	for _, a := range acls {
		if a.Type == aclType && a.Qualifier == target {
			return a, nil
		}
	}
	return nil, errtypes.NotFound(fmt.Sprintf("%s:%s", aclType, target))

}

// ListACLs returns the list of ACLs present under the given path.
// EOS returns uids/gid for Citrine version and usernames for older versions.
// For Citire we need to convert back the uid back to username.
func (c *Client) ListACLs(ctx context.Context, auth eosclient.Authorization, path string) ([]*acl.Entry, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "ListACLs").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	parsedACLs, err := c.getACLForPath(ctx, auth, path)
	if err != nil {
		return nil, err
	}

	// EOS Citrine ACLs are stored with uid. The UID will be resolved to the
	// user opaque ID at the eosfs level.
	return parsedACLs.Entries, nil
}

func (c *Client) getACLForPath(ctx context.Context, auth eosclient.Authorization, path string) (*acl.ACLs, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "GetACLForPath").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return nil, err
	}

	msg := new(erpc.NSRequest_AclRequest)
	msg.Cmd = erpc.NSRequest_AclRequest_ACL_COMMAND(erpc.NSRequest_AclRequest_ACL_COMMAND_value["LIST"])
	msg.Type = erpc.NSRequest_AclRequest_ACL_TYPE(erpc.NSRequest_AclRequest_ACL_TYPE_value["SYS_ACL"])
	msg.Recursive = true

	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(path)

	rq.Command = &erpc.NSRequest_Acl{Acl: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(context.Background(), rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "GetACLForPath").Str("path", path).Str("err", e.Error()).Msg("")
		return nil, e
	}

	if resp == nil {
		return nil, errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' path: '%s'", auth.Role.UID, path))
	}

	log.Debug().Str("func", "GetACLForPath").Str("path", path).Str("resp:", fmt.Sprintf("%#v", resp)).Msg("grpc response")

	if resp.Acl == nil {
		return nil, errtypes.InternalError(fmt.Sprintf("nil acl for uid: '%s' path: '%s'", auth.Role.UID, path))
	}

	if resp.GetError() != nil {
		log.Error().Str("func", "GetACLForPath").Str("uid", auth.Role.UID).Str("path", path).Int64("errcode", resp.GetError().Code).Str("errmsg", resp.GetError().Msg).Msg("EOS negative resp")
	}

	aclret, err := acl.Parse(resp.Acl.Rule, acl.ShortTextForm)

	// Now loop and build the correct return value

	return aclret, err
}

// GetFileInfoByInode returns the FileInfo by the given inode
func (c *Client) GetFileInfoByInode(ctx context.Context, auth eosclient.Authorization, inode uint64) (*eosclient.FileInfo, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "GetFileInfoByInode").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Uint64("inode", inode).Msg("")

	// Initialize the common fields of the MDReq
	mdrq, err := c.initMDRequest(ctx, auth)
	if err != nil {
		return nil, err
	}

	// Stuff filename, uid, gid into the MDRequest type
	mdrq.Type = erpc.TYPE_STAT
	mdrq.Id = new(erpc.MDId)
	mdrq.Id.Ino = inode

	// Now send the req and see what happens
	resp, err := c.cl.MD(context.Background(), mdrq)
	if err != nil {
		log.Error().Err(err).Uint64("inode", inode).Str("err", err.Error())

		return nil, err
	}
	rsp, err := resp.Recv()
	if err != nil {
		log.Error().Err(err).Uint64("inode", inode).Str("err", err.Error())
		return nil, err
	}

	if rsp == nil {
		return nil, errtypes.InternalError(fmt.Sprintf("nil response for inode: '%d'", inode))
	}

	log.Debug().Uint64("inode", inode).Str("rsp:", fmt.Sprintf("%#v", rsp)).Msg("grpc response")

	info, err := c.grpcMDResponseToFileInfo(rsp)
	if err != nil {
		return nil, err
	}

	if c.opt.VersionInvariant && isVersionFolder(info.File) {
		info, err = c.getFileInfoFromVersion(ctx, auth, info.File)
		if err != nil {
			return nil, err
		}
		info.Inode = inode
	}

	log.Debug().Str("func", "GetFileInfoByInode").Uint64("inode", inode).Msg("")
	return c.fixupACLs(ctx, auth, info), nil
}

func (c *Client) fixupACLs(ctx context.Context, auth eosclient.Authorization, info *eosclient.FileInfo) *eosclient.FileInfo {

	// Append the ACLs that are described by the xattr sys.acl entry
	a, err := acl.Parse(info.Attrs["sys.acl"], acl.ShortTextForm)
	if err == nil {
		if info.SysACL != nil {
			info.SysACL.Entries = append(info.SysACL.Entries, a.Entries...)
		} else {
			info.SysACL = a
		}
	}

	// We need to inherit the ACLs for the parent directory as these are not available for files
	if !info.IsDir {
		parentInfo, err := c.GetFileInfoByPath(ctx, auth, path.Dir(info.File))
		// Even if this call fails, at least return the current file object
		if err == nil {
			info.SysACL.Entries = append(info.SysACL.Entries, parentInfo.SysACL.Entries...)
		}
	}
	return info
}

// SetAttr sets an extended attributes on a path.
func (c *Client) SetAttr(ctx context.Context, auth eosclient.Authorization, attr *eosclient.Attribute, errorIfExists, recursive bool, path string) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "SetAttr").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_SetXAttrRequest)

	var m = map[string][]byte{attr.GetKey(): []byte(attr.Val)}
	msg.Xattrs = m
	msg.Recursive = recursive

	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(path)

	if errorIfExists {
		msg.Create = true
	}

	rq.Command = &erpc.NSRequest_Xattr{Xattr: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(ctx, rq)
	e := c.getRespError(resp, err)

	if resp != nil && resp.Error != nil && resp.Error.Code == 17 {
		return eosclient.AttrAlreadyExistsError
	}

	if e != nil {
		log.Error().Str("func", "SetAttr").Str("path", path).Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' gid: '%s' path: '%s'", auth.Role.UID, auth.Role.GID, path))
	}

	if resp.GetError() != nil {
		log.Error().Str("func", "setAttr").Str("path", path).Int64("errcode", resp.GetError().Code).Str("errmsg", resp.GetError().Msg).Msg("EOS negative result")
	}

	return err

}

// UnsetAttr unsets an extended attribute on a path.
func (c *Client) UnsetAttr(ctx context.Context, auth eosclient.Authorization, attr *eosclient.Attribute, recursive bool, path string) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "UnsetAttr").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_SetXAttrRequest)

	var ktd = []string{attr.GetKey()}
	msg.Keystodelete = ktd
	msg.Recursive = recursive

	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(path)

	rq.Command = &erpc.NSRequest_Xattr{Xattr: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(ctx, rq)

	if resp != nil && resp.Error != nil && resp.Error.Code == 61 {
		return eosclient.AttrNotExistsError
	}

	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "UnsetAttr").Str("path", path).Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' gid: '%s' path: '%s'", auth.Role.UID, auth.Role.GID, path))
	}

	if resp.GetError() != nil {
		log.Error().Str("func", "UnsetAttr").Str("path", path).Int64("errcode", resp.GetError().Code).Str("errmsg", resp.GetError().Msg).Msg("EOS negative resp")
	}
	return err

}

// GetAttr returns the attribute specified by key
func (c *Client) GetAttr(ctx context.Context, auth eosclient.Authorization, key, path string) (*eosclient.Attribute, error) {
	info, err := c.GetFileInfoByPath(ctx, auth, path)
	if err != nil {
		return nil, err
	}

	for k, v := range info.Attrs {
		if k == key {
			attr, err := getAttribute(k, v)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("eosgrpc: cannot parse attribute key=%s value=%s", k, v))
			}
			return attr, nil
		}
	}
	return nil, errtypes.NotFound(fmt.Sprintf("key %s not found", key))
}

func getAttribute(key, val string) (*eosclient.Attribute, error) {
	// key is in the form sys.forced.checksum
	type2key := strings.SplitN(key, ".", 2) // type2key = ["sys", "forced.checksum"]
	if len(type2key) != 2 {
		return nil, errtypes.InternalError("wrong attr format to deserialize")
	}
	t, err := eosclient.AttrStringToType(type2key[0])
	if err != nil {
		return nil, err
	}
	attr := &eosclient.Attribute{
		Type: t,
		Key:  type2key[1],
		Val:  val,
	}
	return attr, nil
}

// GetFileInfoByPath returns the FilInfo at the given path
func (c *Client) GetFileInfoByPath(ctx context.Context, auth eosclient.Authorization, path string) (*eosclient.FileInfo, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "GetFileInfoByPath").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	// Initialize the common fields of the MDReq
	mdrq, err := c.initMDRequest(ctx, auth)
	if err != nil {
		return nil, err
	}

	mdrq.Type = erpc.TYPE_STAT
	mdrq.Id = new(erpc.MDId)
	mdrq.Id.Path = []byte(path)

	// Now send the req and see what happens
	resp, err := c.cl.MD(ctx, mdrq)
	if err != nil {
		log.Error().Str("func", "GetFileInfoByPath").Err(err).Str("path", path).Str("err", err.Error()).Msg("")

		return nil, err
	}
	rsp, err := resp.Recv()
	if err != nil {
		log.Error().Str("func", "GetFileInfoByPath").Err(err).Str("path", path).Str("err", err.Error()).Msg("")

		// FIXME: this is very very bad and poisonous for the project!!!!!!!
		// Apparently here we have to assume that an error in Recv() means "file not found"
		// - "File not found is not an error", it's a legitimate result of a legitimate check
		// - Assuming that any error means file not found is doubly poisonous
		return nil, errtypes.NotFound(err.Error())
		// return nil, nil
	}

	if rsp == nil {
		return nil, errtypes.NotFound(fmt.Sprintf("%s:%s", "acltype", path))
	}

	log.Debug().Str("func", "GetFileInfoByPath").Str("path", path).Str("rsp:", fmt.Sprintf("%#v", rsp)).Msg("grpc response")

	info, err := c.grpcMDResponseToFileInfo(rsp)
	if err != nil {
		return nil, err
	}

	if c.opt.VersionInvariant && !isVersionFolder(path) && !info.IsDir {
		inode, err := c.getVersionFolderInode(ctx, auth, path)
		if err != nil {
			return nil, err
		}
		info.Inode = inode
	}

	return c.fixupACLs(ctx, auth, info), nil
}

// GetFileInfoByFXID returns the FileInfo by the given file id in hexadecimal
func (c *Client) GetFileInfoByFXID(ctx context.Context, auth eosclient.Authorization, fxid string) (*eosclient.FileInfo, error) {
	return nil, errtypes.NotSupported("eosgrpc: GetFileInfoByFXID not implemented")
}

// GetQuota gets the quota of a user on the quota node defined by path
func (c *Client) GetQuota(ctx context.Context, username string, rootAuth eosclient.Authorization, path string) (*eosclient.QuotaInfo, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "GetQuota").Str("rootuid,rootgid", rootAuth.Role.UID+","+rootAuth.Role.GID).Str("username", username).Str("path", path).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, rootAuth)
	if err != nil {
		return nil, err
	}

	msg := new(erpc.NSRequest_QuotaRequest)
	msg.Path = []byte(path)
	msg.Id = new(erpc.RoleId)
	msg.Op = erpc.QUOTAOP_GET
	// Eos filters the returned quotas by username. This means that EOS must know it, someone
	// must have created an user with that name
	msg.Id.Username = username
	rq.Command = &erpc.NSRequest_Quota{Quota: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(ctx, rq)
	e := c.getRespError(resp, err)
	if e != nil {
		return nil, e
	}

	if resp == nil {
		return nil, errtypes.InternalError(fmt.Sprintf("nil response for username: '%s' path: '%s'", username, path))
	}

	if resp.GetError() != nil {
		log.Error().Str("func", "GetQuota").Str("username", username).Str("info:", fmt.Sprintf("%#v", resp)).Int64("eoserrcode", resp.GetError().Code).Str("errmsg", resp.GetError().Msg).Msg("EOS negative resp")
	} else {
		log.Debug().Str("func", "GetQuota").Str("username", username).Str("info:", fmt.Sprintf("%#v", resp)).Msg("grpc response")
	}

	if resp.Quota == nil {
		return nil, errtypes.InternalError(fmt.Sprintf("nil quota response? path: '%s'", path))
	}

	if resp.Quota.Code != 0 {
		return nil, errtypes.InternalError(fmt.Sprintf("Quota error from eos. info: '%#v'", resp.Quota))
	}

	qi := new(eosclient.QuotaInfo)
	if resp == nil {
		return nil, errtypes.InternalError("Out of memory")
	}

	// Let's loop on all the quotas that match this uid (apparently there can be many)
	// If there are many for this node, we sum them up
	for i := 0; i < len(resp.Quota.Quotanode); i++ {
		log.Debug().Str("func", "GetQuota").Str("quotanode:", fmt.Sprintf("%d: %#v", i, resp.Quota.Quotanode[i])).Msg("")

		mx := int64(resp.Quota.Quotanode[i].Maxlogicalbytes) - int64(resp.Quota.Quotanode[i].Usedbytes)
		if mx < 0 {
			mx = 0
		}
		qi.AvailableBytes += uint64(mx)
		qi.UsedBytes += resp.Quota.Quotanode[i].Usedbytes

		mx = int64(resp.Quota.Quotanode[i].Maxfiles) - int64(resp.Quota.Quotanode[i].Usedfiles)
		if mx < 0 {
			mx = 0
		}
		qi.AvailableInodes += uint64(mx)
		qi.UsedInodes += resp.Quota.Quotanode[i].Usedfiles
	}

	return qi, err

}

// SetQuota sets the quota of a user on the quota node defined by path
func (c *Client) SetQuota(ctx context.Context, rootAuth eosclient.Authorization, info *eosclient.SetQuotaInfo) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "SetQuota").Str("info:", fmt.Sprintf("%#v", info)).Msg("")

	// EOS does not have yet this command... work in progress, this is a draft piece of code
	// return errtypes.NotSupported("eosgrpc: SetQuota not implemented")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, rootAuth)
	if err != nil {
		return err

	}

	msg := new(erpc.NSRequest_QuotaRequest)
	msg.Path = []byte(info.QuotaNode)
	msg.Id = new(erpc.RoleId)
	uidInt, err := strconv.ParseUint(info.UID, 10, 64)
	if err != nil {
		return err
	}

	// We set a quota for an user, not a group!
	msg.Id.Uid = uidInt
	msg.Id.Gid = 0
	msg.Id.Username = info.Username
	msg.Op = erpc.QUOTAOP_SET
	msg.Maxbytes = info.MaxBytes
	msg.Maxfiles = info.MaxFiles
	rq.Command = &erpc.NSRequest_Quota{Quota: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(ctx, rq)
	e := c.getRespError(resp, err)
	if e != nil {
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for info: '%#v'", info))
	}

	if resp.GetError() != nil {
		log.Error().Str("func", "SetQuota").Str("info:", fmt.Sprintf("%#v", resp)).Int64("errcode", resp.GetError().Code).Str("errmsg", resp.GetError().Msg).Msg("EOS negative resp")
	} else {
		log.Debug().Str("func", "SetQuota").Str("info:", fmt.Sprintf("%#v", resp)).Msg("grpc response")
	}

	if resp.Quota == nil {
		return errtypes.InternalError(fmt.Sprintf("nil quota response? info: '%#v'", info))
	}

	if resp.Quota.Code != 0 {
		return errtypes.InternalError(fmt.Sprintf("Quota error from eos. quota: '%#v'", resp.Quota))
	}

	log.Debug().Str("func", "GetQuota").Str("quotanodes", fmt.Sprintf("%d", len(resp.Quota.Quotanode))).Msg("grpc response")

	return err
}

// Touch creates a 0-size,0-replica file in the EOS namespace.
func (c *Client) Touch(ctx context.Context, auth eosclient.Authorization, path string) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "Touch").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_TouchRequest)

	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(path)

	rq.Command = &erpc.NSRequest_Touch{Touch: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(ctx, rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "Touch").Str("path", path).Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' path: '%s'", auth.Role.UID, path))
	}

	log.Debug().Str("func", "Touch").Str("path", path).Str("resp:", fmt.Sprintf("%#v", resp)).Msg("grpc response")

	return err

}

// Chown given path
func (c *Client) Chown(ctx context.Context, auth, chownAuth eosclient.Authorization, path string) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "Chown").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("chownuid,chowngid", chownAuth.Role.UID+","+chownAuth.Role.GID).Str("path", path).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_ChownRequest)
	msg.Owner = new(erpc.RoleId)
	msg.Owner.Uid, err = strconv.ParseUint(chownAuth.Role.UID, 10, 64)
	if err != nil {
		return err
	}
	msg.Owner.Gid, err = strconv.ParseUint(chownAuth.Role.GID, 10, 64)
	if err != nil {
		return err
	}

	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(path)

	rq.Command = &erpc.NSRequest_Chown{Chown: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(ctx, rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "Chown").Str("path", path).Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' chownuid: '%s' path: '%s'", auth.Role.UID, chownAuth.Role.UID, path))
	}

	log.Debug().Str("func", "Chown").Str("path", path).Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("chownuid,chowngid", chownAuth.Role.UID+","+chownAuth.Role.GID).Str("resp:", fmt.Sprintf("%#v", resp)).Msg("grpc response")

	return err

}

// Chmod given path
func (c *Client) Chmod(ctx context.Context, auth eosclient.Authorization, mode, path string) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "Chmod").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("mode", mode).Str("path", path).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_ChmodRequest)

	md, err := strconv.ParseUint(mode, 8, 64)
	if err != nil {
		return err
	}
	msg.Mode = int64(md)

	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(path)

	rq.Command = &erpc.NSRequest_Chmod{Chmod: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(ctx, rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "Chmod").Str("path ", path).Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' mode: '%s' path: '%s'", auth.Role.UID, mode, path))
	}

	log.Debug().Str("func", "Chmod").Str("path", path).Str("resp:", fmt.Sprintf("%#v", resp)).Msg("grpc response")

	return err

}

// CreateDir creates a directory at the given path
func (c *Client) CreateDir(ctx context.Context, auth eosclient.Authorization, path string) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "Createdir").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_MkdirRequest)

	// Let's put 750 as permissions, assuming that EOS will apply some mask
	md, err := strconv.ParseUint("750", 8, 64)
	if err != nil {
		return err
	}
	msg.Mode = int64(md)
	msg.Recursive = true
	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(path)

	rq.Command = &erpc.NSRequest_Mkdir{Mkdir: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(ctx, rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "Createdir").Str("path", path).Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' path: '%s'", auth.Role.UID, path))
	}

	log.Debug().Str("func", "Createdir").Str("path", path).Str("resp:", fmt.Sprintf("%#v", resp)).Msg("grpc response")

	return err

}

func (c *Client) rm(ctx context.Context, auth eosclient.Authorization, path string, noRecycle bool) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "rm").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_UnlinkRequest)

	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(path)
	msg.Norecycle = noRecycle

	rq.Command = &erpc.NSRequest_Unlink{Unlink: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(ctx, rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "rm").Str("path", path).Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' path: '%s'", auth.Role.UID, path))
	}

	log.Debug().Str("func", "rm").Str("path", path).Str("resp:", fmt.Sprintf("%#v", resp)).Msg("grpc response")

	return err

}

func (c *Client) rmdir(ctx context.Context, auth eosclient.Authorization, path string, noRecycle bool) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "rmdir").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_RmRequest)

	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(path)
	msg.Recursive = true
	msg.Norecycle = noRecycle

	rq.Command = &erpc.NSRequest_Rm{Rm: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(ctx, rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "rmdir").Str("path", path).Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' path: '%s'", auth.Role.UID, path))
	}

	log.Debug().Str("func", "rmdir").Str("path", path).Str("resp:", fmt.Sprintf("%#v", resp)).Msg("grpc response")

	return err
}

// Remove removes the resource at the given path
func (c *Client) Remove(ctx context.Context, auth eosclient.Authorization, path string, noRecycle bool) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "Remove").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	nfo, err := c.GetFileInfoByPath(ctx, auth, path)
	if err != nil {
		log.Warn().Err(err).Str("func", "Remove").Str("path", path).Str("err", err.Error())
		return err
	}

	if nfo.IsDir {
		return c.rmdir(ctx, auth, path, noRecycle)
	}

	return c.rm(ctx, auth, path, noRecycle)
}

// Rename renames the resource referenced by oldPath to newPath
func (c *Client) Rename(ctx context.Context, auth eosclient.Authorization, oldPath, newPath string) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "Rename").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("oldPath", oldPath).Str("newPath", newPath).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_RenameRequest)

	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(oldPath)
	msg.Target = []byte(newPath)
	rq.Command = &erpc.NSRequest_Rename{Rename: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(ctx, rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "Rename").Str("oldPath", oldPath).Str("newPath", newPath).Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' oldpath: '%s' newpath: '%s'", auth.Role.UID, oldPath, newPath))
	}

	log.Debug().Str("func", "Rename").Str("oldPath", oldPath).Str("newPath", newPath).Str("resp:", fmt.Sprintf("%#v", resp)).Msg("grpc response")

	return err

}

// List the contents of the directory given by path
func (c *Client) List(ctx context.Context, auth eosclient.Authorization, dpath string) ([]*eosclient.FileInfo, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "List").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("dpath", dpath).Msg("")

	// Stuff filename, uid, gid into the FindRequest type
	fdrq := new(erpc.FindRequest)
	fdrq.Maxdepth = 1
	fdrq.Type = erpc.TYPE_LISTING
	fdrq.Id = new(erpc.MDId)
	fdrq.Id.Path = []byte(dpath)

	fdrq.Role = new(erpc.RoleId)

	uidInt, err := strconv.ParseUint(auth.Role.UID, 10, 64)
	if err != nil {
		return nil, err
	}
	gidInt, err := strconv.ParseUint(auth.Role.GID, 10, 64)
	if err != nil {
		return nil, err
	}
	fdrq.Role.Uid = uidInt
	fdrq.Role.Gid = gidInt

	fdrq.Authkey = c.opt.Authkey

	// Now send the req and see what happens
	resp, err := c.cl.Find(context.Background(), fdrq)
	if err != nil {
		log.Error().Err(err).Str("func", "List").Str("path", dpath).Str("err", err.Error()).Msg("grpc response")

		return nil, err
	}

	var mylst []*eosclient.FileInfo
	var parent *eosclient.FileInfo
	i := 0
	for {
		rsp, err := resp.Recv()
		if err != nil {
			if err == io.EOF {
				log.Debug().Str("path", dpath).Int("nitems", i).Msg("OK, no more items, clean exit")
				break
			}

			// We got an error while reading items. We log this as an error and we return
			// the items we have
			log.Error().Err(err).Str("func", "List").Int("nitems", i).Str("path", dpath).Str("got err from EOS", err.Error()).Msg("")
			if i > 0 {
				log.Error().Str("path", dpath).Int("nitems", i).Msg("No more items, dirty exit")
				return mylst, nil
			}
		}

		if rsp == nil {
			log.Error().Int("nitems", i).Err(err).Str("func", "List").Str("path", dpath).Str("err", "rsp is nil").Msg("grpc response")
			return nil, errtypes.NotFound(dpath)
		}

		log.Debug().Str("func", "List").Str("path", dpath).Str("item resp:", fmt.Sprintf("%#v", rsp)).Msg("grpc response")

		myitem, err := c.grpcMDResponseToFileInfo(rsp)
		if err != nil {
			log.Error().Err(err).Str("func", "List").Str("path", dpath).Str("could not convert item:", fmt.Sprintf("%#v", rsp)).Str("err", err.Error()).Msg("")

			return nil, err
		}

		i++
		// The first item is the directory itself... skip
		if i == 1 {
			parent = myitem
			log.Debug().Str("func", "List").Str("path", dpath).Str("skipping first item resp:", fmt.Sprintf("%#v", rsp)).Msg("grpc response")
			continue
		}

		mylst = append(mylst, myitem)
	}

	if parent.SysACL != nil {

		for _, info := range mylst {
			if !info.IsDir && parent != nil {
				if info.SysACL == nil {
					log.Warn().Str("func", "List").Str("path", dpath).Str("SysACL is nil, taking parent", "").Msg("grpc response")
					info.SysACL.Entries = parent.SysACL.Entries
				} else {
					info.SysACL.Entries = append(info.SysACL.Entries, parent.SysACL.Entries...)
				}
			}
		}
	}

	return mylst, nil

}

// Read reads a file from the mgm and returns a handle to read it
// This handle could be directly the body of the response or a local tmp file
// returning a handle to the body is nice, yet it gives less control on the transaction
// itself, e.g. strange timeouts or TCP issues may be more difficult to trace
// Let's consider this experimental for the moment, maybe I'll like to add a config
// parameter to choose between the two behaviours
func (c *Client) Read(ctx context.Context, auth eosclient.Authorization, path string) (io.ReadCloser, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "Read").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")

	var localTarget string
	var err error
	var localfile io.WriteCloser
	localfile = nil

	if c.opt.ReadUsesLocalTemp {
		rand := "eosread-" + uuid.New().String()
		localTarget := fmt.Sprintf("%s/%s", c.opt.CacheDirectory, rand)
		defer os.RemoveAll(localTarget)

		log.Info().Str("func", "Read").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Str("tempfile", localTarget).Msg("")
		localfile, err = os.Create(localTarget)
		if err != nil {
			log.Error().Str("func", "Read").Str("path", path).Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("err", err.Error()).Msg("")
			return nil, errtypes.InternalError(fmt.Sprintf("can't open local temp file '%s'", localTarget))
		}
	}

	bodystream, err := c.httpcl.GETFile(ctx, "", auth, path, localfile)
	if err != nil {
		log.Error().Str("func", "Read").Str("path", path).Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("err", err.Error()).Msg("")
		return nil, errtypes.InternalError(fmt.Sprintf("can't GET local cache file '%s'", localTarget))
	}

	return bodystream, nil
	// return os.Open(localTarget)
}

// Write writes a file to the mgm
// Somehow the same considerations as Read apply
func (c *Client) Write(ctx context.Context, auth eosclient.Authorization, path string, stream io.ReadCloser) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "Write").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Msg("")
	var length int64
	length = -1

	if c.opt.WriteUsesLocalTemp {
		fd, err := os.CreateTemp(c.opt.CacheDirectory, "eoswrite-")
		if err != nil {
			return err
		}
		defer fd.Close()
		defer os.RemoveAll(fd.Name())

		log.Info().Str("func", "Write").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Str("tempfile", fd.Name()).Msg("")
		// copy stream to local temp file
		length, err = io.Copy(fd, stream)
		if err != nil {
			return err
		}

		wfd, err := os.Open(fd.Name())
		if err != nil {
			return err
		}
		defer wfd.Close()
		defer os.RemoveAll(fd.Name())

		return c.httpcl.PUTFile(ctx, "", auth, path, wfd, length)
	}

	return c.httpcl.PUTFile(ctx, "", auth, path, stream, length)

	// return c.httpcl.PUTFile(ctx, remoteuser, auth, urlpathng, stream)
	// return c.WriteFile(ctx, uid, gid, path, fd.Name())
}

// WriteFile writes an existing file to the mgm. Old xrdcp utility
func (c *Client) WriteFile(ctx context.Context, auth eosclient.Authorization, path, source string) error {

	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "WriteFile").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Str("source", source).Msg("")

	xrdPath := fmt.Sprintf("%s//%s", c.opt.URL, path)
	cmd := exec.CommandContext(ctx, c.opt.XrdcopyBinary, "--nopbar", "--silent", "-f", source, xrdPath, fmt.Sprintf("-ODeos.ruid=%s&eos.rgid=%s", auth.Role.UID, auth.Role.GID))
	_, _, err := c.execute(ctx, cmd)
	return err

}

// ListDeletedEntries returns a list of the deleted entries.
func (c *Client) ListDeletedEntries(ctx context.Context, auth eosclient.Authorization) ([]*eosclient.DeletedEntry, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "ListDeletedEntries").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return nil, err
	}

	msg := new(erpc.NSRequest_RecycleRequest)
	msg.Cmd = erpc.NSRequest_RecycleRequest_RECYCLE_CMD(erpc.NSRequest_RecycleRequest_RECYCLE_CMD_value["LIST"])

	rq.Command = &erpc.NSRequest_Recycle{Recycle: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(context.Background(), rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("err", e.Error()).Msg("")
		return nil, e
	}

	if resp == nil {
		return nil, errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s'", auth.Role.UID))
	}

	if resp.GetError() != nil {
		log.Error().Str("func", "ListDeletedEntries").Int64("errcode", resp.GetError().Code).Str("errmsg", resp.GetError().Msg).Msg("EOS negative resp")
	} else {
		log.Debug().Str("func", "ListDeletedEntries").Str("info:", fmt.Sprintf("%#v", resp)).Msg("grpc response")
	}
	// TODO(labkode): add protection if slave is configured and alive to count how many files are in the trashbin before
	// triggering the recycle ls call that could break the instance because of unavailable memory.
	// FF: I agree with labkode, if we think we may have memory problems then the semantics of the grpc call`and
	// the semantics if this func will have to change. For now this is not foreseen

	ret := make([]*eosclient.DeletedEntry, 0)
	for _, f := range resp.Recycle.Recycles {
		if f == nil {
			log.Info().Msg("nil item in response")
			continue
		}

		entry := &eosclient.DeletedEntry{
			RestorePath:   string(f.Id.Path),
			RestoreKey:    f.Key,
			Size:          f.Size,
			DeletionMTime: f.Dtime.Sec,
			IsDir:         (f.Type == erpc.NSResponse_RecycleResponse_RecycleInfo_TREE),
		}

		ret = append(ret, entry)
	}

	return ret, nil
}

// RestoreDeletedEntry restores a deleted entry.
func (c *Client) RestoreDeletedEntry(ctx context.Context, auth eosclient.Authorization, key string) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "RestoreDeletedEntries").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("key", key).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_RecycleRequest)
	msg.Cmd = erpc.NSRequest_RecycleRequest_RECYCLE_CMD(erpc.NSRequest_RecycleRequest_RECYCLE_CMD_value["RESTORE"])

	msg.Key = key

	rq.Command = &erpc.NSRequest_Recycle{Recycle: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(context.Background(), rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "RestoreDeletedEntries").Str("key", key).Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' key: '%s'", auth.Role.UID, key))
	}

	if resp.GetError() != nil {
		log.Error().Str("func", "RestoreDeletedEntries").Str("key", key).Int64("errcode", resp.GetError().Code).Str("errmsg", resp.GetError().Msg).Msg("EOS negative resp")
	} else {
		log.Info().Str("func", "RestoreDeletedEntries").Str("key", key).Str("resp:", fmt.Sprintf("%#v", resp)).Msg("grpc response")
	}
	return err
}

// PurgeDeletedEntries purges all entries from the recycle bin.
func (c *Client) PurgeDeletedEntries(ctx context.Context, auth eosclient.Authorization) error {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "PurgeDeletedEntries").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_RecycleRequest)
	msg.Cmd = erpc.NSRequest_RecycleRequest_RECYCLE_CMD(erpc.NSRequest_RecycleRequest_RECYCLE_CMD_value["PURGE"])

	rq.Command = &erpc.NSRequest_Recycle{Recycle: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(context.Background(), rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "PurgeDeletedEntries").Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' ", auth.Role.UID))
	}

	log.Info().Str("func", "PurgeDeletedEntries").Int64("errcode", resp.GetError().Code).Str("errmsg", resp.GetError().Msg).Msg("grpc response")

	return err
}

// ListVersions list all the versions for a given file.
func (c *Client) ListVersions(ctx context.Context, auth eosclient.Authorization, p string) ([]*eosclient.FileInfo, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "ListVersions").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("p", p).Msg("")

	versionFolder := getVersionFolder(p)
	finfos, err := c.List(ctx, auth, versionFolder)
	if err != nil {
		// we send back an empty list
		return []*eosclient.FileInfo{}, nil
	}
	return finfos, nil
}

// RollbackToVersion rollbacks a file to a previous version.
func (c *Client) RollbackToVersion(ctx context.Context, auth eosclient.Authorization, path, version string) error {

	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "RollbackToVersion").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("path", path).Str("version", version).Msg("")

	// Initialize the common fields of the NSReq
	rq, err := c.initNSRequest(ctx, auth)
	if err != nil {
		return err
	}

	msg := new(erpc.NSRequest_VersionRequest)
	msg.Cmd = erpc.NSRequest_VersionRequest_VERSION_CMD(erpc.NSRequest_VersionRequest_VERSION_CMD_value["GRAB"])
	msg.Id = new(erpc.MDId)
	msg.Id.Path = []byte(path)
	msg.Grabversion = version

	rq.Command = &erpc.NSRequest_Version{Version: msg}

	// Now send the req and see what happens
	resp, err := c.cl.Exec(context.Background(), rq)
	e := c.getRespError(resp, err)
	if e != nil {
		log.Error().Str("func", "RollbackToVersion").Str("err", e.Error()).Msg("")
		return e
	}

	if resp == nil {
		return errtypes.InternalError(fmt.Sprintf("nil response for uid: '%s' ", auth.Role.UID))
	}

	log.Info().Str("func", "RollbackToVersion").Int64("errcode", resp.GetError().Code).Str("errmsg", resp.GetError().Msg).Msg("grpc response")

	return err

}

// ReadVersion reads the version for the given file.
func (c *Client) ReadVersion(ctx context.Context, auth eosclient.Authorization, p, version string) (io.ReadCloser, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "ReadVersion").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("p", p).Str("version", version).Msg("")

	versionFile := path.Join(getVersionFolder(p), version)
	return c.Read(ctx, auth, versionFile)
}

// GenerateToken returns a token on behalf of the resource owner to be used by lightweight accounts
func (c *Client) GenerateToken(ctx context.Context, auth eosclient.Authorization, path string, a *acl.Entry) (string, error) {
	return "", errtypes.NotSupported("TODO")
}

func (c *Client) getVersionFolderInode(ctx context.Context, auth eosclient.Authorization, p string) (uint64, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "getVersionFolderInode").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("p", p).Msg("")

	versionFolder := getVersionFolder(p)
	md, err := c.GetFileInfoByPath(ctx, auth, versionFolder)
	if err != nil {
		if err = c.CreateDir(ctx, auth, versionFolder); err != nil {
			return 0, err
		}
		md, err = c.GetFileInfoByPath(ctx, auth, versionFolder)
		if err != nil {
			return 0, err
		}
	}
	return md.Inode, nil
}

func (c *Client) getFileInfoFromVersion(ctx context.Context, auth eosclient.Authorization, p string) (*eosclient.FileInfo, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Str("func", "getFileInfoFromVersion").Str("uid,gid", auth.Role.UID+","+auth.Role.GID).Str("p", p).Msg("")

	file := getFileFromVersionFolder(p)
	md, err := c.GetFileInfoByPath(ctx, auth, file)
	if err != nil {
		return nil, err
	}
	return md, nil
}

func isVersionFolder(p string) bool {
	return strings.HasPrefix(path.Base(p), versionPrefix)
}

func getVersionFolder(p string) string {
	return path.Join(path.Dir(p), versionPrefix+path.Base(p))
}

func getFileFromVersionFolder(p string) string {
	return path.Join(path.Dir(p), strings.TrimPrefix(path.Base(p), versionPrefix))
}

func (c *Client) grpcMDResponseToFileInfo(st *erpc.MDResponse) (*eosclient.FileInfo, error) {
	if st.Cmd == nil && st.Fmd == nil {
		return nil, errors.Wrap(errtypes.NotSupported(""), "Invalid response (st.Cmd and st.Fmd are nil)")
	}
	fi := new(eosclient.FileInfo)

	if st.Type == erpc.TYPE_CONTAINER {
		fi.IsDir = true
		fi.Inode = st.Fmd.Inode
		fi.FID = st.Cmd.ParentId
		fi.UID = st.Cmd.Uid
		fi.GID = st.Cmd.Gid
		fi.MTimeSec = st.Cmd.Mtime.Sec
		fi.ETag = st.Cmd.Etag
		fi.File = path.Clean(string(st.Cmd.Path))

		fi.Attrs = make(map[string]string)
		for k, v := range st.Cmd.Xattrs {
			fi.Attrs[strings.TrimPrefix(k, "user.")] = string(v)
		}

		fi.Size = uint64(st.Cmd.TreeSize)

		log.Debug().Str("stat info - path", fi.File).Uint64("inode", fi.Inode).Uint64("uid", fi.UID).Uint64("gid", fi.GID).Str("etag", fi.ETag).Msg("grpc response")
	} else {
		fi.Inode = st.Fmd.Inode
		fi.FID = st.Fmd.ContId
		fi.UID = st.Fmd.Uid
		fi.GID = st.Fmd.Gid
		fi.MTimeSec = st.Fmd.Mtime.Sec
		fi.ETag = st.Fmd.Etag
		fi.File = path.Clean(string(st.Fmd.Path))

		fi.Attrs = make(map[string]string)
		for k, v := range st.Fmd.Xattrs {
			fi.Attrs[strings.TrimPrefix(k, "user.")] = string(v)
		}

		fi.Size = st.Fmd.Size

		if st.Fmd.Checksum != nil {
			xs := &eosclient.Checksum{
				XSSum:  hex.EncodeToString(st.Fmd.Checksum.Value),
				XSType: st.Fmd.Checksum.Type,
			}
			fi.XS = xs

			log.Debug().Str("stat info - path", fi.File).Uint64("inode", fi.Inode).Uint64("uid", fi.UID).Uint64("gid", fi.GID).Str("etag", fi.ETag).Str("checksum", fi.XS.XSType+":"+fi.XS.XSSum).Msg("grpc response")
		}
	}
	return fi, nil
}

// exec executes the command and returns the stdout, stderr and return code
func (c *Client) execute(ctx context.Context, cmd *exec.Cmd) (string, string, error) {
	log := appctx.GetLogger(ctx)

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	cmd.Env = []string{
		"EOS_MGM_URL=" + c.opt.URL,
	}

	if c.opt.UseKeytab {
		cmd.Env = append(cmd.Env, "XrdSecPROTOCOL="+c.opt.SecProtocol)
		cmd.Env = append(cmd.Env, "XrdSecSSSKT="+c.opt.Keytab)
	}

	err := cmd.Run()

	var exitStatus int
	if exiterr, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0
		// This works on both Unix and Windows. Although package
		// syscall is generally platform dependent, WaitStatus is
		// defined for both Unix and Windows and in both cases has
		// an ExitStatus() method with the same signature.
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {

			exitStatus = status.ExitStatus()
			switch exitStatus {
			case 0:
				err = nil
			case 2:
				err = errtypes.NotFound(errBuf.String())
			}
		}
	}

	args := fmt.Sprintf("%s", cmd.Args)
	env := fmt.Sprintf("%s", cmd.Env)
	log.Info().Str("args", args).Str("env", env).Int("exit", exitStatus).Msg("eos cmd")

	if err != nil && exitStatus != 2 { // don't wrap the errtypes.NotFoundError
		err = errors.Wrap(err, "eosclient: error while executing command")
	}

	return outBuf.String(), errBuf.String(), err
}
