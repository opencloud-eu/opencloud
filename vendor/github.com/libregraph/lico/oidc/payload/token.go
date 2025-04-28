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
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/libregraph/oidc-go"

	konnectoidc "github.com/libregraph/lico/oidc"
)

// TokenRequest holds the incoming parameters and request data for
// the OpenID Connect 1.0 token endpoint as specified at
// http://openid.net/specs/openid-connect-core-1_0.html#TokenRequest
type TokenRequest struct {
	providerMetadata *oidc.WellKnown

	GrantType       string `schema:"grant_type"`
	Code            string `schema:"code"`
	RawRedirectURI  string `schema:"redirect_uri"`
	RawRefreshToken string `schema:"refresh_token"`
	RawScope        string `schema:"scope"`

	ClientID     string `schema:"client_id"`
	ClientSecret string `schema:"client_secret"`

	CodeVerifier string `schema:"code_verifier"`

	RedirectURI  *url.URL        `schema:"-"`
	RefreshToken *jwt.Token      `schema:"-"`
	Scopes       map[string]bool `schema:"-"`
}

// DecodeTokenRequest return a TokenRequest holding the provided
// request's form data.
func DecodeTokenRequest(req *http.Request, providerMetadata *oidc.WellKnown) (*TokenRequest, error) {
	tr, err := NewTokenRequest(req.PostForm, providerMetadata)
	if err != nil {
		return nil, err
	}

	var clientID string
	var clientSecret string

	auth := strings.SplitN(req.Header.Get("Authorization"), " ", 2)
	switch auth[0] {
	case "Basic":
		// Support client_secret_basic authentication method.
		if len(auth) != 2 {
			return nil, fmt.Errorf("invalid Basic authorization header format")
		}
		var basic []byte
		if basic, err = base64.StdEncoding.DecodeString(auth[1]); err != nil {
			return nil, fmt.Errorf("invalid Basic authorization value: %w", err)
		}
		// Decode username as client ID and password as client secret. See
		// https://tools.ietf.org/html/rfc6749#section-2.3.1 for details.
		check := strings.SplitN(string(basic), ":", 2)
		if len(check) == 2 {
			// Data is encoded application/x-www-form-urlencoded UTF-8. See
			// https://tools.ietf.org/html/rfc6749#appendix-B for details.
			if clientID, err = url.QueryUnescape(check[0]); err == nil {
				clientSecret, _ = url.QueryUnescape(check[1])
			}
		}
	}

	if tr.ClientID == "" {
		if clientID == "" {
			return nil, fmt.Errorf("client_id is missing")
		}
		// Use client ID and secret if no client_id was passed to the request directly.
		tr.ClientID = clientID
		tr.ClientSecret = clientSecret
	} else if clientID != "" {
		if tr.ClientID == clientID {
			// Update the client secret, if the ID is a match. This replaces
			// a directly given secret.
			tr.ClientSecret = clientSecret
		}
	}

	return tr, err
}

// NewTokenRequest returns a TokenRequest holding the provided url values.
func NewTokenRequest(values url.Values, providerMetadata *oidc.WellKnown) (*TokenRequest, error) {
	tr := &TokenRequest{
		providerMetadata: providerMetadata,

		Scopes: make(map[string]bool),
	}

	err := DecodeSchema(tr, values)
	if err != nil {
		return nil, err
	}

	tr.RedirectURI, _ = url.Parse(tr.RawRedirectURI)

	if tr.RawScope != "" {
		for _, scope := range strings.Split(tr.RawScope, " ") {
			tr.Scopes[scope] = true
		}
	}

	return tr, nil
}

// Validate validates the request data of the accociated token request.
func (tr *TokenRequest) Validate(keyFunc jwt.Keyfunc, claims jwt.Claims) error {
	switch tr.GrantType {
	case oidc.GrantTypeAuthorizationCode:
		// breaks
	case oidc.GrantTypeRefreshToken:
		if tr.RawRefreshToken != "" {
			refreshToken, err := jwt.ParseWithClaims(tr.RawRefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
				if keyFunc != nil {
					return keyFunc(token)
				}

				return nil, fmt.Errorf("Not validated")
			})
			if err != nil {
				return konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidRequest, err.Error())
			}
			tr.RefreshToken = refreshToken
		}
		// breaks

	default:
		return konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2UnsupportedGrantType, "unsupported grant_type value")
	}

	return nil
}

// TokenSuccess holds the outgoing data for a successful OpenID
// Connect 1.0 token request as specified at
// http://openid.net/specs/openid-connect-core-1_0.html#TokenResponse.
type TokenSuccess struct {
	AccessToken  string `json:"access_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
}
