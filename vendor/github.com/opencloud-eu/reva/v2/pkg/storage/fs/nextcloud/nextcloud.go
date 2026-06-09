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

package nextcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
)

func init() {
	registry.Register("nextcloud", New)
}

// StorageDriverConfig is the configuration struct for a NextcloudStorageDriver
type StorageDriverConfig struct {
	EndPoint     string `mapstructure:"endpoint"` // e.g. "http://nc/apps/sciencemesh/~alice/"
	SharedSecret string `mapstructure:"shared_secret"`
	MockHTTP     bool   `mapstructure:"mock_http"`
}

// StorageDriver implements the storage.FS interface
// and connects with a StorageDriver server as its backend
type StorageDriver struct {
	endPoint     string
	sharedSecret string
	client       *http.Client
}

func parseConfig(m map[string]interface{}) (*StorageDriverConfig, error) {
	c := &StorageDriverConfig{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

// New returns an implementation to of the storage.FS interface that talks to
// a Nextcloud instance over http.
func New(m map[string]interface{}, _ events.Stream, _ *zerolog.Logger) (storage.FS, error) {
	conf, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	return NewStorageDriver(conf)
}

// NewStorageDriver returns a new NextcloudStorageDriver
func NewStorageDriver(c *StorageDriverConfig) (*StorageDriver, error) {
	var client *http.Client
	if c.MockHTTP {
		// called := make([]string, 0)
		// nextcloudServerMock := GetNextcloudServerMock(&called)
		// client, _ = TestingHTTPClient(nextcloudServerMock)

		// This is only used by the integration tests:
		// (unit tests will call SetHTTPClient later):
		called := make([]string, 0)
		h := GetNextcloudServerMock(&called)
		client, _ = TestingHTTPClient(h)
		// FIXME: defer teardown()
	} else {
		if len(c.EndPoint) == 0 {
			return nil, errors.New("Please specify 'endpoint' in '[grpc.services.storageprovider.drivers.nextcloud]'")
		}
		client = &http.Client{}
	}
	return &StorageDriver{
		endPoint:     c.EndPoint, // e.g. "http://nc/apps/sciencemesh/"
		sharedSecret: c.SharedSecret,
		client:       client,
	}, nil
}

// Action describes a REST request to forward to the Nextcloud backend
type Action struct {
	verb string
	argS string
}

func getUser(ctx context.Context) (*user.User, error) {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		err := errors.Wrap(errtypes.UserRequired(""), "nextcloud storage driver: error getting user from ctx")
		return nil, err
	}
	return u, nil
}

// SetHTTPClient sets the HTTP client
func (nc *StorageDriver) SetHTTPClient(c *http.Client) {
	nc.client = c
}

func (nc *StorageDriver) doUpload(ctx context.Context, filePath string, r io.ReadCloser) error {
	// log := appctx.GetLogger(ctx)
	user, err := getUser(ctx)
	if err != nil {
		return err
	}
	// See https://github.com/pondersource/nc-sciencemesh/issues/5
	// url := nc.endPoint + "~" + user.Username + "/files/" + filePath
	url := nc.endPoint + "~" + user.Username + "/api/storage/Upload/" + filePath
	req, err := http.NewRequest(http.MethodPut, url, r)
	if err != nil {
		panic(err)
	}

	req.Header.Set("X-Reva-Secret", nc.sharedSecret)
	// set the request header Content-Type for the upload
	// FIXME: get the actual content type from somewhere
	req.Header.Set("Content-Type", "text/plain")
	resp, err := nc.client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	return err
}

func (nc *StorageDriver) doDownload(ctx context.Context, filePath string) (io.ReadCloser, error) {
	user, err := getUser(ctx)
	if err != nil {
		return nil, err
	}
	// See https://github.com/pondersource/nc-sciencemesh/issues/5
	// url := nc.endPoint + "~" + user.Username + "/files/" + filePath
	url := nc.endPoint + "~" + user.Username + "/api/storage/Download/" + filePath
	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader(""))
	if err != nil {
		panic(err)
	}

	resp, err := nc.client.Do(req)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		panic("No 200 response code in download request")
	}

	return resp.Body, err
}

func (nc *StorageDriver) doDownloadRevision(ctx context.Context, filePath string, key string) (io.ReadCloser, error) {
	user, err := getUser(ctx)
	if err != nil {
		return nil, err
	}
	// See https://github.com/pondersource/nc-sciencemesh/issues/5
	url := nc.endPoint + "~" + user.Username + "/api/storage/DownloadRevision/" + url.QueryEscape(key) + "/" + filePath
	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader(""))
	if err != nil {
		panic(err)
	}
	req.Header.Set("X-Reva-Secret", nc.sharedSecret)

	resp, err := nc.client.Do(req)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		panic("No 200 response code in download request")
	}

	return resp.Body, err
}

func (nc *StorageDriver) do(ctx context.Context, a Action) (int, []byte, error) {
	log := appctx.GetLogger(ctx)
	user, err := getUser(ctx)
	if err != nil {
		return 0, nil, err
	}
	// See https://github.com/cs3org/reva/issues/2377
	// for discussion of user.Username vs user.Id.OpaqueId
	url := nc.endPoint + "~" + user.Id.OpaqueId + "/api/storage/" + a.verb
	log.Info().Msgf("nc.do req %s %s", url, a.argS)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(a.argS))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("X-Reva-Secret", nc.sharedSecret)

	req.Header.Set("Content-Type", "application/json")
	resp, err := nc.client.Do(req)
	if err != nil {
		return 0, nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return 0, nil, err
	}
	log.Info().Msgf("nc.do res %s %s", url, string(body))

	return resp.StatusCode, body, nil
}

// GetHome as defined in the storage.FS interface
func (nc *StorageDriver) GetHome(ctx context.Context) (string, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Msg("GetHome")

	_, respBody, err := nc.do(ctx, Action{"GetHome", ""})
	return string(respBody), err
}

// CreateHome as defined in the storage.FS interface
func (nc *StorageDriver) CreateHome(ctx context.Context) error {
	log := appctx.GetLogger(ctx)
	log.Info().Msg("CreateHome")

	_, _, err := nc.do(ctx, Action{"CreateHome", ""})
	return err
}

// CreateDir as defined in the storage.FS interface
func (nc *StorageDriver) CreateDir(ctx context.Context, ref *provider.Reference) error {
	bodyStr, err := json.Marshal(ref)
	if err != nil {
		return err
	}
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("CreateDir %s", bodyStr)

	_, _, err = nc.do(ctx, Action{"CreateDir", string(bodyStr)})
	return err
}

// TouchFile as defined in the storage.FS interface
func (nc *StorageDriver) TouchFile(ctx context.Context, ref *provider.Reference, markprocessing bool, mtime string) error {
	return fmt.Errorf("unimplemented: TouchFile")
}

// Delete as defined in the storage.FS interface
func (nc *StorageDriver) Delete(ctx context.Context, ref *provider.Reference) error {
	bodyStr, err := json.Marshal(ref)
	if err != nil {
		return err
	}
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("Delete %s", bodyStr)

	_, _, err = nc.do(ctx, Action{"Delete", string(bodyStr)})
	return err
}

// Move as defined in the storage.FS interface
func (nc *StorageDriver) Move(ctx context.Context, oldRef, newRef *provider.Reference) error {
	type paramsObj struct {
		OldRef *provider.Reference `json:"oldRef"`
		NewRef *provider.Reference `json:"newRef"`
	}
	bodyObj := &paramsObj{
		OldRef: oldRef,
		NewRef: newRef,
	}
	bodyStr, _ := json.Marshal(bodyObj)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("Move %s", bodyStr)

	_, _, err := nc.do(ctx, Action{"Move", string(bodyStr)})
	return err
}

// GetMD as defined in the storage.FS interface
// TODO forward fieldMask
func (nc *StorageDriver) GetMD(ctx context.Context, ref *provider.Reference, mdKeys []string, fieldMask []string) (*provider.ResourceInfo, error) {
	type paramsObj struct {
		Ref    *provider.Reference `json:"ref"`
		MdKeys []string            `json:"mdKeys"`
		// MetaData provider.ResourceInfo `json:"metaData"`
	}
	bodyObj := &paramsObj{
		Ref:    ref,
		MdKeys: mdKeys,
	}
	bodyStr, _ := json.Marshal(bodyObj)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("GetMD %s", bodyStr)

	status, body, err := nc.do(ctx, Action{"GetMD", string(bodyStr)})
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, errtypes.NotFound("")
	}
	var respObj provider.ResourceInfo
	err = json.Unmarshal(body, &respObj)
	if err != nil {
		return nil, err
	}
	return &respObj, nil
}

// ListFolder as defined in the storage.FS interface
func (nc *StorageDriver) ListFolder(ctx context.Context, ref *provider.Reference, mdKeys, fieldMask []string) ([]*provider.ResourceInfo, error) {
	type paramsObj struct {
		Ref    *provider.Reference `json:"ref"`
		MdKeys []string            `json:"mdKeys"`
	}
	bodyObj := &paramsObj{
		Ref:    ref,
		MdKeys: mdKeys,
	}
	bodyStr, err := json.Marshal(bodyObj)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("ListFolder %s", bodyStr)
	if err != nil {
		return nil, err
	}
	status, body, err := nc.do(ctx, Action{"ListFolder", string(bodyStr)})
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, errtypes.NotFound("")
	}

	var respMapArr []provider.ResourceInfo
	err = json.Unmarshal(body, &respMapArr)
	if err != nil {
		return nil, err
	}
	var pointers = make([]*provider.ResourceInfo, len(respMapArr))
	for i := 0; i < len(respMapArr); i++ {
		pointers[i] = &respMapArr[i]
	}
	return pointers, err
}

// InitiateUpload as defined in the storage.FS interface
func (nc *StorageDriver) InitiateUpload(ctx context.Context, ref *provider.Reference, uploadLength int64, metadata map[string]string) (map[string]string, error) {
	type paramsObj struct {
		Ref          *provider.Reference `json:"ref"`
		UploadLength int64               `json:"uploadLength"`
		Metadata     map[string]string   `json:"metadata"`
	}
	bodyObj := &paramsObj{
		Ref:          ref,
		UploadLength: uploadLength,
		Metadata:     metadata,
	}
	bodyStr, _ := json.Marshal(bodyObj)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("InitiateUpload %s", bodyStr)

	_, respBody, err := nc.do(ctx, Action{"InitiateUpload", string(bodyStr)})
	if err != nil {
		return nil, err
	}
	respMap := make(map[string]string)
	err = json.Unmarshal(respBody, &respMap)
	if err != nil {
		return nil, err
	}
	return respMap, err
}

// Upload as defined in the storage.FS interface
func (nc *StorageDriver) Upload(ctx context.Context, req storage.UploadRequest, _ storage.UploadFinishedFunc) (*provider.ResourceInfo, error) {
	err := nc.doUpload(ctx, req.Ref.Path, req.Body)
	if err != nil {
		return &provider.ResourceInfo{}, err
	}

	// return id, etag and mtime
	ri, err := nc.GetMD(ctx, req.Ref, []string{}, []string{"id", "etag", "mtime"})
	if err != nil {
		return &provider.ResourceInfo{}, err
	}

	return ri, nil
}

// Download as defined in the storage.FS interface
func (nc *StorageDriver) Download(ctx context.Context, ref *provider.Reference, openReaderfunc func(*provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	md, err := nc.GetMD(ctx, ref, []string{}, nil)
	if err != nil {
		return nil, nil, err
	}
	if !openReaderfunc(md) {
		return md, nil, nil
	}

	reader, err := nc.doDownload(ctx, ref.Path)
	return md, reader, err
}

// ListRevisions as defined in the storage.FS interface
func (nc *StorageDriver) ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error) {
	bodyStr, _ := json.Marshal(ref)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("ListRevisions %s", bodyStr)

	_, respBody, err := nc.do(ctx, Action{"ListRevisions", string(bodyStr)})

	if err != nil {
		return nil, err
	}
	var respMapArr []provider.FileVersion
	err = json.Unmarshal(respBody, &respMapArr)
	if err != nil {
		return nil, err
	}
	revs := make([]*provider.FileVersion, len(respMapArr))
	for i := 0; i < len(respMapArr); i++ {
		revs[i] = &respMapArr[i]
	}
	return revs, err
}

// DownloadRevision as defined in the storage.FS interface
func (nc *StorageDriver) DownloadRevision(ctx context.Context, ref *provider.Reference, key string, openReaderfunc func(*provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("DownloadRevision %s %s", ref.Path, key)

	md, err := nc.GetMD(ctx, ref, []string{}, nil)
	if err != nil {
		return nil, nil, err
	}
	revs, err := nc.ListRevisions(ctx, ref)
	if err != nil {
		return nil, nil, err
	}
	var ri *provider.ResourceInfo
	for _, rev := range revs {
		if rev.Key == key {

			ri = &provider.ResourceInfo{
				// TODO(jfd) we cannot access version content, this will be a problem when trying to fetch version thumbnails
				// Opaque
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Id: &provider.ResourceId{
					StorageId: "versions",
					OpaqueId:  md.Id.OpaqueId + "@" + rev.GetKey(),
				},
				// Checksum
				Etag: rev.Etag,
				// MimeType
				Mtime: &types.Timestamp{
					Seconds: rev.Mtime,
					// TODO cs3apis FileVersion should use types.Timestamp instead of uint64
				},
				Path: path.Join("v", rev.Key),
				// PermissionSet
				Size:  rev.Size,
				Owner: md.Owner,
			}
		}
	}
	if ri == nil {
		return nil, nil, errtypes.NotFound("revision not found")
	}

	if !openReaderfunc(ri) {
		return ri, nil, nil
	}

	readCloser, err := nc.doDownloadRevision(ctx, ref.Path, key)
	return ri, readCloser, err
}

// RestoreRevision as defined in the storage.FS interface
func (nc *StorageDriver) RestoreRevision(ctx context.Context, ref *provider.Reference, key string) error {
	type paramsObj struct {
		Ref *provider.Reference `json:"ref"`
		Key string              `json:"key"`
	}
	bodyObj := &paramsObj{
		Ref: ref,
		Key: key,
	}
	bodyStr, _ := json.Marshal(bodyObj)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("RestoreRevision %s", bodyStr)

	_, _, err := nc.do(ctx, Action{"RestoreRevision", string(bodyStr)})
	return err
}

// ListRecycle as defined in the storage.FS interface
func (nc *StorageDriver) ListRecycle(ctx context.Context, ref *provider.Reference, key string, relativePath string) ([]*provider.RecycleItem, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Msg("ListRecycle")
	type paramsObj struct {
		Key  string `json:"key"`
		Path string `json:"path"`
	}
	bodyObj := &paramsObj{
		Key:  key,
		Path: relativePath,
	}
	bodyStr, _ := json.Marshal(bodyObj)

	_, respBody, err := nc.do(ctx, Action{"ListRecycle", string(bodyStr)})

	if err != nil {
		return nil, err
	}
	var respMapArr []provider.RecycleItem
	err = json.Unmarshal(respBody, &respMapArr)
	if err != nil {
		return nil, err
	}
	items := make([]*provider.RecycleItem, len(respMapArr))
	for i := 0; i < len(respMapArr); i++ {
		items[i] = &respMapArr[i]
	}
	return items, err
}

// RestoreRecycleItem as defined in the storage.FS interface
func (nc *StorageDriver) RestoreRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string, restoreRef *provider.Reference) error {
	type paramsObj struct {
		Key        string              `json:"key"`
		Path       string              `json:"path"`
		RestoreRef *provider.Reference `json:"restoreRef"`
	}
	bodyObj := &paramsObj{
		Key:        key,
		Path:       relativePath,
		RestoreRef: restoreRef,
	}
	bodyStr, _ := json.Marshal(bodyObj)

	log := appctx.GetLogger(ctx)
	log.Info().Msgf("RestoreRecycleItem %s", bodyStr)

	_, _, err := nc.do(ctx, Action{"RestoreRecycleItem", string(bodyStr)})

	return err
}

// PurgeRecycleItem as defined in the storage.FS interface
func (nc *StorageDriver) PurgeRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string) error {
	type paramsObj struct {
		Key  string `json:"key"`
		Path string `json:"path"`
	}
	bodyObj := &paramsObj{
		Key:  key,
		Path: relativePath,
	}
	bodyStr, _ := json.Marshal(bodyObj)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("PurgeRecycleItem %s", bodyStr)

	_, _, err := nc.do(ctx, Action{"PurgeRecycleItem", string(bodyStr)})
	return err
}

// EmptyRecycle as defined in the storage.FS interface
func (nc *StorageDriver) EmptyRecycle(ctx context.Context, ref *provider.Reference) error {
	log := appctx.GetLogger(ctx)
	log.Info().Msg("EmptyRecycle")

	_, _, err := nc.do(ctx, Action{"EmptyRecycle", ""})
	return err
}

// GetPathByID as defined in the storage.FS interface
func (nc *StorageDriver) GetPathByID(ctx context.Context, id *provider.ResourceId) (string, error) {
	bodyStr, _ := json.Marshal(id)
	_, respBody, err := nc.do(ctx, Action{"GetPathByID", string(bodyStr)})
	return string(respBody), err
}

// AddGrant as defined in the storage.FS interface
func (nc *StorageDriver) AddGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	type paramsObj struct {
		Ref *provider.Reference `json:"ref"`
		G   *provider.Grant     `json:"g"`
	}
	bodyObj := &paramsObj{
		Ref: ref,
		G:   g,
	}
	bodyStr, _ := json.Marshal(bodyObj)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("AddGrant %s", bodyStr)

	_, _, err := nc.do(ctx, Action{"AddGrant", string(bodyStr)})
	return err
}

// DenyGrant as defined in the storage.FS interface
func (nc *StorageDriver) DenyGrant(ctx context.Context, ref *provider.Reference, g *provider.Grantee) error {
	type paramsObj struct {
		Ref *provider.Reference `json:"ref"`
		G   *provider.Grantee   `json:"g"`
	}
	bodyObj := &paramsObj{
		Ref: ref,
		G:   g,
	}
	bodyStr, _ := json.Marshal(bodyObj)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("DenyGrant %s", bodyStr)

	_, _, err := nc.do(ctx, Action{"DenyGrant", string(bodyStr)})
	return err
}

// RemoveGrant as defined in the storage.FS interface
func (nc *StorageDriver) RemoveGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	type paramsObj struct {
		Ref *provider.Reference `json:"ref"`
		G   *provider.Grant     `json:"g"`
	}
	bodyObj := &paramsObj{
		Ref: ref,
		G:   g,
	}
	bodyStr, _ := json.Marshal(bodyObj)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("RemoveGrant %s", bodyStr)

	_, _, err := nc.do(ctx, Action{"RemoveGrant", string(bodyStr)})
	return err
}

// UpdateGrant as defined in the storage.FS interface
func (nc *StorageDriver) UpdateGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	type paramsObj struct {
		Ref *provider.Reference `json:"ref"`
		G   *provider.Grant     `json:"g"`
	}
	bodyObj := &paramsObj{
		Ref: ref,
		G:   g,
	}
	bodyStr, _ := json.Marshal(bodyObj)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("UpdateGrant %s", bodyStr)

	_, _, err := nc.do(ctx, Action{"UpdateGrant", string(bodyStr)})
	return err
}

// ListGrants as defined in the storage.FS interface
func (nc *StorageDriver) ListGrants(ctx context.Context, ref *provider.Reference) ([]*provider.Grant, error) {
	bodyStr, _ := json.Marshal(ref)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("ListGrants %s", bodyStr)

	_, respBody, err := nc.do(ctx, Action{"ListGrants", string(bodyStr)})
	if err != nil {
		return nil, err
	}

	// To avoid this error:
	// json: cannot unmarshal object into Go struct field Grantee.grantee.Id of type providerv1beta1.isGrantee_Id
	// To test:
	// bodyStr, _ := json.Marshal(provider.Grant{
	// 	 Grantee: &provider.Grantee{
	// 		 Type: provider.GranteeType_GRANTEE_TYPE_USER,
	// 		 Id: &provider.Grantee_UserId{
	// 			 UserId: &user.UserId{
	// 				 Idp:      "some-idp",
	// 				 OpaqueId: "some-opaque-id",
	// 				 Type:     user.UserType_USER_TYPE_PRIMARY,
	// 			 },
	// 		 },
	// 	 },
	// 	 Permissions: &provider.ResourcePermissions{},
	// })
	// JSON example:
	// [{"grantee":{"Id":{"UserId":{"idp":"some-idp","opaque_id":"some-opaque-id","type":1}}},"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true}}]
	var respMapArr []map[string]interface{}
	err = json.Unmarshal(respBody, &respMapArr)
	if err != nil {
		return nil, err
	}
	grants := make([]*provider.Grant, len(respMapArr))
	for i := 0; i < len(respMapArr); i++ {
		granteeMap := respMapArr[i]["grantee"].(map[string]interface{})
		granteeIDMap := granteeMap["Id"].(map[string]interface{})
		granteeIDUserIDMap := granteeIDMap["UserId"].(map[string]interface{})

		// if (granteeMap["Id"])
		permsMap := respMapArr[i]["permissions"].(map[string]interface{})
		grants[i] = &provider.Grant{
			Grantee: &provider.Grantee{
				Type: provider.GranteeType_GRANTEE_TYPE_USER, // FIXME: support groups too
				Id: &provider.Grantee_UserId{
					UserId: &user.UserId{
						Idp:      granteeIDUserIDMap["idp"].(string),
						OpaqueId: granteeIDUserIDMap["opaque_id"].(string),
						Type:     user.UserType(granteeIDUserIDMap["type"].(float64)),
					},
				},
			},
			Permissions: &provider.ResourcePermissions{
				AddGrant:             permsMap["add_grant"].(bool),
				CreateContainer:      permsMap["create_container"].(bool),
				Delete:               permsMap["delete"].(bool),
				GetPath:              permsMap["get_path"].(bool),
				GetQuota:             permsMap["get_quota"].(bool),
				InitiateFileDownload: permsMap["initiate_file_download"].(bool),
				InitiateFileUpload:   permsMap["initiate_file_upload"].(bool),
				ListGrants:           permsMap["list_grants"].(bool),
				ListContainer:        permsMap["list_container"].(bool),
				ListFileVersions:     permsMap["list_file_versions"].(bool),
				ListRecycle:          permsMap["list_recycle"].(bool),
				Move:                 permsMap["move"].(bool),
				RemoveGrant:          permsMap["remove_grant"].(bool),
				PurgeRecycle:         permsMap["purge_recycle"].(bool),
				RestoreFileVersion:   permsMap["restore_file_version"].(bool),
				RestoreRecycleItem:   permsMap["restore_recycle_item"].(bool),
				Stat:                 permsMap["stat"].(bool),
				UpdateGrant:          permsMap["update_grant"].(bool),
			},
		}
	}
	return grants, err
}

// GetQuota as defined in the storage.FS interface
func (nc *StorageDriver) GetQuota(ctx context.Context, ref *provider.Reference) (uint64, uint64, uint64, error) {
	log := appctx.GetLogger(ctx)
	log.Info().Msg("GetQuota")

	_, respBody, err := nc.do(ctx, Action{"GetQuota", ""})
	if err != nil {
		return 0, 0, 0, err
	}

	var respMap map[string]interface{}
	err = json.Unmarshal(respBody, &respMap)
	if err != nil {
		return 0, 0, 0, err
	}

	total := uint64(respMap["totalBytes"].(float64))
	used := uint64(respMap["usedBytes"].(float64))
	remaining := total - used
	return total, used, remaining, err
}

// CreateReference as defined in the storage.FS interface
func (nc *StorageDriver) CreateReference(ctx context.Context, path string, targetURI *url.URL) error {
	type paramsObj struct {
		Path string `json:"path"`
		URL  string `json:"url"`
	}
	bodyObj := &paramsObj{
		Path: path,
		URL:  targetURI.String(),
	}
	bodyStr, _ := json.Marshal(bodyObj)

	_, _, err := nc.do(ctx, Action{"CreateReference", string(bodyStr)})
	return err
}

// Shutdown as defined in the storage.FS interface
func (nc *StorageDriver) Shutdown(ctx context.Context) error {
	log := appctx.GetLogger(ctx)
	log.Info().Msg("Shutdown")

	_, _, err := nc.do(ctx, Action{"Shutdown", ""})
	return err
}

// SetArbitraryMetadata as defined in the storage.FS interface
func (nc *StorageDriver) SetArbitraryMetadata(ctx context.Context, ref *provider.Reference, md *provider.ArbitraryMetadata) error {
	type paramsObj struct {
		Ref *provider.Reference         `json:"ref"`
		Md  *provider.ArbitraryMetadata `json:"md"`
	}
	bodyObj := &paramsObj{
		Ref: ref,
		Md:  md,
	}
	bodyStr, _ := json.Marshal(bodyObj)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("SetArbitraryMetadata %s", bodyStr)

	_, _, err := nc.do(ctx, Action{"SetArbitraryMetadata", string(bodyStr)})
	return err
}

// UnsetArbitraryMetadata as defined in the storage.FS interface
func (nc *StorageDriver) UnsetArbitraryMetadata(ctx context.Context, ref *provider.Reference, keys []string) error {
	type paramsObj struct {
		Ref  *provider.Reference `json:"ref"`
		Keys []string            `json:"keys"`
	}
	bodyObj := &paramsObj{
		Ref:  ref,
		Keys: keys,
	}
	bodyStr, _ := json.Marshal(bodyObj)
	log := appctx.GetLogger(ctx)
	log.Info().Msgf("UnsetArbitraryMetadata %s", bodyStr)

	_, _, err := nc.do(ctx, Action{"UnsetArbitraryMetadata", string(bodyStr)})
	return err
}

// GetLock returns an existing lock on the given reference
func (nc *StorageDriver) GetLock(ctx context.Context, ref *provider.Reference) (*provider.Lock, error) {
	return nil, errtypes.NotSupported("unimplemented")
}

// SetLock puts a lock on the given reference
func (nc *StorageDriver) SetLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	return errtypes.NotSupported("unimplemented")
}

// RefreshLock refreshes an existing lock on the given reference
func (nc *StorageDriver) RefreshLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock, existingLockID string) error {
	return errtypes.NotSupported("unimplemented")
}

// Unlock removes an existing lock from the given reference
func (nc *StorageDriver) Unlock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	return errtypes.NotSupported("unimplemented")
}

// AddLabel adds a label to a resource
func (nc *StorageDriver) AddLabel(ctx context.Context, ref *provider.Reference, userID *user.UserId, label string) error {
	return errtypes.NotSupported("AddLabel not implemented")
}

// RemoveLabel removes a label from a resource
func (nc *StorageDriver) RemoveLabel(ctx context.Context, ref *provider.Reference, userID *user.UserId, label string) error {
	return errtypes.NotSupported("RemoveLabel not implemented")
}

// ListStorageSpaces as defined in the storage.FS interface
func (nc *StorageDriver) ListStorageSpaces(ctx context.Context, f []*provider.ListStorageSpacesRequest_Filter, unrestricted bool) ([]*provider.StorageSpace, error) {
	bodyStr, _ := json.Marshal(f)
	_, respBody, err := nc.do(ctx, Action{"ListStorageSpaces", string(bodyStr)})
	if err != nil {
		return nil, err
	}

	// https://github.com/cs3org/go-cs3apis/blob/970eec3/cs3/storage/provider/v1beta1/resources.pb.go#L1341-L1366
	var respMapArr []provider.StorageSpace
	err = json.Unmarshal(respBody, &respMapArr)
	if err != nil {
		return nil, err
	}
	var spaces = make([]*provider.StorageSpace, len(respMapArr))
	for i := 0; i < len(respMapArr); i++ {
		spaces[i] = &respMapArr[i]
	}
	return spaces, err
}

// CreateStorageSpace creates a storage space
func (nc *StorageDriver) CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error) {
	bodyStr, _ := json.Marshal(req)
	_, respBody, err := nc.do(ctx, Action{"CreateStorageSpace", string(bodyStr)})
	if err != nil {
		return nil, err
	}
	var respObj provider.CreateStorageSpaceResponse
	err = json.Unmarshal(respBody, &respObj)
	if err != nil {
		return nil, err
	}
	return &respObj, nil
}

// UpdateStorageSpace updates a storage space
func (nc *StorageDriver) UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error) {
	bodyStr, _ := json.Marshal(req)
	_, respBody, err := nc.do(ctx, Action{"UpdateStorageSpace", string(bodyStr)})
	if err != nil {
		return nil, err
	}
	var respObj provider.UpdateStorageSpaceResponse
	err = json.Unmarshal(respBody, &respObj)
	if err != nil {
		return nil, err
	}
	return &respObj, nil
}

// DeleteStorageSpace deletes a storage space
func (nc *StorageDriver) DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) error {
	bodyStr, _ := json.Marshal(req)
	_, respBody, err := nc.do(ctx, Action{"DeleteStorageSpace", string(bodyStr)})
	if err != nil {
		return err
	}
	var respObj provider.DeleteStorageSpaceResponse
	err = json.Unmarshal(respBody, &respObj)
	if err != nil {
		return err
	}
	return nil
}
