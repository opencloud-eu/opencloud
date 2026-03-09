// Copyright 2018-2023 CERN
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

package ocm

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	ocmpb "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typepb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/rs/zerolog"
	"github.com/studio-b12/gowebdav"

	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/mime"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/router"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/opencloud-eu/reva/v2/pkg/utils/cfg"
)

func init() {
	registry.Register("ocmreceived", New)
}

type driver struct {
	c       *config
	gateway gateway.GatewayAPIClient
}

type config struct {
	GatewaySVC           string `mapstructure:"gatewaysvc"`
	Insecure             bool   `mapstructure:"insecure"`
	StorageRoot          string `mapstructure:"storage_root"`
	ServiceAccountID     string `mapstructure:"service_account_id"`
	ServiceAccountSecret string `mapstructure:"service_account_secret"`
}

func (c *config) ApplyDefaults() {
	c.GatewaySVC = sharedconf.GetGatewaySVC(c.GatewaySVC)
}

// BearerAuthenticator represents an authenticator that adds a Bearer token to the Authorization header of HTTP requests.
type BearerAuthenticator struct {
	Token string
}

// Authorize adds the Bearer token to the Authorization header of the provided HTTP request.
func (b BearerAuthenticator) Authorize(_ *http.Client, r *http.Request, _ string) error {
	r.Header.Add("Authorization", "Bearer "+b.Token)
	return nil
}

// Verify is not implemented for the BearerAuthenticator. It always returns false and nil error.
func (BearerAuthenticator) Verify(*http.Client, *http.Response, string) (bool, error) {
	return false, nil
}

// Clone creates a new instance of the BearerAuthenticator.
func (b BearerAuthenticator) Clone() gowebdav.Authenticator {
	return BearerAuthenticator{Token: b.Token}
}

// Close is not implemented for the BearerAuthenticator. It always returns nil.
func (BearerAuthenticator) Close() error {
	return nil
}

// New creates an OCM storage driver.
func New(m map[string]interface{}, _ events.Stream, _ *zerolog.Logger) (storage.FS, error) {
	var c config
	if err := cfg.Decode(m, &c); err != nil {
		return nil, err
	}

	gateway, err := pool.GetGatewayServiceClient(c.GatewaySVC)
	if err != nil {
		return nil, err
	}

	d := &driver{
		c:       &c,
		gateway: gateway,
	}

	return d, nil
}

func shareInfoFromPath(path string) (*ocmpb.ShareId, string) {
	// the path is of the type /share_id[/rel_path]
	shareID, rel := router.ShiftPath(path)
	return &ocmpb.ShareId{OpaqueId: shareID}, rel
}

func shareInfoFromReference(ref *provider.Reference) (*ocmpb.ShareId, string) {
	if ref.ResourceId == nil {
		return shareInfoFromPath(ref.Path)
	}

	if ref.ResourceId.SpaceId == ref.ResourceId.OpaqueId {
		return &ocmpb.ShareId{OpaqueId: ref.ResourceId.SpaceId}, ref.Path
	}
	decodedBytes, err := base64.StdEncoding.DecodeString(ref.ResourceId.OpaqueId)
	if err != nil {
		// this should never happen
		return &ocmpb.ShareId{OpaqueId: ref.ResourceId.SpaceId}, ref.Path
	}
	return &ocmpb.ShareId{OpaqueId: ref.ResourceId.SpaceId}, filepath.Join(string(decodedBytes), ref.Path)

}

func (d *driver) getWebDAVFromShare(ctx context.Context, forUser *userpb.UserId, shareID *ocmpb.ShareId) (*ocmpb.ReceivedShare, string, string, error) {
	// TODO: we may want to cache the share
	req := &ocmpb.GetReceivedOCMShareRequest{
		Ref: &ocmpb.ShareReference{
			Spec: &ocmpb.ShareReference_Id{
				Id: shareID,
			},
		},
	}
	if forUser != nil {
		req.Opaque = utils.AppendJSONToOpaque(nil, "userid", forUser)
	}
	res, err := d.gateway.GetReceivedOCMShare(ctx, req)
	if err != nil {
		return nil, "", "", err
	}

	if res.Status.Code != rpc.Code_CODE_OK {
		if res.Status.Code == rpc.Code_CODE_NOT_FOUND {
			return nil, "", "", errtypes.NotFound("share not found")
		}
		return nil, "", "", errtypes.InternalError(res.Status.Message)
	}

	dav, ok := getWebDAVProtocol(res.Share.Protocols)
	if !ok {
		return nil, "", "", errtypes.NotFound("share does not contain a WebDAV endpoint")
	}

	return res.Share, dav.Uri, dav.SharedSecret, nil
}

func getWebDAVProtocol(protocols []*ocmpb.Protocol) (*ocmpb.WebDAVProtocol, bool) {
	for _, p := range protocols {
		if dav, ok := p.Term.(*ocmpb.Protocol_WebdavOptions); ok {
			return dav.WebdavOptions, true
		}
	}
	return nil, false
}

func (d *driver) webdavClient(ctx context.Context, forUser *userpb.UserId, ref *provider.Reference) (*gowebdav.Client, *ocmpb.ReceivedShare, string, error) {
	id, rel := shareInfoFromReference(ref)

	share, endpoint, secret, err := d.getWebDAVFromShare(ctx, forUser, id)
	if err != nil {
		return nil, nil, "", err
	}

	endpoint, err = url.PathUnescape(endpoint)
	if err != nil {
		return nil, nil, "", err
	}

	// FIXME: it's still not clear from the OCM APIs how to use the shared secret
	// will use as a token in the bearer authentication as this is the reva implementation
	c := gowebdav.NewAuthClient(endpoint, gowebdav.NewPreemptiveAuth(BearerAuthenticator{Token: secret}))
	if d.c.Insecure {
		c.SetTransport(&http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		})
	}

	return c, share, rel, nil
}

func (d *driver) CreateDir(ctx context.Context, ref *provider.Reference) error {
	client, _, rel, err := d.webdavClient(ctx, nil, ref)
	if err != nil {
		return err
	}
	return client.MkdirAll(rel, 0)
}

func (d *driver) Delete(ctx context.Context, ref *provider.Reference) error {
	client, _, rel, err := d.webdavClient(ctx, nil, ref)
	if err != nil {
		return err
	}
	return client.RemoveAll(rel)
}

func (d *driver) TouchFile(ctx context.Context, ref *provider.Reference, markprocessing bool, mtime string) error {
	client, _, rel, err := d.webdavClient(ctx, nil, ref)
	if err != nil {
		return err
	}
	return client.Write(rel, []byte{}, 0)
}

func (d *driver) Move(ctx context.Context, oldRef, newRef *provider.Reference) error {
	client, _, relOld, err := d.webdavClient(ctx, nil, oldRef)
	if err != nil {
		return err
	}
	_, relNew := shareInfoFromReference(newRef)

	return client.Rename(relOld, relNew, false)
}

func getPathFromShareIDAndRelPath(shareID *ocmpb.ShareId, relPath string) string {
	return filepath.Join("/", shareID.OpaqueId, relPath)
}

func convertStatToResourceInfo(ref *provider.Reference, f fs.FileInfo, share *ocmpb.ReceivedShare) (*provider.ResourceInfo, error) {
	t := provider.ResourceType_RESOURCE_TYPE_FILE
	if f.IsDir() {
		t = provider.ResourceType_RESOURCE_TYPE_CONTAINER
	}

	webdavFile, ok := f.(gowebdav.File)
	if !ok {
		return nil, errtypes.InternalError("could not get webdav props")
	}

	var name string
	switch {
	case share.ResourceType == provider.ResourceType_RESOURCE_TYPE_FILE: //nolint:staticcheck // we will update our ocm implementation later
		name = share.Name
	case webdavFile.Path() == "/":
		name = share.Name
	default:
		name = webdavFile.Name()
	}

	opaqueid := base64.StdEncoding.EncodeToString([]byte(webdavFile.Path()))

	// ids are of the format <ocmstorageproviderid>$<shareid>!<opaqueid>
	id := &provider.ResourceId{
		StorageId: utils.OCMStorageProviderID,
		SpaceId:   share.Id.OpaqueId,
		OpaqueId:  opaqueid,
	}
	webdavProtocol, _ := getWebDAVProtocol(share.Protocols)

	ri := provider.ResourceInfo{
		Type:     t,
		Id:       id,
		MimeType: mime.Detect(f.IsDir(), f.Name()),
		Path:     name,
		Name:     name,
		Size:     uint64(f.Size()),
		Mtime: &typepb.Timestamp{
			Seconds: uint64(f.ModTime().Unix()),
		},
		Etag:          webdavFile.ETag(),
		Owner:         share.Creator,
		PermissionSet: webdavProtocol.Permissions.Permissions,
	}

	if t == provider.ResourceType_RESOURCE_TYPE_FILE {
		// get SHA1 checksum from OpenCloud specific properties if available
		propstat := webdavFile.Sys().(gowebdav.Props)
		ri.Checksum = extractChecksum(propstat)
	}

	if f.(gowebdav.File).StatusCode() == 425 {
		ri.Opaque = utils.AppendPlainToOpaque(ri.Opaque, "status", "processing")
	}

	return &ri, nil
}

func extractChecksum(props gowebdav.Props) *provider.ResourceChecksum {
	checksums := props.GetString(xml.Name{Space: "http://owncloud.org/ns", Local: "checksums"})
	if checksums == "" {
		return &provider.ResourceChecksum{
			Type: provider.ResourceChecksumType_RESOURCE_CHECKSUM_TYPE_INVALID,
		}
	}
	re := regexp.MustCompile("SHA1:(.*)")
	matches := re.FindStringSubmatch(checksums)
	if len(matches) == 2 {
		return &provider.ResourceChecksum{
			Type: provider.ResourceChecksumType_RESOURCE_CHECKSUM_TYPE_SHA1,
			Sum:  matches[1],
		}
	}
	return &provider.ResourceChecksum{
		Type: provider.ResourceChecksumType_RESOURCE_CHECKSUM_TYPE_INVALID,
	}
}

func (d *driver) GetMD(ctx context.Context, ref *provider.Reference, _ []string, _ []string) (*provider.ResourceInfo, error) {
	client, share, rel, err := d.webdavClient(ctx, nil, ref)
	if err != nil {
		return nil, err
	}

	info, err := client.StatWithProps(rel, []string{}) // request all properties by giving an empty list
	if err != nil {
		if gowebdav.IsErrNotFound(err) {
			return nil, errtypes.NotFound(ref.GetPath())
		}
		return nil, err
	}

	return convertStatToResourceInfo(ref, info, share)
}

func (d *driver) ListFolder(ctx context.Context, ref *provider.Reference, _ []string, _ []string) ([]*provider.ResourceInfo, error) {
	client, share, rel, err := d.webdavClient(ctx, nil, ref)
	if err != nil {
		return nil, err
	}

	list, err := client.ReadDirWithProps(rel, []string{}) // request all properties by giving an empty list
	if err != nil {
		return nil, err
	}

	res := make([]*provider.ResourceInfo, 0, len(list))
	for _, r := range list {
		info, err := convertStatToResourceInfo(ref, r, share)
		if err != nil {
			return nil, err
		}
		res = append(res, info)
	}
	return res, nil
}

func (d *driver) Download(ctx context.Context, ref *provider.Reference, openReaderfunc func(*provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	client, share, rel, err := d.webdavClient(ctx, nil, ref)
	if err != nil {
		return nil, nil, err
	}

	info, err := client.StatWithProps(rel, []string{}) // request all properties by giving an empty list
	if err != nil {
		if gowebdav.IsErrNotFound(err) {
			return nil, nil, errtypes.NotFound(ref.GetPath())
		}
		return nil, nil, err
	}
	md, err := convertStatToResourceInfo(ref, info, share)
	if err != nil {
		return nil, nil, err
	}

	if !openReaderfunc(md) {
		return md, nil, nil
	}

	reader, err := client.ReadStream(rel)
	return md, reader, err
}

func (d *driver) GetPathByID(ctx context.Context, id *provider.ResourceId) (string, error) {
	shareID, rel := shareInfoFromReference(&provider.Reference{
		ResourceId: id,
	})
	return getPathFromShareIDAndRelPath(shareID, rel), nil
}

func (d *driver) Shutdown(ctx context.Context) error {
	return nil
}

func (d *driver) CreateHome(ctx context.Context) error {
	return errtypes.NotSupported("operation not supported")
}

func (d *driver) GetHome(ctx context.Context) (string, error) {
	return "", errtypes.NotSupported("operation not supported")
}

func (d *driver) ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error) {
	return nil, errtypes.NotSupported("operation not supported")
}

func (d *driver) DownloadRevision(ctx context.Context, ref *provider.Reference, key string, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	return nil, nil, errtypes.NotSupported("operation not supported")
}

func (d *driver) RestoreRevision(ctx context.Context, ref *provider.Reference, key string) error {
	return errtypes.NotSupported("operation not supported")
}

func (d *driver) ListRecycle(ctx context.Context, ref *provider.Reference, key, relativePath string) ([]*provider.RecycleItem, error) {
	return nil, errtypes.NotSupported("operation not supported")
}

func (d *driver) RestoreRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string, restoreRef *provider.Reference) error {
	return errtypes.NotSupported("operation not supported")
}

func (d *driver) PurgeRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string) error {
	return errtypes.NotSupported("operation not supported")
}

func (d *driver) EmptyRecycle(ctx context.Context, ref *provider.Reference) error {
	return errtypes.NotSupported("operation not supported")
}

func (d *driver) AddGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	return errtypes.NotSupported("operation not supported")
}

func (d *driver) DenyGrant(ctx context.Context, ref *provider.Reference, g *provider.Grantee) error {
	return errtypes.NotSupported("operation not supported")
}

func (d *driver) RemoveGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	return errtypes.NotSupported("operation not supported")
}

func (d *driver) UpdateGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	return errtypes.NotSupported("operation not supported")
}

func (d *driver) ListGrants(ctx context.Context, ref *provider.Reference) ([]*provider.Grant, error) {
	return nil, errtypes.NotSupported("operation not supported")
}

func (d *driver) GetQuota(ctx context.Context, ref *provider.Reference) ( /*TotalBytes*/ uint64 /*UsedBytes*/, uint64, uint64, error) {
	return 0, 0, 0, errtypes.NotSupported("operation not supported")
}

func (d *driver) CreateReference(ctx context.Context, path string, targetURI *url.URL) error {
	return errtypes.NotSupported("operation not supported")
}

func (d *driver) SetArbitraryMetadata(ctx context.Context, ref *provider.Reference, md *provider.ArbitraryMetadata) error {
	return errtypes.NotSupported("operation not supported")
}

func (d *driver) UnsetArbitraryMetadata(ctx context.Context, ref *provider.Reference, keys []string) error {
	return errtypes.NotSupported("operation not supported")
}

// SetLock sets a lock on a file
func (d *driver) SetLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	client, _, rel, err := d.webdavClient(ctx, nil, ref)
	if err != nil {
		return err
	}

	return client.Lock(rel, lock.GetLockId())
}

func (d *driver) GetLock(ctx context.Context, ref *provider.Reference) (*provider.Lock, error) {
	client, _, rel, err := d.webdavClient(ctx, nil, ref)
	if err != nil {
		return nil, err
	}

	token, err := client.GetLock(rel)
	if err != nil {
		return nil, err
	}

	return &provider.Lock{LockId: token, Type: provider.LockType_LOCK_TYPE_EXCL}, nil
}

func (d *driver) RefreshLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock, existingLockID string) error {
	client, _, rel, err := d.webdavClient(ctx, nil, ref)
	if err != nil {
		return err
	}

	return client.RefreshLock(rel, lock.GetLockId())
}

// Unlock removes a lock from a file
func (d *driver) Unlock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	client, _, rel, err := d.webdavClient(ctx, nil, ref)
	if err != nil {
		return err
	}

	return client.Unlock(rel, lock.GetLockId())
}

func (d *driver) ListStorageSpaces(ctx context.Context, filters []*provider.ListStorageSpacesRequest_Filter, _ bool) ([]*provider.StorageSpace, error) {
	spaceTypes := map[string]struct{}{}
	var exists = struct{}{}
	appendTypes := []string{}
	for _, f := range filters {
		if f.Type == provider.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE {
			spaceType := f.GetSpaceType()
			if spaceType == "+mountpoint" {
				appendTypes = append(appendTypes, strings.TrimPrefix(spaceType, "+"))
				continue
			}
			spaceTypes[spaceType] = exists
		}
	}

	lrsRes, err := d.gateway.ListReceivedOCMShares(ctx, &ocmpb.ListReceivedOCMSharesRequest{})
	if err != nil {
		return nil, err
	}
	// FIXME This might have to be ListOCMShares
	// these are grants

	lsRes, err := d.gateway.ListOCMShares(ctx, &ocmpb.ListOCMSharesRequest{})
	if err != nil {
		return nil, err
	}

	if len(spaceTypes) == 0 {
		spaceTypes["mountpoint"] = exists
	}
	for _, s := range appendTypes {
		spaceTypes[s] = exists
	}

	spaces := []*provider.StorageSpace{}
	for k := range spaceTypes {
		if k == "mountpoint" {
			for _, share := range lrsRes.Shares {
				root := &provider.ResourceId{
					StorageId: utils.OCMStorageProviderID,
					SpaceId:   share.Id.OpaqueId,
					OpaqueId:  share.Id.OpaqueId,
				}
				space := &provider.StorageSpace{
					Id: &provider.StorageSpaceId{
						OpaqueId: storagespace.FormatResourceID(root),
					},
					SpaceType: "mountpoint",
					Owner: &userpb.User{
						Id: share.Grantee.GetUserId(),
					},
					Root: root,
				}

				spaces = append(spaces, space)
			}
			for _, share := range lsRes.Shares {
				root := &provider.ResourceId{
					StorageId: utils.OCMStorageProviderID,
					SpaceId:   share.Id.OpaqueId,
					OpaqueId:  share.Id.OpaqueId,
				}
				space := &provider.StorageSpace{
					Id: &provider.StorageSpaceId{
						OpaqueId: storagespace.FormatResourceID(root),
					},
					SpaceType: "mountpoint",
					Owner: &userpb.User{
						Id: share.Grantee.GetUserId(),
					},
					Root: root,
				}

				spaces = append(spaces, space)
			}
		}
	}

	return spaces, nil
}

func (d *driver) CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error) {
	return nil, errtypes.NotSupported("operation not supported")
}

func (d *driver) UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error) {
	return nil, errtypes.NotSupported("operation not supported")
}

func (d *driver) DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) error {
	return errtypes.NotSupported("operation not supported")
}
