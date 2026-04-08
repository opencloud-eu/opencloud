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

package sharesstorageprovider

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	gstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/share"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
)

const (
	_defaultSharesJailEtag = "DECAFC00FEE"
)

func init() {
	rgrpc.Register("sharesstorageprovider", NewDefault)
}

type config struct {
	GatewayAddr               string `mapstructure:"gateway_addr"`
	UserShareProviderEndpoint string `mapstructure:"usershareprovidersvc"`
	MaxConcurrency            int    `mapstructure:"max_concurrency"`
}

type service struct {
	gatewaySelector              pool.Selectable[gateway.GatewayAPIClient]
	sharingCollaborationSelector pool.Selectable[collaboration.CollaborationAPIClient]
	maxConcurrency               int
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

// NewDefault returns a new instance of the SharesStorageProvider service with default dependencies
func NewDefault(m map[string]interface{}, _ *grpc.Server, _ *zerolog.Logger) (rgrpc.Service, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}

	gatewaySelector, err := pool.GatewaySelector(sharedconf.GetGatewaySVC(c.GatewayAddr))
	if err != nil {
		return nil, err
	}

	sharingCollaborationSelector, err := pool.SharingCollaborationSelector(sharedconf.GetGatewaySVC(c.UserShareProviderEndpoint))
	if err != nil {
		return nil, errors.Wrap(err, "sharesstorageprovider: error getting UserShareProvider client")
	}

	if c.MaxConcurrency <= 0 {
		c.MaxConcurrency = 5
	}

	return New(gatewaySelector, sharingCollaborationSelector, c.MaxConcurrency)
}

// New returns a new instance of the SharesStorageProvider service
func New(gatewaySelector pool.Selectable[gateway.GatewayAPIClient], sharingCollaborationSelector pool.Selectable[collaboration.CollaborationAPIClient], maxConcurrency int) (rgrpc.Service, error) {
	s := &service{
		gatewaySelector:              gatewaySelector,
		sharingCollaborationSelector: sharingCollaborationSelector,
		maxConcurrency:               maxConcurrency,
	}
	return s, nil
}

func (s *service) SetArbitraryMetadata(ctx context.Context, req *provider.SetArbitraryMetadataRequest) (*provider.SetArbitraryMetadataResponse, error) {
	receivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, req.Ref)
	appctx.GetLogger(ctx).Debug().
		Interface("ref", req.Ref).
		Interface("received_share", receivedShare).
		Msg("sharesstorageprovider: Got SetArbitraryMetadata request")
	if err != nil {
		return nil, err
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.SetArbitraryMetadataResponse{
			Status: rpcStatus,
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	return gatewayClient.SetArbitraryMetadata(ctx, &provider.SetArbitraryMetadataRequest{
		Opaque:            req.Opaque,
		Ref:               buildReferenceInShare(req.Ref, receivedShare),
		ArbitraryMetadata: req.ArbitraryMetadata,
	})
}

func (s *service) UnsetArbitraryMetadata(ctx context.Context, req *provider.UnsetArbitraryMetadataRequest) (*provider.UnsetArbitraryMetadataResponse, error) {
	receivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, req.Ref)
	appctx.GetLogger(ctx).Debug().
		Interface("ref", req.Ref).
		Interface("received_share", receivedShare).
		Msg("sharesstorageprovider: Got UnsetArbitraryMetadata request")
	if err != nil {
		return nil, err
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.UnsetArbitraryMetadataResponse{
			Status: rpcStatus,
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	return gatewayClient.UnsetArbitraryMetadata(ctx, &provider.UnsetArbitraryMetadataRequest{
		Opaque:                req.Opaque,
		Ref:                   buildReferenceInShare(req.Ref, receivedShare),
		ArbitraryMetadataKeys: req.ArbitraryMetadataKeys,
	})
}

func (s *service) InitiateFileDownload(ctx context.Context, req *provider.InitiateFileDownloadRequest) (*provider.InitiateFileDownloadResponse, error) {
	receivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, req.Ref)
	appctx.GetLogger(ctx).Debug().
		Interface("ref", req.Ref).
		Interface("received_share", receivedShare).
		Msg("sharesstorageprovider: Got InitiateFileDownload request")
	if err != nil {
		return nil, err
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.InitiateFileDownloadResponse{
			Status: rpcStatus,
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	gwres, err := gatewayClient.InitiateFileDownload(ctx, &provider.InitiateFileDownloadRequest{
		Opaque: req.Opaque,
		Ref:    buildReferenceInShare(req.Ref, receivedShare),
		LockId: req.LockId,
	})
	if err != nil {
		return nil, err
	}
	if gwres.Status.Code != rpc.Code_CODE_OK {
		return &provider.InitiateFileDownloadResponse{
			Status: gwres.Status,
		}, nil
	}

	protocols := []*provider.FileDownloadProtocol{}
	for p := range gwres.Protocols {
		if !strings.HasSuffix(gwres.Protocols[p].DownloadEndpoint, "/") {
			gwres.Protocols[p].DownloadEndpoint += "/"
		}
		gwres.Protocols[p].DownloadEndpoint += gwres.Protocols[p].Token

		protocols = append(protocols, &provider.FileDownloadProtocol{
			Opaque:           gwres.Protocols[p].Opaque,
			Protocol:         gwres.Protocols[p].Protocol,
			DownloadEndpoint: gwres.Protocols[p].DownloadEndpoint,
			Expose:           true, // the gateway already has encoded the upload endpoint
		})
	}

	return &provider.InitiateFileDownloadResponse{
		Opaque:    gwres.GetOpaque(),
		Status:    gwres.Status,
		Protocols: protocols,
	}, nil

}

func (s *service) InitiateFileUpload(ctx context.Context, req *provider.InitiateFileUploadRequest) (*provider.InitiateFileUploadResponse, error) {
	receivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, req.Ref)
	appctx.GetLogger(ctx).Debug().
		Interface("ref", req.Ref).
		Interface("received_share", receivedShare).
		Msg("sharesstorageprovider: Got InitiateFileUpload request")
	switch {
	case err != nil:
		return nil, err
	case rpcStatus.Code == rpc.Code_CODE_NOT_FOUND:
		// the user has access (it showed up in the clist of shares), but we cannot write here
		return &provider.InitiateFileUploadResponse{
			Status: status.NewFailedPrecondition(ctx, nil, rpcStatus.GetMessage()),
		}, nil
	case rpcStatus.Code != rpc.Code_CODE_OK:
		return &provider.InitiateFileUploadResponse{
			Status: rpcStatus,
		}, nil
	}

	if !receivedShare.GetShare().GetPermissions().GetPermissions().GetInitiateFileUpload() {
		return &provider.InitiateFileUploadResponse{
			Status: status.NewPermissionDenied(ctx, nil, "share does not grant InitiateFileDownload permission"),
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	gwres, err := gatewayClient.InitiateFileUpload(ctx, &provider.InitiateFileUploadRequest{
		Opaque:  req.Opaque,
		Ref:     buildReferenceInShare(req.Ref, receivedShare),
		LockId:  req.LockId,
		Options: req.Options,
	})
	if err != nil {
		return nil, err
	}
	if gwres.Status.Code != rpc.Code_CODE_OK {
		return &provider.InitiateFileUploadResponse{
			Status: gwres.Status,
		}, nil
	}

	protocols := []*provider.FileUploadProtocol{}
	for p := range gwres.Protocols {
		if !strings.HasSuffix(gwres.Protocols[p].UploadEndpoint, "/") {
			gwres.Protocols[p].UploadEndpoint += "/"
		}
		gwres.Protocols[p].UploadEndpoint += gwres.Protocols[p].Token

		protocols = append(protocols, &provider.FileUploadProtocol{
			Opaque:             gwres.Protocols[p].Opaque,
			Protocol:           gwres.Protocols[p].Protocol,
			UploadEndpoint:     gwres.Protocols[p].UploadEndpoint,
			AvailableChecksums: gwres.Protocols[p].AvailableChecksums,
			Expose:             true, // the gateway already has encoded the upload endpoint
		})
	}
	return &provider.InitiateFileUploadResponse{
		Opaque:    gwres.GetOpaque(),
		Status:    gwres.Status,
		Protocols: protocols,
	}, nil
}

func (s *service) GetPath(ctx context.Context, req *provider.GetPathRequest) (*provider.GetPathResponse, error) {
	// TODO: Needs to find a path for a given resourceID
	// It should
	// - getPath of the resourceID - probably requires owner permissions -> needs machine auth
	// - getPath of every received share on the same space - needs also owner permissions -> needs machine auth
	// - find the shortest root path that is a prefix of the resource path
	// alternatively implement this on storageprovider - it needs to know about grants to do so

	if isShareJailRoot(req.ResourceId) {
		return &provider.GetPathResponse{
			Status: status.NewOK(ctx),
			Path:   "/",
		}, nil
	}

	receivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, &provider.Reference{
		ResourceId: req.ResourceId,
	})
	appctx.GetLogger(ctx).Debug().
		Interface("resourceId", req.ResourceId).
		Interface("received_share", receivedShare).
		Msg("sharesstorageprovider: Got GetPath request")
	if err != nil {
		return nil, err
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.GetPathResponse{
			Status: rpcStatus,
		}, nil
	}

	return &provider.GetPathResponse{
		Status: status.NewOK(ctx),
		Path:   filepath.Clean("/" + receivedShare.MountPoint.Path),
	}, nil

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

// ListStorageSpaces returns a list storage spaces with type "share" the current user has acces to.
// Do owners of shares see type "shared"? Do they see andyhing? They need to if the want a fast lookup of shared with others
// -> but then a storage sprovider has to do everything? not everything but permissions (= shares) related operations, yes
// The root node of every storag space is the (spaceid, nodeid) of the shared node.
// Since real space roots have (spaceid=nodeid) shares can be correlated with the space using the (spaceid, ) part of the reference.

// However, when the space registry tries
// to find a storage provider for a specific space it returns an empty list, so the actual storage provider
// should be found.

func (s *service) ListStorageSpaces(ctx context.Context, req *provider.ListStorageSpacesRequest) (*provider.ListStorageSpacesResponse, error) {
	spaceTypes := map[string]struct{}{}
	var exists = struct{}{}
	var fetchShares bool
	appendTypes := []string{}
	var spaceID *provider.ResourceId
	for _, f := range req.Filters {
		switch f.Type {
		case provider.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE:
			spaceType := f.GetSpaceType()
			// do we need to fetch the shares?
			if spaceType == "+mountpoint" || spaceType == "+grant" {
				appendTypes = append(appendTypes, strings.TrimPrefix(spaceType, "+"))
				fetchShares = true
				continue
			}
			if spaceType == "mountpoint" || spaceType == "grant" {
				fetchShares = true
			}
			spaceTypes[spaceType] = exists
		case provider.ListStorageSpacesRequest_Filter_TYPE_ID:
			storageid, spaceid, shareid, err := storagespace.SplitID(f.GetId().OpaqueId)
			if err != nil {
				continue
			}
			if spaceid != utils.ShareStorageSpaceID {
				return &provider.ListStorageSpacesResponse{
					// a specific id was requested, return not found instead of empty list
					Status: &rpc.Status{Code: rpc.Code_CODE_NOT_FOUND},
				}, nil
			}

			spaceID = &provider.ResourceId{StorageId: storageid, SpaceId: spaceid, OpaqueId: shareid}
		}
	}

	if len(spaceTypes) == 0 {
		spaceTypes["virtual"] = exists
		spaceTypes["mountpoint"] = exists
		fetchShares = true
	}

	for _, s := range appendTypes {
		spaceTypes[s] = exists
	}

	var receivedShares []*collaboration.ReceivedShare
	var shareInfo map[string]*provider.ResourceInfo
	var err error
	if fetchShares {
		receivedShares, shareInfo, err = s.fetchAcceptedShares(ctx, req.Opaque, []string{}, &fieldmaskpb.FieldMask{ /*TODO mtime and etag only?*/ })
		if err != nil {
			return nil, errors.Wrap(err, "sharesstorageprovider: error calling ListReceivedSharesRequest")
		}
	}

	res := &provider.ListStorageSpacesResponse{
		Status: status.NewOK(ctx),
	}
	for k := range spaceTypes {
		switch k {
		case "virtual":
			virtualRootID := &provider.ResourceId{
				StorageId: utils.ShareStorageProviderID,
				SpaceId:   utils.ShareStorageSpaceID,
				OpaqueId:  utils.ShareStorageSpaceID,
			}
			if spaceID == nil || isShareJailRoot(spaceID) {
				earliestShare := findEarliestShare(receivedShares, shareInfo)
				var opaque *typesv1beta1.Opaque
				var mtime *typesv1beta1.Timestamp
				if earliestShare != nil {
					if info, ok := shareInfo[earliestShare.GetId().GetOpaqueId()]; ok {
						mtime = info.Mtime
						opaque = utils.AppendPlainToOpaque(opaque, "etag", info.Etag)
					}
				} else {
					opaque = utils.AppendPlainToOpaque(opaque, "etag", _defaultSharesJailEtag)
				}
				// only display the shares jail if we have accepted shares
				opaque = utils.AppendPlainToOpaque(opaque, "spaceAlias", "virtual/shares")
				space := &provider.StorageSpace{
					Opaque: opaque,
					Id: &provider.StorageSpaceId{
						OpaqueId: storagespace.FormatResourceID(virtualRootID),
					},
					SpaceType: "virtual",
					//Owner:     &userv1beta1.User{Id: receivedShare.Share.Owner}, // FIXME actually, the mount point belongs to the recipient
					// the sharesstorageprovider keeps track of mount points
					Root:  virtualRootID,
					Name:  "Shares",
					Mtime: mtime,
				}
				res.StorageSpaces = append(res.StorageSpaces, space)
			}
		case "grant":
			for _, receivedShare := range receivedShares {
				root := receivedShare.Share.ResourceId
				// do we filter by id?
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
					Owner:     &userv1beta1.User{Id: receivedShare.Share.Owner},
					// the sharesstorageprovider keeps track of mount points
					Root:     root,
					RootInfo: shareInfo[receivedShare.Share.Id.OpaqueId],
				}

				res.StorageSpaces = append(res.StorageSpaces, space)
			}
		case "mountpoint":
			for _, receivedShare := range receivedShares {
				if receivedShare.State != collaboration.ShareState_SHARE_STATE_ACCEPTED {
					continue
				}
				root := &provider.ResourceId{
					StorageId: utils.ShareStorageProviderID,
					SpaceId:   utils.ShareStorageSpaceID,
					OpaqueId:  receivedShare.Share.Id.OpaqueId,
				}
				// do we filter by id
				if spaceID != nil {
					switch {
					case utils.ResourceIDEqual(spaceID, root):
						// we have a virtual node
					case utils.ResourceIDEqual(spaceID, receivedShare.Share.ResourceId):
						// we have a mount point
						root = receivedShare.Share.ResourceId
					default:
						// none of our business
						continue
					}
				}
				var opaque *typesv1beta1.Opaque
				if _, ok := shareInfo[receivedShare.Share.Id.OpaqueId]; !ok {
					// we could not stat the share, skip it
					continue
				}
				// add the resourceID for the grant
				if receivedShare.Share.ResourceId != nil {
					opaque = utils.AppendPlainToOpaque(opaque, "grantStorageID", receivedShare.Share.ResourceId.StorageId)
					opaque = utils.AppendPlainToOpaque(opaque, "grantSpaceID", receivedShare.Share.ResourceId.SpaceId)
					opaque = utils.AppendPlainToOpaque(opaque, "grantOpaqueID", receivedShare.Share.ResourceId.OpaqueId)
				}

				// prefix storageid if we are responsible
				if root.SpaceId == utils.ShareStorageSpaceID {
					root.StorageId = utils.ShareStorageProviderID
				}

				space := &provider.StorageSpace{
					Opaque: opaque,
					Id: &provider.StorageSpaceId{
						OpaqueId: storagespace.FormatResourceID(root),
					},
					SpaceType: "mountpoint",
					Owner:     &userv1beta1.User{Id: receivedShare.Share.Owner}, // FIXME actually, the mount point belongs to the recipient
					// the sharesstorageprovider keeps track of mount points
					Root:     root,
					RootInfo: shareInfo[receivedShare.Share.Id.OpaqueId],
				}

				// TODO in the future the spaces registry will handle the alias for share spaces.
				// for now use the name from the share to override the name determined by stat
				if receivedShare.MountPoint != nil {
					space.Name = receivedShare.MountPoint.Path
					space.Opaque = utils.AppendPlainToOpaque(space.Opaque, "spaceAlias", space.SpaceType+"/"+strings.ReplaceAll(strings.ToLower(space.Name), " ", "-"))
				}

				// what if we don't have a name?
				res.StorageSpaces = append(res.StorageSpaces, space)
			}
		}
	}
	return res, nil
}

func (s *service) UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) (*provider.DeleteStorageSpaceResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) CreateContainer(ctx context.Context, req *provider.CreateContainerRequest) (*provider.CreateContainerResponse, error) {
	receivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, req.Ref)
	appctx.GetLogger(ctx).Debug().
		Interface("ref", req.Ref).
		Interface("received_share", receivedShare).
		Msg("sharesstorageprovider: Got CreateContainer request")
	if err != nil {
		return nil, err
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.CreateContainerResponse{
			Status: rpcStatus,
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	return gatewayClient.CreateContainer(ctx, &provider.CreateContainerRequest{
		Opaque: req.Opaque,
		Ref:    buildReferenceInShare(req.Ref, receivedShare),
	})
}

func (s *service) Delete(ctx context.Context, req *provider.DeleteRequest) (*provider.DeleteResponse, error) {
	receivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, req.Ref)
	appctx.GetLogger(ctx).Debug().
		Interface("ref", req.Ref).
		Interface("received_share", receivedShare).
		Err(err).
		Msg("sharesstorageprovider: Got Delete request")
	if err != nil {
		return nil, err
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.DeleteResponse{
			Status: rpcStatus,
		}, nil
	}

	// the root of a share always has the path "."
	if req.Ref.ResourceId.StorageId == utils.ShareStorageProviderID && req.Ref.ResourceId.SpaceId == utils.ShareStorageSpaceID && req.Ref.Path == "." {
		err := s.rejectReceivedShare(ctx, receivedShare)
		if err != nil {
			return &provider.DeleteResponse{
				Status: status.NewInternal(ctx, "sharesstorageprovider: error rejecting share"),
			}, nil
		}
		return &provider.DeleteResponse{
			Status: status.NewOK(ctx),
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	return gatewayClient.Delete(ctx, &provider.DeleteRequest{
		Opaque: req.Opaque,
		Ref:    buildReferenceInShare(req.Ref, receivedShare),
	})
}

func (s *service) Move(ctx context.Context, req *provider.MoveRequest) (*provider.MoveResponse, error) {
	appctx.GetLogger(ctx).Debug().
		Interface("source", req.Source).
		Interface("destination", req.Destination).
		Msg("sharesstorageprovider: Got Move request")

	// TODO moving inside a shared tree should just be a forward of the move
	//      but when do we rename a mounted share? Does that request even hit us?
	//      - the registry needs to invalidate the alias
	//      - the rhe share manager needs to change the name
	//      ... but which storageprovider will receive the move request???
	srcReceivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, req.Source)
	if err != nil {
		return nil, err
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.MoveResponse{
			Status: rpcStatus,
		}, nil
	}

	// we can do a rename
	if isRename(req.Source, req.Destination) {

		// Change the MountPoint of the share, it has no relative prefix
		srcReceivedShare.MountPoint = &provider.Reference{
			// FIXME actually it does have a resource id: the one of the sharesstorageprovider
			Path: filepath.Base(req.Destination.Path),
		}

		sharingCollaborationClient, err := s.sharingCollaborationSelector.Next()
		if err != nil {
			return nil, err
		}

		_, err = sharingCollaborationClient.UpdateReceivedShare(ctx, &collaboration.UpdateReceivedShareRequest{
			Share:      srcReceivedShare,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state", "mount_point"}},
		})
		if err != nil {
			return &provider.MoveResponse{
				Status: status.NewInternal(ctx, "sharesstorageprovider: can not change mountpoint of share"),
			}, nil
		}
		return &provider.MoveResponse{
			Status: status.NewOK(ctx),
		}, nil
	}

	dstReceivedShare, rpcStatus, err2 := s.resolveAcceptedShare(ctx, req.Destination)
	if err2 != nil {
		return nil, err2
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.MoveResponse{
			Status: rpcStatus,
		}, nil
	}

	if dstReceivedShare.Share.Id.OpaqueId != srcReceivedShare.Share.Id.OpaqueId {
		return &provider.MoveResponse{
			Status: status.NewUnimplemented(ctx, nil, "cross storage moves are not supported, use copy and delete"),
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	return gatewayClient.Move(ctx, &provider.MoveRequest{
		Opaque:      req.Opaque,
		Source:      buildReferenceInShare(req.Source, srcReceivedShare),
		Destination: buildReferenceInShare(req.Destination, dstReceivedShare),
	})
}

// SetLock puts a lock on the given reference
func (s *service) SetLock(ctx context.Context, req *provider.SetLockRequest) (*provider.SetLockResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

// GetLock returns an existing lock on the given reference
func (s *service) GetLock(ctx context.Context, req *provider.GetLockRequest) (*provider.GetLockResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

// RefreshLock refreshes an existing lock on the given reference
func (s *service) RefreshLock(ctx context.Context, req *provider.RefreshLockRequest) (*provider.RefreshLockResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

// Unlock removes an existing lock from the given reference
func (s *service) Unlock(ctx context.Context, req *provider.UnlockRequest) (*provider.UnlockResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) Stat(ctx context.Context, req *provider.StatRequest) (*provider.StatResponse, error) {
	if isVirtualRoot(req.Ref) {
		owner, ok := ctxpkg.ContextGetUser(ctx)
		if !ok {
			return nil, fmt.Errorf("missing user in context")
		}
		receivedShares, shareMd, err := s.fetchAcceptedShares(ctx, req.Opaque, req.ArbitraryMetadataKeys, req.FieldMask)
		if err != nil {
			return nil, err
		}
		earliestShare := findEarliestShare(receivedShares, shareMd)
		var mtime *typesv1beta1.Timestamp
		etag := _defaultSharesJailEtag
		if earliestShare != nil {
			if info, ok := shareMd[earliestShare.GetId().GetOpaqueId()]; ok {
				mtime = info.Mtime
				etag = info.Etag
			}
		}
		return &provider.StatResponse{
			Status: status.NewOK(ctx),
			Info: &provider.ResourceInfo{
				Opaque: &typesv1beta1.Opaque{
					Map: map[string]*typesv1beta1.OpaqueEntry{
						"root": {
							Decoder: "plain",
							Value:   []byte(utils.ShareStorageProviderID),
						},
					},
				},
				Id: &provider.ResourceId{
					StorageId: utils.ShareStorageProviderID,
					SpaceId:   utils.ShareStorageSpaceID,
					OpaqueId:  utils.ShareStorageSpaceID,
				},
				Type:          provider.ResourceType_RESOURCE_TYPE_CONTAINER,
				Mtime:         mtime,
				Path:          "/",
				MimeType:      "httpd/unix-directory",
				Size:          0,
				PermissionSet: &provider.ResourcePermissions{
					// TODO
				},
				Space: &provider.StorageSpace{
					SpaceType: "virtual",
				},
				Etag:  etag,
				Owner: owner.Id,
			},
		}, nil
	}
	receivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, req.Ref)
	appctx.GetLogger(ctx).Debug().
		Interface("ref", req.Ref).
		Interface("received_share", receivedShare).
		Err(err).
		Msg("sharesstorageprovider: Got Stat request")
	if err != nil {
		return nil, err
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.StatResponse{
			Status: rpcStatus,
		}, nil
	}
	if receivedShare.State != collaboration.ShareState_SHARE_STATE_ACCEPTED {
		return &provider.StatResponse{
			Status: &rpc.Status{Code: rpc.Code_CODE_NOT_FOUND},
			// not mounted yet
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	statRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{
		Opaque:                req.Opaque,
		Ref:                   buildReferenceInShare(req.Ref, receivedShare),
		ArbitraryMetadataKeys: req.ArbitraryMetadataKeys,
	})
	if err != nil {
		return nil, err
	}

	// when stating a share jail mountpoint we need to rewrite the ids
	if statRes.GetStatus().GetCode() == rpc.Code_CODE_OK && receivedShare.MountPoint.Path == strings.TrimPrefix(req.Ref.Path, "./") && statRes.Info != nil {
		// overwrite id with the share jail mountpoint id
		statRes.Info.Id = &provider.ResourceId{
			StorageId: utils.ShareStorageProviderID,
			SpaceId:   utils.ShareStorageSpaceID,
			OpaqueId:  receivedShare.GetShare().GetId().GetOpaqueId(),
		}
		// overwrite parent id with the share jail root
		statRes.Info.ParentId = &provider.ResourceId{
			StorageId: utils.ShareStorageProviderID,
			SpaceId:   utils.ShareStorageSpaceID,
			OpaqueId:  utils.ShareStorageSpaceID,
		}
	}

	return statRes, nil
}

func (s *service) ListContainerStream(req *provider.ListContainerStreamRequest, ss provider.ProviderAPI_ListContainerStreamServer) error {
	return gstatus.Errorf(codes.Unimplemented, "method not implemented")
}
func (s *service) ListContainer(ctx context.Context, req *provider.ListContainerRequest) (*provider.ListContainerResponse, error) {
	if isVirtualRoot(req.Ref) {
		// The root is empty, it is filled by mountpoints
		// so, when accessing the root via /dav/spaces, we need to list the accepted shares with their mountpoint

		receivedShares, shareMd, err := s.fetchAcceptedShares(ctx, req.Opaque, req.ArbitraryMetadataKeys, req.FieldMask)
		if err != nil {
			return nil, errors.Wrap(err, "sharesstorageprovider: error calling ListReceivedSharesRequest")
		}

		// Create map of shares that contains only the oldest share per shared resource. This is to avoid
		// returning multiple resourceInfos for the same resource. But still be able to maintain a
		// "somewhat" stable resourceID
		oldestReceivedSharesByResourceID := make(map[string]*collaboration.ReceivedShare, len(receivedShares))
		for _, receivedShare := range receivedShares {
			if receivedShare.GetState() != collaboration.ShareState_SHARE_STATE_ACCEPTED {
				continue
			}
			rIDStr := storagespace.FormatResourceID(receivedShare.GetShare().GetResourceId())
			if oldest, ok := oldestReceivedSharesByResourceID[rIDStr]; ok {
				// replace if older than current oldest
				if utils.TSToTime(receivedShare.GetShare().GetCtime()).Before(utils.TSToTime(oldest.GetShare().GetCtime())) {
					oldestReceivedSharesByResourceID[rIDStr] = receivedShare
				}
			} else {
				oldestReceivedSharesByResourceID[rIDStr] = receivedShare
			}
		}

		// now compose the resourceInfos for the unified list of shares
		infos := []*provider.ResourceInfo{}
		for _, share := range oldestReceivedSharesByResourceID {
			info := shareMd[share.GetShare().GetId().GetOpaqueId()]
			if info == nil {
				appctx.GetLogger(ctx).Debug().
					Interface("share", share).
					Msg("sharesstorageprovider: no resource info for share")
				continue
			}

			// override resource id info
			info.Id = &provider.ResourceId{
				StorageId: utils.ShareStorageProviderID,
				SpaceId:   utils.ShareStorageSpaceID,
				OpaqueId:  share.GetShare().GetId().GetOpaqueId(),
			}
			info.Path = filepath.Base(share.MountPoint.Path)
			info.Name = info.Path

			infos = append(infos, info)
		}
		return &provider.ListContainerResponse{
			Status: status.NewOK(ctx),
			Infos:  infos,
		}, nil
	}
	receivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, req.Ref)
	appctx.GetLogger(ctx).Debug().
		Interface("ref", req.Ref).
		Interface("received_share", receivedShare).
		Err(err).
		Msg("sharesstorageprovider: Got ListContainer request")
	if err != nil {
		return nil, err
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.ListContainerResponse{
			Status: rpcStatus,
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	return gatewayClient.ListContainer(ctx, &provider.ListContainerRequest{
		Opaque:                req.Opaque,
		Ref:                   buildReferenceInShare(req.Ref, receivedShare),
		ArbitraryMetadataKeys: req.ArbitraryMetadataKeys,
	})
}
func (s *service) ListFileVersions(ctx context.Context, req *provider.ListFileVersionsRequest) (*provider.ListFileVersionsResponse, error) {
	receivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, req.Ref)
	appctx.GetLogger(ctx).Debug().
		Interface("ref", req.Ref).
		Interface("received_share", receivedShare).
		Err(err).
		Msg("sharesstorageprovider: Got ListFileVersions request")
	if err != nil {
		return nil, err
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.ListFileVersionsResponse{
			Status: rpcStatus,
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	return gatewayClient.ListFileVersions(ctx, &provider.ListFileVersionsRequest{
		Opaque: req.Opaque,
		Ref:    buildReferenceInShare(req.Ref, receivedShare),
	})
}

func (s *service) RestoreFileVersion(ctx context.Context, req *provider.RestoreFileVersionRequest) (*provider.RestoreFileVersionResponse, error) {
	receivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, req.Ref)
	appctx.GetLogger(ctx).Debug().
		Interface("ref", req.Ref).
		Interface("received_share", receivedShare).
		Err(err).
		Msg("sharesstorageprovider: Got RestoreFileVersion request")
	if err != nil {
		return nil, err
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.RestoreFileVersionResponse{
			Status: rpcStatus,
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	return gatewayClient.RestoreFileVersion(ctx, &provider.RestoreFileVersionRequest{
		Opaque: req.Opaque,
		Ref:    buildReferenceInShare(req.Ref, receivedShare),
	})
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

func (s *service) DenyGrant(ctx context.Context, ref *provider.DenyGrantRequest) (*provider.DenyGrantResponse, error) {
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

func (s *service) TouchFile(ctx context.Context, req *provider.TouchFileRequest) (*provider.TouchFileResponse, error) {
	receivedShare, rpcStatus, err := s.resolveAcceptedShare(ctx, req.Ref)
	appctx.GetLogger(ctx).Debug().
		Interface("ref", req.Ref).
		Interface("received_share", receivedShare).
		Msg("sharesstorageprovider: Got TouchFile request")
	if err != nil {
		return nil, err
	}
	if rpcStatus.Code != rpc.Code_CODE_OK {
		return &provider.TouchFileResponse{
			Status: rpcStatus,
		}, nil
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	return gatewayClient.TouchFile(ctx, &provider.TouchFileRequest{
		Opaque: req.Opaque,
		Ref:    buildReferenceInShare(req.Ref, receivedShare),
	})
}

// GetQuota returns 0 free quota. It is virtual ... the shares may have a different quota ...
func (s *service) GetQuota(ctx context.Context, req *provider.GetQuotaRequest) (*provider.GetQuotaResponse, error) {
	// FIXME use req.Ref to get real quota
	return &provider.GetQuotaResponse{
		Status: status.NewOK(ctx),
	}, nil
}

func (s *service) AddFavorite(ctx context.Context, req *provider.AddFavoriteRequest) (*provider.AddFavoriteResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) RemoveFavorite(ctx context.Context, req *provider.RemoveFavoriteRequest) (*provider.RemoveFavoriteResponse, error) {
	return nil, gstatus.Errorf(codes.Unimplemented, "method not implemented")
}

func (s *service) resolveAcceptedShare(ctx context.Context, ref *provider.Reference) (*collaboration.ReceivedShare, *rpc.Status, error) {
	// treat absolute id based references as relative ones
	if ref.Path == "" {
		ref.Path = "."
	}
	if !utils.IsRelativeReference(ref) {
		return nil, status.NewInvalid(ctx, "sharesstorageprovider: can only handle relative references"), nil
	}

	if ref.ResourceId.SpaceId != utils.ShareStorageSpaceID {
		return nil, status.NewNotFound(ctx, "sharesstorageprovider: not found "+ref.String()), nil
	}

	sharingCollaborationClient, err := s.sharingCollaborationSelector.Next()
	if err != nil {
		return nil, nil, err
	}

	// we can get the share if the reference carries a share id
	if ref.ResourceId.OpaqueId != utils.ShareStorageProviderID {
		// look up share for this resourceid
		lsRes, err := sharingCollaborationClient.GetReceivedShare(ctx, &collaboration.GetReceivedShareRequest{
			Ref: &collaboration.ShareReference{
				Spec: &collaboration.ShareReference_Id{
					Id: &collaboration.ShareId{
						OpaqueId: ref.ResourceId.OpaqueId,
					},
				},
			},
		})

		if err != nil {
			return nil, nil, errors.Wrap(err, "sharesstorageprovider: error calling GetReceivedShare")
		}
		if lsRes.Status.Code != rpc.Code_CODE_OK {
			return nil, lsRes.Status, nil
		}
		if lsRes.Share.State != collaboration.ShareState_SHARE_STATE_ACCEPTED {
			return nil, status.NewNotFound(ctx, "sharesstorageprovider: not found "+ref.String()), nil
		}
		return lsRes.Share, lsRes.Status, nil
	}

	// we currently need to list all accepted shares and match the path if the
	// request is relative to the share jail root. Also we need to Stat() the
	// shared resource's id to check whether that still exist. There might be
	// old shares using the same path but for an already vanished resource id.
	if ref.ResourceId.OpaqueId == utils.ShareStorageProviderID && ref.Path != "." {
		lsRes, err := sharingCollaborationClient.ListReceivedShares(ctx, &collaboration.ListReceivedSharesRequest{
			Filters: []*collaboration.Filter{
				share.StateFilter(collaboration.ShareState_SHARE_STATE_ACCEPTED),
				share.SpaceRootFilter(false),
				// TODO filter by mountpoint?
			},
		})
		if err != nil {
			return nil, nil, errors.Wrap(err, "sharesstorageprovider: error calling GetReceivedShare")
		}
		if lsRes.Status.Code != rpc.Code_CODE_OK {
			return nil, lsRes.Status, nil
		}
		for _, receivedShare := range lsRes.Shares {
			if isMountPointForPath(receivedShare.MountPoint.Path, ref.Path) {
				// Only return this share if the resource still exists.
				gatewayClient, err := s.gatewaySelector.Next()
				if err != nil {
					return nil, nil, err
				}
				sRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{
					Ref: &provider.Reference{ResourceId: receivedShare.GetShare().GetResourceId()},
				})
				if err != nil {
					appctx.GetLogger(ctx).Debug().
						Err(err).
						Interface("resourceID", receivedShare.GetShare().GetResourceId()).
						Msg("resolveAcceptedShare: failed to stat shared resource")
					continue
				}
				if sRes.Status.Code != rpc.Code_CODE_OK {
					appctx.GetLogger(ctx).Debug().
						Interface("resourceID", receivedShare.GetShare().GetResourceId()).
						Interface("status", sRes.Status).
						Msg("resolveAcceptedShare: failed to stat shared resource")
					continue
				}
				return receivedShare, lsRes.Status, nil
			}
		}
	}

	return nil, status.NewNotFound(ctx, "sharesstorageprovider: not found "+ref.String()), nil
}

func isMountPointForPath(mountpoint, path string) bool {
	requiredSegments := strings.Split(strings.TrimPrefix(mountpoint, "./"), "/")
	pathSegments := strings.Split(strings.TrimPrefix(path, "./"), "/")
	for i := range requiredSegments {
		if pathSegments[i] != requiredSegments[i] {
			return false
		}
	}
	return true
}

func (s *service) rejectReceivedShare(ctx context.Context, receivedShare *collaboration.ReceivedShare) error {
	receivedShare.State = collaboration.ShareState_SHARE_STATE_REJECTED
	receivedShare.MountPoint = nil

	sharingCollaborationClient, err := s.sharingCollaborationSelector.Next()
	if err != nil {
		return err
	}

	res, err := sharingCollaborationClient.UpdateReceivedShare(ctx, &collaboration.UpdateReceivedShareRequest{
		Share:      receivedShare,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state", "mount_point"}},
	})
	if err != nil {
		return err
	}

	return errtypes.NewErrtypeFromStatus(res.Status)
}

func (s *service) fetchAcceptedShares(ctx context.Context, opaque *typesv1beta1.Opaque, arbitraryMetadataKeys []string, fieldMask *field_mask.FieldMask) ([]*collaboration.ReceivedShare, map[string]*provider.ResourceInfo, error) {
	sharingCollaborationClient, err := s.sharingCollaborationSelector.Next()
	if err != nil {
		return nil, nil, err
	}

	lsRes, err := sharingCollaborationClient.ListReceivedShares(ctx, &collaboration.ListReceivedSharesRequest{
		Filters: []*collaboration.Filter{
			share.StateFilter(collaboration.ShareState_SHARE_STATE_ACCEPTED),
			share.SpaceRootFilter(false),
		},
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "sharesstorageprovider: error calling ListReceivedSharesRequest")
	}
	if lsRes.Status.Code != rpc.Code_CODE_OK {
		return nil, nil, fmt.Errorf("sharesstorageprovider: error calling ListReceivedSharesRequest")
	}

	numWorkers := s.maxConcurrency
	if len(lsRes.Shares) < numWorkers {
		numWorkers = len(lsRes.Shares)
	}
	type res struct {
		shareid string
		info    *provider.ResourceInfo
	}
	work := make(chan *collaboration.ReceivedShare, len(lsRes.Shares))
	results := make(chan res, len(lsRes.Shares))

	g, ctx := errgroup.WithContext(ctx)

	// Distribute work
	g.Go(func() error {
		defer close(work)
		for _, share := range lsRes.Shares {
			select {
			case work <- share:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// Spawn workers that'll concurrently work the queue
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for rs := range work {

				// only stat accepted shares
				if rs.State != collaboration.ShareState_SHARE_STATE_ACCEPTED {
					continue
				}
				if rs.Share.ResourceId.SpaceId == "" {
					// convert backwards compatible share id
					rs.Share.ResourceId.StorageId, rs.Share.ResourceId.SpaceId = storagespace.SplitStorageID(rs.Share.ResourceId.StorageId)
				}

				gatewayClient, err := s.gatewaySelector.Next()
				if err != nil {
					appctx.GetLogger(ctx).Error().
						Err(err).
						Interface("resourceID", rs.Share.ResourceId).
						Msg("ListRecievedShares: failed to select next gateway client")
					return err
				}
				sRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{
					Opaque:                opaque,
					Ref:                   &provider.Reference{ResourceId: rs.Share.ResourceId},
					ArbitraryMetadataKeys: arbitraryMetadataKeys,
					FieldMask:             fieldMask,
				})
				if err != nil {
					appctx.GetLogger(ctx).Error().
						Err(err).
						Interface("resourceID", rs.Share.ResourceId).
						Msg("ListRecievedShares: failed to make stat call")
					return err
				}
				if sRes.Status.Code != rpc.Code_CODE_OK {
					appctx.GetLogger(ctx).Debug().
						Interface("resourceID", rs.Share.ResourceId).
						Interface("status", sRes.Status).
						Msg("ListRecievedShares: failed to stat the resource")
					continue
				}
				select {
				case results <- res{shareid: rs.Share.Id.OpaqueId, info: sRes.Info}:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}

	// Wait for things to settle down, then close results chan
	go func() {
		_ = g.Wait() // error is checked later
		close(results)
	}()

	// some results might have been skipped, so we cannot preallocate the map
	shareMetaData := make(map[string]*provider.ResourceInfo)
	for r := range results {
		shareMetaData[r.shareid] = r.info
	}

	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	return lsRes.Shares, shareMetaData, nil
}

func findEarliestShare(receivedShares []*collaboration.ReceivedShare, shareInfo map[string]*provider.ResourceInfo) (earliestShare *collaboration.Share) {
	for _, rs := range receivedShares {
		var hasCurrentMd bool
		var hasEarliestMd bool

		current := rs.Share
		// We cannot assume that every share has metadata
		if current.Id != nil {
			_, hasCurrentMd = shareInfo[current.Id.OpaqueId]
		}
		if earliestShare != nil && earliestShare.Id != nil {
			_, hasEarliestMd = shareInfo[earliestShare.Id.OpaqueId]
		}

		switch {
		case earliestShare == nil && hasCurrentMd:
			earliestShare = current
		// ignore if one of the shares has no metadata
		case !hasEarliestMd || !hasCurrentMd:
			continue
		case shareInfo[current.Id.OpaqueId].Mtime.Seconds > shareInfo[earliestShare.Id.OpaqueId].Mtime.Seconds:
			earliestShare = current
		case shareInfo[current.Id.OpaqueId].Mtime.Seconds == shareInfo[earliestShare.Id.OpaqueId].Mtime.Seconds &&
			shareInfo[current.Id.OpaqueId].Mtime.Nanos > shareInfo[earliestShare.Id.OpaqueId].Mtime.Nanos:
			earliestShare = current
		}
	}
	return earliestShare
}

func buildReferenceInShare(ref *provider.Reference, s *collaboration.ReceivedShare) *provider.Reference {
	path := ref.Path
	if isShareJailRoot(ref.ResourceId) {
		// we need to cut off the mountpoint from the path in the request reference
		path = utils.MakeRelativePath(strings.TrimPrefix(strings.TrimPrefix(path, "./"), s.MountPoint.Path))
	}
	return &provider.Reference{
		ResourceId: s.Share.ResourceId,
		Path:       path,
	}
}

// isRename checks if the two references lie in the responsibility of the sharesstorageprovider and if a rename occurs
func isRename(s, d *provider.Reference) bool {
	// if the source is a share jail child where the path is .
	return ((isShareJailChild(s.ResourceId) && s.Path == ".") ||
		// or if the source is the share jail with a single path segment, e.g. './old'
		(isShareJailRoot(s.ResourceId) && len(strings.SplitN(s.Path, "/", 3)) == 2)) &&
		// and if the destination is the share jail a single path segment, e.g. './new'
		isShareJailRoot(d.ResourceId) && len(strings.SplitN(d.Path, "/", 3)) == 2
}

func isShareJailChild(id *provider.ResourceId) bool {
	return id.SpaceId == utils.ShareStorageSpaceID && id.OpaqueId != utils.ShareStorageSpaceID
}

func isShareJailRoot(id *provider.ResourceId) bool {
	return id.SpaceId == utils.ShareStorageSpaceID && id.OpaqueId == utils.ShareStorageSpaceID
}

func isVirtualRoot(ref *provider.Reference) bool {
	return isShareJailRoot(ref.ResourceId) && (ref.Path == "" || ref.Path == "." || ref.Path == "./")
}
