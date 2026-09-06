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

package scope

import (
	"context"
	"strings"

	appprovider "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	appregistry "github.com/cs3org/go-cs3apis/cs3/app/registry/v1beta1"
	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	permissionsv1beta1 "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	registry "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
)

// PublicStorageProviderID is the space id used for the public links storage space
const PublicStorageProviderID = "7993447f-687f-490d-875c-ac95e89a62a4"

func publicshareScope(ctx context.Context, scope *authpb.Scope, resource interface{}, logger *zerolog.Logger) (bool, error) {
	var share link.PublicShare
	err := utils.UnmarshalJSONToProtoV1(scope.Resource.Value, &share)
	if err != nil {
		return false, err
	}

	switch v := resource.(type) {
	// Viewer role
	case *registry.GetStorageProvidersRequest:
		return checkStorageRef(ctx, &share, v.GetRef()), nil
	case *registry.ListStorageProvidersRequest:
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
		return checkStorageRef(ctx, &share, ref), nil
	case *provider.CreateHomeRequest:
		return false, nil
	case *provider.GetPathRequest:
		return checkStorageRef(ctx, &share, &provider.Reference{ResourceId: v.GetResourceId()}), nil
	case *provider.StatRequest:
		return checkStorageRef(ctx, &share, v.GetRef()), nil
	case *provider.GetLockRequest:
		return checkStorageRef(ctx, &share, v.GetRef()), nil
	case *provider.UnlockRequest:
		return checkStorageRef(ctx, &share, v.GetRef()), nil
	case *provider.RefreshLockRequest:
		return checkStorageRef(ctx, &share, v.GetRef()), nil
	case *provider.SetLockRequest:
		return checkStorageRef(ctx, &share, v.GetRef()), nil
	case *provider.ListContainerRequest:
		return checkStorageRef(ctx, &share, v.GetRef()), nil
	case *provider.InitiateFileDownloadRequest:
		return checkStorageRef(ctx, &share, v.GetRef()), nil
	case *appprovider.OpenInAppRequest:
		return checkStorageRef(ctx, &share, &provider.Reference{ResourceId: v.ResourceInfo.Id}), nil
	case *gateway.OpenInAppRequest:
		return checkStorageRef(ctx, &share, v.GetRef()), nil
	case *permissionsv1beta1.CheckPermissionRequest:
		return true, nil

	// Editor role
	// need to return appropriate status codes in the ocs/ocdav layers.
	case *provider.CreateContainerRequest:
		return hasRoleEditor(scope) && checkStorageRef(ctx, &share, v.GetRef()), nil
	case *provider.TouchFileRequest:
		return hasRoleEditor(scope) && checkStorageRef(ctx, &share, v.GetRef()), nil
	case *provider.DeleteRequest:
		return hasRoleEditor(scope) && checkStorageRef(ctx, &share, v.GetRef()), nil
	case *provider.MoveRequest:
		return hasRoleEditor(scope) && checkStorageRef(ctx, &share, v.GetSource()) && checkStorageRef(ctx, &share, v.GetDestination()), nil
	case *provider.InitiateFileUploadRequest:
		return hasRoleEditor(scope) && checkStorageRef(ctx, &share, v.GetRef()), nil
	case *provider.SetArbitraryMetadataRequest:
		return hasRoleEditor(scope) && checkStorageRef(ctx, &share, v.GetRef()), nil
	case *provider.UnsetArbitraryMetadataRequest:
		return hasRoleEditor(scope) && checkStorageRef(ctx, &share, v.GetRef()), nil

	// App provider requests
	case *appregistry.GetDefaultAppProviderForMimeTypeRequest:
		return true, nil

	case *appregistry.GetAppProvidersRequest:
		return true, nil
	case *userv1beta1.GetUserByClaimRequest:
		return true, nil
	case *userv1beta1.GetUserRequest:
		return true, nil

	case *provider.ListStorageSpacesRequest:
		return true, nil
	case *link.GetPublicShareRequest:
		return checkPublicShareRef(&share, v.GetRef()), nil
	case *link.ListPublicSharesRequest:
		// public links must not leak info about other links
		return false, nil

	case *collaboration.ListReceivedSharesRequest:
		// public links must not leak info about collaborative shares
		return false, nil
	case string:
		return checkResourcePath(v) || checkGraphDrivesPath(v), nil
	}

	msg := "public resource type assertion failed"
	logger.Debug().Str("scope", "publicshareScope").Interface("resource", resource).Msg(msg)
	return false, errtypes.InternalError(msg)
}

// checkGraphDrivesPath opens the graph drive routes; what a token may read is
// enforced per CS3 request.
func checkGraphDrivesPath(path string) bool {
	return strings.HasPrefix(path, "/graph/v1.0/drives/") || strings.HasPrefix(path, "/graph/v1beta1/drives/")
}

func checkStorageRef(ctx context.Context, s *link.PublicShare, r *provider.Reference) bool {
	// r: <resource_id:<storage_id:$storageID space_id:$spaceID opaque_id:$opaqueID> path:$path > >
	if utils.ResourceIDEqual(s.ResourceId, r.GetResourceId()) {
		return true
	}

	// r: <path:"/public/$token" >
	if strings.HasPrefix(r.GetPath(), "/public/"+s.Token) || strings.HasPrefix(r.GetPath(), "./"+s.Token) {
		return true
	}

	// r: <resource_id:<storage_id: space_id: opaque_id:$token> path:$path>
	if id := r.GetResourceId(); id.GetStorageId() == PublicStorageProviderID {
		// access to /public
		if id.GetOpaqueId() == PublicStorageProviderID {
			return true
		}
		// access relative to /public/$token
		if id.GetOpaqueId() == s.Token {
			return true
		}
	}
	return false
}

func checkPublicShareRef(s *link.PublicShare, ref *link.PublicShareReference) bool {
	// ref: <token:$token >
	return ref.GetToken() == s.Token
}

// AddPublicShareScope adds the scope to allow access to a public share and
// the shared resource.
func AddPublicShareScope(share *link.PublicShare, role authpb.Role, scopes map[string]*authpb.Scope) (map[string]*authpb.Scope, error) {
	// Create a new "scope share" to only expose the required fields `ResourceId` and `Token` to the scope.
	scopeShare := &link.PublicShare{ResourceId: share.ResourceId, Token: share.Token}
	val, err := utils.MarshalProtoV1ToJSON(scopeShare)
	if err != nil {
		return nil, err
	}
	if scopes == nil {
		scopes = make(map[string]*authpb.Scope)
	}
	scopes["publicshare:"+share.Id.OpaqueId] = &authpb.Scope{
		Resource: &types.OpaqueEntry{
			Decoder: "json",
			Value:   val,
		},
		Role: role,
	}
	return scopes, nil
}
