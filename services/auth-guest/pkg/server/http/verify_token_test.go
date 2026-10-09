// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest/mocks"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newVerifyTokenHandler(t *testing.T, svc authguest.AuthGuest) http.HandlerFunc {
	t.Helper()
	cfg := &config.Config{
		JWT: config.JWT{
			CookieName: "__Host-oc_guest_session",
			TTL:        time.Hour,
		},
	}
	return VerifyTokenHandler(log.NopLogger(), svc, cfg)
}

func TestVerifyTokenHandler(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	svcMock.On("VerifyToken", mock.Anything, "valid-token").Return(&authguest.SessionResponse{SessionToken: "session-token", ShareID: "share-1"}, nil)

	body, err := json.Marshal(VerifyTokenRequest{Token: "valid-token"})
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	newVerifyTokenHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body))))

	assert.Equal(t, http.StatusOK, rr.Code)

	var cookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == "__Host-oc_guest_session" {
			cookie = c
		}
	}
	require.NotNil(t, cookie)
	assert.Equal(t, "session-token", cookie.Value)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, "/", cookie.Path)

	var resp sessionResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "share-1", resp.PermissionID)
}

func TestVerifyTokenHandlerErrorMapping(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantStatus     int
		wantType       string
		wantPermission string
	}{
		{
			name:           "token expired",
			err:            &authguest.GuestError{ErrorType: authguest.ErrExpired, ShareID: "share-1"},
			wantStatus:     http.StatusUnauthorized,
			wantType:       "tokenExpired",
			wantPermission: "share-1",
		},
		{
			name:       "token invalid",
			err:        &authguest.GuestError{ErrorType: token.ErrInvalidToken},
			wantStatus: http.StatusUnauthorized,
			wantType:   "tokenInvalid",
		},
		{
			name:       "token not found",
			err:        &authguest.GuestError{ErrorType: storage.ErrNotFound},
			wantStatus: http.StatusNotFound,
			wantType:   "tokenNotFound",
		},
		{
			name:       "token already redeemed",
			err:        &authguest.GuestError{ErrorType: authguest.ErrAlreadyRedeemed},
			wantStatus: http.StatusConflict,
			wantType:   "tokenAlreadyRedeemed",
		},
		{
			name:       "share not found",
			err:        &authguest.GuestError{ErrorType: authguest.ErrShareNotFound},
			wantStatus: http.StatusNotFound,
			wantType:   "shareNotFound",
		},
		{
			name:       "share expired",
			err:        &authguest.GuestError{ErrorType: authguest.ErrShareExpired},
			wantStatus: http.StatusGone,
			wantType:   "shareExpired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := mocks.NewAuthGuest(t)
			svcMock.On("VerifyToken", mock.Anything, "token").Return(nil, tt.err)

			body, err := json.Marshal(VerifyTokenRequest{Token: "token"})
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			newVerifyTokenHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body))))

			assert.Equal(t, tt.wantStatus, rr.Code)

			var resp errorResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
			assert.Equal(t, tt.wantType, resp.ErrorType)
			assert.Equal(t, tt.wantPermission, resp.PermissionID)
		})
	}
}

func TestVerifyTokenHandlerBodyTooLarge(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)

	body := `{"token":"` + strings.Repeat("a", maxVerifyTokenBodySize) + `"}`
	rr := httptest.NewRecorder()
	newVerifyTokenHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	svcMock.AssertNotCalled(t, "VerifyToken", mock.Anything, mock.Anything)
}

func TestVerifyTokenHandlerMalformedBody(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)

	rr := httptest.NewRecorder()
	newVerifyTokenHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-json")))

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp errorResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "invalidRequest", resp.ErrorType)
}

func TestVerifyTokenHandlerMissingToken(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)

	body, err := json.Marshal(VerifyTokenRequest{})
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	newVerifyTokenHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body))))

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp errorResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "invalidRequest", resp.ErrorType)
}
