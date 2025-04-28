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

package payload

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt/v5"
	"github.com/libregraph/oidc-go"

	konnectoidc "github.com/libregraph/lico/oidc"
)

// EndSessionRequest holds the incoming parameters and request data for OpenID
// Connect Session Management 1.0 RP initiaed logout requests as specified at
// https://openid.net/specs/openid-connect-session-1_0.html#RPLogout
type EndSessionRequest struct {
	providerMetadata *oidc.WellKnown

	RawIDTokenHint           string `schema:"id_token_hint"`
	RawPostLogoutRedirectURI string `schema:"post_logout_redirect_uri"`
	State                    string `schema:"state"`

	IDTokenHint           *jwt.Token `schema:"-"`
	PostLogoutRedirectURI *url.URL   `schema:"-"`
}

// DecodeEndSessionRequest returns a EndSessionRequest holding the
// provided requests form data.
func DecodeEndSessionRequest(req *http.Request, providerMetadata *oidc.WellKnown) (*EndSessionRequest, error) {
	return NewEndSessionRequest(req.Form, providerMetadata)
}

// NewEndSessionRequest returns a EndSessionRequest holding the
// provided url values.
func NewEndSessionRequest(values url.Values, providerMetadata *oidc.WellKnown) (*EndSessionRequest, error) {
	esr := &EndSessionRequest{
		providerMetadata: providerMetadata,
	}
	err := DecodeSchema(esr, values)
	if err != nil {
		return nil, err
	}

	esr.PostLogoutRedirectURI, _ = url.Parse(esr.RawPostLogoutRedirectURI)

	return esr, nil
}

// Validate validates the request data of the accociated endSession request.
func (esr *EndSessionRequest) Validate(keyFunc jwt.Keyfunc) error {
	if esr.RawIDTokenHint != "" {
		parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		idTokenHint, err := parser.ParseWithClaims(esr.RawIDTokenHint, &konnectoidc.IDTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
			if keyFunc != nil {
				return keyFunc(token)
			}

			return nil, fmt.Errorf("Not validated")
		})
		if err != nil {
			return esr.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, err.Error())
		}
		esr.IDTokenHint = idTokenHint
	}

	return nil
}

// Verify checks that the passed parameters match the accociated requirements.
func (esr *EndSessionRequest) Verify(userID string) error {
	if esr.IDTokenHint != nil {
		// Compare userID with IDTokenHint.
		if userID != esr.IDTokenHint.Claims.(*konnectoidc.IDTokenClaims).Subject {
			return esr.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, "userid mismatch")
		}
	}

	return nil
}

// NewError creates a new error with id and string and the associated request's
// state.
func (esr *EndSessionRequest) NewError(id string, description string) *AuthenticationError {
	return &AuthenticationError{
		ErrorID:          id,
		ErrorDescription: description,
		State:            esr.State,
	}
}

// NewBadRequest creates a new error with id and string and the associated
// request's state.
func (esr *EndSessionRequest) NewBadRequest(id string, description string) *AuthenticationBadRequest {
	return &AuthenticationBadRequest{
		ErrorID:          id,
		ErrorDescription: description,
		State:            esr.State,
	}
}

func (esr *EndSessionRequest) MakeRedirectEndSessionRequestURL() *url.URL {
	if esr.PostLogoutRedirectURI == nil || esr.PostLogoutRedirectURI.String() == "" {
		return nil
	}

	if esr.State == "" {
		return esr.PostLogoutRedirectURI
	}
	uri, _ := url.Parse(esr.PostLogoutRedirectURI.String())
	query := uri.Query()
	query.Add("state", esr.State)
	uri.RawQuery = query.Encode()
	return uri
}
