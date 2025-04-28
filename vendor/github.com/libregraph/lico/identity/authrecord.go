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

package identity

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/libregraph/lico/oidc/payload"
)

type authRecord struct {
	manager Manager

	sub              string
	authorizedScopes map[string]bool
	authorizedClaims *payload.ClaimsRequest
	claimsByScope    map[string]jwt.Claims

	user     PublicUser
	authTime time.Time
}

// NewAuthRecord returns a implementation of identity.AuthRecord holding
// the provided data in memory.
func NewAuthRecord(manager Manager, sub string, authorizedScopes map[string]bool, authorizedClaims *payload.ClaimsRequest, claimsByScope map[string]jwt.Claims) AuthRecord {
	if authorizedScopes == nil {
		authorizedScopes = make(map[string]bool)
	}

	return &authRecord{
		manager: manager,

		sub:              sub,
		authorizedScopes: authorizedScopes,
		authorizedClaims: authorizedClaims,
		claimsByScope:    claimsByScope,
	}
}

// Manager implements the identity.AuthRecord interface, returning the
// accociated identities manager.
func (r *authRecord) Manager() Manager {
	return r.manager
}

// Subject implements the identity.AuthRecord  interface.
func (r *authRecord) Subject() string {
	return r.sub
}

// AuthorizedScopes implements the identity.AuthRecord  interface.
func (r *authRecord) AuthorizedScopes() map[string]bool {
	return r.authorizedScopes
}

// AuthorizeScopes implements the identity.AuthRecord  interface.
func (r *authRecord) AuthorizeScopes(scopes map[string]bool) {
	authorizedScopes, unauthorizedScopes := AuthorizeScopes(r.manager, r.User(), scopes)

	for scope, grant := range authorizedScopes {
		if grant {
			r.authorizedScopes[scope] = grant
		} else {
			delete(r.authorizedScopes, scope)
		}
	}
	for scope := range unauthorizedScopes {
		delete(r.authorizedScopes, scope)
	}
}

// AuthorizedClaims implements the identity.AuthRecord interface.
func (r *authRecord) AuthorizedClaims() *payload.ClaimsRequest {
	return r.authorizedClaims
}

// AuthorizeClaims implements the identity.AuthRecord interface.
func (r *authRecord) AuthorizeClaims(claims *payload.ClaimsRequest) {
	r.authorizedClaims = claims
}

// Claims implements the identity.AuthRecord  interface.
func (r *authRecord) Claims(scopes ...string) []jwt.Claims {
	result := make([]jwt.Claims, len(scopes))
	for idx, scope := range scopes {
		if claimsForScope, ok := r.claimsByScope[scope]; ok {
			result[idx] = claimsForScope
		}
	}

	return result
}

// User implements the identity.AuthRecord interface.
func (r *authRecord) User() PublicUser {
	return r.user
}

// SetUser implements the identity.AuthRecord interface.
func (r *authRecord) SetUser(u PublicUser) {
	r.user = u
}

// LoggedOn implements the identity.AuthRecord interface
func (r *authRecord) LoggedOn() (bool, time.Time) {
	return !r.authTime.IsZero(), r.authTime
}

// SetAuthTime implements the identity.AuthRecord interface.
func (r *authRecord) SetAuthTime(authTime time.Time) {
	r.authTime = authTime
}
