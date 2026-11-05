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
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/libregraph/lico/oidc/payload"
)

// LookupContext carries additional context of the request which triggered a
// user lookup. Backends can use it to decide how to treat their own sessions.
// All fields are optional, a zero value means that the value is not known at
// the place of the lookup.
type LookupContext struct {
	// Audience is the client ID this lookup is made for.
	Audience string

	// NewGrant is true when the lookup belongs to an authorization which
	// creates a new grant, which means that a refresh token was requested.
	// Backends can use this to hand out a new session per grant.
	NewGrant bool

	// RefreshTokenID is the ID of the refresh token which is currently
	// exchanged, if any.
	RefreshTokenID string

	// NewAccessTokenID is the ID of the access token which is about to be
	// issued for this lookup, if any.
	NewAccessTokenID string

	// NewRefreshTokenID is the ID of the refresh token which is about to be
	// issued for this lookup. Empty when this lookup issues no refresh token,
	// so backends can tell a rotating grant from a one shot lookup.
	NewRefreshTokenID string
}

// Manager is a interface to define a identity manager.
type Manager interface {
	Authenticate(ctx context.Context, rw http.ResponseWriter, req *http.Request, ar *payload.AuthenticationRequest, next Manager) (AuthRecord, error)
	Authorize(ctx context.Context, rw http.ResponseWriter, req *http.Request, ar *payload.AuthenticationRequest, auth AuthRecord) (AuthRecord, error)
	EndSession(ctx context.Context, rw http.ResponseWriter, req *http.Request, esr *payload.EndSessionRequest) error

	ApproveScopes(ctx context.Context, sub string, audience string, approvedScopesList map[string]bool) (string, error)
	ApprovedScopes(ctx context.Context, sub string, audience string, ref string) (map[string]bool, error)

	Fetch(ctx context.Context, userID string, sessionRef *string, scopes map[string]bool, requestedClaimsMaps []*payload.ClaimsRequestMap, requestedScopes map[string]bool, lookup *LookupContext) (AuthRecord, bool, error)

	Name() string
	ScopesSupported(scopes map[string]bool) []string
	ClaimsSupported(claims []string) []string

	AddRoutes(ctx context.Context, router *mux.Router)

	OnSetLogon(func(ctx context.Context, rw http.ResponseWriter, user User) error) error
	OnUnsetLogon(func(ctx context.Context, rw http.ResponseWriter) error) error
}
