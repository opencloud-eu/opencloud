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

package auth

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	appprovider "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	appregistry "github.com/cs3org/go-cs3apis/cs3/app/registry/v1beta1"
	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	ocmv1beta1 "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	registry "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	statuspkg "github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/token"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"google.golang.org/grpc/metadata"
)

const (
	scopeDelimiter       = "#"
	scopeCacheExpiration = 3600
)

func expandAndVerifyScope(ctx context.Context, req interface{}, tokenScope map[string]*authpb.Scope, user *userpb.User, gatewayAddr string, mgr token.Manager) error {
	log := appctx.GetLogger(ctx)
	client, err := pool.GetGatewayServiceClient(gatewayAddr)
	if err != nil {
		return err
	}

	if ref, ok := extractRef(req, tokenScope); ok {
		// The request is for a storage reference. This can be the case for multiple scenarios:
		// - If the path is not empty, the request might be coming from a share where the accessor is
		//   trying to impersonate the owner, since the share manager doesn't know the
		//   share path.
		// - If the ID not empty, the request might be coming from
		//   - a resource present inside a shared folder, or
		//   - a share created for a lightweight account after the token was minted.
		log.Info().Msgf("resolving storage reference to check token scope %s", ref.String())
		for k := range tokenScope {
			switch {
			case strings.HasPrefix(k, "publicshare"):
				if err = resolvePublicShare(ctx, ref, tokenScope[k], client, mgr); err == nil {
					return nil
				}

			case strings.HasPrefix(k, "lightweight"):
				if err = resolveLightweightScope(ctx, ref, tokenScope[k], user, client, mgr); err == nil {
					return nil
				}
			case strings.HasPrefix(k, "ocmshare"):
				if err = resolveOCMShare(ctx, ref, tokenScope[k], client, mgr); err == nil {
					return nil
				}
			}
			log.Err(err).Interface("ref", ref).Interface("scope", k).Msg("error resolving reference under scope")
		}

	} else if ref, ok := extractShareRef(req); ok {
		// It's a share ref
		// The request might be coming from a share created for a lightweight account
		// after the token was minted.
		log.Info().Msgf("resolving share reference against received shares to verify token scope %+v", ref.String())
		for k := range tokenScope {
			if strings.HasPrefix(k, "lightweight") {
				// Check if this ID is cached
				key := "lw:" + user.Id.OpaqueId + scopeDelimiter + ref.GetId().OpaqueId
				if _, err := scopeExpansionCache.Get(key); err == nil {
					return nil
				}

				shares, err := client.ListReceivedShares(ctx, &collaboration.ListReceivedSharesRequest{})
				if err != nil || shares.Status.Code != rpc.Code_CODE_OK {
					log.Warn().Err(err).Msg("error listing received shares")
					continue
				}
				for _, s := range shares.Shares {
					shareKey := "lw:" + user.Id.OpaqueId + scopeDelimiter + s.Share.Id.OpaqueId
					_ = scopeExpansionCache.SetWithExpire(shareKey, nil, scopeCacheExpiration*time.Second)

					if ref.GetId() != nil && ref.GetId().OpaqueId == s.Share.Id.OpaqueId {
						return nil
					}
					if key := ref.GetKey(); key != nil && (utils.UserEqual(key.Owner, s.Share.Owner) || utils.UserEqual(key.Owner, s.Share.Creator)) &&
						utils.ResourceIDEqual(key.ResourceId, s.Share.ResourceId) && utils.GranteeEqual(key.Grantee, s.Share.Grantee) {
						return nil
					}
				}
			}
		}
	}

	return errtypes.PermissionDenied(fmt.Sprintf("access to resource %+v not allowed within the assigned scope", req))
}

func resolveLightweightScope(ctx context.Context, ref *provider.Reference, scope *authpb.Scope, user *userpb.User, client gateway.GatewayAPIClient, mgr token.Manager) error {
	refString, err := storagespace.FormatReference(ref)
	if err != nil {
		// cannot format reference, so cannot be valid
		return errtypes.PermissionDenied("invalid reference")
	}
	// Check if this ref is cached
	key := "lw:" + user.Id.OpaqueId + scopeDelimiter + refString
	if _, err := scopeExpansionCache.Get(key); err == nil {
		return nil
	}

	shares, err := client.ListReceivedShares(ctx, &collaboration.ListReceivedSharesRequest{})
	if err != nil || shares.Status.Code != rpc.Code_CODE_OK {
		return errtypes.InternalError("error listing received shares")
	}

	for _, share := range shares.Shares {
		shareKey := "lw:" + user.Id.OpaqueId + scopeDelimiter + storagespace.FormatResourceID(share.Share.ResourceId)
		_ = scopeExpansionCache.SetWithExpire(shareKey, nil, scopeCacheExpiration*time.Second)

		if ref.ResourceId != nil && utils.ResourceIDEqual(share.Share.ResourceId, ref.ResourceId) {
			return nil
		}
		if ok, err := checkIfNestedResource(ctx, ref, share.Share.ResourceId, client, mgr); err == nil && ok {
			_ = scopeExpansionCache.SetWithExpire(key, nil, scopeCacheExpiration*time.Second)
			return nil
		}
	}

	return errtypes.PermissionDenied("request is not for a nested resource")
}

func resolvePublicShare(ctx context.Context, ref *provider.Reference, scope *authpb.Scope, client gateway.GatewayAPIClient, mgr token.Manager) error {
	var share link.PublicShare
	err := utils.UnmarshalJSONToProtoV1(scope.Resource.Value, &share)
	if err != nil {
		return err
	}

	return checkCacheForNestedResource(ctx, ref, share.ResourceId, client, mgr)
}

func resolveOCMShare(ctx context.Context, ref *provider.Reference, scope *authpb.Scope, client gateway.GatewayAPIClient, mgr token.Manager) error {
	var share ocmv1beta1.Share
	if err := utils.UnmarshalJSONToProtoV1(scope.Resource.Value, &share); err != nil {
		return err
	}

	// for ListOCMSharesRequest, the ref resource id is empty and we set path to . to indicate the root of the share
	if ref.GetResourceId() == nil && ref.Path == "." {
		ref.ResourceId = share.GetResourceId()
	}

	return checkCacheForNestedResource(ctx, ref, share.ResourceId, client, mgr)
}

func checkCacheForNestedResource(ctx context.Context, ref *provider.Reference, resource *provider.ResourceId, client gateway.GatewayAPIClient, mgr token.Manager) error {
	refString, err := storagespace.FormatReference(ref)
	if err != nil {
		// cannot format reference, so cannot be valid
		return errtypes.PermissionDenied("invalid reference")
	}

	// Check if this ref is cached
	key := storagespace.FormatResourceID(resource) + scopeDelimiter + refString
	if _, err := scopeExpansionCache.Get(key); err == nil {
		return nil
	}

	if ok, err := checkIfNestedResource(ctx, ref, resource, client, mgr); err == nil && ok {
		_ = scopeExpansionCache.SetWithExpire(key, nil, scopeCacheExpiration*time.Second)
		return nil
	}

	return errtypes.PermissionDenied("request is not for a nested resource")
}

func checkIfNestedResource(ctx context.Context, ref *provider.Reference, parent *provider.ResourceId, client gateway.GatewayAPIClient, mgr token.Manager) (bool, error) {
	// Since the resource ID is obtained from the scope, the current token
	// has access to it.
	statResponse, err := client.Stat(ctx, &provider.StatRequest{Ref: &provider.Reference{ResourceId: parent}})
	if err != nil {
		return false, err
	}
	if statResponse.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return false, statuspkg.NewErrorFromCode(statResponse.Status.Code, "auth interceptor")
	}

	parentInfo := statResponse.GetInfo()

	// We need to find out if the requested resource is child of the `parent` (coming from token scope)
	// We mint a token as the owner of the public share and try to stat the reference
	// TODO(ishank011): We need to find a better alternative to this
	// NOTE: did somebody say service accounts? ...
	var user *userpb.User
	if parentInfo.GetOwner().GetType() == userpb.UserType_USER_TYPE_SPACE_OWNER {
		// fake a space owner user
		user = &userpb.User{
			Id: parentInfo.GetOwner(),
		}
	} else {
		userResp, err := client.GetUser(ctx, &userpb.GetUserRequest{UserId: parentInfo.GetOwner(), SkipFetchingUserGroups: true})
		if err != nil || userResp.Status.Code != rpc.Code_CODE_OK {
			return false, err
		}
		user = userResp.User
	}
	scope, err := scope.AddOwnerScope(map[string]*authpb.Scope{})
	if err != nil {
		return false, err
	}
	token, err := mgr.MintToken(ctx, user, scope)
	if err != nil {
		return false, err
	}
	ctx = metadata.AppendToOutgoingContext(context.Background(), ctxpkg.TokenHeader, token)

	childStat, err := client.Stat(ctx, &provider.StatRequest{Ref: ref})
	if err != nil {
		return false, err
	}
	if childStat.GetStatus().GetCode() == rpc.Code_CODE_NOT_FOUND && ref.GetPath() != "" && ref.GetPath() != "." {
		// The resource does not seem to exist (yet?). We might be part of an initiate upload request.
		// Stat the parent to get its path and check that against the root path.
		childStat, err = client.Stat(ctx, &provider.StatRequest{Ref: &provider.Reference{ResourceId: ref.GetResourceId()}})
		if err != nil {
			return false, err
		}
	}
	if childStat.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return false, statuspkg.NewErrorFromCode(childStat.Status.Code, "auth interceptor")
	}
	childInfo := childStat.GetInfo()

	// child can only be a nested resource if it is within the same space as parent
	if childInfo.GetId().GetStorageId() != parentInfo.GetId().GetStorageId() ||
		childInfo.GetId().GetSpaceId() != parentInfo.GetId().GetSpaceId() {
		return false, nil
	}

	// Both resources are in the same space, now check paths
	pathResp, err := client.GetPath(ctx, &provider.GetPathRequest{ResourceId: statResponse.GetInfo().GetId()})
	if err != nil {
		return false, err
	}
	if pathResp.Status.Code != rpc.Code_CODE_OK {
		return false, statuspkg.NewErrorFromCode(pathResp.Status.Code, "auth interceptor")
	}
	parentPath := pathResp.Path

	pathResp, err = client.GetPath(ctx, &provider.GetPathRequest{ResourceId: childStat.GetInfo().GetId()})
	if err != nil {
		return false, err
	}
	if pathResp.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return false, statuspkg.NewErrorFromCode(pathResp.Status.Code, "auth interceptor")
	}
	childPath := pathResp.Path
	rel, err := filepath.Rel(parentPath, childPath)
	if err != nil {
		return false, err
	}
	return !strings.HasPrefix(rel, ".."), nil
}

func extractRefFromListProvidersReq(v *registry.ListStorageProvidersRequest) (*provider.Reference, bool) {
	ref := &provider.Reference{}
	if v.Opaque != nil && v.Opaque.Map != nil {
		if e, ok := v.Opaque.Map["storage_id"]; ok {
			if ref.ResourceId == nil {
				ref.ResourceId = &provider.ResourceId{}
			}
			ref.ResourceId.StorageId = string(e.Value)
		}
		if e, ok := v.Opaque.Map["space_id"]; ok {
			if ref.ResourceId == nil {
				ref.ResourceId = &provider.ResourceId{}
			}
			ref.ResourceId.SpaceId = string(e.Value)
		}
		if e, ok := v.Opaque.Map["opaque_id"]; ok {
			if ref.ResourceId == nil {
				ref.ResourceId = &provider.ResourceId{}
			}
			ref.ResourceId.OpaqueId = string(e.Value)
		}
		if e, ok := v.Opaque.Map["path"]; ok {
			ref.Path = string(e.Value)
		}
	}
	return ref, true
}

func extractRefForReaderRole(req interface{}) (*provider.Reference, bool) {
	switch v := req.(type) {
	// Read requests
	case *registry.GetStorageProvidersRequest:
		return v.GetRef(), true
	case *registry.ListStorageProvidersRequest:
		return extractRefFromListProvidersReq(v)
	case *provider.StatRequest:
		return v.GetRef(), true
	case *provider.ListContainerRequest:
		return v.GetRef(), true
	case *provider.InitiateFileDownloadRequest:
		return v.GetRef(), true

	// App provider requests
	case *appregistry.GetAppProvidersRequest:
		return &provider.Reference{ResourceId: v.ResourceInfo.Id}, true
	case *appprovider.OpenInAppRequest:
		return &provider.Reference{ResourceId: v.ResourceInfo.Id}, true
	case *gateway.OpenInAppRequest:
		return v.GetRef(), true

	// Locking
	case *provider.GetLockRequest:
		return v.GetRef(), true
	case *provider.SetLockRequest:
		return v.GetRef(), true
	case *provider.RefreshLockRequest:
		return v.GetRef(), true
	case *provider.UnlockRequest:
		return v.GetRef(), true

	// OCM shares
	case *ocmv1beta1.ListReceivedOCMSharesRequest:
		return &provider.Reference{Path: "."}, true // we will try to stat the shared node

	}

	return nil, false

}

func extractRefForUploaderRole(req interface{}) (*provider.Reference, bool) {
	switch v := req.(type) {
	// Write Requests
	case *registry.GetStorageProvidersRequest:
		return v.GetRef(), true
	case *registry.ListStorageProvidersRequest:
		return extractRefFromListProvidersReq(v)
	case *provider.StatRequest:
		return v.GetRef(), true
	case *provider.CreateContainerRequest:
		return v.GetRef(), true
	case *provider.TouchFileRequest:
		return v.GetRef(), true
	case *provider.InitiateFileUploadRequest:
		return v.GetRef(), true

	// App provider requests
	case *appregistry.GetAppProvidersRequest:
		return &provider.Reference{ResourceId: v.ResourceInfo.Id}, true
	case *appprovider.OpenInAppRequest:
		return &provider.Reference{ResourceId: v.ResourceInfo.Id}, true
	case *gateway.OpenInAppRequest:
		return v.GetRef(), true

	// Locking
	case *provider.GetLockRequest:
		return v.GetRef(), true
	case *provider.SetLockRequest:
		return v.GetRef(), true
	case *provider.RefreshLockRequest:
		return v.GetRef(), true
	case *provider.UnlockRequest:
		return v.GetRef(), true
	}

	return nil, false

}

func extractRefForEditorRole(req interface{}) (*provider.Reference, bool) {
	switch v := req.(type) {
	// Remaining edit Requests
	case *provider.DeleteRequest:
		return v.GetRef(), true
	case *provider.MoveRequest:
		return v.GetSource(), true
	case *provider.SetArbitraryMetadataRequest:
		return v.GetRef(), true
	case *provider.UnsetArbitraryMetadataRequest:
		return v.GetRef(), true
	}

	return nil, false

}

func extractRef(req interface{}, tokenScope map[string]*authpb.Scope) (*provider.Reference, bool) {
	var readPerm, uploadPerm, editPerm bool
	for _, v := range tokenScope {
		if v.Role == authpb.Role_ROLE_OWNER || v.Role == authpb.Role_ROLE_EDITOR || v.Role == authpb.Role_ROLE_VIEWER {
			readPerm = true
		}
		if v.Role == authpb.Role_ROLE_OWNER || v.Role == authpb.Role_ROLE_EDITOR || v.Role == authpb.Role_ROLE_UPLOADER {
			uploadPerm = true
		}
		if v.Role == authpb.Role_ROLE_OWNER || v.Role == authpb.Role_ROLE_EDITOR {
			editPerm = true
		}
	}

	if readPerm {
		ref, ok := extractRefForReaderRole(req)
		if ok {
			return ref, true
		}
	}
	if uploadPerm {
		ref, ok := extractRefForUploaderRole(req)
		if ok {
			return ref, true
		}
	}
	if editPerm {
		ref, ok := extractRefForEditorRole(req)
		if ok {
			return ref, true
		}
	}

	return nil, false
}

func extractShareRef(req interface{}) (*collaboration.ShareReference, bool) {
	switch v := req.(type) {
	case *collaboration.GetReceivedShareRequest:
		return v.GetRef(), true
	case *collaboration.UpdateReceivedShareRequest:
		return &collaboration.ShareReference{Spec: &collaboration.ShareReference_Id{Id: v.GetShare().GetShare().GetId()}}, true
	}
	return nil, false
}
