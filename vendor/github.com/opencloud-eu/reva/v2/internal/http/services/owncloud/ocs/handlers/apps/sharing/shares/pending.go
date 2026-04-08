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

package shares

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strconv"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	ocmv1beta1 "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/opencloud-eu/reva/v2/internal/grpc/services/usershareprovider"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocs/response"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
)

const (
	// shareidkey is the key user to obtain the id of the share to update. It is present in the request URL.
	shareidkey string = "shareid"
)

// AcceptReceivedShare handles Post Requests on /apps/files_sharing/api/v1/shares/{shareid}
func (h *Handler) AcceptReceivedShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	shareID := chi.URLParam(r, shareidkey)

	if h.isFederatedReceivedShare(r, shareID) {
		h.updateReceivedFederatedShare(w, r, shareID, false)
		return
	}

	client, err := h.getClient()
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error getting grpc gateway client", err)
		return
	}

	receivedShare, ocsResponse := getReceivedShareFromID(ctx, client, shareID)
	if ocsResponse != nil {
		response.WriteOCSResponse(w, r, *ocsResponse, nil)
		return
	}

	sharedResource, ocsResponse := getSharedResource(ctx, client, receivedShare.Share.ResourceId)
	if ocsResponse != nil {
		response.WriteOCSResponse(w, r, *ocsResponse, nil)
		return
	}
	mount, unmountedShares, err := usershareprovider.GetMountpointAndUnmountedShares(
		ctx,
		client,
		sharedResource.GetInfo().GetId(),
		sharedResource.GetInfo().GetName(),
		nil,
	)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "could not determine mountpoint", err)
		return
	}

	// first update the requested share
	receivedShare.State = collaboration.ShareState_SHARE_STATE_ACCEPTED
	// we need to add a path to the share
	receivedShare.MountPoint = &provider.Reference{
		Path: mount,
	}

	updateMask := &fieldmaskpb.FieldMask{Paths: []string{"state", "mount_point"}}
	data, meta, err := h.updateReceivedShare(r.Context(), receivedShare, updateMask)
	if err != nil {
		// we log an error for affected shares, for the actual share we return an error
		response.WriteOCSData(w, r, meta, data, err)
		return
	}
	response.WriteOCSSuccess(w, r, []*conversions.ShareData{data})

	// then update other unmounted shares to the same resource
	for _, rs := range unmountedShares {
		if rs.GetShare().GetId().GetOpaqueId() == shareID {
			// we already updated this one
			continue
		}

		rs.State = collaboration.ShareState_SHARE_STATE_ACCEPTED
		// set the same mountpoint as for the requested received share
		rs.MountPoint = &provider.Reference{
			Path: mount,
		}

		_, _, err := h.updateReceivedShare(r.Context(), rs, updateMask)
		if err != nil {
			// we log an error for affected shares, the actual share was successful
			appctx.GetLogger(ctx).Error().Err(err).Str("received_share", shareID).Str("affected_share", rs.GetShare().GetId().GetOpaqueId()).Msg("could not update affected received share")
		}
	}
}

// RejectReceivedShare handles DELETE Requests on /apps/files_sharing/api/v1/shares/{shareid}
func (h *Handler) RejectReceivedShare(w http.ResponseWriter, r *http.Request) {
	shareID := chi.URLParam(r, "shareid")

	if h.isFederatedReceivedShare(r, shareID) {
		h.updateReceivedFederatedShare(w, r, shareID, true)
		return
	}

	// we need to add a path to the share
	receivedShare := &collaboration.ReceivedShare{
		Share: &collaboration.Share{
			Id: &collaboration.ShareId{OpaqueId: shareID},
		},
		State: collaboration.ShareState_SHARE_STATE_REJECTED,
	}
	updateMask := &fieldmaskpb.FieldMask{Paths: []string{"state"}}

	data, meta, err := h.updateReceivedShare(r.Context(), receivedShare, updateMask)
	if err != nil {
		response.WriteOCSData(w, r, meta, nil, err)
	}
	response.WriteOCSSuccess(w, r, []*conversions.ShareData{data})
}

func (h *Handler) UpdateReceivedShare(w http.ResponseWriter, r *http.Request) {
	shareID := chi.URLParam(r, "shareid")
	hideFlag, _ := strconv.ParseBool(r.URL.Query().Get("hidden"))

	// unfortunately we need to get the share first to read the state
	client, err := h.getClient()
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error getting grpc gateway client", err)
		return
	}

	// we need to add a path to the share
	receivedShare := &collaboration.ReceivedShare{
		Share: &collaboration.Share{
			Id: &collaboration.ShareId{OpaqueId: shareID},
		},
		Hidden: hideFlag,
	}
	updateMask := &fieldmaskpb.FieldMask{Paths: []string{"state", "hidden"}}

	rs, _ := getReceivedShareFromID(r.Context(), client, shareID)
	if rs != nil && rs.Share != nil {
		receivedShare.State = rs.State
	}

	data, meta, err := h.updateReceivedShare(r.Context(), receivedShare, updateMask)
	if err != nil {
		response.WriteOCSData(w, r, meta, nil, err)
	}
	response.WriteOCSSuccess(w, r, []*conversions.ShareData{data})
}

func (h *Handler) ReceivedShareNotFound(w http.ResponseWriter, r *http.Request) {
	response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "cannot find share", nil)
}

func (h *Handler) updateReceivedShare(ctx context.Context, receivedShare *collaboration.ReceivedShare, fieldMask *fieldmaskpb.FieldMask) (*conversions.ShareData, response.Meta, error) {
	logger := appctx.GetLogger(ctx)

	updateShareRequest := &collaboration.UpdateReceivedShareRequest{
		Share:      receivedShare,
		UpdateMask: fieldMask,
	}

	client, err := h.getClient()
	if err != nil {
		return nil, response.MetaServerError, errors.Wrap(err, "error getting grpc gateway client")
	}

	shareRes, err := client.UpdateReceivedShare(ctx, updateShareRequest)
	if err != nil {
		return nil, response.MetaServerError, errors.Wrap(err, "grpc update received share request failed")
	}

	if shareRes.Status.Code != rpc.Code_CODE_OK {
		if shareRes.Status.Code == rpc.Code_CODE_NOT_FOUND {
			return nil, response.MetaNotFound, errors.New(shareRes.Status.Message)
		}
		return nil, response.MetaServerError, errors.Errorf("grpc update received share request failed: code: %d, message: %s", shareRes.Status.Code, shareRes.Status.Message)
	}

	rs := shareRes.GetShare()

	info, status, err := h.getResourceInfoByID(ctx, client, rs.Share.ResourceId)
	if err != nil || status.Code != rpc.Code_CODE_OK {
		h.logProblems(logger, status, err, "could not stat, skipping")
		return nil, response.MetaServerError, errors.Errorf("grpc get resource info failed: code: %d, message: %s", status.Code, status.Message)
	}

	data := conversions.CS3Share2ShareData(ctx, rs.Share)

	data.State = mapState(rs.GetState())
	data.Hidden = rs.GetHidden()

	h.addFileInfo(ctx, data, info)
	h.mapUserIds(ctx, client, data)

	if data.State == ocsStateAccepted {
		// Needed because received shares can be jailed in a folder in the users home
		data.Path = path.Join(h.sharePrefix, path.Base(info.Path))
	}

	return data, response.MetaOK, nil
}

func (h *Handler) updateReceivedFederatedShare(w http.ResponseWriter, r *http.Request, shareID string, rejectShare bool) {
	ctx := r.Context()

	client, err := h.getClient()
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "error getting grpc gateway client", err)
		return
	}

	share, err := client.GetReceivedOCMShare(ctx, &ocmv1beta1.GetReceivedOCMShareRequest{
		Ref: &ocmv1beta1.ShareReference{
			Spec: &ocmv1beta1.ShareReference_Id{
				Id: &ocmv1beta1.ShareId{
					OpaqueId: shareID,
				},
			},
		},
	})
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "grpc update received share request failed", err)
		return
	}
	if share.Status.Code != rpc.Code_CODE_OK {
		if share.Status.Code == rpc.Code_CODE_NOT_FOUND {
			response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "not found", nil)
			return
		}
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "grpc update received share request failed", errors.Errorf("code: %d, message: %s", share.Status.Code, share.Status.Message))
		return
	}

	req := &ocmv1beta1.UpdateReceivedOCMShareRequest{
		Share: &ocmv1beta1.ReceivedShare{
			Id: &ocmv1beta1.ShareId{
				OpaqueId: shareID,
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
	}
	if rejectShare {
		req.Share.State = ocmv1beta1.ShareState_SHARE_STATE_REJECTED
	} else {
		req.Share.State = ocmv1beta1.ShareState_SHARE_STATE_ACCEPTED
	}

	updateRes, err := client.UpdateReceivedOCMShare(ctx, req)
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "grpc update received share request failed", err)
		return
	}

	if updateRes.Status.Code != rpc.Code_CODE_OK {
		if updateRes.Status.Code == rpc.Code_CODE_NOT_FOUND {
			response.WriteOCSError(w, r, response.MetaNotFound.StatusCode, "not found", nil)
			return
		}
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "grpc update received share request failed", errors.Errorf("code: %d, message: %s", updateRes.Status.Code, updateRes.Status.Message))
		return
	}

	data, err := conversions.ReceivedOCMShare2ShareData(share.Share, h.ocmLocalMount(share.Share))
	if err != nil {
		response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "grpc update received share request failed", err)
		return
	}
	h.mapUserIdsReceivedFederatedShare(ctx, client, data)
	data.State = mapOCMState(req.Share.State)
	response.WriteOCSSuccess(w, r, []*conversions.ShareData{data})
}

// getReceivedShareFromID uses a client to the gateway to fetch a share based on its ID.
func getReceivedShareFromID(ctx context.Context, client gateway.GatewayAPIClient, shareID string) (*collaboration.ReceivedShare, *response.Response) {
	s, err := client.GetReceivedShare(ctx, &collaboration.GetReceivedShareRequest{
		Ref: &collaboration.ShareReference{
			Spec: &collaboration.ShareReference_Id{
				Id: &collaboration.ShareId{
					OpaqueId: shareID,
				}},
		},
	})

	if err != nil {
		e := errors.Wrap(err, fmt.Sprintf("could not get share with ID: `%s`", shareID))
		return nil, arbitraryOcsResponse(response.MetaServerError.StatusCode, e.Error())
	}

	if s.Status.Code != rpc.Code_CODE_OK {
		if s.Status.Code == rpc.Code_CODE_NOT_FOUND {
			e := fmt.Errorf("share not found")
			return nil, arbitraryOcsResponse(response.MetaNotFound.StatusCode, e.Error())
		}

		e := fmt.Errorf("invalid share: %s", s.GetStatus().GetMessage())
		return nil, arbitraryOcsResponse(response.MetaBadRequest.StatusCode, e.Error())
	}

	return s.Share, nil
}

// getSharedResource attempts to get a shared resource from the storage from the resource reference.
func getSharedResource(ctx context.Context, client gateway.GatewayAPIClient, resID *provider.ResourceId) (*provider.StatResponse, *response.Response) {
	res, err := client.Stat(ctx, &provider.StatRequest{
		Ref: &provider.Reference{
			ResourceId: resID,
		},
	})
	if err != nil {
		return nil, arbitraryOcsResponse(response.MetaServerError.StatusCode, "could not get reference")
	}

	if res.Status.Code != rpc.Code_CODE_OK {
		if res.Status.Code == rpc.Code_CODE_NOT_FOUND {
			return nil, arbitraryOcsResponse(response.MetaNotFound.StatusCode, "not found")
		}
		return nil, arbitraryOcsResponse(response.MetaServerError.StatusCode, res.GetStatus().GetMessage())
	}

	return res, nil
}

// listReceivedShares list all received shares for the current user.
func listReceivedShares(ctx context.Context, client gateway.GatewayAPIClient) ([]*collaboration.ReceivedShare, error) {
	res, err := client.ListReceivedShares(ctx, &collaboration.ListReceivedSharesRequest{})
	if err != nil {
		return nil, errtypes.InternalError("grpc list received shares request failed")
	}

	if err := errtypes.NewErrtypeFromStatus(res.Status); err != nil {
		return nil, err
	}
	return res.Shares, nil
}

// arbitraryOcsResponse abstracts the boilerplate that is creating a response.Response struct.
func arbitraryOcsResponse(statusCode int, message string) *response.Response {
	r := response.Response{
		OCS: &response.Payload{
			XMLName: struct{}{},
			Meta:    response.Meta{},
			Data:    nil,
		},
	}

	r.OCS.Meta.StatusCode = statusCode
	r.OCS.Meta.Message = message
	return &r
}
