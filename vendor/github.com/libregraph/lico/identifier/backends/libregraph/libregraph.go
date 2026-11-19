/*
 * Copyright 2021 Kopano and its licensors
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

package libregraph

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cevaris/ordered_map"
	"github.com/libregraph/oidc-go"
	"github.com/sirupsen/logrus"

	konnect "github.com/libregraph/lico"
	"github.com/libregraph/lico/config"
	"github.com/libregraph/lico/identifier"
	"github.com/libregraph/lico/identifier/backends"
	"github.com/libregraph/lico/identifier/meta/scopes"
	"github.com/libregraph/lico/identity"
	identityClients "github.com/libregraph/lico/identity/clients"
	"github.com/libregraph/lico/utils"
)

const libreGraphIdentifierBackendName = "identifier-libregraph"

const (
	OpenTypeExtensionType = "#microsoft.graph.openTypeExtension"

	IdentityClaimsExtensionName    = "libregraph.identityClaims"
	IDTokenClaimsExtensionName     = "libregraph.idTokenClaims"
	AccessTokenClaimsExtensionName = "libregraph.accessTokenClaims"
	RequestedScopesExtensionName   = "libregraph.requestedScopes"
	SessionExtensionName           = "libregraph.session"
)

const (
	apiPathMe                  = "/api/v1/me"
	apiPathUsers               = "/api/v1/users"
	apiPathRevokeSignInSession = "/api/v1/me/revokeSignInSession"
)

var libreGraphSpportedScopes = []string{
	oidc.ScopeProfile,
	oidc.ScopeEmail,
	konnect.ScopeUniqueUserID,
	konnect.ScopeRawSubject,
}

// ErrUnauthorized is returned when the backend answered a request as
// unauthorized. For user lookups this usually means that the session which was
// sent along is no longer valid, for example because it was revoked. It is
// deliberately not the same as an unknown user, which is no error at all.
var ErrUnauthorized = errors.New("libregraph identifier backend request unauthorized")

type LibreGraphIdentifierBackend struct {
	config *config.Config

	supportedScopes []string

	logger    logrus.FieldLogger
	tlsConfig *tls.Config

	client *http.Client

	baseURLMap          *ordered_map.OrderedMap
	useMultipleBackends bool

	clients *identityClients.Registry
}

type libreGraphUser struct {
	AccountEnabled    bool   `json:"accountEnabled"`
	DisplayName       string `json:"displayName"`
	RawGivenName      string `json:"givenName"`
	ID                string `json:"id"`
	Mail              string `json:"mail"`
	Surname           string `json:"surname"`
	UserPrincipalName string `json:"userPrincipalName"`

	Extensions []map[string]interface{} `json:"extensions"`

	identityClaims  map[string]interface{}
	requestedScopes []string
	requiredScopes  []string
}

func decodeLibreGraphUser(r io.Reader) (*libreGraphUser, error) {
	decoder := json.NewDecoder(r)
	u := &libreGraphUser{}
	if err := decoder.Decode(u); err != nil {
		return nil, err
	}

	identityClaims := make(map[string]interface{})
	identityClaims[konnect.IdentifiedUserIDClaim] = u.ID

	var idTokenClaims map[string]interface{}
	var accessTokenClaims map[string]interface{}
	var requestedScopes []string

	for _, extension := range u.Extensions {
		if odataType, ok := extension["@odata.type"]; ok && odataType.(string) != OpenTypeExtensionType {
			continue
		}
		if extensionName, ok := extension["extensionName"].(string); ok {
			switch extensionName {
			case IdentityClaimsExtensionName:
				if v, ok := extension["claims"].(map[string]interface{}); ok {
					for k, v := range v {
						if k == konnect.InternalExtraIDTokenClaimsClaim || k == konnect.InternalExtraAccessTokenClaimsClaim {
							// Ignore keys which areused internally by IDTokenClaimsExtensionName
							// and AccessTokenClaimsExtensionName.
							continue
						}
						identityClaims[k] = v
					}
				}
			case IDTokenClaimsExtensionName:
				if idTokenClaims == nil {
					idTokenClaims = make(map[string]interface{})
				}
				if v, ok := extension["claims"].(map[string]interface{}); ok {
					for k, v := range v {
						idTokenClaims[k] = v
					}
				}
			case AccessTokenClaimsExtensionName:
				if accessTokenClaims == nil {
					accessTokenClaims = make(map[string]interface{})
				}
				if v, ok := extension["claims"].(map[string]interface{}); ok {
					for k, v := range v {
						accessTokenClaims[k] = v
					}
				}
			case RequestedScopesExtensionName:
				if values, ok := extension["scopes"].([]interface{}); ok {
					for _, v := range values {
						if s, ok := v.(string); ok {
							requestedScopes = append(requestedScopes, s)
						}
					}
				}
			case SessionExtensionName:
				if sid, ok := extension[oidc.SessionIDClaim].(string); ok {
					if sid != "" {
						if accessTokenClaims == nil {
							accessTokenClaims = make(map[string]interface{})
						}
						accessTokenClaims[oidc.SessionIDClaim] = sid
					}
				}
			}
		}
	}

	if idTokenClaims != nil {
		// Inject claims as nested identity claim. The key is picket up by the
		// token signer and used to extend ID token root claims.
		identityClaims[konnect.InternalExtraIDTokenClaimsClaim] = idTokenClaims
	}
	if accessTokenClaims != nil {
		// Inject claims as nested identity claims. The key is picked up by the
		// token signer and userinfo handler  to extend ID and access token root
		// claims based on the request.
		identityClaims[konnect.InternalExtraAccessTokenClaimsClaim] = accessTokenClaims
	}
	if requestedScopes != nil {
		u.requestedScopes = requestedScopes
	}

	u.identityClaims = identityClaims

	return u, nil
}

func (u *libreGraphUser) Subject() string {
	return u.ID
}

func (u *libreGraphUser) Email() string {
	return u.Mail
}
func (u *libreGraphUser) EmailVerified() bool {
	return true
}
func (u *libreGraphUser) Name() string {
	return u.DisplayName
}

func (u *libreGraphUser) FamilyName() string {
	return u.Surname
}

func (u *libreGraphUser) GivenName() string {
	return u.RawGivenName
}

func (u *libreGraphUser) Username() string {
	return u.UserPrincipalName
}

func (u *libreGraphUser) UniqueID() string {
	// Provide our ID as unique ID.
	return u.ID
}

func (u *libreGraphUser) BackendClaims() map[string]interface{} {
	return u.identityClaims
}

func (u *libreGraphUser) BackendScopes() []string {
	return u.requestedScopes
}

func (u *libreGraphUser) RequiredScopes() []string {
	return u.requiredScopes
}

func (u *libreGraphUser) setRequiredScopes(selectedScope string, scopeMap *ordered_map.OrderedMap) []string {
	var requiredScopes []string

	if selectedScope != "" {
		requiredScopes = []string{selectedScope}
	}
	iter := scopeMap.IterFunc()
	for kv, ok := iter(); ok; kv, ok = iter() {
		if scope := kv.Key.(string); scope != selectedScope {
			requiredScopes = append(requiredScopes, "!"+scope)
		}
	}
	u.requiredScopes = requiredScopes
	return requiredScopes
}

// SessionRef implements the backends.UserFromBackendWithSessionRef interface.
func (u *libreGraphUser) SessionRef() *string {
	if sessionID := u.sessionID(); sessionID != "" {
		return &sessionID
	}
	return nil
}

func (u *libreGraphUser) sessionID() string {
	if accessTokenClaims, ok := u.identityClaims[konnect.InternalExtraAccessTokenClaimsClaim].(map[string]interface{}); ok {
		if sessionID, withSessionID := accessTokenClaims[oidc.SessionIDClaim].(string); withSessionID {
			if sessionID != "" {
				return sessionID
			}
		}
	}
	return ""
}

func withSelectQuery(r *http.Request) {
	if r.Form == nil {
		r.Form = make(url.Values)
	}
	r.Form.Set("$select", "accountEnabled,displayName,givenName,id,mail,surname,userPrincipalName,extensions")
}

func NewLibreGraphIdentifierBackend(
	c *config.Config,
	tlsConfig *tls.Config,
	baseURI string,
	baseURIByScope *ordered_map.OrderedMap,
	clients *identityClients.Registry,
) (*LibreGraphIdentifierBackend, error) {

	if baseURI == "" {
		return nil, fmt.Errorf("base uri must not be empty")
	}

	// Build supported scopes based on default scopes.
	supportedScopes := make([]string, len(libreGraphSpportedScopes))
	copy(supportedScopes, libreGraphSpportedScopes)

	baseURLMap := ordered_map.NewOrderedMapWithArgs([]*ordered_map.KVPair{{
		Key:   "",
		Value: baseURI,
	}})
	if baseURIByScope != nil {
		iter := baseURIByScope.IterFunc()
		for kv, ok := iter(); ok; kv, ok = iter() {
			if kv.Key == "" {
				return nil, fmt.Errorf("scoped base uri with empty scope is not allowed")
			}
			baseURLMap.Set(kv.Key, kv.Value)
		}
	}

	transport := utils.HTTPTransportWithTLSClientConfig(tlsConfig)
	transport.MaxIdleConns = 100
	transport.IdleConnTimeout = 30 * time.Second

	b := &LibreGraphIdentifierBackend{
		config: c,

		supportedScopes: supportedScopes,

		logger:    c.Logger,
		tlsConfig: tlsConfig,

		client: &http.Client{
			Transport: transport,
			Timeout:   60 * time.Second,
		},

		baseURLMap:          baseURLMap,
		useMultipleBackends: baseURLMap.Len() > 1,

		clients: clients,
	}

	b.logger.WithField("map", baseURLMap).Infoln("libregraph server identified backend connection set up")

	return b, nil
}

// RunWithContext implements the Backend interface.
func (b *LibreGraphIdentifierBackend) RunWithContext(ctx context.Context) error {
	return nil
}

// Logon implements the Backend interface, enabling Logon with user name and
// password as provided. Requests are bound to the provided context.
func (b *LibreGraphIdentifierBackend) Logon(ctx context.Context, audience, username, password string) (bool, *string, *string, backends.UserFromBackend, error) {
	record, _ := identifier.FromRecordContext(ctx)
	var requestedScopes map[string]bool
	if record != nil && record.HelloRequest != nil {
		requestedScopes = record.HelloRequest.Scopes
	}

	// Inject implicit scopes set by client registration. This is needed here,
	// as the requested scopes might not have the implicit scopes applied yet,
	// based on the calling stack chain and since we use the scopes to select
	// the backend.
	registration, _ := b.clients.Get(ctx, audience)
	if registration != nil {
		_ = registration.ApplyImplicitScopes(requestedScopes)
	}

	selectedScope, meURL := b.getMeURL(requestedScopes)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, meURL, nil)
	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("libregraph identifier backend logon request error: %w", err)
	}
	req.SetBasicAuth(username, password)

	if audience != "" {
		req.Header.Set("X-Audience", audience)
	}

	if record == nil {
		record, _ = identifier.RecordFromRequestContext(ctx, b.config)
	}
	if record != nil {
		// Inject HTTP headers.
		if record.HelloRequest != nil {
			if record.HelloRequest.Flow != "" {
				req.Header.Set("X-Flow", record.HelloRequest.Flow)
			}
			if record.HelloRequest.RawScope != "" {
				req.Header.Set("X-Scope", record.HelloRequest.RawScope)
			}
			if record.HelloRequest.RawPrompt != "" {
				req.Header.Set("X-Prompt", record.HelloRequest.RawPrompt)
			}
		}
		if record.RealIP != "" {
			req.Header.Set("X-User-Real-IP", record.RealIP)
		}
		if record.UserAgent != "" {
			req.Header.Set("X-User-Real-User-Agent", record.UserAgent)
		}
	}
	req.Header.Set("User-Agent", utils.DefaultHTTPUserAgent)

	// Inject select parameter.
	withSelectQuery(req)

	response, err := b.client.Do(req)
	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("libregraph identifier backend logon request failed: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		// breaks
	case http.StatusNotFound:
		return false, nil, nil, nil, nil
	case http.StatusUnauthorized:
		return false, nil, nil, nil, nil
	default:
		return false, nil, nil, nil, fmt.Errorf("libregraph identifier backend logon request unexpected response status: %d", response.StatusCode)
	}

	user, err := decodeLibreGraphUser(response.Body)
	if err != nil {
		return false, nil, nil, nil, fmt.Errorf("libregraph identifier backend logon json decode error: %w", err)
	}

	if !user.AccountEnabled {
		return false, nil, nil, nil, nil
	}

	requiredScopes := user.setRequiredScopes(selectedScope, b.baseURLMap)

	// Use the users subject as user id.
	userID := user.Subject()

	sessionID := user.sessionID()

	b.logger.WithFields(logrus.Fields{
		"username":  user.Username(),
		"id":        userID,
		"scope":     requiredScopes,
		"sessionID": sessionID,
	}).Debugln("libregraph identifier backend logon")

	// Put the user into the record (if any).
	if record != nil {
		record.BackendUser = user
	}

	return true, &userID, &sessionID, user, nil
}

// GetUser implements the Backend interface, providing user meta data retrieval
// for the user specified by the userID. Requests are bound to the provided
// context.
func (b *LibreGraphIdentifierBackend) GetUser(ctx context.Context, entryID string, sessionRef *string, requestedScopes map[string]bool, lookup *identity.LookupContext) (backends.UserFromBackend, error) {
	record, _ := identifier.FromRecordContext(ctx)
	if record != nil {
		if record.BackendUser != nil {
			if user, ok := record.BackendUser.(*libreGraphUser); ok {
				// Fastpath, if logon previously injected the user.
				if user.ID == entryID {
					return user, nil
				}
			}
		}
		if requestedScopes == nil && record.HelloRequest != nil {
			requestedScopes = record.HelloRequest.Scopes
		}
	}

	selectedScope, userURL := b.getUserURL(requestedScopes)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL+"/"+entryID, nil)
	if err != nil {
		return nil, fmt.Errorf("libregraph identifier backend get user request error: %w", err)
	}

	// Inject HTTP headers.
	if requestedScopes != nil {
		rawRequestedScopes := make([]string, 0)
		for scope, enabled := range requestedScopes {
			if enabled {
				rawRequestedScopes = append(rawRequestedScopes, scope)
			}
		}
		req.Header.Set("X-Scope", strings.Join(rawRequestedScopes, " "))
	}
	if sessionRef != nil {
		sessionID := *sessionRef
		if identity.IsBackendSessionRef(libreGraphIdentifierBackendName, sessionID) {
			// Only send the session ID if the backend can resolve it.
			req.Header.Set("X-SessionID", sessionID)
		}
	}
	if lookup != nil {
		// Tell the backend about the request which triggered this lookup, so it
		// can decide how to treat its own session.
		if lookup.Audience != "" {
			req.Header.Set("X-Audience", lookup.Audience)
		}
		if lookup.NewGrant {
			req.Header.Set("X-New-Grant", "1")
		}
		if lookup.RefreshTokenID != "" {
			req.Header.Set("X-Refresh-Token-ID", lookup.RefreshTokenID)
		}
		if lookup.NewAccessTokenID != "" {
			req.Header.Set("X-New-Access-Token-ID", lookup.NewAccessTokenID)
		}
		if lookup.NewRefreshTokenID != "" {
			req.Header.Set("X-New-Refresh-Token-ID", lookup.NewRefreshTokenID)
		}
	}

	if record == nil {
		record, _ = identifier.RecordFromRequestContext(ctx, b.config)
	}
	if record != nil {
		if record.IdentifiedUser != nil {
			if ok, ts := record.IdentifiedUser.LoggedOn(); ok {
				req.Header.Set("X-User-Logon-At", ts.UTC().Format(http.TimeFormat))
			}
		}
		if record.RealIP != "" {
			req.Header.Set("X-User-Real-IP", record.RealIP)
		}
		if record.UserAgent != "" {
			req.Header.Set("X-User-Real-User-Agent", record.UserAgent)
		}
	}
	req.Header.Set("User-Agent", utils.DefaultHTTPUserAgent)

	// Inject select parameter.
	withSelectQuery(req)

	response, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("libregraph identifier backend get user request failed: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		// breaks
	case http.StatusNotFound:
		return nil, nil
	case http.StatusUnauthorized:
		// The session which was sent with this lookup got rejected, for example
		// because it was revoked. Keep this apart from an unknown user, so the
		// two cases can be told apart.
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("libregraph identifier backend get user request unexpected response status: %d", response.StatusCode)
	}

	user, err := decodeLibreGraphUser(response.Body)
	if err != nil {
		return nil, fmt.Errorf("libregraph identifier backend logon json decode error: %w", err)
	}

	if !user.AccountEnabled {
		return nil, nil
	}

	user.setRequiredScopes(selectedScope, b.baseURLMap)

	return user, nil
}

// ResolveUserByUsername implements the Backend interface, providing lookup for
// user by providing the username. Requests are bound to the provided context.
func (b *LibreGraphIdentifierBackend) ResolveUserByUsername(ctx context.Context, username string) (backends.UserFromBackend, error) {
	// Libregraph backend accept both user name and ID lookups, so this is
	// the same as GetUser without a session.
	return b.GetUser(ctx, username, nil, nil, nil)
}

// RefreshSession implements the Backend interface.
func (b *LibreGraphIdentifierBackend) RefreshSession(ctx context.Context, userID string, sessionRef *string, claims map[string]interface{}) error {
	user, err := b.GetUser(ctx, userID, sessionRef, nil, nil)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("refresh session did not yield a user")
	}
	return nil
}

// DestroySession implements the Backend interface providing explicit revoke
// of the backend session.
func (b *LibreGraphIdentifierBackend) DestroySession(ctx context.Context, sessionRef *string) error {
	var requestedScopes map[string]bool
	record, _ := identifier.FromRecordContext(ctx)
	if record != nil {
		if record.HelloRequest != nil {
			requestedScopes = record.HelloRequest.Scopes
		}
	}

	_, revokeSessionURL := b.getRevokeSigninSessionURL(requestedScopes)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, revokeSessionURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("libregraph identifier backend destroy session request error: %w", err)
	}

	if sessionRef != nil {
		sessionID := *sessionRef
		if identity.IsBackendSessionRef(libreGraphIdentifierBackendName, sessionID) {
			// Only send the session ID if the backend can resolve it.
			req.Header.Set("X-SessionID", sessionID)
		}
	}

	if record == nil {
		record, _ = identifier.RecordFromRequestContext(ctx, b.config)
	}
	if record != nil {
		// Inject HTTP headers.
		if record.RealIP != "" {
			req.Header.Set("X-User-Real-IP", record.RealIP)
		}
		if record.UserAgent != "" {
			req.Header.Set("X-User-Real-User-Agent", record.UserAgent)
		}
	}
	req.Header.Set("User-Agent", utils.DefaultHTTPUserAgent)

	response, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("libregraph identifier backend destroy session request failed: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		// breaks
	case http.StatusNotFound:
		return nil
	case http.StatusUnauthorized:
		return nil
	default:
		return fmt.Errorf("libregraph identifier backend logon request unexpected response status: %d", response.StatusCode)
	}

	return nil
}

// UserClaims implements the Backend interface, providing user specific claims
// for the user specified by the userID.
func (b *LibreGraphIdentifierBackend) UserClaims(userID string, authorizedScopes map[string]bool) map[string]interface{} {
	return nil
}

// ScopesSupported implements the Backend interface, providing supported scopes
// when running this backend.
func (b *LibreGraphIdentifierBackend) ScopesSupported() []string {
	return b.supportedScopes
}

// ScopesMeta implements the Backend interface, providing meta data for
// supported scopes.
func (b *LibreGraphIdentifierBackend) ScopesMeta() *scopes.Scopes {
	return nil
}

// Name implements the Backend interface.
func (b *LibreGraphIdentifierBackend) Name() string {
	return libreGraphIdentifierBackendName
}

func (b *LibreGraphIdentifierBackend) getBaseURL(requestedScopes map[string]bool) (string, string) {
	if b.useMultipleBackends && requestedScopes != nil {
		// Loop through configured backends for each requested scope.
		for s, v := range requestedScopes {
			if !v {
				continue
			}
			if u, ok := b.baseURLMap.Get(s); ok {
				return s, u.(string)
			}
		}
	}
	// If nothing found, return default.
	u, _ := b.baseURLMap.Get("")
	return "", u.(string)
}

func (b *LibreGraphIdentifierBackend) getMeURL(requestedScopes map[string]bool) (string, string) {
	scope, baseURL := b.getBaseURL(requestedScopes)

	return scope, baseURL + apiPathMe
}

func (b *LibreGraphIdentifierBackend) getUserURL(requestedScopes map[string]bool) (string, string) {
	scope, baseURL := b.getBaseURL(requestedScopes)

	return scope, baseURL + apiPathUsers
}

func (b *LibreGraphIdentifierBackend) getRevokeSigninSessionURL(requestedScopes map[string]bool) (string, string) {
	scope, baseURL := b.getBaseURL(requestedScopes)

	return scope, baseURL + apiPathRevokeSignInSession
}
