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
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/libregraph/oidc-go"

	konnectoidc "github.com/libregraph/lico/oidc"
)

// AuthenticationRequest holds the incoming parameters and request data for
// the OpenID Connect 1.0 authorization endpoint as specified at
// http://openid.net/specs/openid-connect-core-1_0.html#AuthRequest and
// http://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthRequest
type AuthenticationRequest struct {
	providerMetadata *oidc.WellKnown

	RawScope        string         `schema:"scope"`
	Claims          *ClaimsRequest `schema:"claims"`
	RawResponseType string         `schema:"response_type"`
	ResponseMode    string         `schema:"response_mode"`
	ClientID        string         `schema:"client_id"`
	RawRedirectURI  string         `schema:"redirect_uri"`
	State           string         `schema:"state"`
	Nonce           string         `schema:"nonce"`
	RawPrompt       string         `schema:"prompt"`
	RawIDTokenHint  string         `schema:"id_token_hint"`
	RawMaxAge       string         `schema:"max_age"`

	RawRequest      string `schema:"request"`
	RawRequestURI   string `schema:"request_uri"`
	RawRegistration string `schema:"registration"`

	CodeChallenge       string `schema:"code_challenge"`
	CodeChallengeMethod string `schema:"code_challenge_method"`

	Scopes        map[string]bool `schema:"-"`
	ResponseTypes map[string]bool `schema:"-"`
	Prompts       map[string]bool `schema:"-"`
	RedirectURI   *url.URL        `schema:"-"`
	IDTokenHint   *jwt.Token      `schema:"-"`
	MaxAge        time.Duration   `schema:"-"`
	Request       *jwt.Token      `schema:"-"`

	UseFragment bool   `schema:"-"`
	Flow        string `schema:"-"`

	Session *Session `schema:"-"`
}

// DecodeAuthenticationRequest returns a AuthenticationRequest holding the
// provided requests form data.
func DecodeAuthenticationRequest(req *http.Request, providerMetadata *oidc.WellKnown, keyFunc jwt.Keyfunc) (*AuthenticationRequest, error) {
	return NewAuthenticationRequest(req.Form, providerMetadata, keyFunc)
}

// NewAuthenticationRequest returns a AuthenticationRequest holding the
// provided url values.
func NewAuthenticationRequest(values url.Values, providerMetadata *oidc.WellKnown, keyFunc jwt.Keyfunc) (*AuthenticationRequest, error) {
	ar := &AuthenticationRequest{
		providerMetadata: providerMetadata,

		Scopes:        make(map[string]bool),
		ResponseTypes: make(map[string]bool),
		Prompts:       make(map[string]bool),
	}
	err := DecodeSchema(ar, values)
	if err != nil {
		return nil, fmt.Errorf("failed to decode authentication request: %v", err)
	}

	if ar.RawScope != "" {
		// Parse scope early, since the value is needed to handle the request
		// parameter properly.
		for _, scope := range strings.Split(ar.RawScope, " ") {
			ar.Scopes[scope] = true
		}
	}

	if ar.RawRequest != "" {
		parser := &jwt.Parser{}
		request, err := parser.ParseWithClaims(ar.RawRequest, &RequestObjectClaims{}, func(token *jwt.Token) (interface{}, error) {
			if keyFunc != nil {
				return keyFunc(token)
			}

			return nil, fmt.Errorf("Not validated")
		})
		if err != nil {
			return nil, ar.NewBadRequest(oidc.ErrorCodeOIDCInvalidRequestObject, err.Error())
		}

		if claims, ok := request.Claims.(*RequestObjectClaims); ok {
			err = ar.ApplyRequestObject(claims, request.Method)
			if err != nil {
				return nil, err
			}
		}

		ar.Request = request
	}

	ar.RedirectURI, _ = url.Parse(ar.RawRedirectURI)

	if ar.RawResponseType != "" {
		for _, rt := range strings.Split(ar.RawResponseType, " ") {
			ar.ResponseTypes[rt] = true
		}
	}
	if ar.RawPrompt != "" {
		for _, prompt := range strings.Split(ar.RawPrompt, " ") {
			ar.Prompts[prompt] = true
		}
	}

	switch ar.RawResponseType {
	case oidc.ResponseTypeCode:
		// Code flow.
		ar.Flow = oidc.FlowCode
		// breaks
	case oidc.ResponseTypeIDToken:
		// Implicit flow.
		fallthrough
	case oidc.ResponseTypeIDTokenToken:
		// Implicit flow with access token.
		ar.UseFragment = true
		ar.Flow = oidc.FlowImplicit
	case oidc.ResponseTypeCodeIDToken:
		// Hybrid flow.
		fallthrough
	case oidc.ResponseTypeCodeToken:
		// Hybgrid flow.
		fallthrough
	case oidc.ResponseTypeCodeIDTokenToken:
		// Hybrid flow.
		ar.UseFragment = true
		ar.Flow = oidc.FlowHybrid
	}

	switch ar.ResponseMode {
	case oidc.ResponseModeFragment:
		ar.UseFragment = true
		// breaks
	case oidc.ResponseModeQuery:
		ar.UseFragment = false
		// breaks
	}

	if ar.RawMaxAge != "" {
		maxAgeInt, err := strconv.ParseInt(ar.RawMaxAge, 10, 64)
		if err != nil {
			return nil, err
		}
		ar.MaxAge = time.Duration(maxAgeInt) * time.Second
	}

	if ar.Claims != nil && ar.Claims.Passthru != nil {
		// Remove pass thru claims when not provided in a secure manner. This
		// means that pass through claims can only be passed via a signed request
		// objects and its claims.
		if ar.Request == nil || ar.Request.Method == jwt.SigningMethodNone || ar.Request.Claims == nil {
			ar.Claims.Passthru = nil
		}
	}

	return ar, nil
}

// ApplyRequestObject applies the provided request object claims to the
// associated authentication request data with validation as required.
func (ar *AuthenticationRequest) ApplyRequestObject(roc *RequestObjectClaims, method jwt.SigningMethod) error {
	// Basic consistency validation following spec at
	// https://openid.net/specs/openid-connect-core-1_0.html#SignedRequestObject
	if ok := ar.Scopes[oidc.ScopeOpenID]; !ok {
		return ar.NewBadRequest(oidc.ErrorCodeOIDCInvalidRequestObject, "openid scope required when using the request parameter")
	}
	if roc.RawScope != "" {
		ar.Scopes = make(map[string]bool)
		// Parse scope directly, since the accociated authentication request
		// has already parsed it when this is called.
		for _, scope := range strings.Split(roc.RawScope, " ") {
			ar.Scopes[scope] = true
		}
	}
	if roc.RawResponseType != "" {
		if roc.RawResponseType != ar.RawResponseType {
			return ar.NewBadRequest(oidc.ErrorCodeOIDCInvalidRequestObject, "request object response_type mismatch")
		}
	}
	if roc.ClientID != "" {
		if roc.ClientID != ar.ClientID {
			return ar.NewBadRequest(oidc.ErrorCodeOIDCInvalidRequestObject, "request object client_id mismatch")
		}
	}

	if method != jwt.SigningMethodNone {
		// Additional claim validation when signed. The spec says that iss and
		// aud SHOULD have defined values. So for now we do not enforce here.
	}

	// Apply rest of the provided request object values to the accociated
	// authentication request.
	if roc.Claims != nil {
		// NOTE(longsleep): Overwrite request claims with the signed claims
		// from the request object. This ensures that only signed claims are
		// processed if any have been given. If no signed claims have been
		// given, the unsigned claims are kept, leaving it to further checks
		// to ensure that only signed claims are used by checking that the
		// roc object has claims.
		ar.Claims = roc.Claims
	}
	if roc.RawRedirectURI != "" {
		ar.RawRedirectURI = roc.RawRedirectURI
	}
	if roc.State != "" {
		ar.State = roc.State
	}
	if roc.Nonce != "" {
		ar.Nonce = roc.Nonce
	}
	if roc.RawPrompt != "" {
		ar.RawPrompt = roc.RawPrompt
	}
	if roc.RawIDTokenHint != "" {
		ar.RawIDTokenHint = roc.RawIDTokenHint
	}
	if roc.RawMaxAge != "" {
		ar.RawMaxAge = roc.RawMaxAge
	}
	if roc.RawRegistration != "" {
		ar.RawRegistration = roc.RawRegistration
	}
	if roc.CodeChallengeMethod != "" {
		ar.CodeChallengeMethod = roc.CodeChallengeMethod
	}
	if roc.CodeChallenge != "" {
		ar.CodeChallenge = roc.CodeChallenge
	}

	return nil
}

// Validate validates the request data of the accociated authentication request.
func (ar *AuthenticationRequest) Validate(keyFunc jwt.Keyfunc) error {
	switch ar.RawResponseType {
	case oidc.ResponseTypeCode:
		// Code flow.
		// breaks
	case oidc.ResponseTypeCodeIDToken:
		// Hybgrid flow.
		if _, ok := ar.Scopes[oidc.ScopeOpenID]; !ok {
			return ar.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, "missing openid scope in request")
		}
		// breaks
	case oidc.ResponseTypeCodeToken:
		// Hybgrid flow.
		// breaks
	case oidc.ResponseTypeCodeIDTokenToken:
		// Hybgrid flow.
		if _, ok := ar.Scopes[oidc.ScopeOpenID]; !ok {
			return ar.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, "missing openid scope in request")
		}
		// breaks
	case oidc.ResponseTypeIDToken:
		// Implicit flow.
		if _, ok := ar.Scopes[oidc.ScopeOpenID]; !ok {
			return ar.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, "missing openid scope in request")
		}
		fallthrough
	case oidc.ResponseTypeIDTokenToken:
		// Implicit flow with access token.
		if _, ok := ar.Scopes[oidc.ScopeOpenID]; !ok {
			return ar.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, "missing openid scope in request")
		}
		if ar.Nonce == "" {
			return ar.NewError(oidc.ErrorCodeOAuth2InvalidRequest, "nonce is required for implicit flow")
		}
	case oidc.ResponseTypeToken:
		// OAuth2 flow implicit grant.
		// breaks
	default:
		return ar.NewError(oidc.ErrorCodeOAuth2UnsupportedResponseType, "")
	}

	// Additional checks for flows with code.
	if ar.Flow == oidc.FlowCode || ar.Flow == oidc.FlowHybrid {
		switch ar.CodeChallengeMethod {
		case "":
			// breaks
		case oidc.S256CodeChallengeMethod:
			// breaks
		case oidc.PlainCodeChallengeMethod:
			// Plain is discouraged, and thus not supported.
			fallthrough
		default:
			return ar.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, "transform algorithm not supported")
		}
	}

	if _, hasNonePrompt := ar.Prompts[oidc.PromptNone]; hasNonePrompt {
		if len(ar.Prompts) > 1 {
			// Cannot have other prompts if none is requested.
			return ar.NewError(oidc.ErrorCodeOAuth2InvalidRequest, "cannot request other prompts together with none")
		}
	}

	if ar.ClientID == "" {
		return ar.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, "missing client_id")
	}
	// TODO(longsleep): implement client_id white list.

	if ar.RedirectURI == nil || !ar.RedirectURI.IsAbs() {
		return ar.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, "invalid or missing redirect_uri")
	}

	if ar.RawIDTokenHint != "" {
		parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		idTokenHint, err := parser.ParseWithClaims(ar.RawIDTokenHint, &konnectoidc.IDTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
			if keyFunc != nil {
				return keyFunc(token)
			}

			return nil, fmt.Errorf("Not validated")
		})
		if err != nil {
			return ar.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, err.Error())
		}
		ar.IDTokenHint = idTokenHint
	}

	// Offline access validation.
	// http://openid.net/specs/openid-connect-core-1_0.html#OfflineAccess
	if ok, _ := ar.Scopes[oidc.ScopeOfflineAccess]; ok {
		if _, withCodeResponseType := ar.ResponseTypes[oidc.ResponseTypeCode]; !withCodeResponseType {
			// Ignore the offline_access request unless the Client is using a
			// response_type value that would result in an Authorization Code
			// being returned.
			delete(ar.Scopes, oidc.ScopeOfflineAccess)
		}
	}

	if ar.RawRequestURI != "" {
		return ar.NewError(oidc.ErrorCodeOIDCRequestURINotSupported, "")
	}
	if ar.RawRegistration != "" {
		return ar.NewError(oidc.ErrorCodeOIDCRegistrationNotSupported, "")
	}

	return nil
}

// Verify checks that the passed parameters match the accociated requirements.
func (ar *AuthenticationRequest) Verify(userID string) error {
	if ar.IDTokenHint != nil {
		// Compare userID with IDTokenHint.
		if userID != ar.IDTokenHint.Claims.(*konnectoidc.IDTokenClaims).Subject {
			return ar.NewError(oidc.ErrorCodeOIDCLoginRequired, "userid mismatch")
		}
	}

	return nil
}

// NewError creates a new error with id and string and the associated request's
// state.
func (ar *AuthenticationRequest) NewError(id string, description string) *AuthenticationError {
	return &AuthenticationError{
		ErrorID:          id,
		ErrorDescription: description,
		State:            ar.State,
	}
}

// NewBadRequest creates a new error with id and string and the associated
// request's state.
func (ar *AuthenticationRequest) NewBadRequest(id string, description string) *AuthenticationBadRequest {
	return &AuthenticationBadRequest{
		ErrorID:          id,
		ErrorDescription: description,
		State:            ar.State,
	}
}

// AuthenticationSuccess holds the outgoind data for a successful OpenID
// Connect 1.0 authorize request as specified at
// http://openid.net/specs/openid-connect-core-1_0.html#AuthResponse and
// http://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthResponse.
// https://openid.net/specs/openid-connect-session-1_0.html#CreatingUpdatingSessions
type AuthenticationSuccess struct {
	Code        string `url:"code,omitempty"`
	AccessToken string `url:"access_token,omitempty"`
	TokenType   string `url:"token_type,omitempty"`
	IDToken     string `url:"id_token,omitempty"`
	State       string `url:"state"`
	ExpiresIn   int64  `url:"expires_in,omitempty"`

	Scope string `url:"scope,omitempty"`

	SessionState string `url:"session_state,omitempty"`
}

// AuthenticationError holds the outgoind data for a failed OpenID
// Connect 1.0 authorize request as specified at
// http://openid.net/specs/openid-connect-core-1_0.html#AuthError and
// http://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthError.
type AuthenticationError struct {
	ErrorID          string `url:"error" json:"error"`
	ErrorDescription string `url:"error_description,omitempty" json:"error_description,omitempty"`
	State            string `url:"state,omitempty" json:"state,omitempty"`
}

// Error interface implementation.
func (ae *AuthenticationError) Error() string {
	return ae.ErrorID
}

// Description implements ErrorWithDescription interface.
func (ae *AuthenticationError) Description() string {
	return ae.ErrorDescription
}

// AuthenticationBadRequest holds the outgoing data for a failed OpenID Connect
// 1.0 authorize request with bad request parameters which make it impossible to
// continue with normal auth.
type AuthenticationBadRequest struct {
	ErrorID          string `url:"error" json:"error"`
	ErrorDescription string `url:"error_description,omitempty" json:"error_description,omitempty"`
	State            string `url:"state,omitempty" json:"state,omitempty"`
}

// Error interface implementation.
func (ae *AuthenticationBadRequest) Error() string {
	return ae.ErrorID
}

// Description implements ErrorWithDescription interface.
func (ae *AuthenticationBadRequest) Description() string {
	return ae.ErrorDescription
}
