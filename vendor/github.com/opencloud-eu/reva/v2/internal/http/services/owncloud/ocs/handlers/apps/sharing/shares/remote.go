// Copyright 2018-2023 CERN
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

package shares

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	invitepb "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
	providerpb "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocs/response"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	"github.com/opencloud-eu/reva/v2/pkg/ocm/share"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/pkg/errors"
)

func (h *Handler) createFederatedCloudShare(w http.ResponseWriter, r *http.Request, resource *provider.ResourceInfo, role *conversions.Role, roleVal []byte) {
	ctx := r.Context()

	c, err := pool.GetGatewayServiceClient(h.gatewayAddr)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error getting grpc gateway client", err)
		return
	}

	shareWithUser, shareWithProvider := r.FormValue("shareWith"), r.FormValue("shareWithProvider")
	if shareWithUser == "" || shareWithProvider == "" {
		response.WriteOCSError(w, r, response.MetaBadRequest.StatusCode, "missing shareWith parameters", nil)
		return
	}

	providerInfoResp, err := c.GetInfoByDomain(ctx, &providerpb.GetInfoByDomainRequest{
		Domain: shareWithProvider,
	})
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error sending a grpc get invite by domain info request", err)
		return
	}

	if providerInfoResp.Status.Code != rpc.Code_CODE_OK {
		// return proper error
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error from provider info response", errors.New(providerInfoResp.Status.Message))
		return
	}

	remoteUserRes, err := c.GetAcceptedUser(ctx, &invitepb.GetAcceptedUserRequest{
		RemoteUserId: &userpb.UserId{OpaqueId: shareWithUser, Idp: shareWithProvider, Type: userpb.UserType_USER_TYPE_FEDERATED},
	})
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error searching recipient", err)
		return
	}
	if remoteUserRes.Status.Code != rpc.Code_CODE_OK {
		response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "user not found", err)
		return
	}

	createShareResponse, err := c.CreateOCMShare(ctx, &ocm.CreateOCMShareRequest{
		ResourceId: resource.Id,
		Grantee: &provider.Grantee{
			Type: provider.GranteeType_GRANTEE_TYPE_USER,
			Id: &provider.Grantee_UserId{
				UserId: remoteUserRes.RemoteUser.Id,
			},
		},
		RecipientMeshProvider: providerInfoResp.ProviderInfo,
		AccessMethods: []*ocm.AccessMethod{
			share.NewWebDavAccessMethod(role.CS3ResourcePermissions()),
			share.NewWebappAccessMethod(getViewModeFromRole(role)),
		},
	})
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error sending a grpc create ocm share request", err)
		return
	}
	if createShareResponse.Status.Code != rpc.Code_CODE_OK {
		if createShareResponse.Status.Code == rpc.Code_CODE_NOT_FOUND {
			response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "not found", nil)
			return
		}
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "grpc create ocm share request failed", err)
		return
	}

	s := createShareResponse.Share
	data, err := conversions.OCMShare2ShareData(s)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error converting share", err)
		return
	}
	h.mapUserIdsFederatedShare(ctx, c, data)

	info, status, err := h.getResourceInfoByID(ctx, c, s.ResourceId)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error statting resource id", err)
		return
	}
	if status.Code != rpc.Code_CODE_OK {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error statting resource id", errors.New(status.Message))
		return
	}

	h.addFileInfo(ctx, data, info)

	response.WriteOCSSuccess(w, r, data)
}

func getViewModeFromRole(role *conversions.Role) providerv1beta1.ViewMode {
	switch role.Name {
	case conversions.RoleViewer:
		return providerv1beta1.ViewMode_VIEW_MODE_READ_ONLY
	case conversions.RoleEditor:
		return providerv1beta1.ViewMode_VIEW_MODE_READ_WRITE
	}
	return providerv1beta1.ViewMode_VIEW_MODE_INVALID
}

// GetFederatedShare handles GET requests on /apps/files_sharing/api/v1/shares/remote_shares/{shareid}.
func (h *Handler) GetFederatedShare(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement response with HAL schemating
	ctx := r.Context()

	shareID := chi.URLParam(r, "shareid")
	gatewayClient, err := pool.GetGatewayServiceClient(h.gatewayAddr)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error getting grpc gateway client", err)
		return
	}

	listOCMSharesRequest := &ocm.GetOCMShareRequest{
		Ref: &ocm.ShareReference{
			Spec: &ocm.ShareReference_Id{
				Id: &ocm.ShareId{
					OpaqueId: shareID,
				},
			},
		},
	}
	ocmShareResponse, err := gatewayClient.GetOCMShare(ctx, listOCMSharesRequest)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error sending a grpc get ocm share request", err)
		return
	}

	share := ocmShareResponse.GetShare()
	if share == nil {
		response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "share not found", err)
		return
	}
	response.WriteOCSSuccess(w, r, share)
}

// ListFederatedShares handles GET requests on /apps/files_sharing/api/v1/shares/remote_shares.
func (h *Handler) ListFederatedShares(w http.ResponseWriter, r *http.Request) {
	// TODO Implement pagination.
	// TODO Implement response with HAL schemating
}

func (h *Handler) listReceivedFederatedShares(ctx context.Context, gw gatewayv1beta1.GatewayAPIClient, state ocm.ShareState) ([]*conversions.ShareData, error) {
	listRes, err := gw.ListReceivedOCMShares(ctx, &ocm.ListReceivedOCMSharesRequest{})
	if err != nil {
		return nil, err
	}

	shares := []*conversions.ShareData{}
	for _, s := range listRes.Shares {
		if state != ocsStateUnknown && s.State != state {
			continue
		}
		sd, err := conversions.ReceivedOCMShare2ShareData(s, h.ocmLocalMount(s))
		if err != nil {
			continue
		}
		h.mapUserIdsReceivedFederatedShare(ctx, gw, sd)
		sd.State = mapOCMState(s.State)
		shares = append(shares, sd)
	}
	return shares, nil
}

func (h *Handler) ocmLocalMount(share *ocm.ReceivedShare) string {
	return filepath.Join("/", h.ocmMountPoint, share.Id.OpaqueId)
}

func (h *Handler) mapUserIdsReceivedFederatedShare(ctx context.Context, gw gatewayv1beta1.GatewayAPIClient, sd *conversions.ShareData) {
	if sd.ShareWith != "" {
		user := h.mustGetIdentifiers(ctx, gw, sd.ShareWith, false)
		sd.ShareWith = user.Username
		sd.ShareWithDisplayname = user.DisplayName
	}

	if sd.UIDOwner != "" {
		user := h.mustGetRemoteUser(ctx, gw, sd.UIDOwner)
		sd.DisplaynameOwner = user.DisplayName
	}

	if sd.UIDFileOwner != "" {
		user := h.mustGetRemoteUser(ctx, gw, sd.UIDFileOwner)
		sd.DisplaynameFileOwner = user.DisplayName
	}
}

func (h *Handler) mapUserIdsFederatedShare(ctx context.Context, gw gatewayv1beta1.GatewayAPIClient, sd *conversions.ShareData) {
	if sd.ShareWith != "" {
		user := h.mustGetRemoteUser(ctx, gw, sd.ShareWith)
		sd.ShareWith = user.Username
		sd.ShareWithDisplayname = user.DisplayName
	}

	if sd.UIDOwner != "" {
		user := h.mustGetIdentifiers(ctx, gw, sd.UIDOwner, false)
		sd.DisplaynameOwner = user.DisplayName
	}

	if sd.UIDFileOwner != "" {
		user := h.mustGetIdentifiers(ctx, gw, sd.UIDFileOwner, false)
		sd.DisplaynameFileOwner = user.DisplayName
	}
}

func getIDAndMeshProvider(user string) (id, provider string, err error) {
	last := strings.LastIndex(user, "@")
	if last == -1 {
		return "", "", errors.New("not in the form <id>@<provider>")
	}
	if len(user[:last]) == 0 {
		return "", "", errors.New("empty id")
	}
	if len(user[last+1:]) == 0 {
		return "", "", errors.New("empty provider")
	}
	return user[:last], user[last+1:], nil
}

func (h *Handler) mustGetRemoteUser(ctx context.Context, gw gatewayv1beta1.GatewayAPIClient, id string) *userIdentifiers {
	id, provider, err := getIDAndMeshProvider(id)
	if err != nil {
		return &userIdentifiers{}
	}
	userRes, err := gw.GetAcceptedUser(ctx, &invitepb.GetAcceptedUserRequest{
		RemoteUserId: &userpb.UserId{
			Type:     userpb.UserType_USER_TYPE_FEDERATED,
			Idp:      provider,
			OpaqueId: id,
		},
	})
	if err != nil {
		return &userIdentifiers{}
	}
	if userRes.Status.Code != rpc.Code_CODE_OK {
		return &userIdentifiers{}
	}

	user := userRes.RemoteUser
	return &userIdentifiers{
		DisplayName: user.DisplayName,
		Username:    user.Username,
		Mail:        user.Mail,
	}
}

func (h *Handler) listOutcomingFederatedShares(ctx context.Context, gw gatewayv1beta1.GatewayAPIClient, filters []*ocm.ListOCMSharesRequest_Filter) ([]*conversions.ShareData, error) {
	listRes, err := gw.ListOCMShares(ctx, &ocm.ListOCMSharesRequest{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	shares := []*conversions.ShareData{}
	for _, s := range listRes.Shares {
		sd, err := conversions.OCMShare2ShareData(s)
		if err != nil {
			continue
		}
		h.mapUserIdsFederatedShare(ctx, gw, sd)

		info, status, err := h.getResourceInfoByID(ctx, gw, s.ResourceId)
		if err != nil {
			return nil, err
		}

		if status.Code != rpc.Code_CODE_OK {
			return nil, err
		}

		h.addFileInfo(ctx, sd, info)
		shares = append(shares, sd)
	}
	return shares, nil
}
