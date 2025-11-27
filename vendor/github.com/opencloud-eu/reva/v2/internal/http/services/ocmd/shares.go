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
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	providerpb "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocmcore "github.com/cs3org/go-cs3apis/cs3/ocm/core/v1beta1"
	ocmprovider "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/go-playground/validator/v10"
	"github.com/opencloud-eu/reva/v2/internal/http/services/reqres"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

var validate = validator.New()

type sharesHandler struct {
	gatewaySelector            *pool.Selector[gateway.GatewayAPIClient]
	exposeRecipientDisplayName bool
}

func (h *sharesHandler) init(c *config) error {
	var err error

	gatewaySelector, err := pool.GatewaySelector(c.GatewaySvc)
	if err != nil {
		return err
	}
	h.gatewaySelector = gatewaySelector

	h.exposeRecipientDisplayName = c.ExposeRecipientDisplayName
	return nil
}

type createShareRequest struct {
	ShareWith         string    `json:"shareWith" validate:"required"`                  // identifier of the recipient of the share
	Name              string    `json:"name" validate:"required"`                       // name of the resource
	Description       string    `json:"description"`                                    // (optional) description of the resource
	ProviderID        string    `json:"providerId" validate:"required"`                 // unique identifier of the resource at provider side
	Owner             string    `json:"owner" validate:"required"`                      // unique identifier of the owner at provider side
	Sender            string    `json:"sender" validate:"required"`                     // unique indentifier of the user who wants to share the resource at provider side
	OwnerDisplayName  string    `json:"ownerDisplayName"`                               // display name of the owner of the resource
	SenderDisplayName string    `json:"senderDisplayName"`                              // dispay name of the user who wants to share the resource
	ShareType         string    `json:"shareType" validate:"required,oneof=user group"` // recipient share type (user or group)
	ResourceType      string    `json:"resourceType" validate:"required,oneof=file folder"`
	Expiration        uint64    `json:"expiration"`
	Protocols         Protocols `json:"protocol" validate:"required"`
}

// CreateShare sends all the informations to the consumer needed to start
// synchronization between the two services.
func (h *sharesHandler) CreateShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := appctx.GetLogger(ctx)
	req, err := getCreateShareRequest(r)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, err.Error(), nil)
		return
	}

	_, meshProvider, err := getIDAndMeshProvider(req.Sender)
	log.Debug().Msgf("Determined Mesh Provider '%s' from req.Sender '%s'", meshProvider, req.Sender)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, err.Error(), nil)
		return
	}

	clientIP, err := utils.GetClientIP(r)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, fmt.Sprintf("error retrieving client IP from request: %s", r.RemoteAddr), err)
		return
	}
	providerInfo := ocmprovider.ProviderInfo{
		Domain: meshProvider,
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
		reqres.WriteError(w, r, reqres.APIErrorUnauthenticated, "provider not authorized", errors.New(providerAllowedResp.Status.Message))
		return
	}

	shareWith, _, err := getLocalUserID(req.ShareWith)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, err.Error(), nil)
		return
	}

	userRes, err := gatewayClient.GetUser(ctx, &userpb.GetUserRequest{
		UserId: &userpb.UserId{OpaqueId: shareWith}, SkipFetchingUserGroups: true,
	})
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error searching recipient", err)
		return
	}
	if userRes.Status.Code != rpc.Code_CODE_OK {
		reqres.WriteError(w, r, reqres.APIErrorNotFound, "user not found", errors.New(userRes.Status.Message))
		return
	}

	owner, err := getUserIDFromOCMUser(req.Owner)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, err.Error(), nil)
		return
	}

	sender, err := getUserIDFromOCMUser(req.Sender)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorInvalidParameter, err.Error(), nil)
		return
	}

	createShareReq := &ocmcore.CreateOCMCoreShareRequest{
		Description:  req.Description,
		Name:         req.Name,
		ResourceId:   req.ProviderID,
		Owner:        owner,
		Sender:       sender,
		ShareWith:    userRes.User.Id,
		ResourceType: getResourceTypeFromOCMRequest(req.ResourceType),
		ShareType:    getOCMShareType(req.ShareType),
		Protocols:    getProtocols(req.Protocols),
	}

	if req.Expiration != 0 {
		createShareReq.Expiration = &types.Timestamp{
			Seconds: req.Expiration,
		}
	}

	createShareResp, err := gatewayClient.CreateOCMCoreShare(ctx, createShareReq)
	if err != nil {
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error creating ocm share", err)
		return
	}

	if userRes.Status.Code != rpc.Code_CODE_OK {
		// TODO: define errors in the cs3apis
		reqres.WriteError(w, r, reqres.APIErrorServerError, "error creating ocm share", errors.New(createShareResp.Status.Message))
		return
	}

	response := map[string]any{}

	if h.exposeRecipientDisplayName {
		response["recipientDisplayName"] = userRes.User.DisplayName
	}

	_ = json.NewEncoder(w).Encode(response)
	w.WriteHeader(http.StatusCreated)
}

func getLocalUserID(user string) (id, provider string, err error) {
	idPart, provider, err := getIDAndMeshProvider(user)
	if err != nil {
		return "", "", err
	}

	// Handle nested @ in idPart (e.g. "user@idp@provider")
	if inner := strings.LastIndex(idPart, "@"); inner != -1 {
		id = idPart[:inner]
	} else {
		id = idPart
	}

	return id, provider, nil
}

func getUserIDFromOCMUser(user string) (*userpb.UserId, error) {
	id, idp, err := getIDAndMeshProvider(user)
	if err != nil {
		return nil, err
	}
	return &userpb.UserId{
		OpaqueId: id,
		Idp:      idp,
		// the remote user is a federated account for the local reva
		Type: userpb.UserType_USER_TYPE_FEDERATED,
	}, nil
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

func getCreateShareRequest(r *http.Request) (*createShareRequest, error) {
	var req createShareRequest
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err == nil && contentType == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("body request not recognised")
	}
	// validate the request
	if err := validate.Struct(req); err != nil {
		return nil, err
	}
	return &req, nil
}

func getResourceTypeFromOCMRequest(t string) providerpb.ResourceType {
	switch t {
	case "file":
		return providerpb.ResourceType_RESOURCE_TYPE_FILE
	case "folder":
		return providerpb.ResourceType_RESOURCE_TYPE_CONTAINER
	default:
		return providerpb.ResourceType_RESOURCE_TYPE_INVALID
	}
}

func getOCMShareType(t string) ocm.ShareType {
	if t == "user" {
		return ocm.ShareType_SHARE_TYPE_USER
	}
	return ocm.ShareType_SHARE_TYPE_GROUP
}

func getProtocols(p Protocols) []*ocm.Protocol {
	prot := make([]*ocm.Protocol, 0, len(p))
	for _, data := range p {
		prot = append(prot, data.ToOCMProtocol())
	}
	return prot
}
