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

// Package publicstorageprovider provides a CS3 storageprovider implementation for public links.
// It will list spaces with type `grant` and `mountpoint` when a public scope is present.
package publicstorageprovider

import (
	"context"
	"encoding/json"
	"path"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	gstatus "google.golang.org/grpc/status"
)

// name is the Tracer name used to identify this instrumentation library.
const tracerName = "publicstorageprovider"

func init() {
	rgrpc.Register("publicstorageprovider", New)
}

type config struct {
	GatewayAddr string `mapstructure:"gateway_addr"`
}

type service struct {
	conf            *config
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
}

func (s *service) Close() error {
	return nil
}

func (s *service) UnprotectedEndpoints() []string {
	return []string{}
}

func (s *service) Register(ss *grpc.Server) {
	provider.RegisterProviderAPIServer(ss, s)
	provider.RegisterSpacesAPIServer(ss, s)
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

// New creates a new publicstorageprovider service.
func New(m map[string]interface{}, ss *grpc.Server, _ *zerolog.Logger) (rgrpc.Service, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	gatewaySelector, err := pool.GatewaySelector(c.GatewayAddr)
	if err != nil {
		return nil, err
	}

	service := &service{
		conf:            c,
		gatewaySelector: gatewaySelector,
	}

	return service, nil
}

func (s *service) SetArbitraryMetadata(ctx context.Context, req *provider.SetArbitraryMetadataRequest) (*provider.SetArbitraryMetadataResponse, error) {
	ref, _, _, err := s.translatePublicRefToCS3Ref(ctx, req.Ref)
	if err != nil {
		return &provider.SetArbitraryMetadataResponse{
			Status: status.NewStatusFromErrType(ctx, "failed to resolve reference", err),
		}, nil
	}
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	return gatewayClient.SetArbitraryMetadata(ctx, &provider.SetArbitraryMetadataRequest{Opaque: req.Opaque, Ref: ref, ArbitraryMetadata: req.ArbitraryMetadata})
}

func (s *service) UnsetArbitraryMetadata(ctx context.Context, req *provider.UnsetArbitraryMetadataRequest) (*provider.UnsetArbitraryMetadataResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

// SetLock puts a lock on the given reference
func (s *service) SetLock(ctx context.Context, req *provider.SetLockRequest) (*provider.SetLockResponse, error) {
	ref, _, _, err := s.translatePublicRefToCS3Ref(ctx, req.Ref)
	if err != nil {
		return &provider.SetLockResponse{
			Status: status.NewStatusFromErrType(ctx, "failed to resolve reference", err),
		}, nil
	}
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	return gatewayClient.SetLock(ctx, &provider.SetLockRequest{Opaque: req.Opaque, Ref: ref, Lock: req.Lock})
}

// GetLock returns an existing lock on the given reference
func (s *service) GetLock(ctx context.Context, req *provider.GetLockRequest) (*provider.GetLockResponse, error) {
	ref, _, _, err := s.translatePublicRefToCS3Ref(ctx, req.Ref)
	if err != nil {
		return &provider.GetLockResponse{
			Status: status.NewStatusFromErrType(ctx, "failed to resolve reference", err),
		}, nil
	}
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	return gatewayClient.GetLock(ctx, &provider.GetLockRequest{Opaque: req.Opaque, Ref: ref})
}

// RefreshLock refreshes an existing lock on the given reference
func (s *service) RefreshLock(ctx context.Context, req *provider.RefreshLockRequest) (*provider.RefreshLockResponse, error) {
	ref, _, _, err := s.translatePublicRefToCS3Ref(ctx, req.Ref)
	if err != nil {
		return &provider.RefreshLockResponse{
			Status: status.NewStatusFromErrType(ctx, "failed to resolve reference", err),
		}, nil
	}
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	return gatewayClient.RefreshLock(ctx, &provider.RefreshLockRequest{Opaque: req.Opaque, Ref: ref, Lock: req.Lock})
}

// Unlock removes an existing lock from the given reference
func (s *service) Unlock(ctx context.Context, req *provider.UnlockRequest) (*provider.UnlockResponse, error) {
	ref, _, _, err := s.translatePublicRefToCS3Ref(ctx, req.Ref)
	if err != nil {
		return &provider.UnlockResponse{
			Status: status.NewStatusFromErrType(ctx, "failed to resolve reference", err),
		}, nil
	}
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	return gatewayClient.Unlock(ctx, &provider.UnlockRequest{Opaque: req.Opaque, Ref: ref, Lock: req.Lock})
}

func (s *service) InitiateFileDownload(ctx context.Context, req *provider.InitiateFileDownloadRequest) (*provider.InitiateFileDownloadResponse, error) {
	statReq := &provider.StatRequest{Ref: req.Ref}
	statRes, err := s.Stat(ctx, statReq)
	if err != nil {
		return &provider.InitiateFileDownloadResponse{
			Status: status.NewInternal(ctx, "InitiateFileDownload: error stating ref:"+req.Ref.String()),
		}, nil
	}
	if statRes.Status.Code != rpc.Code_CODE_OK {
		if statRes.Status.Code == rpc.Code_CODE_NOT_FOUND {
			return &provider.InitiateFileDownloadResponse{
				Status: status.NewNotFound(ctx, "InitiateFileDownload: file not found"),
			}, nil
		}
		return &provider.InitiateFileDownloadResponse{
			Status: status.NewInternal(ctx, "InitiateFileDownload: error stating ref"),
		}, nil
	}

	req.Opaque = statRes.Info.Opaque
	return s.initiateFileDownload(ctx, req)
}

func (s *service) translatePublicRefToCS3Ref(ctx context.Context, ref *provider.Reference) (*provider.Reference, *provider.ResourceInfo, string, error) {
	log := appctx.GetLogger(ctx)

	info, _, _, token, err := s.extractLinkFromScope(ctx)
	if err != nil {
		return nil, nil, "", err
	}

	var path string
	switch info.Type {
	case provider.ResourceType_RESOURCE_TYPE_CONTAINER:
		// folders point to the folder -> path needs to be added
		path = utils.MakeRelativePath(ref.Path)
	case provider.ResourceType_RESOURCE_TYPE_FILE:
		// files already point to the correct id
		path = "."
	default:
		// TODO: can this happen?
		// path = utils.MakeRelativePath(relativePath)
	}

	cs3Ref := &provider.Reference{
		ResourceId: info.Id,
		Path:       path,
	}

	log.Debug().
		Interface("sourceRef", ref).
		Interface("cs3Ref", cs3Ref).
		Str("tkn", token).
		Str("originalPath", info.Path).
		Str("relativePath", path).
		Msg("translatePublicRefToCS3Ref")
	return cs3Ref, info, token, nil
}

func (s *service) initiateFileDownload(ctx context.Context, req *provider.InitiateFileDownloadRequest) (*provider.InitiateFileDownloadResponse, error) {
	ref, info, _, err := s.translatePublicRefToCS3Ref(ctx, req.Ref)
	switch {
	case err != nil:
		return &provider.InitiateFileDownloadResponse{
			Status: status.NewStatusFromErrType(ctx, "failed to resolve reference", err),
		}, nil
	case info.PermissionSet == nil || !info.PermissionSet.InitiateFileDownload:
		return &provider.InitiateFileDownloadResponse{
			Status: status.NewPermissionDenied(ctx, nil, "share does not grant InitiateFileDownload permission"),
		}, nil
	}
	dReq := &provider.InitiateFileDownloadRequest{
		Ref: ref,
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	dRes, err := gatewayClient.InitiateFileDownload(ctx, dReq)
	if err != nil {
		return &provider.InitiateFileDownloadResponse{
			Status: status.NewInternal(ctx, "initiateFileDownload: error calling InitiateFileDownload"),
		}, nil
	}

	if dRes.Status.Code != rpc.Code_CODE_OK {
		return &provider.InitiateFileDownloadResponse{
			Status: dRes.Status,
		}, nil
	}

	protocols := make([]*provider.FileDownloadProtocol, len(dRes.Protocols))
	for p := range dRes.Protocols {
		if !strings.HasSuffix(dRes.Protocols[p].DownloadEndpoint, "/") {
			dRes.Protocols[p].DownloadEndpoint += "/"
		}
		dRes.Protocols[p].DownloadEndpoint += dRes.Protocols[p].Token

		protocols = append(protocols, &provider.FileDownloadProtocol{
			Opaque:           dRes.Protocols[p].Opaque,
			Protocol:         dRes.Protocols[p].Protocol,
			DownloadEndpoint: dRes.Protocols[p].DownloadEndpoint,
			Expose:           true, // the gateway already has encoded the upload endpoint
		})
	}

	return &provider.InitiateFileDownloadResponse{
		Opaque:    dRes.GetOpaque(),
		Status:    dRes.Status,
		Protocols: protocols,
	}, nil
}

func (s *service) InitiateFileUpload(ctx context.Context, req *provider.InitiateFileUploadRequest) (*provider.InitiateFileUploadResponse, error) {
	cs3Ref, info, _, err := s.translatePublicRefToCS3Ref(ctx, req.Ref)
	switch {
	case err != nil:
		return &provider.InitiateFileUploadResponse{
			Status: status.NewStatusFromErrType(ctx, "failed to resolve reference", err),
		}, nil
	case info.PermissionSet == nil || !info.PermissionSet.InitiateFileUpload:
		return &provider.InitiateFileUploadResponse{
			Status: status.NewPermissionDenied(ctx, nil, "share does not grant InitiateFileUpload permission"),
		}, nil
	}
	uReq := &provider.InitiateFileUploadRequest{
		Ref:    cs3Ref,
		Opaque: req.Opaque,
		LockId: req.LockId,
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	uRes, err := gatewayClient.InitiateFileUpload(ctx, uReq)
	if err != nil {
		return &provider.InitiateFileUploadResponse{
			Status: status.NewInternal(ctx, "InitiateFileUpload: error calling InitiateFileUpload"),
		}, nil
	}

	if uRes.Status.Code != rpc.Code_CODE_OK {
		return &provider.InitiateFileUploadResponse{
			Status: uRes.Status,
		}, nil
	}

	protocols := make([]*provider.FileUploadProtocol, len(uRes.Protocols))
	for p := range uRes.Protocols {
		if !strings.HasSuffix(uRes.Protocols[p].UploadEndpoint, "/") {
			uRes.Protocols[p].UploadEndpoint += "/"
		}
		uRes.Protocols[p].UploadEndpoint += uRes.Protocols[p].Token

		protocols = append(protocols, &provider.FileUploadProtocol{
			Opaque:             uRes.Protocols[p].Opaque,
			Protocol:           uRes.Protocols[p].Protocol,
			UploadEndpoint:     uRes.Protocols[p].UploadEndpoint,
			AvailableChecksums: uRes.Protocols[p].AvailableChecksums,
			Expose:             true, // the gateway already has encoded the upload endpoint
		})
	}

	res := &provider.InitiateFileUploadResponse{
		Opaque:    uRes.GetOpaque(),
		Status:    uRes.Status,
		Protocols: protocols,
	}

	return res, nil
}

func (s *service) GetPath(ctx context.Context, req *provider.GetPathRequest) (*provider.GetPathResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) GetHome(ctx context.Context, req *provider.GetHomeRequest) (*provider.GetHomeResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) CreateHome(ctx context.Context, req *provider.CreateHomeRequest) (*provider.CreateHomeResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

// ListStorageSpaces returns storage spaces when a public scope is present
// in the context.
//
// On the one hand, it lists a `mountpoint` space that can be used by the
// registry to construct a mount path. These spaces have their root
// storageid set to 7993447f-687f-490d-875c-ac95e89a62a4 and the
// opaqueid set to the link token.
//
// On the other hand, it lists a `grant` space for the shared resource id,
// so id based requests can find the correct storage provider. These spaces
// have their root set to the shared resource.
func (s *service) ListStorageSpaces(ctx context.Context, req *provider.ListStorageSpacesRequest) (*provider.ListStorageSpacesResponse, error) {
	spaceTypes := map[string]struct{}{}
	var exists = struct{}{}
	appendTypes := []string{}
	var spaceID *provider.ResourceId
	for _, f := range req.Filters {
		switch f.Type {
		case provider.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE:
			spaceType := f.GetSpaceType()
			if spaceType == "+mountpoint" || spaceType == "+grant" {
				appendTypes = append(appendTypes, strings.TrimPrefix(spaceType, "+"))
				continue
			}
			spaceTypes[spaceType] = exists
		case provider.ListStorageSpacesRequest_Filter_TYPE_ID:
			resID, err := storagespace.ParseID(f.GetId().GetOpaqueId())
			if err != nil {
				return &provider.ListStorageSpacesResponse{
					Status: &rpc.Status{Code: rpc.Code_CODE_INVALID_ARGUMENT, Message: err.Error()},
				}, nil
			}
			if resID.SpaceId != utils.PublicStorageSpaceID && resID.SpaceId != utils.OCMStorageSpaceID {
				return &provider.ListStorageSpacesResponse{
					// a specific id was requested, return not found instead of empty list
					Status: &rpc.Status{Code: rpc.Code_CODE_NOT_FOUND},
				}, nil
			}
			spaceID = &resID
		}
	}

	info, share, grantee, token, err := s.extractLinkFromScope(ctx)
	if err != nil {
		switch err.(type) {
		case errtypes.NotFound:
			// if there is no public scope there are no publicstorage spaces
			return &provider.ListStorageSpacesResponse{
				Status: &rpc.Status{Code: rpc.Code_CODE_OK},
			}, nil
		default:
			return &provider.ListStorageSpacesResponse{
				Status: &rpc.Status{Code: rpc.Code_CODE_INTERNAL},
			}, nil
		}
	}

	if len(spaceTypes) == 0 {
		spaceTypes["mountpoint"] = exists
	}
	for _, s := range appendTypes {
		spaceTypes[s] = exists
	}

	res := &provider.ListStorageSpacesResponse{
		Status: status.NewOK(ctx),
	}
	for k := range spaceTypes {
		switch k {
		case "grant":
			// when a list storage space with the resourceid of an external
			// resource is made we may have a grant for it
			root := info.Id
			if spaceID != nil && !utils.ResourceIDEqual(spaceID, root) {
				// none of our business
				continue
			}
			// we know a grant for this resource
			space := &provider.StorageSpace{
				Id: &provider.StorageSpaceId{
					OpaqueId: storagespace.FormatResourceID(root),
				},
				SpaceType: "grant",
				Owner:     &userv1beta1.User{Id: grantee},
				// the publicstorageprovider keeps track of mount points
				Root: root,
			}

			res.StorageSpaces = append(res.StorageSpaces, space)
		case "mountpoint":
			root := &provider.ResourceId{
				StorageId: utils.PublicStorageProviderID,
				SpaceId:   utils.PublicStorageSpaceID,
				OpaqueId:  token, // the link share has no id, only the token
			}
			if ocmShare, ok := share.(*ocm.Share); ok {
				root.OpaqueId = ocmShare.GetId().GetOpaqueId()
			}
			if spaceID != nil {
				switch {
				case utils.ResourceIDEqual(spaceID, root):
					// we have a virtual node
				case utils.ResourceIDEqual(spaceID, info.Id):
					// we have a mount point
					root = info.Id
				default:
					// none of our business
					continue
				}
			}
			space := &provider.StorageSpace{
				Id: &provider.StorageSpaceId{
					OpaqueId: storagespace.FormatResourceID(root),
				},
				SpaceType: "mountpoint",
				Owner:     &userv1beta1.User{Id: grantee}, // FIXME actually, the mount point belongs to no one?
				// the publicstorageprovider keeps track of mount points
				Root: root,
			}

			res.StorageSpaces = append(res.StorageSpaces, space)
		}
	}
	return res, nil
}

func (s *service) extractLinkFromScope(ctx context.Context) (*provider.ResourceInfo, interface{}, *userv1beta1.UserId, string, error) {
	scopes, ok := ctxpkg.ContextGetScopes(ctx)
	if !ok {
		return nil, nil, nil, "", errtypes.NotFound("No scopes found in context")
	}
	for k, v := range scopes {
		if strings.HasPrefix(k, "ocmshare:") && v.Resource.Decoder == "json" {
			share := &ocm.Share{}
			err := utils.UnmarshalJSONToProtoV1(v.Resource.Value, share)
			if err != nil {
				return nil, nil, nil, "", errtypes.InternalError("failed to unmarshal ocm share")
			}

			// the share is minimally populated, we need more than the token
			// look up complete share
			info, resolvedShare, err := s.resolveToken(ctx, share)
			if err != nil {
				return nil, nil, nil, "", err
			}
			return info, resolvedShare, share.Owner, share.Token, nil
		} else if strings.HasPrefix(k, "publicshare:") && v.Resource.Decoder == "json" {
			share := &link.PublicShare{}
			err := utils.UnmarshalJSONToProtoV1(v.Resource.Value, share)
			if err != nil {
				return nil, nil, nil, "", errtypes.InternalError("failed to unmarshal public share")
			}

			// the share is minimally populated, we need more than the token
			// look up complete share
			info, resolvedShare, err := s.resolveToken(ctx, share)
			if err != nil {
				return nil, nil, nil, "", err
			}
			return info, resolvedShare, share.Owner, share.Token, nil
		}
	}
	return nil, nil, nil, "", errtypes.NotFound("No public storage info found in scopes")
}
func (s *service) UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) (*provider.DeleteStorageSpaceResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) CreateContainer(ctx context.Context, req *provider.CreateContainerRequest) (*provider.CreateContainerResponse, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "CreateContainer")
	defer span.End()

	span.SetAttributes(attribute.KeyValue{
		Key:   "reference",
		Value: attribute.StringValue(req.Ref.String()),
	})

	cs3Ref, info, _, err := s.translatePublicRefToCS3Ref(ctx, req.Ref)
	switch {
	case err != nil:
		return &provider.CreateContainerResponse{
			Status: status.NewStatusFromErrType(ctx, "failed to resolve reference", err),
		}, nil
	case info.PermissionSet == nil || !info.PermissionSet.CreateContainer:
		return &provider.CreateContainerResponse{
			Status: status.NewPermissionDenied(ctx, nil, "share does not grant CreateContainer permission"),
		}, nil
	}

	var res *provider.CreateContainerResponse
	// the call has to be made to the gateway instead of the storage.
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	res, err = gatewayClient.CreateContainer(ctx, &provider.CreateContainerRequest{
		Ref: cs3Ref,
	})
	if err != nil {
		return &provider.CreateContainerResponse{
			Status: status.NewInternal(ctx, "createContainer: error calling CreateContainer for ref:"+req.Ref.String()),
		}, nil
	}
	if res.Status.Code == rpc.Code_CODE_INTERNAL {
		return res, nil
	}

	return res, nil
}

func (s *service) TouchFile(ctx context.Context, req *provider.TouchFileRequest) (*provider.TouchFileResponse, error) {
	ref, _, _, err := s.translatePublicRefToCS3Ref(ctx, req.Ref)
	if err != nil {
		return &provider.TouchFileResponse{
			Status: status.NewStatusFromErrType(ctx, "failed to resolve reference", err),
		}, nil
	}
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	return gatewayClient.TouchFile(ctx, &provider.TouchFileRequest{Opaque: req.Opaque, Ref: ref})
}

func (s *service) Delete(ctx context.Context, req *provider.DeleteRequest) (*provider.DeleteResponse, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Delete")
	defer span.End()

	span.SetAttributes(attribute.KeyValue{
		Key:   "reference",
		Value: attribute.StringValue(req.Ref.String()),
	})

	cs3Ref, info, _, err := s.translatePublicRefToCS3Ref(ctx, req.Ref)
	switch {
	case err != nil:
		return &provider.DeleteResponse{
			Status: status.NewStatusFromErrType(ctx, "failed to resolve reference", err),
		}, nil
	case info.PermissionSet == nil || !info.PermissionSet.Delete:
		return &provider.DeleteResponse{
			Status: status.NewPermissionDenied(ctx, nil, "share does not grant Delete permission"),
		}, nil
	}

	var res *provider.DeleteResponse
	// the call has to be made to the gateway instead of the storage.
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	res, err = gatewayClient.Delete(ctx, &provider.DeleteRequest{
		Ref: cs3Ref,
	})
	if err != nil {
		return &provider.DeleteResponse{
			Status: status.NewInternal(ctx, "Delete: error calling Delete for ref:"+req.Ref.String()),
		}, nil
	}
	if res.Status.Code == rpc.Code_CODE_INTERNAL {
		return res, nil
	}

	return res, nil
}

func (s *service) Move(ctx context.Context, req *provider.MoveRequest) (*provider.MoveResponse, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Move")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "source",
			Value: attribute.StringValue(req.Source.String()),
		},
		attribute.KeyValue{
			Key:   "destination",
			Value: attribute.StringValue(req.Destination.String()),
		},
	)

	cs3RefSource, info, tknSource, err := s.translatePublicRefToCS3Ref(ctx, req.Source)
	switch {
	case err != nil:
		return &provider.MoveResponse{
			Status: status.NewStatusFromErrType(ctx, "failed to resolve source reference", err),
		}, nil
	case info.PermissionSet == nil || !info.PermissionSet.Move:
		return &provider.MoveResponse{
			Status: status.NewPermissionDenied(ctx, nil, "share does not grant Move permission"),
		}, nil
	}
	// FIXME: maybe there's a shortcut possible here using the source path
	cs3RefDestination, _, tknDest, err := s.translatePublicRefToCS3Ref(ctx, req.Destination)
	if err != nil {
		return &provider.MoveResponse{
			Status: status.NewStatusFromErrType(ctx, "failed to resolve destination reference", err),
		}, nil
	}

	if tknSource != tknDest {
		return &provider.MoveResponse{
			Status: status.NewInvalid(ctx, "Source and destination token must be the same"),
		}, nil
	}

	var res *provider.MoveResponse
	// the call has to be made to the gateway instead of the storage.
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	res, err = gatewayClient.Move(ctx, &provider.MoveRequest{
		Source:      cs3RefSource,
		Destination: cs3RefDestination,
	})
	if err != nil {
		return &provider.MoveResponse{
			Status: status.NewInternal(ctx, "Move: error calling Move for source ref "+req.Source.String()+" to destination ref "+req.Destination.String()),
		}, nil
	}
	if res.Status.Code == rpc.Code_CODE_INTERNAL {
		return res, nil
	}

	return res, nil
}

func (s *service) Stat(ctx context.Context, req *provider.StatRequest) (*provider.StatResponse, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Stat")
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "source",
			Value: attribute.StringValue(req.Ref.String()),
		})

	info, share, _, token, err := s.extractLinkFromScope(ctx)
	if err != nil {
		switch err.(type) {
		case errtypes.NotFound:
			return &provider.StatResponse{
				Status: status.NewNotFound(ctx, "share or token not found"),
			}, nil
		default:
			return &provider.StatResponse{
				Status: status.NewInternal(ctx, "share or token not found"),
			}, nil
		}
	}

	if info.Type == provider.ResourceType_RESOURCE_TYPE_FILE || req.Ref.Path == "" {
		res := &provider.StatResponse{
			Status: status.NewOK(ctx),
			Info:   info,
		}
		s.augmentStatResponse(ctx, res.Info, info, share, token)
		return res, nil
	}

	ref := &provider.Reference{
		ResourceId: info.Id,
		Path:       utils.MakeRelativePath(req.Ref.Path),
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	statResponse, err := gatewayClient.Stat(ctx, &provider.StatRequest{Ref: ref})
	if err != nil {
		return &provider.StatResponse{
			Status: status.NewInternal(ctx, "Stat: error calling Stat for ref:"+req.Ref.String()),
		}, nil
	}

	s.augmentStatResponse(ctx, statResponse.Info, info, share, token)

	return statResponse, nil
}

func (s *service) augmentStatResponse(ctx context.Context, statInfo *provider.ResourceInfo, shareInfo *provider.ResourceInfo, share interface{}, tkn string) {
	// prevent leaking internal paths
	if statInfo != nil {
		if err := addShare(statInfo, share); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Interface("share", share).Interface("info", statInfo).Msg("error when adding share")
		}

		var sharePath string
		if shareInfo.Type == provider.ResourceType_RESOURCE_TYPE_FILE {
			sharePath = path.Base(shareInfo.Path)
		} else {
			sharePath = strings.TrimPrefix(statInfo.Path, shareInfo.Path)
		}

		statInfo.Path = path.Join("/", sharePath)
		filterPermissions(statInfo.PermissionSet, shareInfo.PermissionSet)
	}
}

func addShare(i *provider.ResourceInfo, share interface{}) error {
	if i.Opaque == nil {
		i.Opaque = &typesv1beta1.Opaque{}
	}
	if i.Opaque.Map == nil {
		i.Opaque.Map = map[string]*typesv1beta1.OpaqueEntry{}
	}
	val, err := json.Marshal(share)
	if err != nil {
		return err
	}
	i.Opaque.Map["link-share"] = &typesv1beta1.OpaqueEntry{Decoder: "json", Value: val}
	return nil
}

func (s *service) ListContainerStream(req *provider.ListContainerStreamRequest, ss provider.ProviderAPI_ListContainerStreamServer) error {
	return gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) ListContainer(ctx context.Context, req *provider.ListContainerRequest) (*provider.ListContainerResponse, error) {
	info, share, _, _, err := s.extractLinkFromScope(ctx)
	if err != nil {
		switch err.(type) {
		case errtypes.NotFound:
			return &provider.ListContainerResponse{
				Status: status.NewNotFound(ctx, "share or token not found"),
			}, nil
		default:
			return &provider.ListContainerResponse{
				Status: status.NewInternal(ctx, "share or token not found"),
			}, nil
		}
	}
	if info.PermissionSet == nil || !info.PermissionSet.ListContainer {
		return &provider.ListContainerResponse{
			Status: status.NewPermissionDenied(ctx, nil, "share does not grant ListContainer permission"),
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	listContainerR, err := gatewayClient.ListContainer(
		ctx,
		&provider.ListContainerRequest{
			Ref: &provider.Reference{
				ResourceId: info.Id,
				// prefix relative path with './' to make it a CS3 relative path
				Path: utils.MakeRelativePath(req.Ref.Path),
			},
		},
	)
	if err != nil {
		return &provider.ListContainerResponse{
			Status: status.NewInternal(ctx, "ListContainer: error calling ListContainer for ref:"+req.Ref.String()),
		}, nil
	}

	for i := range listContainerR.Infos {
		// FIXME how do we reduce permissions to what is granted by the public link?
		// only a problem for id based access -> middleware
		filterPermissions(listContainerR.Infos[i].PermissionSet, info.PermissionSet)
		if err := addShare(listContainerR.Infos[i], share); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Interface("share", share).Interface("info", listContainerR.Infos[i]).Msg("error when adding share")
		}
	}

	return listContainerR, nil
}

func filterPermissions(l *provider.ResourcePermissions, r *provider.ResourcePermissions) {
	l.AddGrant = l.AddGrant && r.AddGrant
	l.CreateContainer = l.CreateContainer && r.CreateContainer
	l.Delete = l.Delete && r.Delete
	l.GetPath = l.GetPath && r.GetPath
	l.GetQuota = l.GetQuota && r.GetQuota
	l.InitiateFileDownload = l.InitiateFileDownload && r.InitiateFileDownload
	l.InitiateFileUpload = l.InitiateFileUpload && r.InitiateFileUpload
	l.ListContainer = l.ListContainer && r.ListContainer
	l.ListFileVersions = l.ListFileVersions && r.ListFileVersions
	l.ListGrants = l.ListGrants && r.ListGrants
	l.ListRecycle = l.ListRecycle && r.ListRecycle
	l.Move = l.Move && r.Move
	l.PurgeRecycle = l.PurgeRecycle && r.PurgeRecycle
	l.RemoveGrant = l.RemoveGrant && r.RemoveGrant
	l.RestoreFileVersion = l.RestoreFileVersion && r.RestoreFileVersion
	l.RestoreRecycleItem = l.RestoreRecycleItem && r.RestoreRecycleItem
	l.Stat = l.Stat && r.Stat
	l.UpdateGrant = l.UpdateGrant && r.UpdateGrant
}

func (s *service) ListFileVersions(ctx context.Context, req *provider.ListFileVersionsRequest) (*provider.ListFileVersionsResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) RestoreFileVersion(ctx context.Context, req *provider.RestoreFileVersionRequest) (*provider.RestoreFileVersionResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) ListRecycleStream(req *provider.ListRecycleStreamRequest, ss provider.ProviderAPI_ListRecycleStreamServer) error {
	return gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) ListRecycle(ctx context.Context, req *provider.ListRecycleRequest) (*provider.ListRecycleResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) RestoreRecycleItem(ctx context.Context, req *provider.RestoreRecycleItemRequest) (*provider.RestoreRecycleItemResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) PurgeRecycle(ctx context.Context, req *provider.PurgeRecycleRequest) (*provider.PurgeRecycleResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) ListGrants(ctx context.Context, req *provider.ListGrantsRequest) (*provider.ListGrantsResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) AddGrant(ctx context.Context, req *provider.AddGrantRequest) (*provider.AddGrantResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) DenyGrant(ctx context.Context, req *provider.DenyGrantRequest) (*provider.DenyGrantResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) CreateReference(ctx context.Context, req *provider.CreateReferenceRequest) (*provider.CreateReferenceResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) CreateSymlink(ctx context.Context, req *provider.CreateSymlinkRequest) (*provider.CreateSymlinkResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) UpdateGrant(ctx context.Context, req *provider.UpdateGrantRequest) (*provider.UpdateGrantResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) RemoveGrant(ctx context.Context, req *provider.RemoveGrantRequest) (*provider.RemoveGrantResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) GetQuota(ctx context.Context, req *provider.GetQuotaRequest) (*provider.GetQuotaResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) AddLabel(ctx context.Context, req *provider.AddLabelRequest) (*provider.AddLabelResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) RemoveLabel(ctx context.Context, req *provider.RemoveLabelRequest) (*provider.RemoveLabelResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

// resolveToken returns the resource info for the publicly shared resource.
func (s *service) resolveToken(ctx context.Context, share interface{}) (*provider.ResourceInfo, interface{}, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, nil, err
	}

	resourceID := &provider.ResourceId{}
	perms := &provider.ResourcePermissions{}
	var resolvedShare interface{}
	switch v := share.(type) {
	case *link.PublicShare:
		publicShareResponse, err := gatewayClient.GetPublicShare(
			ctx,
			&link.GetPublicShareRequest{
				Ref: &link.PublicShareReference{
					Spec: &link.PublicShareReference_Token{
						Token: v.Token,
					},
				},
				Sign: true,
			},
		)
		switch {
		case err != nil:
			return nil, nil, err
		case publicShareResponse.Status.Code != rpc.Code_CODE_OK:
			return nil, nil, errtypes.NewErrtypeFromStatus(publicShareResponse.Status)
		}
		resolvedShare = publicShareResponse.GetShare()
		resourceID = publicShareResponse.GetShare().GetResourceId()
		perms = publicShareResponse.GetShare().GetPermissions().GetPermissions()
	case *ocm.Share:
		gsr, err := gatewayClient.GetOCMShareByToken(ctx, &ocm.GetOCMShareByTokenRequest{
			Token: v.Token,
		})
		switch {
		case err != nil:
			return nil, nil, err
		case gsr.Status.Code != rpc.Code_CODE_OK:
			return nil, nil, errtypes.NewErrtypeFromStatus(gsr.Status)
		}
		accessMethods := gsr.GetShare().GetAccessMethods()
		if len(accessMethods) == 0 {
			return nil, nil, errtypes.PermissionDenied("failed to get access to the requested resource")
		}
		resolvedShare = gsr.GetShare()
		resourceID = gsr.GetShare().GetResourceId()
		perms = accessMethods[0].GetWebdavOptions().Permissions
	}

	sRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{
		Ref: &provider.Reference{
			ResourceId: resourceID,
		},
	})
	switch {
	case err != nil:
		return nil, nil, err
	case sRes.Status.Code != rpc.Code_CODE_OK:
		return nil, nil, errtypes.NewErrtypeFromStatus(sRes.Status)
	}

	// Set permissions
	sRes.Info.PermissionSet = perms
	return sRes.Info, resolvedShare, nil
}
