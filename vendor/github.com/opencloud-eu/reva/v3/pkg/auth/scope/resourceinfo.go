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
	"path"
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

// checkResourceInfo checks whether ref references the resource of the scope,
// or a resource below it.
//
// Only references with the id of the resource are accepted, with an optional
// relative path that doesn't leave the resource. References with a path only
// are rejected: the path of the resource in the scope is whatever the storage
// provider returned for it (e.g. only its name for a relative reference), while
// the path of a path only reference is in the namespace of the gateway, so the
// two can't be compared.
func checkResourceInfo(inf *provider.ResourceInfo, ref *provider.Reference) bool {
	id := ref.GetResourceId()
	if id == nil || inf.GetId().GetSpaceId() == "" || inf.GetId().GetOpaqueId() == "" {
		return false
	}
	if id.GetSpaceId() != inf.GetId().GetSpaceId() || id.GetOpaqueId() != inf.GetId().GetOpaqueId() {
		return false
	}
	return isPathInsideResource(ref.GetPath())
}

// isPathInsideResource checks whether p, the path of an id based reference, is
// the resource itself ("" or ".") or a path below it ("./a", "a/b"). Absolute
// paths and paths leaving the resource ("..", "./a/../../b") are rejected.
func isPathInsideResource(p string) bool {
	if p == "" {
		return true
	}
	if strings.HasPrefix(p, "/") {
		return false
	}
	p = path.Clean(p)
	return p != ".." && !strings.HasPrefix(p, "../")
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

// AddResourceInfoScope adds the scope to allow access to the resource with the
// id of r, and to the resources below it. r must have a complete id.
//
// Only the id is stored in the scope: the path of a resource info is no stable
// address of the resource, and its form depends on how it was looked up.
func AddResourceInfoScope(r *provider.ResourceInfo, role authpb.Role, scopes map[string]*authpb.Scope) (map[string]*authpb.Scope, error) {
	if r.GetId().GetSpaceId() == "" || r.GetId().GetOpaqueId() == "" {
		return nil, errtypes.BadRequest("resourceinfo scope: resource info without a complete id")
	}
	scopeInfo := &provider.ResourceInfo{Id: r.GetId()}
	val, err := utils.MarshalProtoV1ToJSON(scopeInfo)
	if err != nil {
		return nil, err
	}
	if scopes == nil {
		scopes = make(map[string]*authpb.Scope)
	}
	scopes["resourceinfo:"+r.GetId().String()] = &authpb.Scope{
		Resource: &types.OpaqueEntry{
			Decoder: "json",
			Value:   val,
		},
		Role: role,
	}
	return scopes, nil
}
