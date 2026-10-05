// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/token"
)

type errorResponse struct {
	ErrorType    string `json:"errorType"`
	Message      string `json:"message"`
	PermissionID string `json:"permissionId"`
}

func writeError(w http.ResponseWriter, status int, body errorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeGuestError(w http.ResponseWriter, err error) {
	var ge *authguest.GuestError
	if !errors.As(err, &ge) {
		writeError(w, http.StatusInternalServerError, errorResponse{ErrorType: "internalError", Message: "An internal error occurred."})
		return
	}

	status := http.StatusInternalServerError
	errorType := "internalError"
	switch {
	case errors.Is(ge.ErrorType, authguest.ErrExpired):
		status, errorType = http.StatusUnauthorized, "tokenExpired"
	case errors.Is(ge.ErrorType, token.ErrInvalidToken):
		status, errorType = http.StatusUnauthorized, "tokenInvalid"
	case errors.Is(ge.ErrorType, storage.ErrNotFound):
		status, errorType = http.StatusNotFound, "tokenNotFound"
	case errors.Is(ge.ErrorType, storage.ErrInvalidHash):
		status, errorType = http.StatusUnauthorized, "tokenInvalid"
	case errors.Is(ge.ErrorType, authguest.ErrAlreadyRedeemed):
		status, errorType = http.StatusConflict, "tokenAlreadyRedeemed"
	case errors.Is(ge.ErrorType, authguest.ErrShareNotFound):
		status, errorType = http.StatusNotFound, "shareNotFound"
	case errors.Is(ge.ErrorType, authguest.ErrShareExpired):
		status, errorType = http.StatusGone, "shareExpired"
	case errors.Is(ge.ErrorType, authguest.ErrPinInvalid):
		status, errorType = http.StatusUnauthorized, "pinInvalid"
	case errors.Is(ge.ErrorType, authguest.ErrPinExpired):
		status, errorType = http.StatusUnauthorized, "pinExpired"
	}

	message := ge.ErrorType.Error()
	if errorType == "internalError" {
		message = "An internal error occurred."
	}

	writeError(w, status, errorResponse{ErrorType: errorType, Message: message, PermissionID: ge.ShareID})
}
