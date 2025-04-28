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
	"errors"

	"github.com/golang-jwt/jwt/v5"

	"github.com/libregraph/lico/identity/clients"
)

// RequestObjectClaims holds the incoming request object claims provided as
// JWT via request parameter to OpenID Connect 1.0 authorization endpoint
// requests specified at
// https://openid.net/specs/openid-connect-core-1_0.html#JWTRequests
type RequestObjectClaims struct {
	jwt.RegisteredClaims

	RawScope        string         `json:"scope"`
	Claims          *ClaimsRequest `json:"claims"`
	RawResponseType string         `json:"response_type"`
	ResponseMode    string         `json:"response_mode"`
	ClientID        string         `json:"client_id"`
	RawRedirectURI  string         `json:"redirect_uri"`
	State           string         `json:"state"`
	Nonce           string         `json:"nonce"`
	RawPrompt       string         `json:"prompt"`
	RawIDTokenHint  string         `json:"id_token_hint"`
	RawMaxAge       string         `json:"max_age"`

	RawRegistration string `json:"registration"`

	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`

	client *clients.Secured
}

// SetSecure sets the provided client as owner of the accociated claims.
func (roc *RequestObjectClaims) SetSecure(client *clients.Secured) error {
	if roc.ClientID != client.ID {
		return errors.New("client ID mismatch")
	}

	roc.client = client

	return nil
}

// Secure returns the accociated secure client or nil if not secure.
func (roc *RequestObjectClaims) Secure() *clients.Secured {
	return roc.client
}
