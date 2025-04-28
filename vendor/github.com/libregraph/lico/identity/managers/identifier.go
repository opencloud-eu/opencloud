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
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/libregraph/oidc-go"
	"github.com/longsleep/rndm"
	"github.com/sirupsen/logrus"

	"github.com/libregraph/lico/identifier"
	"github.com/libregraph/lico/identity"
	"github.com/libregraph/lico/identity/clients"
	"github.com/libregraph/lico/managers"
	konnectoidc "github.com/libregraph/lico/oidc"
	"github.com/libregraph/lico/oidc/payload"
	"github.com/libregraph/lico/utils"
)

// IdentifierIdentityManager implements an identity manager which relies on
// Konnect its identifier to provide identity.
type IdentifierIdentityManager struct {
	signInFormURI string
	signedOutURI  string

	scopesSupported []string
	claimsSupported []string

	identifier *identifier.Identifier
	clients    *clients.Registry
	logger     logrus.FieldLogger
}

type identifierUser struct {
	*identifier.IdentifiedUser
}

func (u *identifierUser) Raw() string {
	return u.IdentifiedUser.Subject()
}

func (u *identifierUser) Subject() string {
	sub, _ := getPublicSubject([]byte(u.Raw()), []byte(u.IdentifiedUser.BackendName()))
	return sub
}

func (u *identifierUser) Scopes() []string {
	return u.IdentifiedUser.Scopes()
}

func (u *identifierUser) RequiredScopes() map[string]bool {
	lockedScopes := u.IdentifiedUser.LockedScopes()
	if lockedScopes == nil {
		return nil
	}
	requiredScopes := make(map[string]bool)
	for _, scope := range lockedScopes {
		if strings.HasPrefix(scope, "!") {
			scope = strings.TrimLeft(scope, "!")
			requiredScopes[scope] = false
		} else {
			requiredScopes[scope] = true
		}
	}
	return requiredScopes
}

func asIdentifierUser(user *identifier.IdentifiedUser) *identifierUser {
	return &identifierUser{user}
}

// NewIdentifierIdentityManager creates a new IdentifierIdentityManager from the provided
// parameters.
func NewIdentifierIdentityManager(c *identity.Config, i *identifier.Identifier) *IdentifierIdentityManager {
	im := &IdentifierIdentityManager{
		signInFormURI: c.SignInFormURI.String(),
		signedOutURI:  c.SignedOutURI.String(),

		scopesSupported: setupSupportedScopes([]string{
			oidc.ScopeOfflineAccess,
		}, nil, c.ScopesSupported),
		claimsSupported: []string{
			oidc.NameClaim,
			oidc.FamilyNameClaim,
			oidc.GivenNameClaim,
			oidc.EmailClaim,
			oidc.EmailVerifiedClaim,
		},

		identifier: i,
		logger:     c.Logger,
	}

	return im
}

// RegisterManagers registers the provided managers,
func (im *IdentifierIdentityManager) RegisterManagers(mgrs *managers.Managers) error {
	im.clients = mgrs.Must("clients").(*clients.Registry)

	return im.identifier.RegisterManagers(mgrs)
}

// Authenticate implements the identity.Manager interface.
func (im *IdentifierIdentityManager) Authenticate(ctx context.Context, rw http.ResponseWriter, req *http.Request, ar *payload.AuthenticationRequest, next identity.Manager) (identity.AuthRecord, error) {
	var user *identifierUser
	var err error

	if authenticationErrorID := req.Form.Get("error"); authenticationErrorID != "" {
		// Incoming with error. Directly abort and return.
		return nil, ar.NewError(authenticationErrorID, req.Form.Get("error_description"))
	}

	u, _ := im.identifier.GetUserFromLogonCookie(ctx, req, ar.MaxAge, true)
	if u != nil {
		// TODO(longsleep): Add other user meta data.
		user = asIdentifierUser(u)
	} else {
		// Not signed in.
		if mode := req.Form.Get("identifier"); mode == identifier.MustBeSignedIn {
			// Identifier mode is set to must, this means that this flow must be authenticated here, and everything
			// else is an error. This is for example set, when coming back from an external authority.
			im.logger.WithField("mode", mode).Debugln("identifier mode is set, but not signed in")
		} else if next != nil {
			// Give next handler a chance if any.
			if auth, authErr := next.Authenticate(ctx, rw, req, ar, nil); authErr == nil {
				// Inner handler success.
				// TODO(longsleep): Add check and option to avoid that the inner
				// handler can ever return users which exist at the outer.
				return auth, authErr
			} else {
				switch authErr.(type) {
				case *payload.AuthenticationError:
					// ignore, breaks
				case *identity.LoginRequiredError:
					// ignore, breaks
				case *identity.IsHandledError:
					// breaks, breaks
				default:
					im.logger.WithFields(utils.ErrorAsFields(authErr)).Errorln("inner authorize request failed")
				}
			}
		}
		err = ar.NewError(oidc.ErrorCodeOIDCLoginRequired, "IdentifierIdentityManager: not signed in")
	}

	// Check prompt value.
	switch {
	case ar.Prompts[oidc.PromptNone] == true:
		if err != nil {
			// Never show sign-in, directly return error.
			return nil, err
		}
	case ar.Prompts[oidc.PromptLogin] == true:
		if err == nil {
			// Enforce to show sign-in, when signed in.
			err = ar.NewError(oidc.ErrorCodeOIDCLoginRequired, "IdentifierIdentityManager: prompt=login request")
		}
	case ar.Prompts[oidc.PromptSelectAccount] == true:
		if err == nil {
			// Enforce to show sign-in, when signed in.
			err = ar.NewError(oidc.ErrorCodeOIDCLoginRequired, "IdentifierIdentityManager: prompt=select_account request")
		}
	default:
		// Let all other prompt values pass.
	}

	var auth identity.AuthRecord

	// More checks.
	if err == nil {
		var sub string
		if user != nil {
			sub = user.Subject()
		}
		err = ar.Verify(sub)
		if err != nil {
			return nil, err
		}

		if user != nil {
			record := identifier.NewRecord(req, im.identifier.Config.Config)
			record.IdentifiedUser = user.IdentifiedUser
			ctx = identifier.NewRecordContext(ctx, record)

			// Inject required scopes into request.
			for scope, ok := range user.RequiredScopes() {
				ar.Scopes[scope] = ok
			}
			// Load user record from identitymanager, without any scopes or claims
			// to ensure that the user data is refreshed and that the user still
			// exists.
			var found bool
			auth, found, err = im.Fetch(ctx, user.Raw(), user.SessionRef(), nil, nil, ar.Scopes)
			if !found {
				err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2ServerError, "user not found")
			} else {
				// Update ar.Scopes with the ones gotten from backend.
				if bu, ok := auth.User().(*identifierUser); ok {
					scopes := bu.Scopes()
					if scopes != nil {
						expanded := make(map[string]bool)
						for _, scope := range scopes {
							if enabled, ok := ar.Scopes[scope]; ok && !enabled {
								// Skip already known but not enabled scopes.
								continue
							}
							expanded[scope] = true
						}
						ar.Scopes = expanded
					}
				}
			}
		}
	}

	if err != nil {
		if ar.Prompts[oidc.PromptNone] == true {
			// Never show sign-in, directly return error.
			return nil, err
		}

		// Build login URL.
		query, err := url.ParseQuery(req.URL.RawQuery)
		if err != nil {
			return nil, err
		}
		query.Set("flow", identifier.FlowOIDC)
		if ar.Claims != nil {
			// Add derived scope list from claims request.
			claimsScopes := ar.Claims.Scopes(ar.Scopes)
			if len(claimsScopes) > 0 {
				query.Set("claims_scope", strings.Join(claimsScopes, " "))
			}
		}
		u, _ := url.Parse(im.signInFormURI)
		u.RawQuery = query.Encode()
		utils.WriteRedirect(rw, http.StatusFound, u, nil, false)

		return nil, &identity.IsHandledError{}
	}

	if auth == nil {
		// In case no existing user was fetched and that was not an error, make
		// sure that we actually create a new auth record. This should not
		// happen and is kept here for potential backwards compatibility.
		auth = identity.NewAuthRecord(im, user.Subject(), nil, nil, nil)
		auth.SetUser(user)
	}

	if loggedOn, logonAt := u.LoggedOn(); loggedOn {
		auth.SetAuthTime(logonAt)
	}

	return auth, nil
}

// Authorize implements the identity.Manager interface.
func (im *IdentifierIdentityManager) Authorize(ctx context.Context, rw http.ResponseWriter, req *http.Request, ar *payload.AuthenticationRequest, auth identity.AuthRecord) (identity.AuthRecord, error) {
	promptConsent := false
	var approvedScopes map[string]bool

	// Check prompt value.
	switch {
	case ar.Prompts[oidc.PromptConsent] == true:
		promptConsent = true
	default:
		// Let all other prompt values pass.
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

	// If not trusted, always force consent.
	if clientDetails.Trusted {
		approvedScopes = ar.Scopes
	} else {
		promptConsent = true
	}

	// Check given consent.
	consent, err := im.identifier.GetConsentFromConsentCookie(req.Context(), rw, req, req.Form.Get("konnect"))
	if err != nil {
		return nil, err
	}
	if consent != nil {
		if !consent.Allow {
			return auth, ar.NewError(oidc.ErrorCodeOAuth2AccessDenied, "consent denied")
		}

		promptConsent = false
		filteredApprovedScopes, allApprovedScopes := consent.Scopes(ar.Scopes)

		// Filter claims request by approved scopes.
		if ar.Claims != nil {
			err = ar.Claims.ApplyScopes(allApprovedScopes)
			if err != nil {
				return nil, err
			}
		}

		approvedScopes = filteredApprovedScopes
	}

	if promptConsent {
		if ar.Prompts[oidc.PromptNone] == true {
			return auth, ar.NewError(oidc.ErrorCodeOIDCInteractionRequired, "consent required")
		}

		// Build consent URL.
		query, err := url.ParseQuery(req.URL.RawQuery)
		if err != nil {
			return nil, err
		}
		query.Set("flow", identifier.FlowConsent)
		if ar.Claims != nil {
			// Add derived scope list from claims request.
			claimsScopes := ar.Claims.Scopes(ar.Scopes)
			if len(claimsScopes) > 0 {
				query.Set("claims_scope", strings.Join(claimsScopes, " "))
			}
		}
		if ar.Scopes != nil {
			scopes := make([]string, 0)
			for scope, ok := range ar.Scopes {
				if ok {
					scopes = append(scopes, scope)
				}
				query.Set("scope", strings.Join(scopes, " "))
			}
		}
		u, _ := url.Parse(im.signInFormURI)
		u.RawQuery = query.Encode()
		utils.WriteRedirect(rw, http.StatusFound, u, nil, false)

		return nil, &identity.IsHandledError{}
	}

	// Offline access validation.
	// http://openid.net/specs/openid-connect-core-1_0.html#OfflineAccess
	if ok, _ := approvedScopes[oidc.ScopeOfflineAccess]; ok {
		var ignoreOfflineAccessErr error
		for {
			if ok, _ := ar.ResponseTypes[oidc.ResponseTypeCode]; !ok {
				// MUST ignore the offline_access request unless the Client is using
				// a response_type value that would result in an Authorization
				// Code being returned,
				ignoreOfflineAccessErr = fmt.Errorf("response_type=code required, %#v", ar.ResponseTypes)
				break
			}

			if clientDetails.Trusted {
				// Always allow offline access for trusted clients. This qualifies
				// for other conditions.
				break
			}

			if ok, _ := ar.Prompts[oidc.PromptConsent]; !ok && consent == nil {
				// Ensure that the prompt parameter contains consent unless
				// other conditions for processing the request permitting offline
				// access to the requested resources are in place; unless one or
				// both of these conditions are fulfilled, then it MUST ignore the
				// offline_access request,
				ignoreOfflineAccessErr = fmt.Errorf("prompt=consent required, %#v", ar.Prompts)
				break
			}

			break
		}

		if ignoreOfflineAccessErr != nil {
			delete(approvedScopes, oidc.ScopeOfflineAccess)
			im.logger.WithError(ignoreOfflineAccessErr).Debugln("removed offline_access scope")
		}
	}

	auth.AuthorizeScopes(approvedScopes)
	auth.AuthorizeClaims(ar.Claims)
	return auth, nil
}

// EndSession implements the identity.Manager interface.
func (im *IdentifierIdentityManager) EndSession(ctx context.Context, rw http.ResponseWriter, req *http.Request, esr *payload.EndSessionRequest) error {
	var err error
	var esrClaims *konnectoidc.IDTokenClaims
	var clientDetails *clients.Details

	origin := utils.OriginFromRequestHeaders(req.Header)

	clientID := ""
	if esr.IDTokenHint != nil {
		// Extended request, verify IDTokenHint and its claims if available.
		esrClaims = esr.IDTokenHint.Claims.(*konnectoidc.IDTokenClaims)
		if len(esrClaims.Audience) == 1 {
			clientID = esrClaims.Audience[0]
		}
		clientDetails, err = im.clients.Lookup(ctx, clientID, "", esr.PostLogoutRedirectURI, origin, true)
		if err != nil {
			// This error is not fatal since according to
			// the spec in https://openid.net/specs/openid-connect-session-1_0.html#RPLogout the
			// id_token_hint is not enforced to match the audience. Instead of fail
			// we treat it as untrusted client.
			im.logger.WithError(err).Debugln("IdentifierIdentityManager: id_token_hint does not match request")
			esrClaims = nil
			clientDetails = nil
		}
	}

	var user *identifierUser
	u, _ := im.identifier.GetUserFromLogonCookie(ctx, req, 0, false)
	if u != nil {
		user = asIdentifierUser(u)
		// More checks.
		if clientDetails != nil && user != nil {
			sub := user.Subject()
			err = esr.Verify(sub)
			if err != nil {
				return err
			}
		}
		if clientDetails != nil && clientDetails.Trusted {
			// Directly end identifier session when a trusted client requests
			// and honor redirect wish if any.
			var uri *url.URL
			uri, err = im.identifier.EndSession(ctx, u, rw, esr.PostLogoutRedirectURI, esr.State)
			if err != nil {
				// Do nothing if err.
				im.logger.WithError(err).Errorln("IdentifierIdentityManager: failed to end session")
				return err
			}
			if uri != nil {
				// Redirect to uri if end session returned any.
				return identity.NewRedirectError("", uri)
			}
		}
	} else {
		// Ignore when not signed in, for end session.
	}

	if clientDetails == nil || !clientDetails.Trusted || esr.PostLogoutRedirectURI == nil || esr.PostLogoutRedirectURI.String() == "" {
		// Handle directly by redirecting to our logout confirm url for untrusted
		// clients or when no URL was set.
		u, _ := url.Parse(im.signedOutURI)
		query := &url.Values{}

		if clientDetails != nil {
			query.Add("flow", identifier.FlowOIDC)
		}
		if esrClaims != nil {
			query.Add("client_id", clientID)
		}

		u.RawQuery = query.Encode()
		return identity.NewRedirectError(oidc.ErrorCodeOIDCInteractionRequired, u)
	}

	return nil
}

// ApproveScopes implements the Backend interface.
func (im *IdentifierIdentityManager) ApproveScopes(ctx context.Context, sub string, audience string, approvedScopes map[string]bool) (string, error) {
	ref := rndm.GenerateRandomString(32)

	// TODO(longsleep): Store generated ref with provided data.
	return ref, nil
}

// ApprovedScopes implements the Backend interface.
func (im *IdentifierIdentityManager) ApprovedScopes(ctx context.Context, sub string, audience string, ref string) (map[string]bool, error) {
	if ref == "" {
		return nil, fmt.Errorf("IdentifierIdentityManager: invalid ref")
	}

	return nil, nil
}

// Fetch implements the identity.Manager interface.
func (im *IdentifierIdentityManager) Fetch(ctx context.Context, userID string, sessionRef *string, scopes map[string]bool, requestedClaimsMaps []*payload.ClaimsRequestMap, requestedScopes map[string]bool) (identity.AuthRecord, bool, error) {
	u, err := im.identifier.GetUserFromID(ctx, userID, sessionRef, requestedScopes)
	if err != nil {
		im.logger.WithError(err).Errorln("IdentifierIdentityManager: fetch failed to get user from userID")
		return nil, false, fmt.Errorf("IdentifierIdentityManager: identifier error")
	}

	if u == nil {
		return nil, false, fmt.Errorf("IdentifierIdentityManager: no user")
	}

	user := asIdentifierUser(u)
	authorizedScopes, _ := identity.AuthorizeScopes(im, user, scopes)
	claims := identity.GetUserClaimsForScopes(user, authorizedScopes, requestedClaimsMaps)

	auth := identity.NewAuthRecord(im, user.Subject(), authorizedScopes, nil, claims)
	auth.SetUser(user)

	return auth, true, nil
}

// Name implements the identity.Manager interface.
func (im *IdentifierIdentityManager) Name() string {
	return im.identifier.Name()
}

// ScopesSupported implements the identity.Manager interface.
func (im *IdentifierIdentityManager) ScopesSupported(scopes map[string]bool) []string {
	scopesSupported := make([]string, len(im.scopesSupported))
	copy(scopesSupported, im.scopesSupported)

	for _, scope := range im.identifier.ScopesSupported() {
		scopesSupported = append(scopesSupported, scope)
	}

	return scopesSupported
}

// ClaimsSupported implements the identity.Manager interface.
func (im *IdentifierIdentityManager) ClaimsSupported(claims []string) []string {
	return im.claimsSupported
}

// AddRoutes implements the identity.Manager interface.
func (im *IdentifierIdentityManager) AddRoutes(ctx context.Context, router *mux.Router) {
	im.identifier.AddRoutes(ctx, router)
}

// OnSetLogon implements the identity.Manager interface.
func (im *IdentifierIdentityManager) OnSetLogon(cb func(ctx context.Context, rw http.ResponseWriter, user identity.User) error) error {
	return im.identifier.OnSetLogon(cb)
}

// OnUnsetLogon implements the identity.Manager interface.
func (im *IdentifierIdentityManager) OnUnsetLogon(cb func(ctx context.Context, rw http.ResponseWriter) error) error {
	return im.identifier.OnUnsetLogon(cb)
}
