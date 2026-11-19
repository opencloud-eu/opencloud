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
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	ocmcore "github.com/cs3org/go-cs3apis/cs3/ocm/core/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/render"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
)

const (
	SHARE_UNSHARED          = "SHARE_UNSHARED"
	SHARE_CHANGE_PERMISSION = "SHARE_CHANGE_PERMISSION"
)

// var validate = validator.New()

type notifHandler struct {
	gatewaySelector *pool.Selector[gateway.GatewayAPIClient]
}

func (h *notifHandler) init(c *config) error {
	gatewaySelector, err := pool.GatewaySelector(c.GatewaySvc)
	if err != nil {
		return err
	}
	h.gatewaySelector = gatewaySelector

	return nil
}

// https://cs3org.github.io/OCM-API/docs.html?branch=develop&repo=OCM-API&user=cs3org#/paths/~1notifications/post
type notificationRequest struct {
	NotificationType string `json:"notificationType" validate:"required"`
	ResourceType     string `json:"resourceType" validate:"required"`
	// Identifier to identify the shared resource at the provider side. This is unique per provider such that if the same resource is shared twice, this providerId will not be repeated.
	ProviderId string `json:"providerId" validate:"required"`
	// Optional additional parameters, depending on the notification and the resource type.
	Notification *notification `json:"notification,omitempty"`
}

type notification struct {
	Owner        string    `json:"owner,omitempty"`
	Grantee      string    `json:"grantee,omitempty"`
	SharedSecret string    `json:"sharedSecret,omitempty"`
	Expiration   uint64    `json:"expiration,omitempty"`
	Protocols    Protocols `json:"protocol,omitempty" validate:"required"`
}

// ErrorMessageResponse is the response returned by the OCM API in case of an error.
// https://cs3org.github.io/OCM-API/docs.html?branch=develop&repo=OCM-API&user=cs3org#/paths/~1notifications/post
type ErrorMessageResponse struct {
	Message          string             `json:"message"`
	ValidationErrors []*ValidationError `json:"validationErrors,omitempty"`
}

// ValidationError is the payload for the validationErrors field in the ErrorMessageResponse.
type ValidationError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// Notifications dispatches any notifications received from remote OCM sites
// according to the specifications at:
// https://cs3org.github.io/OCM-API/docs.html?branch=v1.1.0&repo=OCM-API&user=cs3org#/paths/~1notifications/post
func (h *notifHandler) Notifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := appctx.GetLogger(ctx)
	req, err := getNotification(r)
	if err != nil {
		renderErrorBadRequest(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// TODO(lopresti) this is all to be implemented. For now we just log what we got
	log.Debug().Msgf("Received OCM notification: %+v", req)

	var status *rpc.Status
	if req.Notification == nil || req.Notification.Grantee == "" {
		renderErrorBadRequest(w, r, http.StatusBadRequest, "grantee is required")
		return
	}
	switch req.NotificationType {
	case SHARE_UNSHARED:
		status, err = h.handleShareUnshared(ctx, req)
		if err != nil {
			log.Err(err).Any("notificationRequest", req).Msg("error handling the share unshared notification")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case SHARE_CHANGE_PERMISSION:
		if req.Notification.Protocols == nil {
			renderErrorBadRequest(w, r, http.StatusBadRequest, "protocol is required")
			return
		}
		status, err = h.handleShareChangePermission(ctx, req)
		if err != nil {
			log.Err(err).Any("notificationRequest", req).Msg("error handling the share change permission notification")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		renderErrorBadRequest(w, r, http.StatusBadRequest, "NotificationType "+req.NotificationType+" is not supported")
		return
	}
	// parse the response status
	switch status.GetCode() {
	case rpc.Code_CODE_OK:
		w.WriteHeader(http.StatusCreated)
		return
	case rpc.Code_CODE_NOT_FOUND:
		w.WriteHeader(http.StatusNotFound)
		return
	case rpc.Code_CODE_INVALID_ARGUMENT:
		renderErrorBadRequest(w, r, http.StatusBadRequest, status.GetMessage())
		return
	case rpc.Code_CODE_UNAUTHENTICATED:
		w.WriteHeader(http.StatusUnauthorized)
		return
	case rpc.Code_CODE_PERMISSION_DENIED:
		w.WriteHeader(http.StatusForbidden)
		return
	default:
		log.Error().Str("code", status.GetCode().String()).Str("message", status.GetMessage()).Str("NotificationType", req.NotificationType).Msg("error handling notification")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *notifHandler) handleShareUnshared(ctx context.Context, req *notificationRequest) (*rpc.Status, error) {
	gatewayClient, err := h.gatewaySelector.Next()
	if err != nil {
		return nil, fmt.Errorf("error getting gateway client: %w", err)
	}

	o := &typesv1beta1.Opaque{}
	utils.AppendPlainToOpaque(o, "grantee", req.Notification.Grantee)

	res, err := gatewayClient.DeleteOCMCoreShare(ctx, &ocmcore.DeleteOCMCoreShareRequest{
		Id:     req.ProviderId,
		Opaque: o,
	})
	if err != nil {
		return nil, fmt.Errorf("error calling DeleteOCMCoreShare: %w", err)
	}
	return res.GetStatus(), nil
}

func (h *notifHandler) handleShareChangePermission(ctx context.Context, req *notificationRequest) (*rpc.Status, error) {
	gatewayClient, err := h.gatewaySelector.Next()
	if err != nil {
		return nil, fmt.Errorf("error getting gateway client: %w", err)
	}

	if req.Notification == nil || req.Notification.Protocols == nil {
		return nil, fmt.Errorf("error getting protocols from notification")
	}

	o := &typesv1beta1.Opaque{}
	utils.AppendPlainToOpaque(o, "grantee", req.Notification.Grantee)
	utils.AppendPlainToOpaque(o, "resourceType", req.ResourceType)

	res, err := gatewayClient.UpdateOCMCoreShare(ctx, &ocmcore.UpdateOCMCoreShareRequest{
		OcmShareId: req.ProviderId,
		Protocols:  getProtocols(req.Notification.Protocols, o),
		Opaque:     o,
	})
	if err != nil {
		return nil, fmt.Errorf("error calling DeleteOCMCoreShare: %w", err)
	}
	return res.GetStatus(), nil
}

func getNotification(r *http.Request) (*notificationRequest, error) {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	if contentType != "application/json" {
		return nil, fmt.Errorf("unsupported content type %q", contentType)
	}
	n := &notificationRequest{}
	if err := json.NewDecoder(r.Body).Decode(n); err != nil {
		return nil, err
	}
	return n, nil
}

func renderJSON(w http.ResponseWriter, r *http.Request, statusCode int, resp any) {
	render.Status(r, statusCode)
	render.JSON(w, r, resp)
}

func renderErrorBadRequest(w http.ResponseWriter, r *http.Request, statusCode int, message string) {
	resp := &ErrorMessageResponse{
		Message: "BAD_REQUEST",
		ValidationErrors: []*ValidationError{
			{
				Name:    "notification",
				Message: message,
			},
		},
	}
	renderJSON(w, r, statusCode, resp)
}
