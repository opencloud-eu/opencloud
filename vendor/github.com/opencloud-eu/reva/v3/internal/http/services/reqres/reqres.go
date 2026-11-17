// Copyright 2018-2023 CERN
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

package reqres

import (
	"encoding/json"
	"net/http"

	"github.com/opencloud-eu/reva/v3/pkg/appctx"
)

// APIErrorCode stores the type of error encountered.
type APIErrorCode string

// The various types of errors that can be expected to occur.
const (
	APIErrorNotFound         APIErrorCode = "RESOURCE_NOT_FOUND"
	APIErrorUnauthenticated  APIErrorCode = "UNAUTHENTICATED"
	APIErrorUntrustedService APIErrorCode = "UNTRUSTED_SERVICE"
	APIErrorUnimplemented    APIErrorCode = "FUNCTION_NOT_IMPLEMENTED"
	APIErrorInvalidParameter APIErrorCode = "INVALID_PARAMETER"
	APIErrorProviderError    APIErrorCode = "PROVIDER_ERROR"
	APIErrorAlreadyExist     APIErrorCode = "ALREADY_EXIST"
	APIErrorServerError      APIErrorCode = "SERVER_ERROR"
)

// APIErrorCodeMapping stores the HTTP error code mapping for various APIErrorCodes.
var APIErrorCodeMapping = map[APIErrorCode]int{
	APIErrorNotFound:         http.StatusNotFound,
	APIErrorUnauthenticated:  http.StatusUnauthorized,
	APIErrorUntrustedService: http.StatusForbidden,
	APIErrorUnimplemented:    http.StatusNotImplemented,
	APIErrorInvalidParameter: http.StatusBadRequest,
	APIErrorProviderError:    http.StatusBadGateway,
	APIErrorAlreadyExist:     http.StatusConflict,
	APIErrorServerError:      http.StatusInternalServerError,
}

// APIError encompasses the error type and message.
type APIError struct {
	Code    APIErrorCode `json:"code"`
	Message string       `json:"message"`
}

// WriteError handles writing error responses.
func WriteError(w http.ResponseWriter, r *http.Request, code APIErrorCode, message string, e error) {
	if e != nil {
		appctx.GetLogger(r.Context()).Error().Err(e).Msg(message)
	}

	var encoded []byte
	var err error
	w.Header().Set("Content-Type", "application/json")
	encoded, err = json.MarshalIndent(APIError{Code: code, Message: message}, "", "  ")

	if err != nil {
		appctx.GetLogger(r.Context()).Error().Err(err).Msg("error encoding response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(APIErrorCodeMapping[code])
	_, err = w.Write(encoded)
	if err != nil {
		appctx.GetLogger(r.Context()).Error().Err(err).Msg("error writing response")
		w.WriteHeader(http.StatusInternalServerError)
	}
}
