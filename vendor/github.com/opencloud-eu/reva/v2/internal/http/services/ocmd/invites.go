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

package ocmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	invitepb "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
	ocmprovider "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/reqres"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ocmuser "github.com/opencloud-eu/reva/v2/pkg/ocm/user"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

type invitesHandler struct {
	gatewaySelector *pool.Selector[gateway.GatewayAPIClient]
}

func (h *invitesHandler) init(c *config) error {
	var err error

	gatewaySelector, err := pool.GatewaySelector(c.GatewaySvc)
	if err != nil {
		return err
	}
	h.gatewaySelector = gatewaySelector
	return nil
}

type acceptInviteRequest struct {
	Token             string `json:"token"`
	UserID            string `json:"userID"`
	RecipientProvider string `json:"recipientProvider"`
	Name              string `json:"name"`
	Email             string `json:"email"`
}

// AcceptInvite informs about an accepted invitation so that the users
// can initiate the OCM share creation.
func (h *invitesHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := appctx.GetLogger(ctx)

	req, err := getAcceptInviteRequest(r)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "missing parameters in request", err)
		return
	}

	if req.Token == "" || req.UserID == "" || req.RecipientProvider == "" {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "token, userID and recipientProvider must not be null", nil)
		return
	}

	clientIP, err := utils.GetClientIP(r)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, fmt.Sprintf("error retrieving client IP from request: %s", r.RemoteAddr), err)
		return
	}

	providerInfo := ocmprovider.ProviderInfo{
		Domain: req.RecipientProvider,
		Services: []*ocmprovider.Service{
			{
				Host: clientIP,
			},
		},
	}
	gatewayClient, err := h.gatewaySelector.Next()
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error getting gateway client", err)
		return
	}
	providerAllowedResp, err := gatewayClient.IsProviderAllowed(ctx, &ocmprovider.IsProviderAllowedRequest{
		Provider: &providerInfo,
	})
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error sending a grpc is provider allowed request", err)
		return
	}
	if providerAllowedResp.Status.Code != rpc.Code_CODE_OK {
		reqres.WriteError(w, r, reqres.APIErrorUntrustedService, "provider not trusted", errors.New(providerAllowedResp.Status.Message))
		return
	}

	userObj := &userpb.User{
		Id: &userpb.UserId{
			OpaqueId: req.UserID,
			Idp:      req.RecipientProvider,
			Type:     userpb.UserType_USER_TYPE_FEDERATED,
		},
		Mail:        req.Email,
		DisplayName: req.Name,
	}
	acceptInviteRequest := &invitepb.AcceptInviteRequest{
		InviteToken: &invitepb.InviteToken{
			Token: req.Token,
		},
		RemoteUser: userObj,
	}
	acceptInviteResponse, err := gatewayClient.AcceptInvite(ctx, acceptInviteRequest)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error sending a grpc accept invite request", err)
		return
	}
	if acceptInviteResponse.Status.Code != rpc.Code_CODE_OK {
		switch acceptInviteResponse.Status.Code {
		case rpc.Code_CODE_NOT_FOUND:
			reqres.WriteError(w, r, reqres.APIErrorNotFound, "token not found", nil)
			return
		case rpc.Code_CODE_INVALID_ARGUMENT:
			reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "token has expired", nil)
			return
		case rpc.Code_CODE_ALREADY_EXISTS:
			reqres.WriteError(w, r, reqres.APIErrorAlreadyExist, "user already known", nil)
			return
		default:
			reqres.WriteError(w, r, reqres.APIErrorServerError, "unexpected error: "+acceptInviteResponse.Status.Message, errors.New(acceptInviteResponse.Status.Message))
			return
		}
	}

	if err := json.NewEncoder(w).Encode(&user{
		UserID: ocmuser.LocalUserFederatedID(acceptInviteResponse.UserId, "").GetOpaqueId(),
		Email:  acceptInviteResponse.Email,
		Name:   acceptInviteResponse.DisplayName,
	}); err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error encoding response", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	log.Info().Str("user", fmt.Sprintf("%s@%s", userObj.Id.OpaqueId, userObj.Id.Idp)).Str("token", req.Token).Msg("added to accepted users")
}

type user struct {
	UserID string `json:"userID"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

func getAcceptInviteRequest(r *http.Request) (*acceptInviteRequest, error) {
	var req acceptInviteRequest
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err == nil && contentType == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, err
		}
	} else {
		req.Token, req.UserID, req.RecipientProvider = r.FormValue("token"), r.FormValue("userID"), r.FormValue("recipientProvider")
		req.Name, req.Email = r.FormValue("name"), r.FormValue("email")
	}
	return &req, nil
}
