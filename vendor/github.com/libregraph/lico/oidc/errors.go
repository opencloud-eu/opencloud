/*
 * Copyright 2017-2019 Kopano and its licensors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package oidc

import (
	"fmt"
	"net/http"

	"github.com/libregraph/lico/utils"
)

const (
	// ErrorCodeOAuth2InvalidTarget is the OAuth2 error code for a resource
	// indicator naming a resource this provider cannot issue a token for, as
	// defined in https://www.rfc-editor.org/rfc/rfc8707.html#section-2. It is
	// not defined by oidc-go.
	ErrorCodeOAuth2InvalidTarget = "invalid_target"

	// ErrorCodeOAuth2InvalidScope is the OAuth2 error code for a token request
	// asking for a scope which is unknown or exceeds the one the resource owner
	// granted, as defined in
	// https://datatracker.ietf.org/doc/html/rfc6749#section-5.2. It is not
	// defined by oidc-go.
	ErrorCodeOAuth2InvalidScope = "invalid_scope"

	// ErrorCodeOAuth2InvalidClient is the OAuth2 error code for a token request
	// whose client is unknown or failed to authenticate, as defined in
	// https://datatracker.ietf.org/doc/html/rfc6749#section-5.2. It is not
	// defined by oidc-go.
	ErrorCodeOAuth2InvalidClient = "invalid_client"
)

// OAuth2Error defines a general OAuth2 error with id and decription.
type OAuth2Error struct {
	ErrorID          string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// Error implements the error interface.
func (err *OAuth2Error) Error() string {
	return err.ErrorID
}

// Description implements the ErrorWithDescription interface.
func (err *OAuth2Error) Description() string {
	return err.ErrorDescription
}

// NewOAuth2Error creates a new error with id and description.
func NewOAuth2Error(id string, description string) utils.ErrorWithDescription {
	return &OAuth2Error{id, description}
}

// WriteWWWAuthenticateError writes the provided error with the provided
// http status code to the provided http response writer as a
// WWW-Authenticate header with comma separated fields for id and
// description.
func WriteWWWAuthenticateError(rw http.ResponseWriter, code int, err error) {
	if code == 0 {
		code = http.StatusUnauthorized
	}

	var description string
	switch err.(type) {
	case utils.ErrorWithDescription:
		description = err.(utils.ErrorWithDescription).Description()
	default:
	}

	rw.Header().Set("WWW-Authenticate", fmt.Sprintf("error=\"%s\", error_description=\"%s\"", err.Error(), description))
	rw.WriteHeader(code)
}

// IsErrorWithID returns true if the given error is an OAuth2Error error with
// the given ID.
func IsErrorWithID(err error, id string) bool {
	if err == nil {
		return false
	}

	oauth2Error, ok := err.(*OAuth2Error)
	if !ok {
		return false
	}

	return oauth2Error.ErrorID == id
}
