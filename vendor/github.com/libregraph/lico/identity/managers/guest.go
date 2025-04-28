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

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/libregraph/oidc-go"
	"github.com/longsleep/rndm"
	"github.com/sirupsen/logrus"

	konnect "github.com/libregraph/lico"
	"github.com/libregraph/lico/identity"
	"github.com/libregraph/lico/identity/clients"
	"github.com/libregraph/lico/managers"
	"github.com/libregraph/lico/oidc/payload"
	"github.com/libregraph/lico/utils"
)

const guestIdentitityManagerName = "guest"

// GuestIdentityManager implements an identity manager for guest users.
type GuestIdentityManager struct {
	scopesSupported []string
	claimsSupported []string

	logger  logrus.FieldLogger
	clients *clients.Registry

	onSetLogonCallbacks   []func(ctx context.Context, rw http.ResponseWriter, user identity.User) error
	onUnsetLogonCallbacks []func(ctx context.Context, rw http.ResponseWriter) error
}

// NewGuestIdentityManager creates a new GuestIdentityManager from the
// provided parameters.
func NewGuestIdentityManager(c *identity.Config) *GuestIdentityManager {
	im := &GuestIdentityManager{
		scopesSupported: setupSupportedScopes([]string{}, []string{
			konnect.ScopeNumericID,
			oidc.ScopeProfile,
			oidc.ScopeEmail,
		}, c.ScopesSupported),
		claimsSupported: []string{
			oidc.NameClaim,
			oidc.FamilyNameClaim,
			oidc.GivenNameClaim,
			oidc.EmailClaim,
			oidc.EmailVerifiedClaim,
		},

		logger: c.Logger,

		onSetLogonCallbacks:   make([]func(ctx context.Context, rw http.ResponseWriter, user identity.User) error, 0),
		onUnsetLogonCallbacks: make([]func(ctx context.Context, rw http.ResponseWriter) error, 0),
	}

	return im
}

type guestUser struct {
	raw           string
	email         string
	emailVerified bool
	name          string
	familyName    string
	givenName     string
}

func newGuestUserFromClaims(claims jwt.MapClaims) *guestUser {
	isGuestClaim, ok := claims[konnect.IdentifiedUserIsGuest]
	if !ok {
		return nil
	}
	isGuest, _ := isGuestClaim.(bool)
	if !isGuest {
		return nil
	}

	idClaim, ok := claims[konnect.IdentifiedUserIDClaim]
	if !ok {
		return nil
	}

	dataClaim, ok := claims[konnect.IdentifiedData]
	if !ok {
		return nil
	}

	user := &guestUser{
		raw: idClaim.(string),
	}
	data, _ := dataClaim.(map[string]interface{})
	for name, value := range data {
		switch name {
		case "e":
			user.email, _ = value.(string)
		case "ev":
			if v, _ := value.(int); v == 1 {
				user.emailVerified = true
			}
		case "n":
			user.name, _ = value.(string)
		case "nf":
			user.familyName, _ = value.(string)
		case "ng":
			user.givenName, _ = value.(string)
		}
	}

	return user
}

type minimalGuestUserData struct {
	E  string `json:"e,omitempty"`
	EV int    `json:"ev,omitempty"`
	N  string `json:"n,omitempty"`
	NF string `json:"nf,omitempty"`
	NG string `json:"ng,omitempty"`
}

func (u *guestUser) Raw() string {
	return u.raw
}

func (u *guestUser) Subject() string {
	sub, _ := getPublicSubject([]byte(u.raw), []byte(guestIdentitityManagerName))
	return sub
}

func (u *guestUser) Email() string {
	return u.email
}

func (u *guestUser) EmailVerified() bool {
	return u.emailVerified
}

func (u *guestUser) Name() string {
	return u.name
}

func (u *guestUser) FamilyName() string {
	return u.familyName
}

func (u *guestUser) GivenName() string {
	return u.givenName
}

func (u *guestUser) Claims() jwt.MapClaims {
	claims := make(jwt.MapClaims)
	claims[konnect.IdentifiedUserIDClaim] = u.raw
	claims[konnect.IdentifiedUserIsGuest] = true

	m := &minimalGuestUserData{
		E:  u.email,
		N:  u.name,
		NF: u.familyName,
		NG: u.givenName,
	}
	if u.emailVerified {
		m.EV = 1
	}
	claims[konnect.IdentifiedData] = m

	return claims
}

// RegisterManagers registers the provided managers,
func (im *GuestIdentityManager) RegisterManagers(mgrs *managers.Managers) error {
	im.clients = mgrs.Must("clients").(*clients.Registry)

	return nil
}

// Authenticate implements the identity.Manager interface.
func (im *GuestIdentityManager) Authenticate(ctx context.Context, rw http.ResponseWriter, req *http.Request, ar *payload.AuthenticationRequest, next identity.Manager) (identity.AuthRecord, error) {
	// Check if required scopes are there.
	if !ar.Scopes[konnect.ScopeGuestOK] {
		return nil, ar.NewError(oidc.ErrorCodeOIDCLoginRequired, "GuestIdentityManager: required scope missing")
	}

	// Authenticate with signed client request object, so that must be there.
	if ar.Request == nil {
		return nil, ar.NewError(oidc.ErrorCodeOIDCInvalidRequestObject, "GuestIdentityManager: no request object")
	}

	// Further checks of signed claims.
	roc, ok := ar.Request.Claims.(*payload.RequestObjectClaims)
	if !ok {
		return nil, ar.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, "GuestIdentityManager: invalid claims request")
	}

	// NOTE(longsleep): Require claims in request object to ensure that the
	// claims requested come from there.
	if roc.Claims == nil || ar.Claims == nil {
		return nil, ar.NewError(oidc.ErrorCodeOAuth2InvalidRequest, "GuestIdentityManager: missing claims request")
	}
	// NOTE(longsleep): Guest mode requires ID token claims request with the
	// guest claim set to an expected value.
	if ar.Claims.IDToken == nil {
		return nil, ar.NewError(oidc.ErrorCodeOAuth2InvalidRequest, "GuestIdentityManager: missing claims request for id_token")
	}
	guest, ok := ar.Claims.IDToken.GetStringValue("guest")
	if !ok {
		return nil, ar.NewError(oidc.ErrorCodeOAuth2InvalidRequest, "GuestIdentityManager: missing claim guest in id_token claims request")
	}

	// Ensure that request object claim is signed.
	if ar.Request.Method == jwt.SigningMethodNone {
		return nil, ar.NewBadRequest(oidc.ErrorCodeOIDCInvalidRequestObject, "GuestIdentityManager: request object must be signed")
	}

	if guest == "" {
		return nil, ar.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, "GuestIdentityManager: invalid claim guest in id_token claims request")
	}

	// Additional email and profile claim values will be taken over into the
	// guest user data.
	email, _ := ar.Claims.IDToken.GetStringValue(oidc.EmailClaim)
	var emailVerified bool
	if emailVerifiedRaw, ok := ar.Claims.IDToken.Get(oidc.EmailVerifiedClaim); ok {
		emailVerified, _ = emailVerifiedRaw.Value.(bool)
	}

	name, _ := ar.Claims.IDToken.GetStringValue(oidc.NameClaim)
	familyName, _ := ar.Claims.IDToken.GetStringValue(oidc.FamilyNameClaim)
	givenName, _ := ar.Claims.IDToken.GetStringValue(oidc.GivenNameClaim)

	// Make new user with the provided signed information.
	sub := guest
	user := &guestUser{
		raw:           sub,
		email:         email,
		emailVerified: emailVerified,
		name:          name,
		familyName:    familyName,
		givenName:     givenName,
	}

	// TODO(longsleep): Add additional claims to user from the claims request
	// after filtering.

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
func (im *GuestIdentityManager) Authorize(ctx context.Context, rw http.ResponseWriter, req *http.Request, ar *payload.AuthenticationRequest, auth identity.AuthRecord) (identity.AuthRecord, error) {
	promptConsent := false
	var approvedScopes map[string]bool

	// Check prompt value.
	switch {
	case ar.Prompts[oidc.PromptConsent] == true:
		promptConsent = true
	default:
		// Let all other prompt values pass.
	}

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

	// Authenticate with signed client request object, so that must be there.
	if ar.Request == nil {
		return nil, ar.NewError(oidc.ErrorCodeOIDCInvalidRequestObject, "GuestIdentityManager: authorize without request object")
	}

	// Further checks of signed claims.
	roc, ok := ar.Request.Claims.(*payload.RequestObjectClaims)
	if !ok {
		return nil, ar.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, "GuestIdentityManager: authorize with invalid claims request")
	}

	securedDetails := roc.Secure()
	if securedDetails == nil {
		return nil, ar.NewBadRequest(oidc.ErrorCodeOIDCInvalidRequestObject, "GuestIdentityManager: authorize without secure client")
	}

	// TODO(longsleep): Validate scopes and force prompt.

	if promptConsent {
		if ar.Prompts[oidc.PromptNone] == true {
			return auth, ar.NewError(oidc.ErrorCodeOIDCInteractionRequired, "consent required")
		}

		// TODO(longsleep): Implement consent page.
		return auth, ar.NewError(oidc.ErrorCodeOIDCInteractionRequired, "consent required, but not supported for guests")
	}

	origin := ""
	if false {
		// TODO(longsleep): find a condition when this can be enabled.
		origin = utils.OriginFromRequestHeaders(req.Header)
	}

	clientDetails, err := im.clients.Lookup(req.Context(), ar.ClientID, "", ar.RedirectURI, origin, true)
	if err != nil {
		return nil, ar.NewError(oidc.ErrorCodeOAuth2AccessDenied, err.Error())
	}

	if clientDetails.ID != securedDetails.ID {
		return nil, ar.NewError(oidc.ErrorCodeOAuth2AccessDenied, "client mismatch")
	}

	// If not trusted we need to check request scopes.
	if clientDetails.Trusted && securedDetails.TrustedScopes == nil {
		// NOTE(longsleep):  Guest scope validation takes all client provided
		// scopes when the trusted client configuration has no trusted scopes
		// configured. This can be used for fine grained access control using
		// the trusted client configuration.
		approvedScopes = ar.Scopes
	} else {
		supportedScopes := make(map[string]bool)
		for _, scope := range im.ScopesSupported(nil) {
			supportedScopes[scope] = true
		}

		// Auto approve all supported scopes.
		approvedScopes = make(map[string]bool)
		for scope := range ar.Scopes {
			if _, ok := supportedScopes[scope]; ok {
				approvedScopes[scope] = true
			}
		}
		// Approve all additional scopes which are allowed by the trusted
		// client.
		for _, scope := range securedDetails.TrustedScopes {
			if _, ok := ar.Scopes[scope]; ok {
				approvedScopes[scope] = true
			}
		}
		// Always approve openid scope.
		if _, ok := ar.Scopes[oidc.ScopeOpenID]; ok {
			approvedScopes[oidc.ScopeOpenID] = true
		}

		// Ensure that guest scope was approved.
		if ok, _ := approvedScopes[konnect.ScopeGuestOK]; !ok {
			return nil, ar.NewBadRequest(oidc.ErrorCodeOAuth2InvalidRequest, "GuestIdentityManager: client does not authorize "+konnect.ScopeGuestOK+" scope")
		}
	}

	auth.AuthorizeScopes(approvedScopes)
	auth.AuthorizeClaims(ar.Claims)
	return auth, nil
}

// EndSession implements the identity.Manager interface.
func (im *GuestIdentityManager) EndSession(ctx context.Context, rw http.ResponseWriter, req *http.Request, esr *payload.EndSessionRequest) error {
	// TODO(longsleep): Implement end session for guests.

	// Trigger callbacks.
	for _, f := range im.onUnsetLogonCallbacks {
		err := f(ctx, rw)
		if err != nil {
			return err
		}
	}

	return nil
}

// ApproveScopes implements the Backend interface.
func (im *GuestIdentityManager) ApproveScopes(ctx context.Context, sub string, audience string, approvedScopes map[string]bool) (string, error) {
	ref := rndm.GenerateRandomString(32)

	// TODO(longsleep): Store generated ref with provided data.
	return ref, nil
}

// ApprovedScopes implements the Backend interface.
func (im *GuestIdentityManager) ApprovedScopes(ctx context.Context, sub string, audience string, ref string) (map[string]bool, error) {
	if ref == "" {
		return nil, fmt.Errorf("GuestIdentityManager: invalid ref")
	}

	return nil, nil
}

// Fetch implements the identity.Manager interface.
func (im *GuestIdentityManager) Fetch(ctx context.Context, userID string, sessionRef *string, scopes map[string]bool, requestedClaimsMaps []*payload.ClaimsRequestMap, requestedScopes map[string]bool) (identity.AuthRecord, bool, error) {
	var user identity.PublicUser

	for {
		// First check if current context has auth.
		if auth, ok := identity.FromContext(ctx); ok {
			user = auth.User()
			break
		}
		// Second check if current context has claims with guest identity in it.
		if claims, ok := konnect.FromClaimsContext(ctx); ok {
			var identityClaims jwt.MapClaims
			var identityProvider string
			switch c := claims.(type) {
			case *konnect.AccessTokenClaims:
				identityClaims = c.IdentityClaims
				identityProvider = c.IdentityProvider
			case *konnect.RefreshTokenClaims:
				identityClaims = c.IdentityClaims
				identityProvider = c.IdentityProvider
			}
			if identityClaims != nil && identityProvider == im.Name() {
				user = newGuestUserFromClaims(identityClaims)
				break
			}
		}

		return nil, false, fmt.Errorf("GuestIdentityManager: no user in context")
	}

	if user.Raw() != userID {
		return nil, false, fmt.Errorf("GuestIdentityManager: wrong user")
	}

	authorizedScopes, _ := identity.AuthorizeScopes(im, user, scopes)
	claims := identity.GetUserClaimsForScopes(user, authorizedScopes, requestedClaimsMaps)

	auth := identity.NewAuthRecord(im, user.Subject(), authorizedScopes, nil, claims)
	auth.SetUser(user)

	return auth, true, nil
}

// Name implements the identity.Manager interface.
func (im *GuestIdentityManager) Name() string {
	return guestIdentitityManagerName
}

// ScopesSupported implements the identity.Manager interface.
func (im *GuestIdentityManager) ScopesSupported(scopes map[string]bool) []string {
	if scopes != nil {
		// NOTE(longsleep): Allow scopes as we get them, since we already validated
		// them in authorize.
		supported := make([]string, 0)
		for scope, ok := range scopes {
			if ok {
				supported = append(supported, scope)
			}
		}
		return supported
	}

	return im.scopesSupported
}

// ClaimsSupported implements the identity.Manager interface.
func (im *GuestIdentityManager) ClaimsSupported(claims []string) []string {
	return im.claimsSupported
}

// AddRoutes implements the identity.Manager interface.
func (im *GuestIdentityManager) AddRoutes(ctx context.Context, router *mux.Router) {
}

// OnSetLogon implements the identity.Manager interface.
func (im *GuestIdentityManager) OnSetLogon(cb func(ctx context.Context, rw http.ResponseWriter, user identity.User) error) error {
	im.onSetLogonCallbacks = append(im.onSetLogonCallbacks, cb)
	return nil
}

// OnUnsetLogon implements the identity.Manager interface.
func (im *GuestIdentityManager) OnUnsetLogon(cb func(ctx context.Context, rw http.ResponseWriter) error) error {
	im.onUnsetLogonCallbacks = append(im.onUnsetLogonCallbacks, cb)
	return nil
}
