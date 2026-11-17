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

package sciencemesh

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	invitepb "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
	ocmprovider "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"

	"github.com/opencloud-eu/reva/v3/internal/http/services/reqres"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/opencloud-eu/reva/v3/pkg/events/stream"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
	"github.com/opencloud-eu/reva/v3/pkg/utils/list"
)

type tokenHandler struct {
	gatewaySelector  *pool.Selector[gateway.GatewayAPIClient]
	meshDirectoryURL string
	providerDomain   string
	eventStream      events.Stream
}

func (h *tokenHandler) init(c *config) error {
	var err error
	h.gatewaySelector, err = pool.GatewaySelector(c.GatewaySvc)
	if err != nil {
		return err
	}

	h.meshDirectoryURL = c.MeshDirectoryURL
	h.providerDomain = c.ProviderDomain

	if c.Events.Endpoint != "" {
		es, err := stream.NatsFromConfig("sciencemesh-token-handler", false, stream.NatsConfig(c.Events))
		if err != nil {
			return err
		}
		h.eventStream = es
	}

	return nil
}

type token struct {
	Token       string `json:"token"`
	Description string `json:"description,omitempty"`
	Expiration  uint64 `json:"expiration,omitempty"`
	InviteLink  string `json:"invite_link,omitempty"`
}

// Generate generates an invitation token and if a recipient is specified,
// will send an email containing the link the user will use to accept the
// invitation.
func (h *tokenHandler) Generate(w http.ResponseWriter, r *http.Request) {
	req, err := getGenerateRequest(r)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "missing parameters in request", err)
		return
	}

	ctx := r.Context()
	gc, err := h.gatewaySelector.Next()
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error selecting gateway client", err)
		return
	}
	genTokenRes, err := gc.GenerateInviteToken(ctx, &invitepb.GenerateInviteTokenRequest{
		Description: req.Description,
	})
	switch {
	case err != nil:
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error generating token", err)
		return
	case genTokenRes.GetStatus().GetCode() == rpc.Code_CODE_NOT_FOUND:
		reqres.WriteError(w, r, reqres.APIErrorNotFound, genTokenRes.GetStatus().GetMessage(), nil)
		return
	case genTokenRes.GetStatus().GetCode() != rpc.Code_CODE_OK:
		reqres.WriteError(w, r, reqres.APIErrorServerError, genTokenRes.GetStatus().GetMessage(), errors.New(genTokenRes.GetStatus().GetMessage()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // FIXME this should be a 201 created status. Tracked in https://github.com/cs3org/reva/issues/4838

	tknRes := h.prepareGenerateTokenResponse(genTokenRes.GetInviteToken())
	if err := json.NewEncoder(w).Encode(tknRes); err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error marshalling token data", err)
	}

	if h.eventStream != nil {
		if err := events.Publish(ctx, h.eventStream, events.ScienceMeshInviteTokenGenerated{
			Sharer:        genTokenRes.GetInviteToken().GetUserId(),
			RecipientMail: req.Recipient,
			Token:         tknRes.Token,
			Description:   tknRes.Description,
			Expiration:    tknRes.Expiration,
			InviteLink:    tknRes.InviteLink,
			Timestamp:     utils.TSNow(),
		}); err != nil {
			log := appctx.GetLogger(ctx)
			log.Error().Err(err).
				Msg("failed to publish the science-mesh invite token generated event")
		}
	}

}

// generateRequest is the request body for the Generate endpoint.
type generateRequest struct {
	Description string `json:"description"`
	Recipient   string `json:"recipient" validate:"omitempty,email"`
}

func getGenerateRequest(r *http.Request) (*generateRequest, error) {
	var req generateRequest
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err == nil && contentType == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, err
		}
	}

	// validate the request
	if err := validate.Struct(req); err != nil {
		return nil, err
	}

	return &req, nil
}

func (h *tokenHandler) prepareGenerateTokenResponse(tkn *invitepb.InviteToken) *token {
	res := &token{
		Token:       tkn.Token,
		Description: tkn.Description,
	}
	if h.meshDirectoryURL != "" {
		res.InviteLink = h.meshDirectoryURL + "?token=" + tkn.Token + "&providerDomain=" + h.providerDomain
	}
	if tkn.Expiration != nil {
		res.Expiration = tkn.Expiration.Seconds
	}

	return res
}

type acceptInviteRequest struct {
	Token          string `json:"token"`
	ProviderDomain string `json:"providerDomain"`
}

// AcceptInvite accepts an invitation from the user in the remote provider.
func (h *tokenHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := appctx.GetLogger(ctx)

	req, err := getAcceptInviteRequest(r)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "missing parameters in request", err)
		return
	}

	if req.Token == "" || req.ProviderDomain == "" {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "token and providerDomain must not be null", nil)
		return
	}

	gc, err := h.gatewaySelector.Next()
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error selecting gateway client", err)
		return
	}
	providerInfo, err := gc.GetInfoByDomain(ctx, &ocmprovider.GetInfoByDomainRequest{
		Domain: req.ProviderDomain,
	})
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error sending a grpc get invite by domain info request", err)
		return
	}
	if providerInfo.Status.Code != rpc.Code_CODE_OK {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "grpc forward invite request failed", errors.New(providerInfo.Status.Message))
		return
	}

	gc, err = h.gatewaySelector.Next()
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error selecting gateway client", err)
		return
	}
	forwardInviteReq := &invitepb.ForwardInviteRequest{
		InviteToken: &invitepb.InviteToken{
			Token: req.Token,
		},
		OriginSystemProvider: providerInfo.ProviderInfo,
	}
	forwardInviteResponse, err := gc.ForwardInvite(ctx, forwardInviteReq)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error sending a grpc forward invite request", err)
		return
	}
	if forwardInviteResponse.Status.Code != rpc.Code_CODE_OK {
		switch forwardInviteResponse.Status.Code {
		case rpc.Code_CODE_NOT_FOUND:
			reqres.WriteError(w, r, reqres.APIErrorNotFound, "token not found", nil)
			return
		case rpc.Code_CODE_INVALID_ARGUMENT:
			reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "token has expired", nil)
			return
		case rpc.Code_CODE_ALREADY_EXISTS:
			reqres.WriteError(w, r, reqres.APIErrorAlreadyExist, "user already known", nil)
			return
		case rpc.Code_CODE_PERMISSION_DENIED:
			reqres.WriteError(w, r, reqres.APIErrorUnauthenticated, "remote service not trusted", nil)
			return
		default:
			reqres.WriteError(w, r, reqres.APIErrorServerError, "unexpected error: "+forwardInviteResponse.Status.Message, errors.New(forwardInviteResponse.Status.Message))
			return
		}
	}

	w.WriteHeader(http.StatusOK)

	log.Info().Str("token", req.Token).Str("provider", req.ProviderDomain).Msgf("invite forwarded")
}

func getAcceptInviteRequest(r *http.Request) (*acceptInviteRequest, error) {
	var req acceptInviteRequest
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err == nil && contentType == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, err
		}
	} else {
		req.Token, req.ProviderDomain = r.FormValue("token"), r.FormValue("providerDomain")
	}
	return &req, nil
}

type remoteUser struct {
	DisplayName string `json:"display_name"`
	Idp         string `json:"idp"`
	UserID      string `json:"user_id"`
	Mail        string `json:"mail"`
}

// FindAccepted returns the list of all the users that accepted the invitation
// to the authenticated user.
func (h *tokenHandler) FindAccepted(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	gc, err := h.gatewaySelector.Next()
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error selecting gateway client", err)
		return
	}
	res, err := gc.FindAcceptedUsers(ctx, &invitepb.FindAcceptedUsersRequest{})
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error sending a grpc find accepted users request", err)
		return
	}

	users := list.Map(res.AcceptedUsers, func(u *userpb.User) *remoteUser {
		return &remoteUser{
			DisplayName: u.DisplayName,
			Idp:         u.Id.Idp,
			UserID:      u.Id.OpaqueId,
			Mail:        u.Mail,
		}
	})

	if err := json.NewEncoder(w).Encode(users); err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error marshalling token data", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// DeleteAccepted deletes the given user from the list of the accepted users.
func (h *tokenHandler) DeleteAccepted(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := getDeleteAcceptedRequest(r)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, "missing parameters in request", err)
		return
	}

	gc, err := h.gatewaySelector.Next()
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error selecting gateway client", err)
		return
	}
	res, err := gc.DeleteAcceptedUser(ctx, &invitepb.DeleteAcceptedUserRequest{
		RemoteUserId: &userpb.UserId{
			Idp:      req.Idp,
			OpaqueId: req.UserID,
			Type:     userpb.UserType_USER_TYPE_FEDERATED,
		},
	})
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error sending a grpc get invite by domain info request", err)
		return
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "grpc forward invite request failed", errors.New(res.Status.Message))
		return
	}
	w.WriteHeader(http.StatusOK)
}

type deleteAcceptedRequest struct {
	Idp    string `json:"idp"`
	UserID string `json:"user_id"`
}

func getDeleteAcceptedRequest(r *http.Request) (*deleteAcceptedRequest, error) {
	var req deleteAcceptedRequest
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err == nil && contentType == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, err
		}
	} else {
		req.Idp, req.UserID = r.FormValue("idp"), r.FormValue("user_id")
	}
	return &req, nil
}

func (h *tokenHandler) ListInvite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	gc, err := h.gatewaySelector.Next()
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error selecting gateway client", err)
		return
	}
	res, err := gc.ListInviteTokens(ctx, &invitepb.ListInviteTokensRequest{})
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error listing tokens", err)
		return
	}

	if res.Status.Code != rpc.Code_CODE_OK {
		reqres.WriteError(w, r, reqres.APIErrorServerError, res.Status.Message, errors.New(res.Status.Message))
		return
	}

	tokens := make([]*token, 0, len(res.InviteTokens))
	for _, tkn := range res.InviteTokens {
		tokens = append(tokens, h.prepareGenerateTokenResponse(tkn))
	}

	if err := json.NewEncoder(w).Encode(tokens); err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error marshalling token data", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
