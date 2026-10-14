// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package appprovider

import (
	"encoding/json"
	"net/http"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
)

// appErrorCode stores the type of error encountered
type appErrorCode string

const (
	appErrorNotFound         appErrorCode = "RESOURCE_NOT_FOUND"
	appErrorAlreadyExists    appErrorCode = "RESOURCE_ALREADY_EXISTS"
	appErrorUnauthenticated  appErrorCode = "UNAUTHENTICATED"
	appErrorPermissionDenied appErrorCode = "PERMISSION_DENIED"
	appErrorUnimplemented    appErrorCode = "NOT_IMPLEMENTED"
	appErrorInvalidParameter appErrorCode = "INVALID_PARAMETER"
	appErrorServerError      appErrorCode = "SERVER_ERROR"
	appErrorTooEarly         appErrorCode = "TOO_EARLY"
	appErrorProviderNotFound appErrorCode = "PROVIDER_NOT_FOUND"
)

// appErrorCodeMapping stores the HTTP error code mapping for various APIErrorCodes
var appErrorCodeMapping = map[appErrorCode]int{
	appErrorNotFound:         http.StatusNotFound,
	appErrorAlreadyExists:    http.StatusForbidden,
	appErrorUnauthenticated:  http.StatusUnauthorized,
	appErrorUnimplemented:    http.StatusNotImplemented,
	appErrorInvalidParameter: http.StatusBadRequest,
	appErrorServerError:      http.StatusInternalServerError,
	appErrorPermissionDenied: http.StatusForbidden,
	appErrorTooEarly:         http.StatusTooEarly,
	appErrorProviderNotFound: http.StatusNotFound,
}

// APIError encompasses the error type and message
type appError struct {
	Code    appErrorCode `json:"code"`
	Message string       `json:"message"`
}

// writeError handles writing error responses
func writeError(w http.ResponseWriter, r *http.Request, code appErrorCode, message string, err error) {
	if err != nil {
		appctx.GetLogger(r.Context()).Error().Err(err).Msg(message)
	}

	var encoded []byte
	w.Header().Set("Content-Type", "application/json")
	encoded, err = json.MarshalIndent(appError{Code: code, Message: message}, "", "  ")

	if err != nil {
		appctx.GetLogger(r.Context()).Error().Err(err).Msg("error encoding response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(appErrorCodeMapping[code])
	_, err = w.Write(encoded)
	if err != nil {
		appctx.GetLogger(r.Context()).Error().Err(err).Msg("error writing response")
		w.WriteHeader(http.StatusInternalServerError)
	}
}
