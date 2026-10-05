// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
)

// VerifyPinRequest is the request body for PIN verification.
type VerifyPinRequest struct {
	Pin          string `json:"pin"`
	PermissionID string `json:"permissionId"`
}

type verifyPinResponse = redeemResponse

// VerifyPinHandler validates the PIN submitted to the verify pin endpoint.
func VerifyPinHandler(log log.Logger, s authguest.AuthGuest, cfg *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req VerifyPinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Debug().Err(err).Msg("request body is malformed")
			writeError(w, http.StatusBadRequest, errorResponse{ErrorType: "invalidRequest", Message: "The request body is malformed."})
			return
		}

		if req.PermissionID == "" || req.Pin == "" {
			log.Debug().Msg("request body is missing the permission id or the pin")
			writeError(w, http.StatusBadRequest, errorResponse{ErrorType: "invalidRequest", Message: "The request body is missing the permission id or the pin."})
			return
		}

		result, err := s.VerifyPin(r.Context(), req.PermissionID, req.Pin)
		if err != nil {
			log.Debug().Err(err).Msg("verify pin failed")
			writeGuestError(w, err)
			return
		}

		setSessionCookie(w, cfg, result.SessionToken)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(verifyPinResponse{PermissionID: result.ShareID})
	}
}
