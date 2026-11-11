// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogoutHandler(t *testing.T) {
	tests := []struct {
		name          string
		sessionCookie bool
	}{
		{name: "with session cookie", sessionCookie: true},
		{name: "without session cookie", sessionCookie: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{JWT: config.JWT{CookieName: testCookieName}}
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.sessionCookie {
				req.AddCookie(&http.Cookie{Name: testCookieName, Value: "session-token"})
			}

			rr := httptest.NewRecorder()
			LogoutHandler(cfg)(rr, req)

			assert.Equal(t, http.StatusNoContent, rr.Code)
			cookies := rr.Result().Cookies()
			require.Len(t, cookies, 1)
			cookie := cookies[0]
			assert.Equal(t, testCookieName, cookie.Name)
			assert.Empty(t, cookie.Value)
			assert.Equal(t, -1, cookie.MaxAge)
			assert.Equal(t, "/", cookie.Path)
			assert.True(t, cookie.HttpOnly)
			assert.True(t, cookie.Secure)
			assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		})
	}
}
