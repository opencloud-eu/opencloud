// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
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

func writeRedeemError(w http.ResponseWriter, err error) {
	var re *authguest.RedeemError
	if !errors.As(err, &re) {
		writeError(w, http.StatusInternalServerError, errorResponse{ErrorType: "internalError", Message: "An internal error occurred."})
		return
	}

	status := http.StatusInternalServerError
	errorType := "internalError"
	switch {
	case errors.Is(re.ErrorType, authguest.ErrExpired):
		status, errorType = http.StatusUnauthorized, "tokenExpired"
	case errors.Is(re.ErrorType, token.ErrInvalidToken):
		status, errorType = http.StatusUnauthorized, "tokenInvalid"
	case errors.Is(re.ErrorType, authguest.ErrAlreadyRedeemed):
		status, errorType = http.StatusConflict, "tokenAlreadyRedeemed"
	case errors.Is(re.ErrorType, authguest.ErrShareNotFound):
		status, errorType = http.StatusNotFound, "shareNotFound"
	case errors.Is(re.ErrorType, authguest.ErrShareExpired):
		status, errorType = http.StatusGone, "shareExpired"
	}

	message := re.ErrorType.Error()
	if errorType == "internalError" {
		message = "An internal error occurred."
	}

	writeError(w, status, errorResponse{ErrorType: errorType, Message: message, PermissionID: re.ShareID})
}
