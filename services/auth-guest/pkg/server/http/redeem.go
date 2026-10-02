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

// RedeemRequest is the request body for token redemption.
type RedeemRequest struct {
	Token string `json:"token"`
}

type redeemResponse struct {
	PermissionID string `json:"permissionId"`
}

// RedeemHandler validates the token submitted to the redeem endpoint.
func RedeemHandler(log log.Logger, s authguest.AuthGuest, cfg *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RedeemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Debug().Err(err).Msg("request body is malformed")
			writeError(w, http.StatusBadRequest, errorResponse{ErrorType: "invalidRequest", Message: "The request body is malformed."})
			return
		}

		result, err := s.Redeem(r.Context(), req.Token)
		if err != nil {
			log.Debug().Err(err).Msg("redeem failed")
			writeRedeemError(w, err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     cfg.JWT.CookieName,
			Value:    result.SessionToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(cfg.JWT.TTL.Seconds()),
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(redeemResponse{PermissionID: result.ShareID})
	}
}
