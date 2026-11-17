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
	"fmt"
	"strings"

	appprovider "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	registry "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	"github.com/rs/zerolog"

	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
)

func resourceinfoScope(_ context.Context, scope *authpb.Scope, resource interface{}, logger *zerolog.Logger) (bool, error) {
	var r provider.ResourceInfo
	err := utils.UnmarshalJSONToProtoV1(scope.Resource.Value, &r)
	if err != nil {
		return false, err
	}

	switch v := resource.(type) {
	// Viewer role
	case *registry.GetStorageProvidersRequest:
		return checkResourceInfo(&r, v.GetRef()), nil
	case *registry.ListStorageProvidersRequest:
		// the call will only return spaces the current user has access to
		ref := &provider.Reference{}
		if v.Opaque != nil && v.Opaque.Map != nil {
			if e, ok := v.Opaque.Map["storage_id"]; ok {
				ref.ResourceId = &provider.ResourceId{
					StorageId: string(e.Value),
				}
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
		return checkResourceInfo(&r, ref), nil
	case *provider.ListStorageSpacesRequest:
		// the call will only return spaces the current user has access to
		return true, nil
	case *provider.StatRequest:
		return checkResourceInfo(&r, v.GetRef()), nil
	case *provider.ListContainerRequest:
		return checkResourceInfo(&r, v.GetRef()), nil
	case *provider.InitiateFileDownloadRequest:
		return checkResourceInfo(&r, v.GetRef()), nil
	case *appprovider.OpenInAppRequest:
		return checkResourceInfo(&r, &provider.Reference{ResourceId: v.ResourceInfo.Id}), nil
	case *gateway.OpenInAppRequest:
		return checkResourceInfo(&r, v.GetRef()), nil

	// Editor role
	// need to return appropriate status codes in the ocs/ocdav layers.
	case *provider.CreateContainerRequest:
		return hasRoleEditor(scope) && checkResourceInfo(&r, v.GetRef()), nil
	case *provider.TouchFileRequest:
		return hasRoleEditor(scope) && checkResourceInfo(&r, v.GetRef()), nil
	case *provider.DeleteRequest:
		return hasRoleEditor(scope) && checkResourceInfo(&r, v.GetRef()), nil
	case *provider.MoveRequest:
		return hasRoleEditor(scope) && checkResourceInfo(&r, v.GetSource()) && checkResourceInfo(&r, v.GetDestination()), nil
	case *provider.InitiateFileUploadRequest:
		return hasRoleEditor(scope) && checkResourceInfo(&r, v.GetRef()), nil
	case *provider.SetArbitraryMetadataRequest:
		return hasRoleEditor(scope) && checkResourceInfo(&r, v.GetRef()), nil
	case *provider.UnsetArbitraryMetadataRequest:
		return hasRoleEditor(scope) && checkResourceInfo(&r, v.GetRef()), nil

	case string:
		return checkResourcePath(v), nil
	}

	msg := fmt.Sprintf("resource type assertion failed: %+v", resource)
	logger.Debug().Str("scope", "resourceinfoScope").Msg(msg)
	return false, errtypes.InternalError(msg)
}

func checkResourceInfo(inf *provider.ResourceInfo, ref *provider.Reference) bool {
	// ref: <resource_id:<storage_id:$storageID opaque_id:$opaqueID path:$path> >
	if ref.ResourceId != nil { // path can be empty or a relative path
		if inf.Id.SpaceId == ref.ResourceId.SpaceId && inf.Id.OpaqueId == ref.ResourceId.OpaqueId {
			if ref.Path == "" {
				// id only reference
				return true
			}
			// check path has same prefix below
		} else {
			return false
		}
	}
	// ref: <path:$path >
	if strings.HasPrefix(ref.GetPath(), inf.Path) {
		return true
	}
	return false
}

func checkResourcePath(path string) bool {
	paths := []string{
		"/dataprovider",
		"/data",
		"/app/open",
		"/app/new",
		"/archiver",
		"/ocs/v2.php/cloud/capabilities",
		"/ocs/v1.php/cloud/capabilities",
	}
	for _, p := range paths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// AddResourceInfoScope adds the scope to allow access to a resource info object.
func AddResourceInfoScope(r *provider.ResourceInfo, role authpb.Role, scopes map[string]*authpb.Scope) (map[string]*authpb.Scope, error) {
	// Create a new "scope info" to only expose the required fields `Id` and `Path` to the scope.
	scopeInfo := &provider.ResourceInfo{Id: r.Id, Path: r.Path}
	val, err := utils.MarshalProtoV1ToJSON(scopeInfo)
	if err != nil {
		return nil, err
	}
	if scopes == nil {
		scopes = make(map[string]*authpb.Scope)
	}
	scopes["resourceinfo:"+r.Id.String()] = &authpb.Scope{
		Resource: &types.OpaqueEntry{
			Decoder: "json",
			Value:   val,
		},
		Role: role,
	}
	return scopes, nil
}
