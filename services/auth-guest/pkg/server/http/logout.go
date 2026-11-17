// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
)

// LogoutHandler ends the guest session in this browser by clearing the session
// cookie. The session token stays valid until it expires.
func LogoutHandler(cfg *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		clearSessionCookie(w, cfg)
		w.WriteHeader(http.StatusNoContent)
	}
}
