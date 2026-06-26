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

package identifier

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/gorilla/mux"
	"github.com/longsleep/rndm"
	"github.com/sirupsen/logrus"

	"github.com/libregraph/oidc-go"

	konnect "github.com/libregraph/lico"
	"github.com/libregraph/lico/identifier/backends"
	"github.com/libregraph/lico/identifier/meta"
	"github.com/libregraph/lico/identifier/meta/scopes"
	"github.com/libregraph/lico/identity"
	"github.com/libregraph/lico/identity/authorities"
	"github.com/libregraph/lico/identity/clients"
	"github.com/libregraph/lico/managers"
	"github.com/libregraph/lico/utils"
)

// audienceMarker defines the value which gets included in logon cookies. Valid
// logon cookies must have the first value of this list in their audience claim.
// Increment this value whenever logon cookie claims and format changes in
// non-backwards compatible ways. User will have to sign in again to get a new
// cookie.
var audienceMarker = jwt.Audience([]string{"2019012201"})

// Identifier defines a identification login area with its endpoints using
// a Kopano Core server as backend logon provider.
type Identifier struct {
	Config *Config

	baseURI         *url.URL
	pathPrefix      string
	staticFolder    string
	scopesConf      string
	webappIndexHTML []byte

	logonCookieName     string
	logonCookieSameSite http.SameSite

	consentCookieSameSite http.SameSite

	stateCookieSameSite http.SameSite

	authorizationEndpointURI *url.URL
	signedOutEndpointURI     *url.URL
	oauth2CbEndpointURI      *url.URL

	encrypter   jose.Encrypter
	recipient   *jose.Recipient
	backend     backends.Backend
	clients     *clients.Registry
	authorities *authorities.Registry

	meta *meta.Meta

	defaultBannerLogo       *string
	defaultBannerLogoHeight *string

	onSetLogonCallbacks   []func(ctx context.Context, rw http.ResponseWriter, user identity.User) error
	onUnsetLogonCallbacks []func(ctx context.Context, rw http.ResponseWriter) error

	logger logrus.FieldLogger

	router *mux.Router
}

// NewIdentifier returns a new Identifier.
func NewIdentifier(c *Config) (*Identifier, error) {
	staticFolder := c.StaticFolder
	var webappIndexHTML = make([]byte, 0)

	if !c.WebAppDisabled {
		fn := staticFolder + "/index.html"
		if _, statErr := os.Stat(fn); os.IsNotExist(statErr) {
			return nil, fmt.Errorf("identifier client index.html not found: %w", statErr)
		}
		readData, readErr := ioutil.ReadFile(fn)
		if readErr != nil {
			return nil, fmt.Errorf("identifier failed to read client index.html: %w", readErr)
		}
		webappIndexHTML = bytes.Replace(readData, []byte("__PATH_PREFIX__"), []byte(c.PathPrefix), 1)
	}

	oauth2CbEndpointURI, _ := url.Parse(c.BaseURI.String())
	oauth2CbEndpointURI.Path = c.PathPrefix + "/identifier/oauth2/cb"

	i := &Identifier{
		Config: c,

		baseURI:         c.BaseURI,
		pathPrefix:      c.PathPrefix,
		staticFolder:    staticFolder,
		scopesConf:      c.ScopesConf,
		webappIndexHTML: webappIndexHTML,

		logonCookieName:     c.LogonCookieName,
		logonCookieSameSite: c.LogonCookieSameSite,

		consentCookieSameSite: c.ConsentCookieSameSite,

		stateCookieSameSite: c.StateCookieSameSite,

		authorizationEndpointURI: c.AuthorizationEndpointURI,
		signedOutEndpointURI:     c.SignedOutEndpointURI,
		oauth2CbEndpointURI:      oauth2CbEndpointURI,

		backend: c.Backend,

		onSetLogonCallbacks:   make([]func(ctx context.Context, rw http.ResponseWriter, user identity.User) error, 0),
		onUnsetLogonCallbacks: make([]func(ctx context.Context, rw http.ResponseWriter) error, 0),

		logger: c.Config.Logger,
	}

	var err error
	i.meta = &meta.Meta{}
	i.meta.Scopes, err = scopes.NewScopesFromFile(i.scopesConf, i.logger)
	if err != nil {
		return nil, err
	}

	if c.DefaultBannerLogo != nil {
		defaultBannerLogo, err := encodeImageAsDataURL(c.DefaultBannerLogo)
		if err != nil {
			return nil, fmt.Errorf("failed to encode default banner logo: %w", err)
		}
		i.defaultBannerLogo = &defaultBannerLogo
	}
	i.defaultBannerLogoHeight = c.DefaultBannerLogoHeight

	i.meta.Scopes.Extend(c.Backend.ScopesMeta())

	return i, nil
}

// RegisterManagers registers the provided managers,
func (i *Identifier) RegisterManagers(mgrs *managers.Managers) error {
	i.clients = mgrs.Must("clients").(*clients.Registry)
	i.authorities = mgrs.Must("authorities").(*authorities.Registry)

	if service, ok := i.backend.(managers.ServiceUsesManagers); ok {
		err := service.RegisterManagers(mgrs)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddRoutes adds the endpoint routes of the accociated Identifier to the
// provided router with the provided context.
func (i *Identifier) AddRoutes(ctx context.Context, router *mux.Router) {
	r := router.PathPrefix(i.pathPrefix).Subrouter()

	r.PathPrefix("/static/").Handler(i.staticHandler(http.StripPrefix(i.pathPrefix, http.FileServer(http.Dir(i.staticFolder))), true))
	r.Handle("/service-worker.js", i.staticHandler(http.StripPrefix(i.pathPrefix, http.FileServer(http.Dir(i.staticFolder))), false))
	r.Handle("/identifier", http.HandlerFunc(i.handleIdentifier)).Methods(http.MethodGet).Name("index")
	r.Handle("/chooseaccount", i).Methods(http.MethodGet).Name("chooseaccount")
	r.Handle("/consent", i).Methods(http.MethodGet).Name("consent")
	r.Handle("/welcome", i).Methods(http.MethodGet).Name("welcome")
	r.Handle("/goodbye", i).Methods(http.MethodGet).Name("goodbye")
	r.Handle("/index.html", i).Methods(http.MethodGet) // For service worker.
	r.Handle("/identifier/_/logon", i.secureHandler(http.HandlerFunc(i.handleLogon))).Methods(http.MethodPost)
	r.Handle("/identifier/_/logoff", i.secureHandler(http.HandlerFunc(i.handleLogoff))).Methods(http.MethodPost)
	r.Handle("/identifier/_/hello", i.secureHandler(http.HandlerFunc(i.handleHello))).Methods(http.MethodPost)
	r.Handle("/identifier/_/consent", i.secureHandler(http.HandlerFunc(i.handleConsent))).Methods(http.MethodPost)
	r.Handle("/identifier/oauth2/start", http.HandlerFunc(i.handleOAuth2Start)).Methods(http.MethodGet).Name("oauth2/start")
	r.Handle("/identifier/oauth2/cb", http.HandlerFunc(i.handleOAuth2Cb)).Methods(http.MethodGet).Name("oauth2/cb")
	r.Handle("/identifier/saml2/metadata", http.HandlerFunc(i.handleSAML2Metadata))
	r.Handle("/identifier/saml2/acs", http.HandlerFunc(i.handleSAML2AssertionConsumerService)).Methods(http.MethodPost).Name("saml2/acs")
	r.Handle("/identifier/_/saml2/slo", http.HandlerFunc(i.handleSAML2SingleLogoutService)).Methods(http.MethodGet).Name("saml2/slo")
	r.Handle("/identifier/trampolin", http.HandlerFunc(i.handleTrampolin)).Methods(http.MethodGet).Name("trampolin")
	r.Handle("/identifier/trampolin/trampolin.js", http.HandlerFunc(i.handleTrampolin)).Methods(http.MethodGet)

	i.router = r

	if i.backend != nil {
		i.backend.RunWithContext(ctx)
	}
}

// ServeHTTP implements the http.Handler interface.
func (i *Identifier) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	addCommonResponseHeaders(rw.Header())
	addNoCacheResponseHeaders(rw.Header())

	// Show default.
	i.writeWebappIndexHTML(rw, req)
}

// SetKey sets the provided key for the accociated identifier.
func (i *Identifier) SetKey(key []byte) error {
	var ce jose.ContentEncryption
	var algo jose.KeyAlgorithm
	switch len(key) {
	case 16:
		ce = jose.A128GCM
		algo = jose.A128GCMKW
	case 24:
		ce = jose.A192GCM
		algo = jose.A192GCMKW
	case 32:
		ce = jose.A256GCM
		algo = jose.A256GCMKW
	default:
		return fmt.Errorf("identifier invalid encryption key size. Need 16, 24 or 32 bytes")
	}

	if len(key) < 32 {
		i.logger.Warnf("identifier using encryption key size with %d bytes which is below 32 bytes", len(key))
	} else {
		i.logger.WithField("security", fmt.Sprintf("%s:%s", ce, algo)).Infoln("identifier set up")
	}

	recipient := jose.Recipient{
		Algorithm: algo,
		KeyID:     "",
		Key:       key,
	}
	encrypter, err := jose.NewEncrypter(
		ce,
		recipient,
		nil,
	)
	if err != nil {
		return err
	}

	i.encrypter = encrypter
	i.recipient = &recipient
	return nil
}

// ErrorPage writes a HTML error page to the provided ResponseWriter.
func (i *Identifier) ErrorPage(rw http.ResponseWriter, code int, title string, message string) {
	utils.WriteErrorPage(rw, code, title, message)
}

// WriteLogonCookie serializes the provided user into an encrypted string and
// sets it as a logon cookie without firing any callbacks. Use this when the
// cookie must be written mid-flow (e.g. inside an authorize handler) where
// the caller is responsible for updating browser state separately.
func (i *Identifier) WriteLogonCookie(rw http.ResponseWriter, user *IdentifiedUser) error {
	loggedOn, logonAt := user.LoggedOn()
	if !loggedOn {
		return fmt.Errorf("refused to set cookie for not logged on user")
	}

	claims := jwt.Claims{
		Issuer:   user.BackendName(),
		Audience: audienceMarker,
		Subject:  user.Subject(),
		IssuedAt: jwt.NewNumericDate(logonAt),
	}
	if user.expiresAfter != nil {
		claims.Expiry = jwt.NewNumericDate(*user.expiresAfter)
	}

	userClaims := map[string]interface{}(user.Claims())
	if sessionRef := user.SessionRef(); sessionRef != nil {
		userClaims[SessionIDClaim] = *sessionRef
	}
	if logonRef := user.LogonRef(); logonRef != nil {
		userClaims[LogonRefClaim] = *logonRef
	}
	if externalAuthorityID := user.ExternalAuthorityID(); externalAuthorityID != nil {
		userClaims[ExternalAuthorityIDClaim] = *externalAuthorityID
	}
	if lockedScopes := user.LockedScopes(); lockedScopes != nil {
		userClaims[LockedScopesClaim] = strings.Join(lockedScopes, " ")
	}

	serialized, err := jwt.Encrypted(i.encrypter).Claims(claims).Claims(userClaims).CompactSerialize()
	if err != nil {
		return err
	}

	return i.setLogonCookie(rw, serialized)
}

// SetUserToLogonCookie serializes the provided user into an encrypted string,
// sets it as cookie on the provided http.ResponseWriter, and fires the
// onSetLogon callbacks.
func (i *Identifier) SetUserToLogonCookie(ctx context.Context, rw http.ResponseWriter, user *IdentifiedUser) error {
	if err := i.WriteLogonCookie(rw, user); err != nil {
		return err
	}
	// Trigger callbacks.
	for _, f := range i.onSetLogonCallbacks {
		if err := f(ctx, rw, user); err != nil {
			return err
		}
	}
	return nil
}

// UnsetLogonCookie adds cookie remove headers to the provided http.ResponseWriter
// effectively implementing logout.
func (i *Identifier) UnsetLogonCookie(ctx context.Context, user *IdentifiedUser, rw http.ResponseWriter) error {
	// Remove cookie.
	err := i.removeLogonCookie(rw)
	if err != nil {
		return err
	}
	// Destroy backend user session if any.
	if user != nil {
		if sessionRef := user.SessionRef(); sessionRef != nil {
			err = i.backend.DestroySession(ctx, sessionRef)
			if err != nil {
				i.logger.WithError(err).Warnln("failed to destroy session on unset logon cookie")
			}
		}
	}
	// Trigger callbacks.
	for _, f := range i.onUnsetLogonCallbacks {
		err = f(ctx, rw)
		if err != nil {
			return err
		}
	}

	return nil
}

// EndSession begins the process to end the session either directly or indirectly
// based on the provided user. It optionally returns an uri which shall be used
// as redirection target or an error.
func (i *Identifier) EndSession(ctx context.Context, user *IdentifiedUser, rw http.ResponseWriter, postRedirectURI *url.URL, state string) (*url.URL, error) {
	err := i.UnsetLogonCookie(ctx, user, rw)
	if err != nil {
		return nil, err
	}

	var uri *url.URL
	if user.externalAuthority != nil && user.externalAuthority.EndSessionEnabled {
		// Generate state and set state cookie with postRedirectURI.
		if state == "" {
			state = rndm.GenerateRandomString(32)
		}
		sd := &StateData{
			State: state,
			Mode:  StateModeEndSession,

			Ref: user.externalAuthority.ID,
		}
		var extra map[string]interface{}
		uri, extra, err = user.externalAuthority.MakeRedirectEndSessionRequestURL(user.LogonRef(), sd.State)
		if err != nil {
			return nil, err
		}
		sd.Extra = extra
		if postRedirectURI != nil && postRedirectURI.String() != "" {
			sd.RawQuery = postRedirectURI.String()
		}

		var scope string
		switch user.externalAuthority.AuthorityType {
		case authorities.AuthorityTypeOIDC:
			// Inject post logout url target.
			cb, _ := i.router.GetRoute("oauth2/cb").URL()
			next, _ := url.Parse(i.baseURI.String())
			next.Path = cb.Path
			query := uri.Query()
			query.Set("post_logout_redirect_uri", next.String())
			uri.RawQuery = query.Encode()
			// Redirect using trampolin, to ensure origin checks of external
			// authority can pass.
			sd.Trampolin = &TrampolinData{
				URI:   uri.String(),
				Scope: "oauth2/cb",
			}
			scope = "trampolin"
			uri, _ = i.router.GetRoute("trampolin").URL()
			query = make(url.Values)
			query.Add("state", sd.State)
			uri.RawQuery = query.Encode()

		case authorities.AuthorityTypeSAML2:
			scope = "_/saml2/slo"
		}

		if scope != "" {
			err = i.SetStateToStateCookie(ctx, rw, scope, sd)
			if err != nil {
				return nil, fmt.Errorf("failed to set saml2 slo state cookie: %w", err)
			}
		}
	}

	return uri, nil
}

// GetUserFromLogonCookie looks up the associated cookie name from the provided
// request, parses it and returns the user containing the information found in
// the coookie payload data.
func (i *Identifier) GetUserFromLogonCookie(ctx context.Context, req *http.Request, maxAge time.Duration, refreshSession bool) (*IdentifiedUser, error) {
	cookie, err := i.getLogonCookie(req)
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, nil
		}
		return nil, err
	}

	// Decrypt and parse cookie value.
	token, err := jwt.ParseEncrypted(cookie.Value)
	if err != nil {
		return nil, err
	}

	// Parse claims.
	var claims jwt.Claims
	var userClaims map[string]interface{}
	if claimsErr := token.Claims(i.recipient.Key, &claims, &userClaims); claimsErr != nil {
		return nil, claimsErr
	}

	// Validate claims.
	if claimsErr := claims.Validate(jwt.Expected{
		// Ignore cookie, when issuer does not match our backend name. This usually
		// means that konnect was reconfigured. Users need to sign in again.
		Issuer: i.backend.Name(),
		// Ignore cookie, when audience marker does not match. This happens
		// for cookies from an older version of konnect. Users need to sign in again.
		Audience: jwt.Audience{audienceMarker[0]},
	}); claimsErr != nil {
		i.logger.WithError(claimsErr).Debugln("logon token claims validation failed")
		return nil, nil
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("invalid subject in logon token")
	}
	if userClaims == nil {
		return nil, fmt.Errorf("invalid user claims in logon token")
	}

	// New user with details from claims.
	user := &IdentifiedUser{
		sub: claims.Subject,

		// TODO(longsleep): It is not verified here that the user still exists at
		// our current backend. We still assign the backend happily here - probably
		// needs some sort of veritification / lookup.
		backend: i.backend,

		logonAt: claims.IssuedAt.Time(),
	}
	if claims.Expiry != nil {
		expiresAfter := claims.Expiry.Time()
		user.expiresAfter = &expiresAfter
	}

	loggedOn, logonAt := user.LoggedOn()
	if !loggedOn {
		// Ignore logons which are not valid.
		return nil, nil
	}
	if maxAge > 0 {
		if logonAt.Add(maxAge).Before(time.Now()) {
			// Ignore logon as it is no longer valid within maxAge.
			return nil, nil
		}
	}

	// Get specific data from claims.
	if v := userClaims[SessionIDClaim]; v != nil {
		sessionRef := v.(string)
		if sessionRef != "" {
			// Remember session ref in user.
			user.sessionRef = &sessionRef
			// Ensure the session is still valid, by refreshing it.
			if refreshSession {
				err = i.backend.RefreshSession(ctx, user.Subject(), &sessionRef, userClaims)
				if err != nil {
					// Ignore logons which fail session refresh.
					return nil, nil
				}
			}
		}
	}
	if v := userClaims[LogonRefClaim]; v != nil {
		logonRef := v.(string)
		if logonRef != "" {
			// Remember logon ref in user.
			user.logonRef = &logonRef
		}
	}
	if v := userClaims[ExternalAuthorityIDClaim]; v != nil {
		externalAuthorityID := v.(string)
		if externalAuthorityID != "" {
			authority, err := i.authorities.Lookup(ctx, externalAuthorityID)
			if err != nil {
				// Ignore logons which have set an unknown external authority.
				return nil, nil
			}
			// TODO(longsleep): Check if authority is actually enabled. For now
			// we check if it is ready.
			if !authority.IsReady() {
				// Ignore logons which have sent an authority which is not ready.
				return nil, nil
			}
			user.externalAuthority = authority
		}
	}

	if v := userClaims[LockedScopesClaim]; v != nil {
		lockedScopes := v.(string)
		if lockedScopes != "" {
			user.lockedScopes = strings.Split(lockedScopes, " ")
		}
	}

	// Fill additional claim.
	user.claims = make(map[string]interface{})
	for k, v := range userClaims {
		switch k {
		case konnect.IdentifiedUsernameClaim:
			user.username = v.(string)
		case konnect.IdentifiedDisplayNameClaim:
			user.displayName = v.(string)

		case SessionIDClaim:
			// Already handled above.
			continue
		case LogonRefClaim:
			// Already handled above.
			continue
		case ExternalAuthorityIDClaim:
			// Already handled above.
			continue
		case LockedScopesClaim:
			// Already handled above.
			continue
		case ObsoleteUserClaimsClaim:
			// Keep and ignore for history reasons.
			continue

		case oidc.AudienceClaim, oidc.IssuedAtClaim, oidc.ExpirationClaim, oidc.SubjectIdentifierClaim, oidc.IssuerIdentifierClaim:
			// Ignore default OIDC claims when resurrecting claims data.
			continue

		default:
			// Add the rest.
			user.claims[k] = v
		}
	}

	return user, nil
}

// GetUserFromID looks up the user identified by the provided userID by
// requesting the associated backend.
func (i *Identifier) GetUserFromID(ctx context.Context, userID string, sessionRef *string, requestedScopes map[string]bool) (*IdentifiedUser, error) {
	user, err := i.backend.GetUser(ctx, userID, sessionRef, requestedScopes)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	// XXX(longsleep): This is quite crappy. Move IdentifiedUser to a package
	// which can be imported by backends so they directly can return that shit.
	identifiedUser := &IdentifiedUser{
		sub: user.Subject(),

		username: user.Username(),

		backend: i.backend,

		sessionRef: sessionRef,
		claims:     user.BackendClaims(),
		scopes:     user.BackendScopes(),

		lockedScopes: user.RequiredScopes(),
	}
	if userWithEmail, ok := user.(identity.UserWithEmail); ok {
		identifiedUser.email = userWithEmail.Email()
		identifiedUser.emailVerified = userWithEmail.EmailVerified()
	}
	if userWithProfile, ok := user.(identity.UserWithProfile); ok {
		identifiedUser.displayName = userWithProfile.Name()
		identifiedUser.familyName = userWithProfile.FamilyName()
		identifiedUser.givenName = userWithProfile.GivenName()
	}
	if userWithID, ok := user.(identity.UserWithID); ok {
		identifiedUser.id = userWithID.ID()
	}
	if userWithUniqueID, ok := user.(identity.UserWithUniqueID); ok {
		identifiedUser.uid = userWithUniqueID.UniqueID()
	}

	return identifiedUser, nil
}

// SetConsentToConsentCookie serializses the provided Consent using the provided
// ConsentRequest and sets it as cookie on the provided ReponseWriter.
func (i *Identifier) SetConsentToConsentCookie(ctx context.Context, rw http.ResponseWriter, cr *ConsentRequest, consent *Consent) error {
	serialized, err := jwt.Encrypted(i.encrypter).Claims(consent).CompactSerialize()
	if err != nil {
		return err
	}

	return i.setConsentCookie(rw, cr, serialized)
}

// GetConsentFromConsentCookie extract consent information for the provided
// request and the provide state.
func (i *Identifier) GetConsentFromConsentCookie(ctx context.Context, rw http.ResponseWriter, req *http.Request, state string) (*Consent, error) {
	if state == "" {
		return nil, nil
	}

	cr := &ConsentRequest{
		State:          state,
		ClientID:       req.Form.Get("client_id"),
		RawRedirectURI: req.Form.Get("redirect_uri"),
		Ref:            req.Form.Get("state"),
		Nonce:          req.Form.Get("nonce"),
	}

	cookie, err := i.getConsentCookie(req, cr)
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, nil
		}
		return nil, err
	}

	// Directly remove the cookie again after we used it.
	i.removeConsentCookie(rw, req, cr)

	token, err := jwt.ParseEncrypted(cookie.Value)
	if err != nil {
		return nil, err
	}

	var consent Consent
	if err = token.Claims(i.recipient.Key, &consent); err != nil {
		return nil, err
	}

	return &consent, nil
}

// SetStateToStateCookie serializses the provided StateRequest and sets it
// as cookie on the provided ReponseWriter.
func (i *Identifier) SetStateToStateCookie(ctx context.Context, rw http.ResponseWriter, scope string, sd *StateData) error {
	serialized, err := jwt.Encrypted(i.encrypter).Claims(sd).CompactSerialize()
	if err != nil {
		return err
	}

	return i.setStateCookie(rw, scope, sd.State, serialized)
}

// GetStateFromStateCookie extracts state information for the provided
// request using the provided scope and state.
func (i *Identifier) GetStateFromStateCookie(ctx context.Context, rw http.ResponseWriter, req *http.Request, scope string, state string) (*StateData, error) {
	if state == "" {
		return nil, nil
	}

	cookie, err := i.getStateCookie(req, state)
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, nil
		}
		return nil, err
	}

	// Directly remove the cookie again after we used it.
	i.removeStateCookie(rw, req, scope, state)

	token, err := jwt.ParseEncrypted(cookie.Value)
	if err != nil {
		return nil, err
	}

	sd := &StateData{}
	if err = token.Claims(i.recipient.Key, sd); err != nil {
		return nil, err
	}

	if sd.State != state {
		return nil, fmt.Errorf("state mismatch")
	}

	return sd, nil
}

// Name returns the active identifiers backend's name.
func (i *Identifier) Name() string {
	return i.backend.Name()
}

// ScopesSupported return the scopes supported by the accociated Identifier.
func (i *Identifier) ScopesSupported() []string {
	scopes := mapset.NewThreadUnsafeSet()

	for scope := range i.meta.Scopes.Definitions {
		scopes.Add(scope)
	}
	for _, scope := range i.backend.ScopesSupported() {
		scopes.Add(scope)
	}

	supportedScopes := make([]string, 0)
	it := scopes.Iterator()
	for scope := range it.C {
		supportedScopes = append(supportedScopes, scope.(string))
	}

	return supportedScopes
}

// OnSetLogon implements a way to register hooks whenever logon information is
// set by the accociated Identifier.
func (i *Identifier) OnSetLogon(cb func(ctx context.Context, rw http.ResponseWriter, user identity.User) error) error {
	i.onSetLogonCallbacks = append(i.onSetLogonCallbacks, cb)
	return nil
}

// OnUnsetLogon implements a way to register hooks whenever logon information is
// set by the accociated Identifier.
func (i *Identifier) OnUnsetLogon(cb func(ctx context.Context, rw http.ResponseWriter) error) error {
	i.onUnsetLogonCallbacks = append(i.onUnsetLogonCallbacks, cb)
	return nil
}

func (i *Identifier) absoluteURLForRoute(name string) (*url.URL, error) {
	uri, _ := url.Parse(i.Config.BaseURI.String())

	route := i.router.Get(name)

	path, err := route.URL()
	if err != nil {
		return nil, err
	}

	uri.Path = path.Path

	return uri, nil
}
