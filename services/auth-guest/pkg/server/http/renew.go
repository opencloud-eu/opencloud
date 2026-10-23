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

// RenewRequest is the request body for token renewal. The share id is required;
// the guest is authorized by the session cookie, when present.
type RenewRequest struct {
	PermissionID string `json:"permissionId"`
}

// RenewResponse is the response body of the renew endpoint. WithPin reports
// whether the renewal email contains a PIN, which is true exactly when the
// request was authorized by a valid session.
type RenewResponse struct {
	WithPin bool `json:"withPin"`
}

// RenewHandler generates a new guest link token and PIN for the share
// identified by the permission id, authorized by the session cookie. The
// response body reports whether the renewal email contains a PIN.
func RenewHandler(log log.Logger, s authguest.AuthGuest, cfg *config.Config) func(w http.ResponseWriter, r *http.Request) {
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

		sessionToken := ""
		if cookie, err := r.Cookie(cfg.JWT.CookieName); err == nil {
			sessionToken = cookie.Value
		}

		nonce, err := s.Renew(r.Context(), req.PermissionID, sessionToken)
		if err != nil {
			log.Debug().Err(err).Msg("renew failed")
			writeGuestError(w, err)
			return
		}

		if nonce != "" {
			setNonceCookie(w, cfg, req.PermissionID, nonce)
		}

		// A non-empty nonce means the PIN path ran, so the mail carries a PIN.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RenewResponse{WithPin: nonce != ""})
	}
}
