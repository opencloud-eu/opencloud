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

// maxVerifyTokenBodySize limits the body of the unauthenticated verify token
// request. A token is about 90 bytes, so this leaves plenty of room for the
// JSON wrapping.
const maxVerifyTokenBodySize = 4 << 10

// VerifyTokenRequest is the request body for token verification.
type VerifyTokenRequest struct {
	Token string `json:"token"`
}

type sessionResponse struct {
	PermissionID string `json:"permissionId"`
}

// VerifyTokenHandler validates the token submitted to the verify token endpoint.
func VerifyTokenHandler(log log.Logger, s authguest.AuthGuest, cfg *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxVerifyTokenBodySize)

		var req VerifyTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Debug().Err(err).Msg("request body is malformed")
			writeError(w, http.StatusBadRequest, errorResponse{ErrorType: "invalidRequest", Message: "The request body is malformed."})
			return
		}

		if req.Token == "" {
			log.Debug().Msg("request body is missing the token")
			writeError(w, http.StatusBadRequest, errorResponse{ErrorType: "invalidRequest", Message: "The request body is missing the token."})
			return
		}

		result, err := s.VerifyToken(r.Context(), req.Token)
		if err != nil {
			log.Debug().Err(err).Msg("verify token failed")
			writeGuestError(w, err)
			return
		}

		setSessionCookie(w, cfg, result.SessionToken)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(sessionResponse{PermissionID: result.ShareID})
	}
}

func setSessionCookie(w http.ResponseWriter, cfg *config.Config, sessionToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.JWT.CookieName,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(cfg.JWT.TTL.Seconds()),
	})
}
