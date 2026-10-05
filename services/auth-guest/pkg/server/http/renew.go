// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
)

// RenewRequest is the request body for token renewal.
type RenewRequest struct {
	PermissionID string `json:"permissionId"`
}

// RenewHandler generates a new guest link token for the submitted share.
func RenewHandler(log log.Logger, s authguest.AuthGuest) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RenewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Debug().Err(err).Msg("request body is malformed")
			writeError(w, http.StatusBadRequest, errorResponse{ErrorType: "invalidRequest", Message: "The request body is malformed."})
			return
		}

		if req.PermissionID == "" {
			log.Debug().Msg("request body is missing the permission id")
			writeError(w, http.StatusBadRequest, errorResponse{ErrorType: "invalidRequest", Message: "The request body is missing the permission id."})
			return
		}

		if err := s.Renew(r.Context(), req.PermissionID); err != nil {
			log.Debug().Err(err).Msg("renew failed")
			writeGuestError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
