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

package readonly

import (
	"context"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	registry "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc"
	rstatus "github.com/opencloud-eu/reva/v3/pkg/rgrpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultPriority = 200
)

func init() {
	rgrpc.RegisterUnaryInterceptor("readonly", NewUnary)
}

// NewUnary returns a new unary interceptor
// that checks grpc calls and blocks write requests.
func NewUnary(map[string]interface{}) (grpc.UnaryServerInterceptor, int, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log := appctx.GetLogger(ctx)

		switch req.(type) {
		// handle known non-write request types
		case *provider.GetHomeRequest,
			*provider.GetPathRequest,
			*provider.GetQuotaRequest,
			*registry.GetStorageProvidersRequest,
			*provider.InitiateFileDownloadRequest,
			*provider.ListFileVersionsRequest,
			*provider.ListGrantsRequest,
			*provider.ListRecycleRequest:
			return handler(ctx, req)
		case *provider.ListContainerRequest:
			resp, err := handler(ctx, req)
			if listResp, ok := resp.(*provider.ListContainerResponse); ok && listResp.Infos != nil {
				for _, info := range listResp.Infos {
					// use the existing PermissionsSet and change the writes to false
					if info.PermissionSet != nil {
						info.PermissionSet.AddGrant = false
						info.PermissionSet.CreateContainer = false
						info.PermissionSet.Delete = false
						info.PermissionSet.InitiateFileUpload = false
						info.PermissionSet.Move = false
						info.PermissionSet.RemoveGrant = false
						info.PermissionSet.PurgeRecycle = false
						info.PermissionSet.RestoreFileVersion = false
						info.PermissionSet.RestoreRecycleItem = false
						info.PermissionSet.UpdateGrant = false
					}
				}
			}
			return resp, err
		case *provider.StatRequest:
			resp, err := handler(ctx, req)
			if statResp, ok := resp.(*provider.StatResponse); ok && statResp.Info != nil && statResp.Info.PermissionSet != nil {
				// use the existing PermissionsSet and change the writes to false
				statResp.Info.PermissionSet.AddGrant = false
				statResp.Info.PermissionSet.CreateContainer = false
				statResp.Info.PermissionSet.Delete = false
				statResp.Info.PermissionSet.InitiateFileUpload = false
				statResp.Info.PermissionSet.Move = false
				statResp.Info.PermissionSet.RemoveGrant = false
				statResp.Info.PermissionSet.PurgeRecycle = false
				statResp.Info.PermissionSet.RestoreFileVersion = false
				statResp.Info.PermissionSet.RestoreRecycleItem = false
				statResp.Info.PermissionSet.UpdateGrant = false
			}
			return resp, err
		// Don't allow the following requests types
		case *provider.AddGrantRequest:
			return &provider.AddGrantResponse{
				Status: rstatus.NewPermissionDenied(ctx, nil, "permission denied: tried to add grant on readonly storage"),
			}, nil
		case *provider.CreateContainerRequest:
			return &provider.CreateContainerResponse{
				Status: rstatus.NewPermissionDenied(ctx, nil, "permission denied: tried to create resource on read-only storage"),
			}, nil
		case *provider.TouchFileRequest:
			return &provider.TouchFileResponse{
				Status: rstatus.NewPermissionDenied(ctx, nil, "permission denied: tried to create resource on read-only storage"),
			}, nil
		case *provider.CreateHomeRequest:
			return &provider.CreateHomeResponse{
				Status: rstatus.NewPermissionDenied(ctx, nil, "permission denied: tried to create home on readonly storage"),
			}, nil
		case *provider.DeleteRequest:
			return &provider.DeleteResponse{
				Status: rstatus.NewPermissionDenied(ctx, nil, "permission denied: tried to delete resource on readonly storage"),
			}, nil
		case *provider.InitiateFileUploadRequest:
			return &provider.InitiateFileUploadResponse{
				Status: rstatus.NewPermissionDenied(ctx, nil, "permission denied: tried to upload resource on readonly storage"),
			}, nil
		case *provider.MoveRequest:
			return &provider.MoveResponse{
				Status: rstatus.NewPermissionDenied(ctx, nil, "permission denied: tried to move resource on readonly storage"),
			}, nil
		case *provider.PurgeRecycleRequest:
			return &provider.PurgeRecycleResponse{
				Status: rstatus.NewPermissionDenied(ctx, nil, "permission denied: tried to purge recycle on readonly storage"),
			}, nil
		case *provider.RemoveGrantRequest:
			return &provider.RemoveGrantResponse{
				Status: rstatus.NewPermissionDenied(ctx, nil, "permission denied: tried to remove grant on readonly storage"),
			}, nil
		case *provider.RestoreRecycleItemRequest:
			return &provider.RestoreRecycleItemResponse{
				Status: rstatus.NewPermissionDenied(ctx, nil, "permission denied: tried to restore recycle item on readonly storage"),
			}, nil
		case *provider.SetArbitraryMetadataRequest:
			return &provider.SetArbitraryMetadataResponse{
				Status: rstatus.NewPermissionDenied(ctx, nil, "permission denied: tried to set arbitrary metadata on readonly storage"),
			}, nil
		case *provider.UnsetArbitraryMetadataRequest:
			return &provider.UnsetArbitraryMetadataResponse{
				Status: rstatus.NewPermissionDenied(ctx, nil, "permission denied: tried to unset arbitrary metadata on readonly storage"),
			}, nil
		// block unknown request types and return error
		default:
			log.Debug().Msg("storage is readonly")
			return nil, status.Errorf(codes.PermissionDenied, "permission denied: tried to execute an unknown operation: %T!", req)
		}
	}, defaultPriority, nil
}
