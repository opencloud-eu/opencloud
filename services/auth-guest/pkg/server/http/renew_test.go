// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest/mocks"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/jwt"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testCookieName = "__Host-oc_guest_session"

const testNonceCookieName = "__Host-oc_guest_nonce"

func newRenewHandler(t *testing.T, svc authguest.AuthGuest) http.HandlerFunc {
	t.Helper()
	cfg := &config.Config{
		JWT: config.JWT{
			CookieName: testCookieName,
			TTL:        time.Hour,
		},
		Nonce: config.Nonce{
			CookieBaseName: testNonceCookieName,
		},
		PinAndNonceTTL: time.Hour,
	}
	return RenewHandler(log.NopLogger(), svc, cfg)
}

func TestRenewHandlerWithSessionCookie(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	svcMock.On("Renew", mock.Anything, "share-1", "session-token").Return("nonce-value", nil)

	body, err := json.Marshal(RenewRequest{PermissionID: "share-1"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
	req.AddCookie(&http.Cookie{
		Name:     testCookieName,
		Value:    "session-token",
		HttpOnly: true,
		Secure:   true,
	})

	rr := httptest.NewRecorder()
	newRenewHandler(t, svcMock)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	svcMock.AssertCalled(t, "Renew", mock.Anything, "share-1", "session-token")

	var resp RenewResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.True(t, resp.WithPin)

	wantName := nonceCookieName(testNonceCookieName, "share-1")
	var cookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == wantName {
			cookie = c
		}
	}
	require.NotNil(t, cookie)
	assert.Equal(t, wantName, cookie.Name)
	assert.True(t, strings.HasPrefix(cookie.Name, testNonceCookieName))
	assert.Equal(t, "nonce-value", cookie.Value)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, "/", cookie.Path)
}

func TestRenewHandlerWithoutSessionCookie(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	svcMock.On("Renew", mock.Anything, "share-1", "").Return("", nil)

	body, err := json.Marshal(RenewRequest{PermissionID: "share-1"})
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	newRenewHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body))))

	assert.Equal(t, http.StatusOK, rr.Code)
	svcMock.AssertCalled(t, "Renew", mock.Anything, "share-1", "")

	var resp RenewResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.False(t, resp.WithPin)
}

func TestRenewHandlerSkipsNonceCookieWhenEmpty(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	svcMock.On("Renew", mock.Anything, "share-1", "session-token").Return("", nil)

	body, err := json.Marshal(RenewRequest{PermissionID: "share-1"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
	req.AddCookie(&http.Cookie{
		Name:     testCookieName,
		Value:    "session-token",
		HttpOnly: true,
		Secure:   true,
	})

	rr := httptest.NewRecorder()
	newRenewHandler(t, svcMock)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Result().Cookies())

	var resp RenewResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.False(t, resp.WithPin)
}

func TestRenewHandlerNonceCookiePerShare(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	svcMock.On("Renew", mock.Anything, "share-1", "session-token").Return("nonce-1", nil)
	svcMock.On("Renew", mock.Anything, "share-2", "session-token").Return("nonce-2", nil)

	names := map[string]string{}
	for _, shareID := range []string{"share-1", "share-2"} {
		body, err := json.Marshal(RenewRequest{PermissionID: shareID})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
		req.AddCookie(&http.Cookie{
			Name:     testCookieName,
			Value:    "session-token",
			HttpOnly: true,
			Secure:   true,
		})

		rr := httptest.NewRecorder()
		newRenewHandler(t, svcMock)(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		cookies := rr.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, nonceCookieName(testNonceCookieName, shareID), cookies[0].Name)
		names[shareID] = cookies[0].Name
	}

	assert.NotEqual(t, names["share-1"], names["share-2"])
}

func TestRenewHandlerErrorMapping(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantStatus     int
		wantType       string
		wantPermission string
	}{
		{
			name:           "share not found",
			err:            &authguest.GuestError{ErrorType: authguest.ErrShareNotFound, ShareID: "share-1"},
			wantStatus:     http.StatusNotFound,
			wantType:       "shareNotFound",
			wantPermission: "share-1",
		},
		{
			name:           "share expired",
			err:            &authguest.GuestError{ErrorType: authguest.ErrShareExpired, ShareID: "share-1"},
			wantStatus:     http.StatusGone,
			wantType:       "shareExpired",
			wantPermission: "share-1",
		},
		{
			name:           "permission mismatch",
			err:            &authguest.GuestError{ErrorType: authguest.ErrPermissionMismatch, ShareID: "share-1"},
			wantStatus:     http.StatusBadRequest,
			wantType:       "invalidRequest",
			wantPermission: "share-1",
		},
		{
			name:           "invalid session",
			err:            &authguest.GuestError{ErrorType: jwt.ErrInvalidSession, ShareID: "share-1"},
			wantStatus:     http.StatusUnauthorized,
			wantType:       "sessionInvalid",
			wantPermission: "share-1",
		},
		{
			name:           "concurrent update",
			err:            &authguest.GuestError{ErrorType: storage.ErrConflict, ShareID: "share-1"},
			wantStatus:     http.StatusConflict,
			wantType:       "conflict",
			wantPermission: "share-1",
		},
		{
			name:       "internal error",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantType:   "internalError",
		},
		{
			name:       "events not configured",
			err:        authguest.ErrEventsNotConfigured,
			wantStatus: http.StatusServiceUnavailable,
			wantType:   "serviceUnavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := mocks.NewAuthGuest(t)
			svcMock.On("Renew", mock.Anything, "share-1", "").Return("", tt.err)

			body, err := json.Marshal(RenewRequest{PermissionID: "share-1"})
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			newRenewHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body))))

			assert.Equal(t, tt.wantStatus, rr.Code)

			var resp errorResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
			assert.Equal(t, tt.wantType, resp.ErrorType)
			assert.Equal(t, tt.wantPermission, resp.PermissionID)
		})
	}
}

func TestRenewHandlerMalformedBody(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)

	rr := httptest.NewRecorder()
	newRenewHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-json")))

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp errorResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "invalidRequest", resp.ErrorType)
}

func TestRenewHandlerMissingPermissionID(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)

	body, err := json.Marshal(RenewRequest{})
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	newRenewHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body))))

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp errorResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "invalidRequest", resp.ErrorType)
}
