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

package gateway

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaborationv1beta1 "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	linkv1beta1 "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	registry "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"

	"github.com/golang-jwt/jwt/v5"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	sdk "github.com/opencloud-eu/reva/v2/pkg/sdk/common"
	"github.com/opencloud-eu/reva/v2/pkg/share"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	gstatus "google.golang.org/grpc/status"
)

/*  About caching
    The gateway is doing a lot of requests to look up the responsible storage providers for a reference.
    - when the reference uses an id we can use a global id -> provider cache because it is the same for all users
    - when the reference is an absolute path we
   	 - 1. look up the corresponding space in the space registry
     - 2. can reuse the global id -> provider cache to look up the provider
	 - paths are unique per user: when a rule mounts shares at /shares/{{.Space.Name}}
	   the path /shares/Documents might show different content for einstein than for marie
	   -> path -> spaceid lookup needs a per user cache
	When can we invalidate?
	- the global cache needs to be invalidated when the provider for a space id changes.
		- happens when a space is moved from one provider to another. Not yet implemented
		-> should be good enough to use a TTL. daily should be good enough
	- the user individual file cache is actually a cache of the mount points
	    - we could do a registry.ListProviders (for user) on startup to warm up the cache ...
		- when a share is granted or removed we need to invalidate that path
		- when a share is renamed we need to invalidate the path
		- we can use a ttl for all paths?
		- the findProviders func in the gateway needs to look up in the user cache first
	We want to cache the root etag of spaces
	    - can be invalidated on every write or delete with fallback via TTL?
*/

// transferClaims are custom claims for a JWT token to be used between the metadata and data gateways.
type transferClaims struct {
	jwt.RegisteredClaims
	Target string `json:"target"`
}

func (s *svc) sign(_ context.Context, target string, expiresAt int64) (string, error) {
	// Tus sends a separate request to the datagateway service for every chunk.
	// For large files, this can take a long time, so we extend the expiration
	claims := transferClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(expiresAt, 0)),
			Audience:  jwt.ClaimStrings{"reva"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Target: target,
	}

	t := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), claims)

	tkn, err := t.SignedString([]byte(s.c.TransferSharedSecret))
	if err != nil {
		return "", errors.Wrapf(err, "error signing token with claims %+v", claims)
	}

	return tkn, nil
}

func (s *svc) CreateHome(ctx context.Context, req *provider.CreateHomeRequest) (*provider.CreateHomeResponse, error) {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		return &provider.CreateHomeResponse{
			Status: status.NewPermissionDenied(ctx, nil, "can't create home for anonymous user"),
		}, nil

	}
	quotaStr := utils.ReadPlainFromOpaque(req.Opaque, "quota")
	var quota *provider.Quota
	if quotaStr != "" {
		q, err := strconv.ParseUint(quotaStr, 10, 64)
		if err != nil {
			return &provider.CreateHomeResponse{
				Status: status.NewInvalid(ctx, fmt.Sprintf("can't parse quotaStr: %s", quotaStr)),
			}, nil
		}
		quota = &provider.Quota{
			QuotaMaxBytes: q,
		}
	}
	createReq := &provider.CreateStorageSpaceRequest{
		Type:  "personal",
		Owner: u,
		Name:  u.DisplayName,
		Quota: quota,
	}

	// send the user id as the space id, makes debugging easier
	if u.Id != nil && u.Id.OpaqueId != "" {
		createReq.Opaque = &typesv1beta1.Opaque{
			Map: map[string]*typesv1beta1.OpaqueEntry{
				"space_id": {
					Decoder: "plain",
					Value:   []byte(u.Id.OpaqueId),
				},
			},
		}
	}
	res, err := s.CreateStorageSpace(ctx, createReq)
	if err != nil {
		return &provider.CreateHomeResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call CreateStorageSpace", err),
		}, nil
	}
	return &provider.CreateHomeResponse{
		Opaque: res.Opaque,
		Status: res.Status,
	}, nil
}

func (s *svc) CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error) {
	// TODO change the CreateStorageSpaceRequest to contain a space instead of sending individual properties
	space := &provider.StorageSpace{
		Owner:     req.Owner,
		SpaceType: req.Type,
		Name:      req.Name,
		Quota:     req.Quota,
	}

	if req.Opaque != nil && req.Opaque.Map != nil && req.Opaque.Map["id"] != nil {
		if req.Opaque.Map["space_id"].Decoder == "plain" {
			space.Id = &provider.StorageSpaceId{OpaqueId: string(req.Opaque.Map["id"].Value)}
		}
	}

	srClient, err := s.getStorageRegistryClient(ctx, s.c.StorageRegistryEndpoint)
	if err != nil {
		return &provider.CreateStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could get storage registry client", err),
		}, nil
	}

	spaceJSON, err := json.Marshal(space)
	if err != nil {
		return &provider.CreateStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not marshal space json", err),
		}, nil
	}

	// The registry is responsible for choosing the right provider
	res, err := srClient.GetStorageProviders(ctx, &registry.GetStorageProvidersRequest{
		Opaque: &typesv1beta1.Opaque{
			Map: map[string]*typesv1beta1.OpaqueEntry{
				"space": {
					Decoder: "json",
					Value:   spaceJSON,
				},
			},
		},
	})
	if err != nil {
		return &provider.CreateStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call GetStorageProviders", err),
		}, nil
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return &provider.CreateStorageSpaceResponse{
			Status: res.Status,
		}, nil
	}

	if len(res.Providers) == 0 {
		return &provider.CreateStorageSpaceResponse{
			Status: status.NewNotFound(ctx, fmt.Sprintf("gateway found no provider for space %+v", space)),
		}, nil
	}

	// just pick the first provider, we expect only one
	c, err := s.getSpacesProviderClient(ctx, res.Providers[0])
	if err != nil {
		return &provider.CreateStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not get storage provider client", err),
		}, nil
	}
	createRes, err := c.CreateStorageSpace(ctx, req)
	if err != nil {
		return &provider.CreateStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call CreateStorageSpace", err),
		}, nil
	}

	// If the create space is not a "personal" space, we add a Share to the ShareProvider to give the creator
	// of the space "manager" access to it.
	// Note: the DecomposedFS storagadriver already adds a grant to the space root giving that access, the "CreateShare"
	//       here is happeing so that that grant is also visible when listing permission (shares) on the spaceroot.
	if req.GetType() != "personal" && createRes.GetStatus().GetCode() == rpc.Code_CODE_OK {
		rollbackFn := func() {
			dsRes, dsErr := s.DeleteStorageSpace(ctx, &provider.DeleteStorageSpaceRequest{
				Id: createRes.GetStorageSpace().GetId(),
			})
			if dsErr != nil || dsRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
				log.Error().Err(dsErr).Interface("status", dsRes.GetStatus()).Interface("space_id", createRes.GetStorageSpace().GetId()).Msg("failed to delete space during rollback")
			}
		}

		u := ctxpkg.ContextMustGetUser(ctx)
		shareReq := &collaborationv1beta1.CreateShareRequest{
			ResourceInfo: &provider.ResourceInfo{
				Id: createRes.GetStorageSpace().GetRoot(),
			},
			Grant: &collaborationv1beta1.ShareGrant{
				Grantee: &provider.Grantee{
					Type: provider.GranteeType_GRANTEE_TYPE_USER,
					Id: &provider.Grantee_UserId{
						UserId: u.GetId(),
					},
				},
				Permissions: &collaborationv1beta1.SharePermissions{
					Permissions: conversions.NewManagerRole().CS3ResourcePermissions(),
				},
			},
		}
		csResp, err := s.CreateShare(ctx, shareReq)
		if err != nil || csResp.GetStatus().GetCode() != rpc.Code_CODE_OK {
			log.Error().Err(err).Interface("status", csResp.GetStatus()).Interface("space_id", createRes.GetStorageSpace().GetId()).Msg("failed to create initial share for space creator, rolling back space creation")
			rollbackFn()

			shareErrMsg := "failed to create initial share for space creator"
			if err == nil {
				shareErrMsg = csResp.GetStatus().GetMessage()
			}
			return &provider.CreateStorageSpaceResponse{
				Status: status.NewInternal(ctx, shareErrMsg),
			}, nil
		}
	}

	return createRes, nil
}

func (s *svc) ListStorageSpaces(ctx context.Context, req *provider.ListStorageSpacesRequest) (*provider.ListStorageSpacesResponse, error) {
	// TODO update CS3 api to forward the filters to the registry so it can filter the number of providers the gateway needs to query
	filters := map[string]string{
		// TODO add opaque / CS3 api to expand 'path,root,stat?' properties / field mask
		"mask": "*", // fetch all properties when listing storage spaces
	}

	mask := utils.ReadPlainFromOpaque(req.Opaque, "mask")
	if mask != "" {
		// TODO check for allowed filters
		filters["mask"] = mask
	}
	path := utils.ReadPlainFromOpaque(req.Opaque, "path")
	if path != "" {
		// TODO check for allowed filters
		filters["path"] = path
	}

	for _, f := range req.Filters {
		switch f.Type {
		case provider.ListStorageSpacesRequest_Filter_TYPE_ID:
			sid, spid, oid, err := storagespace.SplitID(f.GetId().GetOpaqueId())
			if err != nil {
				continue
			}
			filters["storage_id"], filters["space_id"], filters["opaque_id"] = sid, spid, oid
		case provider.ListStorageSpacesRequest_Filter_TYPE_OWNER:
			filters["owner_idp"] = f.GetOwner().GetIdp()
			filters["owner_id"] = f.GetOwner().GetOpaqueId()
		case provider.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE:
			filters["space_type"] = f.GetSpaceType()
		case provider.ListStorageSpacesRequest_Filter_TYPE_USER:
			filters["user_idp"] = f.GetUser().GetIdp()
			filters["user_id"] = f.GetUser().GetOpaqueId()
		default:
			return &provider.ListStorageSpacesResponse{
				Status: status.NewInvalid(ctx, fmt.Sprintf("unknown filter %v", f.Type)),
			}, nil
		}
	}

	c, err := s.getStorageRegistryClient(ctx, s.c.StorageRegistryEndpoint)
	if err != nil {
		return &provider.ListStorageSpacesResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not get storage registry client", err),
		}, nil
	}

	listReq := &registry.ListStorageProvidersRequest{Opaque: req.Opaque}
	if listReq.Opaque == nil {
		listReq.Opaque = &typesv1beta1.Opaque{}
	}
	if len(filters) > 0 {
		sdk.EncodeOpaqueMap(listReq.Opaque, filters)
	}
	res, err := c.ListStorageProviders(ctx, listReq)
	if err != nil {
		return &provider.ListStorageSpacesResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call ListStorageSpaces", err),
		}, nil
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return &provider.ListStorageSpacesResponse{
			Status: res.Status,
		}, nil
	}

	spaces := []*provider.StorageSpace{}
	for _, providerInfo := range res.Providers {
		spaces = append(spaces, decodeSpaces(providerInfo)...)
	}

	return &provider.ListStorageSpacesResponse{
		Status:        status.NewOK(ctx),
		StorageSpaces: spaces,
	}, nil
}

func (s *svc) UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error) {
	// TODO: needs to be fixed
	ref := &provider.Reference{ResourceId: req.StorageSpace.Root}
	c, _, err := s.findSpacesProvider(ctx, ref)
	if err != nil {
		return &provider.UpdateStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find reference %+v", ref), err),
		}, nil
	}

	res, err := c.UpdateStorageSpace(ctx, req)
	if err != nil {
		return &provider.UpdateStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call UpdateStorageSpace", err),
		}, nil
	}

	if res.Status.Code == rpc.Code_CODE_OK {
		id := res.StorageSpace.Root
		s.providerCache.RemoveListStorageProviders(id)
	}
	return res, nil
}

func (s *svc) DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) (*provider.DeleteStorageSpaceResponse, error) {
	opaque := req.Opaque
	var purge bool
	// This is just a temporary hack until the CS3 API get's updated to have a dedicated purge parameter or a dedicated PurgeStorageSpace method.
	if opaque != nil {
		_, purge = opaque.Map["purge"]
	}

	rid, err := storagespace.ParseID(req.GetId().GetOpaqueId())
	if err != nil {
		return &provider.DeleteStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not parse space id %s", req.GetId().GetOpaqueId()), err),
		}, nil
	}

	ref := &provider.Reference{ResourceId: &rid}
	c, _, err := s.findSpacesProvider(ctx, ref)
	if err != nil {
		return &provider.DeleteStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find reference %+v", ref), err),
		}, nil
	}

	dsRes, err := c.DeleteStorageSpace(ctx, req)
	if err != nil {
		return &provider.DeleteStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call DeleteStorageSpace", err),
		}, nil
	}

	id := &provider.ResourceId{OpaqueId: req.GetId().GetOpaqueId()}
	s.providerCache.RemoveListStorageProviders(id)

	if dsRes.Status.Code != rpc.Code_CODE_OK {
		return dsRes, nil
	}

	if !purge {
		return dsRes, nil
	}

	log := appctx.GetLogger(ctx)
	log.Debug().Msg("purging storage space")
	// List all shares in this storage space
	lsRes, err := s.ListShares(ctx, &collaborationv1beta1.ListSharesRequest{
		Filters: []*collaborationv1beta1.Filter{share.SpaceIDFilter(id.SpaceId)},
	})
	switch {
	case err != nil:
		return &provider.DeleteStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not delete shares of StorageSpace", err),
		}, nil
	case lsRes.Status.Code != rpc.Code_CODE_OK:
		return &provider.DeleteStorageSpaceResponse{
			Status: status.NewInternal(ctx, "gateway could not delete shares of StorageSpace"),
		}, nil
	}
	for _, share := range lsRes.Shares {
		rsRes, err := s.RemoveShare(ctx, &collaborationv1beta1.RemoveShareRequest{
			Ref: &collaborationv1beta1.ShareReference{
				Spec: &collaborationv1beta1.ShareReference_Id{Id: share.Id},
			},
		})
		if err != nil || rsRes.Status.Code != rpc.Code_CODE_OK {
			log.Error().Err(err).Interface("status", rsRes.Status).Str("share_id", share.Id.OpaqueId).Msg("failed to delete share")
		}
	}

	// List all public shares in this storage space
	lpsRes, err := s.ListPublicShares(ctx, &linkv1beta1.ListPublicSharesRequest{
		Filters: []*linkv1beta1.ListPublicSharesRequest_Filter{publicshare.StorageIDFilter(id.SpaceId)}, // FIXME rename the filter? @c0rby
	})
	switch {
	case err != nil:
		return &provider.DeleteStorageSpaceResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not delete shares of StorageSpace", err),
		}, nil
	case lpsRes.Status.Code != rpc.Code_CODE_OK:
		return &provider.DeleteStorageSpaceResponse{
			Status: status.NewInternal(ctx, "gateway could not delete shares of StorageSpace"),
		}, nil
	}
	for _, share := range lpsRes.Share {
		rsRes, err := s.RemovePublicShare(ctx, &linkv1beta1.RemovePublicShareRequest{
			Ref: &linkv1beta1.PublicShareReference{
				Spec: &linkv1beta1.PublicShareReference_Id{Id: share.Id},
			},
		})
		if err != nil || rsRes.Status.Code != rpc.Code_CODE_OK {
			log.Error().Err(err).Interface("status", rsRes.Status).Str("share_id", share.Id.OpaqueId).Msg("failed to delete share")
		}
	}

	return dsRes, nil
}

func (s *svc) GetHome(ctx context.Context, _ *provider.GetHomeRequest) (*provider.GetHomeResponse, error) {
	currentUser, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		return nil, errors.New("user not found in context")
	}

	srClient, err := s.getStorageRegistryClient(ctx, s.c.StorageRegistryEndpoint)
	if err != nil {
		return &provider.GetHomeResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not get storage registry client", err),
		}, nil
	}

	spaceJSON, err := json.Marshal(&provider.StorageSpace{
		Owner:     currentUser,
		SpaceType: "personal",
	})
	if err != nil {
		return &provider.GetHomeResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not marshal space", err),
		}, nil
	}

	// The registry is responsible for choosing the right provider
	// TODO fix naming GetStorageProviders calls the GetProvider functon on the registry implementation
	res, err := srClient.GetStorageProviders(ctx, &registry.GetStorageProvidersRequest{
		Opaque: &typesv1beta1.Opaque{
			Map: map[string]*typesv1beta1.OpaqueEntry{
				"space": {
					Decoder: "json",
					Value:   spaceJSON,
				},
			},
		},
	})
	if err != nil {
		return &provider.GetHomeResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call GetStorageProviders", err),
		}, nil
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return &provider.GetHomeResponse{
			Status: res.Status,
		}, nil
	}

	if len(res.Providers) == 0 {
		return &provider.GetHomeResponse{
			Status: status.NewNotFound(ctx, fmt.Sprintf("error finding provider for home space of %+v", currentUser)),
		}, nil
	}

	// NOTE: this will cause confusion if len(spaces) > 1
	spaces := decodeSpaces(res.Providers[0])
	for _, space := range spaces {
		return &provider.GetHomeResponse{
			Path:   decodePath(space),
			Status: status.NewOK(ctx),
		}, nil
	}

	return &provider.GetHomeResponse{
		Status: status.NewNotFound(ctx, fmt.Sprintf("error finding home path for provider %+v with spaces %+v ", res.Providers[0], spaces)),
	}, nil
}

func (s *svc) InitiateFileDownload(ctx context.Context, req *provider.InitiateFileDownloadRequest) (*gateway.InitiateFileDownloadResponse, error) {
	// TODO(ishank011): enable downloading references spread across storage providers, eg. /eos
	var c provider.ProviderAPIClient
	var err error
	c, _, req.Ref, err = s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &gateway.InitiateFileDownloadResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	storageRes, err := c.InitiateFileDownload(ctx, req)
	if err != nil {
		return &gateway.InitiateFileDownloadResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not call InitiateFileDownload, ref=%+v", req.Ref), err),
		}, nil
	}

	protocols := make([]*gateway.FileDownloadProtocol, len(storageRes.Protocols))
	for p := range storageRes.Protocols {
		protocols[p] = &gateway.FileDownloadProtocol{
			Opaque:           storageRes.Protocols[p].Opaque,
			Protocol:         storageRes.Protocols[p].Protocol,
			DownloadEndpoint: storageRes.Protocols[p].DownloadEndpoint,
		}

		if !storageRes.Protocols[p].Expose {
			// sign the download location and pass it to the data gateway
			u, err := url.Parse(protocols[p].DownloadEndpoint)
			if err != nil {
				return &gateway.InitiateFileDownloadResponse{
					Status: status.NewStatusFromErrType(ctx, "wrong format for download endpoint", err),
				}, nil
			}

			// TODO(labkode): calculate signature of the whole request? we only sign the URI now. Maybe worth https://tools.ietf.org/html/draft-cavage-http-signatures-11
			target := u.String()
			token, err := s.sign(ctx, target, time.Now().UTC().Add(time.Duration(s.c.TransferExpires)*time.Second).Unix())
			if err != nil {
				return &gateway.InitiateFileDownloadResponse{
					Status: status.NewStatusFromErrType(ctx, "error creating signature for download", err),
				}, nil
			}

			protocols[p].DownloadEndpoint = s.c.DataGatewayEndpoint
			protocols[p].Token = token
		}
	}

	return &gateway.InitiateFileDownloadResponse{
		Opaque:    storageRes.Opaque,
		Status:    storageRes.Status,
		Protocols: protocols,
	}, nil
}

func (s *svc) InitiateFileUpload(ctx context.Context, req *provider.InitiateFileUploadRequest) (*gateway.InitiateFileUploadResponse, error) {
	var c provider.ProviderAPIClient
	var err error
	c, _, req.Ref, err = s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &gateway.InitiateFileUploadResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	storageRes, err := c.InitiateFileUpload(ctx, req)
	if err != nil {
		return &gateway.InitiateFileUploadResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not call InitiateFileUpload, ref=%+v", req.Ref), err),
		}, nil
	}

	if storageRes.Status.Code != rpc.Code_CODE_OK {
		return &gateway.InitiateFileUploadResponse{
			Status: storageRes.Status,
		}, nil
	}

	protocols := make([]*gateway.FileUploadProtocol, len(storageRes.Protocols))
	for p := range storageRes.Protocols {
		protocols[p] = &gateway.FileUploadProtocol{
			Opaque:             storageRes.Protocols[p].Opaque,
			Protocol:           storageRes.Protocols[p].Protocol,
			UploadEndpoint:     storageRes.Protocols[p].UploadEndpoint,
			AvailableChecksums: storageRes.Protocols[p].AvailableChecksums,
		}

		if !storageRes.Protocols[p].Expose {
			// sign the upload location and pass it to the data gateway
			u, err := url.Parse(protocols[p].UploadEndpoint)
			if err != nil {
				return &gateway.InitiateFileUploadResponse{
					Status: status.NewStatusFromErrType(ctx, "wrong format for upload endpoint", err),
				}, nil
			}

			// TODO(labkode): calculate signature of the whole request? we only sign the URI now. Maybe worth https://tools.ietf.org/html/draft-cavage-http-signatures-11
			target := u.String()
			ttl := time.Duration(s.c.TransferExpires) * time.Second
			expiresAt := time.Now().Add(ttl).Unix()
			if storageRes.Protocols[p].Expiration != nil {
				expiresAt = utils.TSToTime(storageRes.Protocols[p].Expiration).Unix()
			}
			token, err := s.sign(ctx, target, expiresAt)
			if err != nil {
				return &gateway.InitiateFileUploadResponse{
					Status: status.NewStatusFromErrType(ctx, "error creating signature for upload", err),
				}, nil
			}

			protocols[p].UploadEndpoint = s.c.DataGatewayEndpoint
			protocols[p].Token = token
		}
	}

	return &gateway.InitiateFileUploadResponse{
		Opaque:    storageRes.Opaque,
		Status:    storageRes.Status,
		Protocols: protocols,
	}, nil
}

func (s *svc) GetPath(ctx context.Context, req *provider.GetPathRequest) (*provider.GetPathResponse, error) {
	c, _, ref, err := s.findAndUnwrap(ctx, &provider.Reference{ResourceId: req.ResourceId})
	if err != nil {
		return &provider.GetPathResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find reference %+v", ref), err),
		}, nil
	}

	req.ResourceId = ref.ResourceId
	return c.GetPath(ctx, req)
}

func (s *svc) CreateContainer(ctx context.Context, req *provider.CreateContainerRequest) (*provider.CreateContainerResponse, error) {
	var c provider.ProviderAPIClient
	var err error
	c, _, req.Ref, err = s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.CreateContainerResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	res, err := c.CreateContainer(ctx, req)
	if err != nil {
		return &provider.CreateContainerResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call CreateContainer", err),
		}, nil
	}

	return res, nil
}

func (s *svc) AddFavorite(ctx context.Context, req *provider.AddFavoriteRequest) (*provider.AddFavoriteResponse, error) {
	var c provider.ProviderAPIClient
	var err error
	c, _, req.Ref, err = s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.AddFavoriteResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	res, err := c.AddFavorite(ctx, req)
	if err != nil {
		return &provider.AddFavoriteResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call AddFavorite", err),
		}, nil
	}

	return res, nil
}

func (s *svc) RemoveFavorite(ctx context.Context, req *provider.RemoveFavoriteRequest) (*provider.RemoveFavoriteResponse, error) {
	var c provider.ProviderAPIClient
	var err error
	c, _, req.Ref, err = s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.RemoveFavoriteResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	res, err := c.RemoveFavorite(ctx, req)
	if err != nil {
		return &provider.RemoveFavoriteResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call RemoveFavorite", err),
		}, nil
	}

	return res, nil
}

func (s *svc) TouchFile(ctx context.Context, req *provider.TouchFileRequest) (*provider.TouchFileResponse, error) {
	var c provider.ProviderAPIClient
	var err error
	c, _, req.Ref, err = s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.TouchFileResponse{
			Status: status.NewStatusFromErrType(ctx, "TouchFile ref="+req.Ref.String(), err),
		}, nil
	}

	res, err := c.TouchFile(ctx, req)
	if err != nil {
		if gstatus.Code(err) == codes.PermissionDenied {
			return &provider.TouchFileResponse{Status: &rpc.Status{Code: rpc.Code_CODE_PERMISSION_DENIED}}, nil
		}
		return nil, errors.Wrap(err, "gateway: error calling TouchFile")
	}

	return res, nil
}

func (s *svc) Delete(ctx context.Context, req *provider.DeleteRequest) (*provider.DeleteResponse, error) {
	// TODO(ishank011): enable deleting references spread across storage providers, eg. /eos
	var c provider.ProviderAPIClient
	var err error
	c, _, req.Ref, err = s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.DeleteResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	res, err := c.Delete(ctx, req)
	if err != nil {
		return &provider.DeleteResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call Delete", err),
		}, nil
	}

	return res, nil
}

func (s *svc) Move(ctx context.Context, req *provider.MoveRequest) (*provider.MoveResponse, error) {
	c, sourceProviderInfo, sref, err := s.findAndUnwrap(ctx, req.Source)
	if err != nil {
		return &provider.MoveResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Source), err),
		}, nil
	}

	_, destProviderInfo, dref, err := s.findAndUnwrap(ctx, req.Destination)
	if err != nil {
		return &provider.MoveResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Source), err),
		}, nil
	}

	if sourceProviderInfo.Address != destProviderInfo.Address {
		return &provider.MoveResponse{
			Status: status.NewUnimplemented(ctx, nil, "cross storage moves are not supported, use copy and delete"),
		}, nil
	}

	req.Source = sref
	req.Destination = dref
	return c.Move(ctx, req)
}

func (s *svc) SetArbitraryMetadata(ctx context.Context, req *provider.SetArbitraryMetadataRequest) (*provider.SetArbitraryMetadataResponse, error) {
	// TODO(ishank011): enable for references spread across storage providers, eg. /eos
	var c provider.ProviderAPIClient
	var err error
	c, _, req.Ref, err = s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.SetArbitraryMetadataResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	res, err := c.SetArbitraryMetadata(ctx, req)
	if err != nil {
		if gstatus.Code(err) == codes.PermissionDenied {
			return &provider.SetArbitraryMetadataResponse{Status: &rpc.Status{Code: rpc.Code_CODE_PERMISSION_DENIED}}, nil
		}
		return nil, errors.Wrap(err, "gateway: error calling SetArbitraryMetadata")
	}

	return res, nil
}

func (s *svc) UnsetArbitraryMetadata(ctx context.Context, req *provider.UnsetArbitraryMetadataRequest) (*provider.UnsetArbitraryMetadataResponse, error) {
	// TODO(ishank011): enable for references spread across storage providers, eg. /eos
	var c provider.ProviderAPIClient
	var err error
	c, _, req.Ref, err = s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.UnsetArbitraryMetadataResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	res, err := c.UnsetArbitraryMetadata(ctx, req)
	if err != nil {
		if gstatus.Code(err) == codes.PermissionDenied {
			return &provider.UnsetArbitraryMetadataResponse{Status: &rpc.Status{Code: rpc.Code_CODE_PERMISSION_DENIED}}, nil
		}
		return nil, errors.Wrap(err, "gateway: error calling UnsetArbitraryMetadata")
	}

	return res, nil
}

// SetLock puts a lock on the given reference
func (s *svc) SetLock(ctx context.Context, req *provider.SetLockRequest) (*provider.SetLockResponse, error) {
	var c provider.ProviderAPIClient
	var err error
	c, _, req.Ref, err = s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.SetLockResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	res, err := c.SetLock(ctx, req)
	if err != nil {
		if gstatus.Code(err) == codes.PermissionDenied {
			return &provider.SetLockResponse{Status: &rpc.Status{Code: rpc.Code_CODE_PERMISSION_DENIED}}, nil
		}
		return nil, errors.Wrap(err, "gateway: error calling SetLock")
	}

	return res, nil
}

// GetLock returns an existing lock on the given reference
func (s *svc) GetLock(ctx context.Context, req *provider.GetLockRequest) (*provider.GetLockResponse, error) {
	c, _, err := s.find(ctx, req.Ref)
	if err != nil {
		return &provider.GetLockResponse{
			Status: status.NewStatusFromErrType(ctx, "GetLock ref="+req.Ref.String(), err),
		}, nil
	}

	res, err := c.GetLock(ctx, req)
	if err != nil {
		if gstatus.Code(err) == codes.PermissionDenied {
			return &provider.GetLockResponse{Status: &rpc.Status{Code: rpc.Code_CODE_PERMISSION_DENIED}}, nil
		}
		return nil, errors.Wrap(err, "gateway: error calling GetLock")
	}

	return res, nil
}

// RefreshLock refreshes an existing lock on the given reference
func (s *svc) RefreshLock(ctx context.Context, req *provider.RefreshLockRequest) (*provider.RefreshLockResponse, error) {
	c, _, err := s.find(ctx, req.Ref)
	if err != nil {
		return &provider.RefreshLockResponse{
			Status: status.NewStatusFromErrType(ctx, "RefreshLock ref="+req.Ref.String(), err),
		}, nil
	}

	res, err := c.RefreshLock(ctx, req)
	if err != nil {
		if gstatus.Code(err) == codes.PermissionDenied {
			return &provider.RefreshLockResponse{Status: &rpc.Status{Code: rpc.Code_CODE_PERMISSION_DENIED}}, nil
		}
		return nil, errors.Wrap(err, "gateway: error calling RefreshLock")
	}

	return res, nil
}

// Unlock removes an existing lock from the given reference
func (s *svc) Unlock(ctx context.Context, req *provider.UnlockRequest) (*provider.UnlockResponse, error) {
	c, _, err := s.find(ctx, req.Ref)
	if err != nil {
		return &provider.UnlockResponse{
			Status: status.NewStatusFromErrType(ctx, "Unlock ref="+req.Ref.String(), err),
		}, nil
	}

	res, err := c.Unlock(ctx, req)
	if err != nil {
		if gstatus.Code(err) == codes.PermissionDenied {
			return &provider.UnlockResponse{Status: &rpc.Status{Code: rpc.Code_CODE_PERMISSION_DENIED}}, nil
		}
		return nil, errors.Wrap(err, "gateway: error calling Unlock")
	}

	return res, nil
}

// Stat returns the Resoure info for a given resource by forwarding the request to the responsible provider.
func (s *svc) Stat(ctx context.Context, req *provider.StatRequest) (*provider.StatResponse, error) {
	c, _, ref, err := s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.StatResponse{
			Status: status.NewNotFound(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref)),
		}, nil
	}

	return c.Stat(ctx, &provider.StatRequest{
		Opaque:                req.Opaque,
		Ref:                   ref,
		ArbitraryMetadataKeys: req.ArbitraryMetadataKeys,
		FieldMask:             req.FieldMask,
	})
}

func (s *svc) ListContainerStream(_ *provider.ListContainerStreamRequest, _ gateway.GatewayAPI_ListContainerStreamServer) error {
	return errtypes.NotSupported("Unimplemented")
}

// ListContainer lists the Resoure infos for a given resource by forwarding the request to the responsible provider.
func (s *svc) ListContainer(ctx context.Context, req *provider.ListContainerRequest) (*provider.ListContainerResponse, error) {
	c, _, ref, err := s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		// we have no provider -> not found
		return &provider.ListContainerResponse{
			Status: status.NewNotFound(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref)),
		}, nil
	}

	return c.ListContainer(ctx, &provider.ListContainerRequest{
		Opaque:                req.Opaque,
		Ref:                   ref,
		ArbitraryMetadataKeys: req.ArbitraryMetadataKeys,
		FieldMask:             req.FieldMask,
	})
}

func (s *svc) CreateSymlink(ctx context.Context, req *provider.CreateSymlinkRequest) (*provider.CreateSymlinkResponse, error) {
	return &provider.CreateSymlinkResponse{
		Status: status.NewUnimplemented(ctx, errtypes.NotSupported("CreateSymlink not implemented"), "CreateSymlink not implemented"),
	}, nil
}

func (s *svc) ListFileVersions(ctx context.Context, req *provider.ListFileVersionsRequest) (*provider.ListFileVersionsResponse, error) {
	var c provider.ProviderAPIClient
	var err error
	c, _, req.Ref, err = s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.ListFileVersionsResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	return c.ListFileVersions(ctx, req)
}

func (s *svc) RestoreFileVersion(ctx context.Context, req *provider.RestoreFileVersionRequest) (*provider.RestoreFileVersionResponse, error) {
	var c provider.ProviderAPIClient
	var err error
	c, _, req.Ref, err = s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.RestoreFileVersionResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	res, err := c.RestoreFileVersion(ctx, req)
	if err != nil {
		return &provider.RestoreFileVersionResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call RestoreFileVersion", err),
		}, nil
	}

	return res, nil
}

func (s *svc) ListRecycleStream(_ *provider.ListRecycleStreamRequest, _ gateway.GatewayAPI_ListRecycleStreamServer) error {
	return errtypes.NotSupported("ListRecycleStream unimplemented")
}

// TODO use the ListRecycleRequest.Ref to only list the trash of a specific storage
func (s *svc) ListRecycle(ctx context.Context, req *provider.ListRecycleRequest) (*provider.ListRecycleResponse, error) {
	c, _, ref, err := s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.ListRecycleResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}
	return c.ListRecycle(ctx, &provider.ListRecycleRequest{
		Opaque: req.Opaque,
		FromTs: req.FromTs,
		ToTs:   req.ToTs,
		Ref:    ref,
		Key:    req.Key,
	})
}

func (s *svc) RestoreRecycleItem(ctx context.Context, req *provider.RestoreRecycleItemRequest) (*provider.RestoreRecycleItemResponse, error) {
	c, si, ref, err := s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.RestoreRecycleItemResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	_, di, rref, err := s.findAndUnwrap(ctx, req.RestoreRef)
	if err != nil {
		return &provider.RestoreRecycleItemResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	if si.Address != di.Address {
		return &provider.RestoreRecycleItemResponse{
			// TODO in Move() we return an unimplemented / supported ... align?
			Status: status.NewPermissionDenied(ctx, err, "gateway: cross-storage restores are forbidden"),
		}, nil
	}

	req.Ref = ref
	req.RestoreRef = rref
	res, err := c.RestoreRecycleItem(ctx, req)
	if err != nil {
		return &provider.RestoreRecycleItemResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call RestoreRecycleItem", err),
		}, nil
	}

	return res, nil
}

func (s *svc) PurgeRecycle(ctx context.Context, req *provider.PurgeRecycleRequest) (*provider.PurgeRecycleResponse, error) {
	c, _, relativeReference, err := s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.PurgeRecycleResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	res, err := c.PurgeRecycle(ctx, &provider.PurgeRecycleRequest{
		Opaque: req.GetOpaque(),
		Ref:    relativeReference,
		Key:    req.Key,
	})
	if err != nil {
		return &provider.PurgeRecycleResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call PurgeRecycle", err),
		}, nil
	}

	return res, nil
}

func (s *svc) GetQuota(ctx context.Context, req *gateway.GetQuotaRequest) (*provider.GetQuotaResponse, error) {
	c, _, relativeReference, err := s.findAndUnwrap(ctx, req.Ref)
	if err != nil {
		return &provider.GetQuotaResponse{
			Status: status.NewStatusFromErrType(ctx, fmt.Sprintf("gateway could not find space for ref=%+v", req.Ref), err),
		}, nil
	}

	res, err := c.GetQuota(ctx, &provider.GetQuotaRequest{
		Opaque: req.GetOpaque(),
		Ref:    relativeReference,
	})
	if err != nil {
		return &provider.GetQuotaResponse{
			Status: status.NewStatusFromErrType(ctx, "gateway could not call GetQuota", err),
		}, nil
	}
	return res, nil
}

// find looks up the provider that is responsible for the given request
// It will return a client that the caller can use to make the call, as well as the ProviderInfo. It:
// - contains the provider path, which is the mount point of the provider
// - may contain a list of storage spaces with their id and space path
func (s *svc) find(ctx context.Context, ref *provider.Reference) (provider.ProviderAPIClient, *registry.ProviderInfo, error) {
	p, err := s.findSingleSpace(ctx, ref)
	if err != nil {
		return nil, nil, err
	}

	client, err := s.getStorageProviderClient(ctx, p[0])
	return client, p[0], err
}

// findSpacesProvider looks up the spaces provider that is responsible for the given request
// It will return a client that the caller can use to make the call, as well as the ProviderInfo. It:
// - contains the provider path, which is the mount point of the provider
// - may contain a list of storage spaces with their id and space path
func (s *svc) findSpacesProvider(ctx context.Context, ref *provider.Reference) (provider.SpacesAPIClient, *registry.ProviderInfo, error) {
	p, err := s.findSingleSpace(ctx, ref)
	if err != nil {
		return nil, nil, err
	}

	client, err := s.getSpacesProviderClient(ctx, p[0])
	return client, p[0], err
}

func (s *svc) findAndUnwrap(ctx context.Context, ref *provider.Reference) (provider.ProviderAPIClient, *registry.ProviderInfo, *provider.Reference, error) {
	c, p, err := s.find(ctx, ref)
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		root      *provider.ResourceId
		mountPath string
	)
	for _, space := range decodeSpaces(p) {
		mountPath = decodePath(space)
		root = space.Root
		break // TODO can there be more than one space for a path?
	}

	relativeReference := unwrap(ref, mountPath, root)

	return c, p, relativeReference, nil
}

func (s *svc) getSpacesProviderClient(_ context.Context, p *registry.ProviderInfo) (provider.SpacesAPIClient, error) {
	c, err := pool.GetSpacesProviderServiceClient(p.Address)
	if err != nil {
		return nil, err
	}

	return &cachedSpacesAPIClient{
		c:                        c,
		createPersonalSpaceCache: s.createPersonalSpaceCache,
	}, nil
}

func (s *svc) getStorageProviderClient(_ context.Context, p *registry.ProviderInfo) (provider.ProviderAPIClient, error) {
	c, err := pool.GetStorageProviderServiceClient(p.Address)
	if err != nil {
		return nil, err
	}

	return &cachedAPIClient{
		c:                        c,
		createPersonalSpaceCache: s.createPersonalSpaceCache,
	}, nil
}

func (s *svc) getStorageRegistryClient(_ context.Context, address string) (registry.RegistryAPIClient, error) {
	c, err := pool.GetStorageRegistryClient(address)
	if err != nil {
		return nil, err
	}
	return &cachedRegistryClient{
		c:     c,
		cache: s.providerCache,
	}, nil
}

func (s *svc) findSingleSpace(ctx context.Context, ref *provider.Reference) ([]*registry.ProviderInfo, error) {
	switch {
	case ref == nil:
		return nil, errtypes.BadRequest("missing reference")
	case ref.ResourceId != nil:
		if ref.ResourceId.OpaqueId == "" {
			ref.ResourceId.OpaqueId = ref.ResourceId.SpaceId
		}
	case ref.Path != "": //  TODO implement a mount path cache in the registry?
		// nothing to do here either
	default:
		return nil, errtypes.BadRequest("invalid reference, at least path or id must be set")
	}

	filters := map[string]string{
		"mask":   "root", // FIXME replace with fieldmask, here we only want to get the root resourceid
		"path":   ref.Path,
		"unique": "true",
	}
	if ref.ResourceId != nil {
		filters["storage_id"] = ref.ResourceId.StorageId
		filters["space_id"] = ref.ResourceId.SpaceId
		filters["opaque_id"] = ref.ResourceId.OpaqueId
	}

	listReq := &registry.ListStorageProvidersRequest{
		Opaque: &typesv1beta1.Opaque{},
	}
	sdk.EncodeOpaqueMap(listReq.Opaque, filters)

	return s.findProvider(ctx, listReq)
}

func (s *svc) findProvider(ctx context.Context, listReq *registry.ListStorageProvidersRequest) ([]*registry.ProviderInfo, error) {
	// lookup
	c, err := pool.GetStorageRegistryClient(s.c.StorageRegistryEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error getting storage registry client")
	}
	res, err := c.ListStorageProviders(ctx, listReq)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling ListStorageProviders")
	}

	if res.Status.Code != rpc.Code_CODE_OK {
		switch res.Status.Code {
		case rpc.Code_CODE_NOT_FOUND:
			// TODO use tombstone cache item?
			return nil, errtypes.NotFound("gateway: storage provider not found for reference:" + listReq.String())
		case rpc.Code_CODE_PERMISSION_DENIED:
			return nil, errtypes.PermissionDenied("gateway: " + res.Status.Message + " for " + listReq.String() + " with code " + res.Status.Code.String())
		case rpc.Code_CODE_INVALID_ARGUMENT, rpc.Code_CODE_FAILED_PRECONDITION, rpc.Code_CODE_OUT_OF_RANGE:
			return nil, errtypes.BadRequest("gateway: " + res.Status.Message + " for " + listReq.String() + " with code " + res.Status.Code.String())
		case rpc.Code_CODE_UNIMPLEMENTED:
			return nil, errtypes.NotSupported("gateway: " + res.Status.Message + " for " + listReq.String() + " with code " + res.Status.Code.String())
		default:
			return nil, status.NewErrorFromCode(res.Status.Code, "gateway")
		}
	}

	if res.Providers == nil {
		return nil, errtypes.NotFound("gateway: provider is nil")
	}

	return res.Providers, nil
}

// unwrap takes a reference and builds a reference for the provider. can be absolute or relative to a root node
func unwrap(ref *provider.Reference, mountPoint string, root *provider.ResourceId) *provider.Reference {
	if utils.IsAbsolutePathReference(ref) {
		providerRef := &provider.Reference{
			Path: strings.TrimPrefix(ref.Path, mountPoint),
		}
		// if we have a root use it and make the path relative
		if root != nil {
			providerRef.ResourceId = root
			providerRef.Path = utils.MakeRelativePath(providerRef.Path)
		}
		return providerRef
	}

	return &provider.Reference{
		ResourceId: &provider.ResourceId{
			StorageId: ref.GetResourceId().GetStorageId(),
			SpaceId:   ref.GetResourceId().GetSpaceId(),
			OpaqueId:  ref.GetResourceId().GetOpaqueId(),
		},
		Path: ref.GetPath(),
	}
}

func decodeSpaces(r *registry.ProviderInfo) []*provider.StorageSpace {
	spaces := []*provider.StorageSpace{}
	if r.Opaque != nil {
		if entry, ok := r.Opaque.Map["spaces"]; ok {
			switch entry.Decoder {
			case "json":
				_ = json.Unmarshal(entry.Value, &spaces)
			case "toml":
				_ = toml.Unmarshal(entry.Value, &spaces)
			case "xml":
				_ = xml.Unmarshal(entry.Value, &spaces)
			}
		}
	}
	if len(spaces) == 0 {
		// we need to convert the provider into a space, needed for the static registry
		spaces = append(spaces, &provider.StorageSpace{
			Opaque: &typesv1beta1.Opaque{Map: map[string]*typesv1beta1.OpaqueEntry{
				"path": {
					Decoder: "plain",
					Value:   []byte(r.ProviderPath),
				},
			}},
		})
	}
	return spaces
}

func decodePath(s *provider.StorageSpace) (path string) {
	if s.Opaque != nil {
		if entry, ok := s.Opaque.Map["path"]; ok {
			switch entry.Decoder {
			case "plain":
				path = string(entry.Value)
			case "json":
				_ = json.Unmarshal(entry.Value, &path)
			case "toml":
				_ = toml.Unmarshal(entry.Value, &path)
			case "xml":
				_ = xml.Unmarshal(entry.Value, &path)
			}
		}
	}
	return
}
