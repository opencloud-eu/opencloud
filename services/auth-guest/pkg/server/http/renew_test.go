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

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newRenewHandler(t *testing.T, svc authguest.AuthGuest) http.HandlerFunc {
	t.Helper()
	return RenewHandler(log.NopLogger(), svc)
}

func TestRenewHandler(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	svcMock.On("Renew", mock.Anything, "share-1").Return(nil)

	body, err := json.Marshal(RenewRequest{PermissionID: "share-1"})
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	newRenewHandler(t, svcMock)(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body))))

	assert.Equal(t, http.StatusOK, rr.Code)
	svcMock.AssertCalled(t, "Renew", mock.Anything, "share-1")
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
			err:            &authguest.RedeemError{ErrorType: authguest.ErrShareNotFound, ShareID: "share-1"},
			wantStatus:     http.StatusNotFound,
			wantType:       "shareNotFound",
			wantPermission: "share-1",
		},
		{
			name:           "share expired",
			err:            &authguest.RedeemError{ErrorType: authguest.ErrShareExpired, ShareID: "share-1"},
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
			svcMock.On("Renew", mock.Anything, "share-1").Return(tt.err)

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
