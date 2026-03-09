// Copyright 2018-2022 CERN
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

package metadata

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"

	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/cs3org/reva/pkg/storage/utils/metadata")
}

// CS3 represents a metadata storage with a cs3 storage backend
type CS3 struct {
	SpaceRoot *provider.ResourceId

	providerAddr      string
	gatewayAddr       string
	useSystemUser     bool
	serviceUser       *user.User
	machineAuthAPIKey string

	dataGatewayClient *http.Client
}

// NewCS3 returns a new CS3 instance. Use an authenticated context and be sure to define SpaceRoot manually.
func NewCS3(gwAddr, providerAddr string) (s *CS3) {
	return &CS3{
		providerAddr:      providerAddr,
		gatewayAddr:       gwAddr,
		dataGatewayClient: http.DefaultClient,
	}
}

// NewCS3Storage returns a new cs3 storage instance. Context passed to methods is irrelevant as the service user will be used.
// Be sure to call Init before using the storage.
func NewCS3Storage(gwAddr, providerAddr, serviceUserID, serviceUserIDP, machineAuthAPIKey string) (s Storage, err error) {
	cs3 := NewCS3(gwAddr, providerAddr)

	cs3.useSystemUser = true
	cs3.machineAuthAPIKey = machineAuthAPIKey
	cs3.serviceUser = &user.User{
		Id: &user.UserId{
			OpaqueId: serviceUserID,
			Idp:      serviceUserIDP,
		},
	}

	return cs3, nil
}

// Backend returns the backend name of the storage
func (cs3 *CS3) Backend() string {
	return "cs3"
}

// Init creates the metadata space
func (cs3 *CS3) Init(ctx context.Context, spaceid string) (err error) {
	logger := appctx.GetLogger(ctx)
	ctx, span := tracer.Start(ctx, "Init")
	defer span.End()

	client, err := cs3.spacesClient()
	if err != nil {
		logger.Err(err).Msg("error getting spaces client")
		return err
	}

	ctx, err = cs3.getAuthContext(ctx)
	if err != nil {
		logger.Err(err).Msg("error getting auth context")
		return err
	}

	lsRes, err := client.ListStorageSpaces(ctx, &provider.ListStorageSpacesRequest{
		Filters: []*provider.ListStorageSpacesRequest_Filter{
			{
				Type: provider.ListStorageSpacesRequest_Filter_TYPE_ID,
				Term: &provider.ListStorageSpacesRequest_Filter_Id{
					Id: &provider.StorageSpaceId{OpaqueId: spaceid + "!" + spaceid},
				},
			},
		},
	})
	switch {
	case err != nil:
		return err
	case lsRes.Status.Code == rpc.Code_CODE_OK && len(lsRes.StorageSpaces) > 0:
		if len(lsRes.StorageSpaces) > 0 {
			cs3.SpaceRoot = lsRes.StorageSpaces[0].Root
			return nil
		}
	}

	// FIXME change CS3 api to allow sending a space id
	cssr, err := client.CreateStorageSpace(ctx, &provider.CreateStorageSpaceRequest{
		Opaque: &types.Opaque{
			Map: map[string]*types.OpaqueEntry{
				"spaceid": {
					Decoder: "plain",
					Value:   []byte(spaceid),
				},
			},
		},
		Owner: cs3.serviceUser,
		Name:  "Metadata",
		Type:  "metadata",
	})
	switch {
	case err != nil:
		logger.Err(err).Msg("error creating storage space")
		return err
	case cssr.Status.Code == rpc.Code_CODE_OK:
		cs3.SpaceRoot = cssr.StorageSpace.Root
	case cssr.Status.Code == rpc.Code_CODE_ALREADY_EXISTS:
		return errtypes.AlreadyExists(fmt.Sprintf("user %s does not have access to metadata space %s, but it exists", cs3.serviceUser.Id.OpaqueId, spaceid))
	default:
		logger.Debug().Str("Status", cssr.Status.Message).Msg("error creating storage space")
		return errtypes.NewErrtypeFromStatus(cssr.Status)
	}
	return nil
}

// SimpleUpload uploads a file to the metadata storage
func (cs3 *CS3) SimpleUpload(ctx context.Context, uploadpath string, content []byte) error {
	ctx, span := tracer.Start(ctx, "SimpleUpload")
	defer span.End()

	_, err := cs3.Upload(ctx, UploadRequest{
		Path:    uploadpath,
		Content: content,
	})
	return err
}

// Upload uploads a file to the metadata storage
func (cs3 *CS3) Upload(ctx context.Context, req UploadRequest) (*UploadResponse, error) {
	ctx, span := tracer.Start(ctx, "Upload")
	defer span.End()

	client, err := cs3.providerClient()
	if err != nil {
		return nil, err
	}
	ctx, err = cs3.getAuthContext(ctx)
	if err != nil {
		return nil, err
	}

	ifuReq := &provider.InitiateFileUploadRequest{
		Opaque: &types.Opaque{},
		Ref: &provider.Reference{
			ResourceId: cs3.SpaceRoot,
			Path:       utils.MakeRelativePath(req.Path),
		},
	}

	if req.IfMatchEtag != "" {
		ifuReq.Options = &provider.InitiateFileUploadRequest_IfMatch{
			IfMatch: req.IfMatchEtag,
		}
	}
	if len(req.IfNoneMatch) > 0 {
		if req.IfNoneMatch[0] == "*" {
			ifuReq.Options = &provider.InitiateFileUploadRequest_IfNotExist{
				IfNotExist: true,
			}
		}
		// else {
		//   the http upload will carry all if-not-match etags
		// }
	}
	if req.IfUnmodifiedSince != (time.Time{}) {
		ifuReq.Options = &provider.InitiateFileUploadRequest_IfUnmodifiedSince{
			IfUnmodifiedSince: utils.TimeToTS(req.IfUnmodifiedSince),
		}
	}
	if req.MTime != (time.Time{}) {
		// The format of the X-OC-Mtime header is <epoch>.<nanoseconds>, e.g. '1691053416.934129485'
		ifuReq.Opaque = utils.AppendPlainToOpaque(ifuReq.Opaque, "X-OC-Mtime", utils.TimeToOCMtime(req.MTime))
	}

	ifuReq.Opaque = utils.AppendPlainToOpaque(ifuReq.Opaque, net.HeaderUploadLength, strconv.FormatInt(int64(len(req.Content)), 10))

	res, err := client.InitiateFileUpload(ctx, ifuReq)
	if err != nil {
		return nil, err
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return nil, errtypes.NewErrtypeFromStatus(res.Status)
	}

	var endpoint string

	for _, proto := range res.GetProtocols() {
		if proto.Protocol == "simple" {
			endpoint = proto.GetUploadEndpoint()
			break
		}
	}
	if endpoint == "" {
		return nil, errors.New("metadata storage doesn't support the simple upload protocol")
	}

	httpReq, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(req.Content))
	if err != nil {
		return nil, err
	}
	for _, etag := range req.IfNoneMatch {
		httpReq.Header.Add(net.HeaderIfNoneMatch, etag)
	}

	md, _ := metadata.FromOutgoingContext(ctx)
	httpReq.Header.Add(ctxpkg.TokenHeader, md.Get(ctxpkg.TokenHeader)[0])
	resp, err := cs3.dataGatewayClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := errtypes.NewErrtypeFromHTTPStatusCode(resp.StatusCode, httpReq.URL.Path); err != nil {
		return nil, err
	}
	etag := resp.Header.Get("Etag")
	if ocEtag := resp.Header.Get("OC-ETag"); ocEtag != "" {
		etag = ocEtag
	}
	return &UploadResponse{
		Etag:   etag,
		FileID: resp.Header.Get("OC-Fileid"),
	}, nil
}

// Stat returns the metadata for the given path
func (cs3 *CS3) Stat(ctx context.Context, path string) (*provider.ResourceInfo, error) {
	ctx, span := tracer.Start(ctx, "Stat")
	defer span.End()

	client, err := cs3.providerClient()
	if err != nil {
		return nil, err
	}
	ctx, err = cs3.getAuthContext(ctx)
	if err != nil {
		return nil, err
	}

	req := provider.StatRequest{
		Ref: &provider.Reference{
			ResourceId: cs3.SpaceRoot,
			Path:       utils.MakeRelativePath(path),
		},
	}

	res, err := client.Stat(ctx, &req)
	if err != nil {
		return nil, err
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return nil, errtypes.NewErrtypeFromStatus(res.Status)
	}

	return res.Info, nil
}

// SimpleDownload reads a file from the metadata storage
func (cs3 *CS3) SimpleDownload(ctx context.Context, downloadpath string) (content []byte, err error) {
	ctx, span := tracer.Start(ctx, "SimpleDownload")
	defer span.End()
	dres, err := cs3.Download(ctx, DownloadRequest{Path: downloadpath})
	if err != nil {
		return nil, err
	}
	return dres.Content, nil
}

// Download reads a file from the metadata storage
func (cs3 *CS3) Download(ctx context.Context, req DownloadRequest) (*DownloadResponse, error) {
	ctx, span := tracer.Start(ctx, "Download")
	defer span.End()

	client, err := cs3.providerClient()
	if err != nil {
		return nil, err
	}
	ctx, err = cs3.getAuthContext(ctx)
	if err != nil {
		return nil, err
	}

	dreq := provider.InitiateFileDownloadRequest{
		Ref: &provider.Reference{
			ResourceId: cs3.SpaceRoot,
			Path:       utils.MakeRelativePath(req.Path),
		},
	}
	// FIXME add a dedicated property on the CS3 InitiateFileDownloadRequest message
	// well the gateway never forwards the initiate request to the storageprovider, so we have to send it in the actual GET request
	// if len(req.IfNoneMatch) > 0 {
	//   dreq.Opaque = utils.AppendPlainToOpaque(dreq.Opaque, "if-none-match", strings.Join(req.IfNoneMatch, ","))
	// }

	res, err := client.InitiateFileDownload(ctx, &dreq)
	if err != nil {
		return nil, errtypes.NotFound(dreq.Ref.Path)
	}

	var endpoint string

	for _, proto := range res.GetProtocols() {
		if proto.Protocol == "spaces" {
			endpoint = proto.GetDownloadEndpoint()
			break
		}
	}
	if endpoint == "" {
		return nil, errors.New("metadata storage doesn't support the spaces download protocol")
	}

	hreq, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	for _, etag := range req.IfNoneMatch {
		hreq.Header.Add(net.HeaderIfNoneMatch, etag)
	}

	md, _ := metadata.FromOutgoingContext(ctx)
	hreq.Header.Add(ctxpkg.TokenHeader, md.Get(ctxpkg.TokenHeader)[0])
	resp, err := cs3.dataGatewayClient.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	dres := DownloadResponse{}

	dres.Etag = resp.Header.Get("etag")
	dres.Etag = resp.Header.Get("oc-etag") // takes precedence

	if err := errtypes.NewErrtypeFromHTTPStatusCode(resp.StatusCode, hreq.URL.Path); err != nil {
		return nil, err
	}

	dres.Mtime, err = time.Parse(time.RFC1123Z, resp.Header.Get("last-modified"))
	if err != nil {
		return nil, err
	}

	dres.Content, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &dres, nil
}

// Delete deletes a path
func (cs3 *CS3) Delete(ctx context.Context, path string) error {
	ctx, span := tracer.Start(ctx, "Delete")
	defer span.End()

	client, err := cs3.providerClient()
	if err != nil {
		return err
	}
	ctx, err = cs3.getAuthContext(ctx)
	if err != nil {
		return err
	}

	res, err := client.Delete(ctx, &provider.DeleteRequest{
		Ref: &provider.Reference{
			ResourceId: cs3.SpaceRoot,
			Path:       utils.MakeRelativePath(path),
		},
	})
	if err != nil {
		return err
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return errtypes.NewErrtypeFromStatus(res.Status)
	}

	return nil
}

// ReadDir returns the entries in a given directory
func (cs3 *CS3) ReadDir(ctx context.Context, path string) ([]string, error) {
	ctx, span := tracer.Start(ctx, "ReadDir")
	defer span.End()

	infos, err := cs3.ListDir(ctx, path)
	if err != nil {
		return nil, err
	}

	entries := []string{}
	for _, ri := range infos {
		entries = append(entries, ri.Path)
	}
	return entries, nil
}

// ListDir returns a list of ResourceInfos for the entries in a given directory
func (cs3 *CS3) ListDir(ctx context.Context, path string) ([]*provider.ResourceInfo, error) {
	ctx, span := tracer.Start(ctx, "ListDir")
	defer span.End()

	client, err := cs3.providerClient()
	if err != nil {
		return nil, err
	}
	ctx, err = cs3.getAuthContext(ctx)
	if err != nil {
		return nil, err
	}

	relPath := utils.MakeRelativePath(path)
	res, err := client.ListContainer(ctx, &provider.ListContainerRequest{
		Ref: &provider.Reference{
			ResourceId: cs3.SpaceRoot,
			Path:       relPath,
		},
	})

	if err != nil {
		return nil, err
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return nil, errtypes.NewErrtypeFromStatus(res.Status)
	}

	return res.Infos, nil
}

// MakeDirIfNotExist will create a root node in the metadata storage. Requires an authenticated context.
func (cs3 *CS3) MakeDirIfNotExist(ctx context.Context, folder string) error {
	ctx, span := tracer.Start(ctx, "MakeDirIfNotExist")
	defer span.End()

	client, err := cs3.providerClient()
	if err != nil {
		return err
	}
	ctx, err = cs3.getAuthContext(ctx)
	if err != nil {
		return err
	}

	var rootPathRef = &provider.Reference{
		ResourceId: cs3.SpaceRoot,
		Path:       utils.MakeRelativePath(folder),
	}

	resp, err := client.CreateContainer(ctx, &provider.CreateContainerRequest{
		Ref: rootPathRef,
	})
	switch {
	case err != nil:
		return err
	case resp.Status.Code == rpc.Code_CODE_OK:
		// nothing to do in this case
		return nil
	case resp.Status.Code == rpc.Code_CODE_ALREADY_EXISTS:
		// nothing to do in this case
		return nil
	default:
		return errtypes.NewErrtypeFromStatus(resp.Status)
	}
}

// CreateSymlink creates a symlink
func (cs3 *CS3) CreateSymlink(ctx context.Context, oldname, newname string) error {
	ctx, span := tracer.Start(ctx, "CreateSymlink")
	defer span.End()

	if _, err := cs3.ResolveSymlink(ctx, newname); err == nil {
		return os.ErrExist
	}

	return cs3.SimpleUpload(ctx, newname, []byte(oldname))
}

// ResolveSymlink resolves a symlink
func (cs3 *CS3) ResolveSymlink(ctx context.Context, name string) (string, error) {
	ctx, span := tracer.Start(ctx, "ResolveSymlink")
	defer span.End()

	b, err := cs3.SimpleDownload(ctx, name)
	if err != nil {
		if errors.Is(err, errtypes.NotFound("")) {
			return "", os.ErrNotExist
		}
		return "", err
	}

	return string(b), err
}

func (cs3 *CS3) providerClient() (provider.ProviderAPIClient, error) {
	return pool.GetStorageProviderServiceClient(cs3.providerAddr)
}

func (cs3 *CS3) spacesClient() (provider.SpacesAPIClient, error) {
	return pool.GetSpacesProviderServiceClient(cs3.providerAddr)
}

func (cs3 *CS3) getAuthContext(ctx context.Context) (context.Context, error) {
	if !cs3.useSystemUser {
		return ctx, nil
	}

	// we need to start a new context to get rid of an existing x-access-token in the outgoing context
	authCtx := context.Background()
	authCtx, span := tracer.Start(authCtx, "getAuthContext", trace.WithLinks(trace.LinkFromContext(ctx)))
	defer span.End()

	selector, err := pool.GatewaySelector(cs3.gatewayAddr)
	if err != nil {
		return nil, err
	}
	client, err := selector.Next()
	if err != nil {
		return nil, err
	}

	authCtx = ctxpkg.ContextSetUser(authCtx, cs3.serviceUser)
	authRes, err := client.Authenticate(authCtx, &gateway.AuthenticateRequest{
		Type:         "machine",
		ClientId:     "userid:" + cs3.serviceUser.Id.OpaqueId,
		ClientSecret: cs3.machineAuthAPIKey,
	})
	if err != nil {
		return nil, err
	}
	if authRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return nil, errtypes.NewErrtypeFromStatus(authRes.GetStatus())
	}
	authCtx = metadata.AppendToOutgoingContext(authCtx, ctxpkg.TokenHeader, authRes.Token)
	return authCtx, nil
}
