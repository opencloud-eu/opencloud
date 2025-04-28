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

package managers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/libregraph/oidc-go"
	"github.com/longsleep/rndm"

	konnect "github.com/libregraph/lico"
	"github.com/libregraph/lico/identity"

	"github.com/libregraph/lico/oidc/payload"
)

const dummyIdentityManagerName = "dummy"

// DummyIdentityManager implements an identity manager which always grants
// access to a fixed user id.
type DummyIdentityManager struct {
	sub string

	scopesSupported []string
}

// NewDummyIdentityManager creates a new DummyIdentityManager from the
// provided parameters.
func NewDummyIdentityManager(c *identity.Config, sub string) *DummyIdentityManager {
	im := &DummyIdentityManager{
		sub: sub,

		scopesSupported: setupSupportedScopes([]string{
			oidc.ScopeProfile,
			oidc.ScopeEmail,
		}, nil, c.ScopesSupported),
	}

	return im
}

type dummyUser struct {
	raw string
}

func (u *dummyUser) Raw() string {
	return u.raw
}

func (u *dummyUser) Subject() string {
	sub, _ := getPublicSubject([]byte(u.raw), []byte(dummyIdentityManagerName))
	return sub
}

func (u *dummyUser) Email() string {
	return fmt.Sprintf("%s@%s.local", u.raw, u.raw)
}

func (u *dummyUser) EmailVerified() bool {
	return false
}

func (u *dummyUser) Name() string {
	return fmt.Sprintf("Foo %s", strings.Title(u.raw))
}

func (u *dummyUser) Claims() jwt.MapClaims {
	claims := make(jwt.MapClaims)
	claims[konnect.IdentifiedUserIDClaim] = u.raw

	return claims
}

// Authenticate implements the identity.Manager interface.
func (im *DummyIdentityManager) Authenticate(ctx context.Context, rw http.ResponseWriter, req *http.Request, ar *payload.AuthenticationRequest, next identity.Manager) (identity.AuthRecord, error) {
	user := &dummyUser{im.sub}

	// Check request.
	err := ar.Verify(user.Subject())
	if err != nil {
		return nil, err
	}

	auth := identity.NewAuthRecord(im, user.Subject(), nil, nil, nil)
	auth.SetUser(user)

	return auth, nil
}

// Authorize implements the identity.Manager interface.
func (im *DummyIdentityManager) Authorize(ctx context.Context, rw http.ResponseWriter, req *http.Request, ar *payload.AuthenticationRequest, auth identity.AuthRecord) (identity.AuthRecord, error) {
	promptConsent := false
	var approvedScopes map[string]bool

	// Check prompt value.
	switch {
	case ar.Prompts[oidc.PromptConsent] == true:
		promptConsent = true
	default:
		// Let all other prompt values pass.
	}

	// TODO(longsleep): Move the code below to general function.
	// TODO(longsleep): Validate scopes and force prompt.
	approvedScopes = ar.Scopes

	// Offline access validation.
	// http://openid.net/specs/openid-connect-core-1_0.html#OfflineAccess
	if ok, _ := ar.Scopes[oidc.ScopeOfflineAccess]; ok {
		if !promptConsent {
			// Ensure that the prompt parameter contains consent unless
			// other conditions for processing the request permitting offline
			// access to the requested resources are in place; unless one or
			// both of these conditions are fulfilled, then it MUST ignore the
			// offline_access request,
			delete(ar.Scopes, oidc.ScopeOfflineAccess)
		}
	}

	if promptConsent {
		if ar.Prompts[oidc.PromptNone] == true {
			return auth, ar.NewError(oidc.ErrorCodeOIDCInteractionRequired, "consent required")
		}

		// TODO(longsleep): Implement consent page.
		return auth, ar.NewError(oidc.ErrorCodeOIDCInteractionRequired, "consent required, but page not implemented")
	}

	auth.AuthorizeScopes(approvedScopes)
	auth.AuthorizeClaims(ar.Claims)
	return auth, nil
}

// EndSession implements the identity.Manager interface.
func (im *DummyIdentityManager) EndSession(ctx context.Context, rw http.ResponseWriter, req *http.Request, esr *payload.EndSessionRequest) error {
	user := &dummyUser{im.sub}

	err := esr.Verify(user.Subject())
	if err != nil {
		return err
	}

	return nil
}

// ApproveScopes implements the Backend interface.
func (im *DummyIdentityManager) ApproveScopes(ctx context.Context, sub string, audience string, approvedScopes map[string]bool) (string, error) {
	ref := rndm.GenerateRandomString(32)

	// TODO(longsleep): Store generated ref with provided data.
	return ref, nil
}

// ApprovedScopes implements the Backend interface.
func (im *DummyIdentityManager) ApprovedScopes(ctx context.Context, sub string, audience string, ref string) (map[string]bool, error) {
	if ref == "" {
		return nil, fmt.Errorf("SimplePasswdBackend: invalid ref")
	}

	return nil, nil
}

// Fetch implements the identity.Manager interface.
func (im *DummyIdentityManager) Fetch(ctx context.Context, userID string, sessionRef *string, scopes map[string]bool, requestedClaimsMaps []*payload.ClaimsRequestMap, requestedScopes map[string]bool) (identity.AuthRecord, bool, error) {
	if userID != im.sub {
		return nil, false, fmt.Errorf("DummyIdentityManager: no user")
	}

	user := &dummyUser{im.sub}

	authorizedScopes, _ := identity.AuthorizeScopes(im, user, scopes)
	claims := identity.GetUserClaimsForScopes(user, authorizedScopes, requestedClaimsMaps)

	return identity.NewAuthRecord(im, user.Subject(), authorizedScopes, nil, claims), true, nil
}

// Name implements the identity.Manager interface.
func (im *DummyIdentityManager) Name() string {
	return dummyIdentityManagerName
}

// ScopesSupported implements the identity.Manager interface.
func (im *DummyIdentityManager) ScopesSupported(scopes map[string]bool) []string {
	return im.scopesSupported
}

// ClaimsSupported implements the identity.Manager interface.
func (im *DummyIdentityManager) ClaimsSupported(claims []string) []string {
	return []string{
		oidc.NameClaim,
		oidc.EmailClaim,
		oidc.EmailVerifiedClaim,
	}
}

// AddRoutes implements the identity.Manager interface.
func (im *DummyIdentityManager) AddRoutes(ctx context.Context, router *mux.Router) {
}

// OnSetLogon implements the identity.Manager interface.
func (im *DummyIdentityManager) OnSetLogon(func(ctx context.Context, rw http.ResponseWriter, user identity.User) error) error {
	return nil
}

// OnUnsetLogon implements the identity.Manager interface.
func (im *DummyIdentityManager) OnUnsetLogon(func(ctx context.Context, rw http.ResponseWriter) error) error {
	return nil
}
