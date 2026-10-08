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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newVerifyPinHandler(t *testing.T, svc authguest.AuthGuest) http.HandlerFunc {
	t.Helper()
	cfg := &config.Config{
		JWT: config.JWT{
			CookieName: "__Host-oc_guest_session",
			TTL:        time.Hour,
		},
		Challenge: config.Challenge{
			CookieName: testChallengeCookieName,
			TTL:        time.Hour,
		},
	}
	return VerifyPinHandler(log.NopLogger(), svc, cfg)
}

func TestVerifyPinHandler(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	expiresAt := time.Date(2026, 10, 15, 10, 30, 0, 0, time.FixedZone("CEST", 2*60*60))
	svcMock.On("VerifyPin", mock.Anything, "share-1", "123456", "challenge-nonce").Return(&authguest.SessionResponse{SessionToken: "session-token", ExpiresAt: expiresAt, ShareID: "share-1"}, nil)

	body, err := json.Marshal(VerifyPinRequest{Pin: "123456", PermissionID: "share-1"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
	req.AddCookie(&http.Cookie{Name: testChallengeCookieName, Value: "challenge-nonce"})

	rr := httptest.NewRecorder()
	newVerifyPinHandler(t, svcMock)(rr, req)

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

	var challengeCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == testChallengeCookieName {
			challengeCookie = c
		}
	}
	require.NotNil(t, challengeCookie)
	assert.Equal(t, -1, challengeCookie.MaxAge)

	var resp verifyPinResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "share-1", resp.PermissionID)
	assert.Equal(t, "2026-10-15T08:30:00Z", resp.ExpirationDateTime.Format(time.RFC3339))
}

func TestVerifyPinHandlerErrorMapping(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantStatus     int
		wantType       string
		wantPermission string
	}{
		{
			name:           "pin invalid",
			err:            &authguest.GuestError{ErrorType: authguest.ErrPinInvalid, ShareID: "share-1"},
			wantStatus:     http.StatusUnauthorized,
			wantType:       "pinInvalid",
			wantPermission: "share-1",
		},
		{
			name:           "pin expired",
			err:            &authguest.GuestError{ErrorType: authguest.ErrPinExpired, ShareID: "share-1"},
			wantStatus:     http.StatusUnauthorized,
			wantType:       "pinExpired",
			wantPermission: "share-1",
		},
		{
			name:           "challenge invalid",
			err:            &authguest.GuestError{ErrorType: authguest.ErrChallengeInvalid, ShareID: "share-1"},
			wantStatus:     http.StatusUnauthorized,
			wantType:       "challengeInvalid",
			wantPermission: "share-1",
		},
		{
			name:           "challenge expired",
			err:            &authguest.GuestError{ErrorType: authguest.ErrChallengeExpired, ShareID: "share-1"},
			wantStatus:     http.StatusUnauthorized,
			wantType:       "challengeExpired",
			wantPermission: "share-1",
		},
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
			name:       "internal error",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantType:   "internalError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := mocks.NewAuthGuest(t)
			svcMock.On("VerifyPin", mock.Anything, "share-1", "123456", mock.Anything).Return(nil, tt.err)

			body, err := json.Marshal(VerifyPinRequest{Pin: "123456", PermissionID: "share-1"})
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
			req.AddCookie(&http.Cookie{Name: testChallengeCookieName, Value: "challenge-nonce"})

			rr := httptest.NewRecorder()
			newVerifyPinHandler(t, svcMock)(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)

			var resp errorResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
			assert.Equal(t, tt.wantType, resp.ErrorType)
			assert.Equal(t, tt.wantPermission, resp.PermissionID)
		})
	}
}

func TestVerifyPinHandlerMissingChallengeCookie(t *testing.T) {
	tests := []struct {
		name        string
		addCookie   bool
		cookieValue string
	}{
		{name: "no cookie"},
		{name: "empty cookie value", addCookie: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := mocks.NewAuthGuest(t)

			body, err := json.Marshal(VerifyPinRequest{Pin: "123456", PermissionID: "share-1"})
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
			if tt.addCookie {
				req.AddCookie(&http.Cookie{Name: testChallengeCookieName, Value: tt.cookieValue})
			}

			rr := httptest.NewRecorder()
			newVerifyPinHandler(t, svcMock)(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
			svcMock.AssertNotCalled(t, "VerifyPin", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

			var resp errorResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
			assert.Equal(t, "invalidRequest", resp.ErrorType)
		})
	}
}

func TestVerifyPinHandlerMalformedBody(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)

	rr := httptest.NewRecorder()
	newVerifyPinHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-json")))

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var resp errorResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "invalidRequest", resp.ErrorType)
}

func TestVerifyPinHandlerMissingFields(t *testing.T) {
	tests := []struct {
		name string
		req  VerifyPinRequest
	}{
		{name: "missing pin", req: VerifyPinRequest{PermissionID: "share-1"}},
		{name: "missing permission id", req: VerifyPinRequest{Pin: "123456"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcMock := mocks.NewAuthGuest(t)

			body, err := json.Marshal(tt.req)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			newVerifyPinHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body))))

			assert.Equal(t, http.StatusBadRequest, rr.Code)

			var resp errorResponse
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
			assert.Equal(t, "invalidRequest", resp.ErrorType)
		})
	}
}
