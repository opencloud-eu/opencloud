/*
 * Copyright 2017-2021 Kopano and its licensors
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

package lico

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"

	"github.com/libregraph/lico/oidc"
	"github.com/libregraph/lico/oidc/payload"
)

// Access token claims used.
const (
	RefClaim              = "lg.r"
	IdentityClaim         = "lg.i"
	IdentityProviderClaim = "lg.p"
	ScopesClaim           = "scp"
)

// Identifier identity sub claims used.
const (
	IdentifiedUserClaim        = "us"
	IdentifiedUserIDClaim      = "id"
	IdentifiedUsernameClaim    = "un"
	IdentifiedDisplayNameClaim = "dn"
	IdentifiedData             = "da"
	IdentifiedUserIsGuest      = "gu"
)

// Internal claim names used for special things.
const (
	InternalExtraIDTokenClaimsClaim     = "$lico.id.extra"
	InternalExtraAccessTokenClaimsClaim = "$lico.at.extra"
)

// TokenType defines the token type value.
type TokenTypeValue string

// Is compares the associated TokenTypeValue to the provided one.
func (ttv TokenTypeValue) Is(value TokenTypeValue) bool {
	return ttv == value
}

// The known token type values.
const (
	TokenTypeIDToken      TokenTypeValue = "" // Just a placeholder, not actually set in ID Tokens.
	TokenTypeAccessToken  TokenTypeValue = "1"
	TokenTypeRefreshToken TokenTypeValue = "2"
)

// AccessTokenClaims define the claims found in access tokens issued.
type AccessTokenClaims struct {
	jwt.RegisteredClaims

	TokenType TokenTypeValue `json:"lg.t"`

	// AuthorizedParty is the client the token was issued to, set only when the
	// audience is a resource rather than that client, see
	// https://www.rfc-editor.org/rfc/rfc8707.html.
	AuthorizedParty string `json:"azp,omitempty"`

	AuthorizedClaimsRequest *payload.ClaimsRequest `json:"lg.acr,omitempty"`

	AuthorizedScopesList payload.ScopesValue `json:"scp"`

	IdentityClaims   jwt.MapClaims `json:"lg.i"`
	IdentityProvider string        `json:"lg.p,omitempty"`

	*oidc.SessionClaims
}

// AuthorizedPartyOrAudience returns the client the token was issued to. It is
// the azp claim when the token is bound to a resource, and the single audience
// value otherwise.
func (c AccessTokenClaims) AuthorizedPartyOrAudience() string {
	if c.AuthorizedParty != "" {
		return c.AuthorizedParty
	}
	if len(c.Audience) == 1 {
		return c.Audience[0]
	}

	return ""
}

// Validate implements the jwt.ClaimsValidator interface.
func (c AccessTokenClaims) Validate() error {
	if !c.TokenType.Is(TokenTypeAccessToken) {
		return errors.New("not an access token")
	}
	if len(c.Audience) != 1 {
		return errors.New("access token must have exactly one audience value")
	}

	return nil
}

// AuthorizedScopes returns a map with scope keys and true value of all scopes
// set in the accociated access token.
func (c AccessTokenClaims) AuthorizedScopes() map[string]bool {
	authorizedScopes := make(map[string]bool)
	for _, scope := range c.AuthorizedScopesList {
		authorizedScopes[scope] = true
	}

	return authorizedScopes
}

// RefreshTokenClaims define the claims used by refresh tokens.
type RefreshTokenClaims struct {
	jwt.RegisteredClaims

	TokenType TokenTypeValue `json:"lg.t"`

	ApprovedScopesList payload.ScopesValue `json:"scp"`

	ApprovedClaimsRequest *payload.ClaimsRequest `json:"lg.acr,omitempty"`
	Ref                   string                 `json:"lg.r"`

	// Resource is the resource indicator the access tokens of this grant are
	// bound to, empty when they are bound to the client. The audience of a
	// refresh token is always the client, see
	// https://www.rfc-editor.org/rfc/rfc8707.html.
	Resource string `json:"lg.res,omitempty"`

	IdentityClaims   jwt.MapClaims `json:"lg.i"`
	IdentityProvider string        `json:"lg.p,omitempty"`

	// Session of the underlaying identity backend, if it provided one. This is
	// the same value as found in access tokens and is nil for backends without
	// sessions.
	*oidc.SessionClaims
}

// Validate implements the jwt.ClaimsValidator interface.
func (c RefreshTokenClaims) Validate() error {
	if !c.TokenType.Is(TokenTypeRefreshToken) {
		return errors.New("not a refresh token")
	}

	if len(c.Audience) != 1 {
		return errors.New("refresh token must have exactly one audience value")
	}

	return nil
}

// NumericIDClaims define the claims used with the konnect/id scope.
type NumericIDClaims struct {
	// NOTE(longsleep): Always keep these claims compatible with the GitLab API
	// https://docs.gitlab.com/ce/api/users.html#for-user.
	NumericID         int64  `json:"id,omitempty"`
	NumericIDUsername string `json:"username,omitempty"`
}

// Validate implements the jwt.ClaimsValidator interface.
func (c NumericIDClaims) Validate() error {
	if c.NumericIDUsername == "" {
		return errors.New("username claim not valid")
	}
	return nil
}

// UniqueUserIDClaims define the claims used with the konnect/uuid scope.
type UniqueUserIDClaims struct {
	UniqueUserID string `json:"lg.uuid,omitempty"`
}

// Validate implements the jwt.ClaimsValidator interface.
func (c UniqueUserIDClaims) Validate() error {
	if c.UniqueUserID == "" {
		return errors.New("lg.uuid claim not valid")
	}
	return nil
}
