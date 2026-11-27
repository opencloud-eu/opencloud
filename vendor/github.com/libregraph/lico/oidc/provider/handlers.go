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

package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-jose/go-jose/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/libregraph/oidc-go"
	"github.com/longsleep/rndm"
	"github.com/sirupsen/logrus"

	konnect "github.com/libregraph/lico"
	"github.com/libregraph/lico/identity"
	"github.com/libregraph/lico/identity/clients"
	"github.com/libregraph/lico/identity/resources"
	konnectoidc "github.com/libregraph/lico/oidc"
	"github.com/libregraph/lico/oidc/code"
	"github.com/libregraph/lico/oidc/payload"
	"github.com/libregraph/lico/utils"
)

const (
	registrationSizeLimit = 1024 * 512
)

// WellKnownHandler implements the HTTP provider configuration endpoint
// for OpenID Connect 1.0 as specified at https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderConfig
// and for OAuth 2.0 as specified at https://datatracker.ietf.org/doc/html/rfc8414#section-3
func (p *Provider) WellKnownHandler(rw http.ResponseWriter, req *http.Request) {
	// The document only changes when this provider is restarted, and clients
	// which discover it on every unauthorized response depend on being able to
	// cache it.
	rw.Header().Set("Cache-Control", "public, max-age=3600")

	err := utils.WriteJSON(rw, http.StatusOK, p.metadataDocument, "")
	if err != nil {
		p.logger.WithError(err).Errorln("well-known request failed writing response")
	}
}

// JwksHandler implements the HTTP provider JWKS endpoint for OpenID provider
// metadata used with OpenID Connect Discovery 1.0 as specified at https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderMetadata
func (p *Provider) JwksHandler(rw http.ResponseWriter, req *http.Request) {
	addResponseHeaders(rw.Header())

	validationKeys := p.validationKeys
	jwks := &jose.JSONWebKeySet{
		Keys: make([]jose.JSONWebKey, 0),
	}
	for kid, key := range validationKeys {
		certificates, _ := p.certificates[kid]
		keyJwk := jose.JSONWebKey{
			Key:          key,
			KeyID:        kid,
			Use:          "sig", // https://tools.ietf.org/html/rfc7517#section-4.2
			Certificates: certificates,
		}
		if keyJwk.Valid() {
			jwks.Keys = append(jwks.Keys, keyJwk.Public())
		}
	}

	err := utils.WriteJSON(rw, http.StatusOK, jwks, "application/jwk-set+json")
	if err != nil {
		p.logger.WithError(err).Errorln("jwks request failed writing response")
	}
}

// AuthorizeHandler implements the HTTP authorization endpoint for OpenID
// Connect 1.0 as specified at http://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthorizationEndpoint
//
// Currently AuthorizeHandler implements only the Implicit Flow as specified at
// http://openid.net/specs/openid-connect-core-1_0.html#ImplicitFlowAuth
func (p *Provider) AuthorizeHandler(rw http.ResponseWriter, req *http.Request) {
	var err error
	var auth identity.AuthRecord
	var nextManager identity.Manager
	var resource *resources.Resource

	addResponseHeaders(rw.Header())

	ctx := konnect.NewRequestContext(req.Context(), req)

	// OpenID Connect 1.0 authentication request validation.
	// http://openid.net/specs/openid-connect-core-1_0.html#ImplicitValidation
	err = req.ParseForm()
	if err != nil {
		p.logger.WithError(err).Errorln("authorize request invalid form data")
		p.ErrorPage(rw, http.StatusBadRequest, oidc.ErrorCodeOAuth2InvalidRequest, err.Error())
		return
	}

	ar, err := payload.DecodeAuthenticationRequest(req, p.metadata, func(token *jwt.Token) (interface{}, error) {
		if claims, ok := token.Claims.(*payload.RequestObjectClaims); ok {
			// Validate signed request tokens according to spec defined at
			// https://openid.net/specs/openid-connect-core-1_0.html#SignedRequestObject
			registration, _ := p.clients.Get(ctx, claims.ClientID)
			if registration != nil {
				if registration.RawRequestObjectSigningAlg != "" {
					if token.Method.Alg() != registration.RawRequestObjectSigningAlg {
						return nil, fmt.Errorf("token alg does not match client registration")
					}
				}
				if token.Method == jwt.SigningMethodNone {
					// Request parameters do not need to be signed to be valid, so
					// none is allowed in this special case.
					return jwt.UnsafeAllowNoneSignatureType, nil
				}
				// Get secure client.
				if registration.JWKS != nil {
					secureClient, err := registration.Secure(token.Header[oidc.JWTHeaderKeyID])
					if err != nil {
						return nil, err
					}
					if err := claims.SetSecure(secureClient); err != nil {
						return nil, err
					}
					return secureClient.PublicKey, err
				}
				return nil, fmt.Errorf("no client keys registered")
			} else {
				// Also allow, when client is not registered and the token is unsigned.
				if token.Method == jwt.SigningMethodNone {
					// Request parameters do not need to be signed to be valid, so
					// none is allowed in this special case.
					return jwt.UnsafeAllowNoneSignatureType, nil
				}
			}
		}

		return nil, fmt.Errorf("not validated")
	})
	if err != nil {
		p.logger.WithFields(utils.ErrorAsFields(err)).Errorln("authorize request invalid request data")
		p.ErrorPage(rw, http.StatusBadRequest, oidc.ErrorCodeOAuth2InvalidRequest, err.Error())
		return
	}
	err = ar.Validate(func(token *jwt.Token) (interface{}, error) {
		// Validator for incoming IDToken hints, looks up key.
		return p.validateJWT(token)
	})
	if err != nil {
		goto done
	}

	// Resource indicator policy, a token is only ever issued for a registered
	// resource. https://www.rfc-editor.org/rfc/rfc8707.html#section-2
	if ar.Resource != "" {
		var found bool
		resource, found = p.getResource(ar.Resource)
		if !found {
			err = ar.NewError(konnectoidc.ErrorCodeOAuth2InvalidTarget, "unknown resource")
			goto done
		}
		// Use the registered form, it is what ends up in the token audience.
		ar.Resource = resource.ID
	}

	// Inject implicit scopes set by client registration.
	if registration, _ := p.clients.Get(ctx, ar.ClientID); registration != nil {
		err = registration.ApplyImplicitScopes(ar.Scopes)
		if err != nil {
			p.logger.WithError(err).Debugln("failed to apply implicit scopes")
		}
	}

	// A resource only ever sees the scopes it accepts, its own implicit ones
	// included. The narrowing runs after the implicit scopes of the client
	// registration have been added, since the audience is what bounds a token
	// and a scope the resource never declared is one it has no way to honor. A
	// scope which has to reach the resource regardless of what a client asks
	// for belongs in the resource's own implicit_scopes.
	if resource != nil {
		resource.ApplyImplicitScopes(ar.Scopes)

		for scope := range ar.Scopes {
			switch scope {
			case oidc.ScopeOpenID:
				fallthrough
			case oidc.ScopeOfflineAccess:
				// Neither is a permission of the resource. openid selects an ID
				// token, which is about the client, and the MCP specification
				// keeps offline_access out of a resource's scopes since a
				// refresh token is not a resource requirement.
				continue
			}
			if !resource.AllowsScope(scope) {
				delete(ar.Scopes, scope)
			}
		}
	}

	// Find session if any, ignoring errors.
	ar.Session, err = p.getSession(req)
	if err != nil {
		p.logger.WithError(err).Debugln("failed to decode client session")
	}

	// Authorization Server Authenticates End-User
	// http://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthenticates
	nextManager = p.guestManager
	auth, err = p.identityManager.Authenticate(ctx, rw, req, ar, nextManager)
	if err != nil {
		goto done
	}

	// Additional validation based on requested ID token claims.
	if ar.Claims != nil && ar.Claims.IDToken != nil {
		// Validate sub claim request
		// https://openid.net/specs/openid-connect-core-1_0.html#ImplicitValidation
		if subRequest, ok := ar.Claims.IDToken.Get(oidc.SubjectIdentifierClaim); ok {
			if !subRequest.Match(auth.Subject()) {
				err = ar.NewError(oidc.ErrorCodeOAuth2AccessDenied, "sub claim request mismatch")
				goto done
			}
		}
	}

	// Authorization Server Obtains End-User Consent/Authorization
	// http://openid.net/specs/openid-connect-core-1_0.html#ImplicitConsent
	auth, err = auth.Manager().Authorize(ctx, rw, req, ar, auth)
	if err != nil {
		goto done
	}

done:
	p.AuthorizeResponse(rw, req, ar, auth, err)
}

// AuthorizeResponse writes the result according to the provided parameters to
// the provided http.ResponseWriter.
func (p *Provider) AuthorizeResponse(rw http.ResponseWriter, req *http.Request, ar *payload.AuthenticationRequest, auth identity.AuthRecord, err error) {
	var codeString string
	var accessTokenString string
	var idTokenString string
	var authorizedScopes map[string]bool
	var session *payload.Session
	var ctx context.Context

	if err != nil {
		goto done
	}

	ctx = identity.NewContext(konnect.NewRequestContext(req.Context(), req), auth)

	// Create session.
	session, err = p.updateOrCreateSession(rw, req, ar, auth)
	if err != nil {
		goto done
	}

	authorizedScopes = auth.AuthorizedScopes()

	// Create code when requested.
	if _, ok := ar.ResponseTypes[oidc.ResponseTypeCode]; ok {
		codeString, err = p.codeManager.Create(&code.Record{
			AuthenticationRequest: ar,
			Auth:                  auth,
			Session:               session,
		})
		if err != nil {
			goto done
		}
	}

	// Create access token when requested.
	if _, ok := ar.ResponseTypes[oidc.ResponseTypeToken]; ok {
		accessTokenString, err = p.makeAccessToken(ctx, ar.ClientID, ar.Resource, auth, nil, nil, "")
		if err != nil {
			goto done
		}
	}

	// Create ID token when requested and granted.
	if authorizedScopes[oidc.ScopeOpenID] {
		if _, ok := ar.ResponseTypes[oidc.ResponseTypeIDToken]; ok {
			idTokenString, err = p.makeIDToken(ctx, ar, auth, session, accessTokenString, codeString, nil, nil)
			if err != nil {
				goto done
			}
		}
	}

done:
	// Always set browser state.
	browserState, browserStateErr := p.makeBrowserState(ar, auth, err)
	if browserStateErr != nil {
		p.logger.WithError(err).Errorln("failed to make browser state")
	}
	if browserStateErr = p.setBrowserStateCookie(rw, browserState); browserStateErr != nil {
		p.logger.WithError(err).Errorln("failed to set browser state cookie")
	}

	if err != nil {
		switch err.(type) {
		case *payload.AuthenticationError:
			p.Found(rw, ar.RedirectURI, err, ar.UseFragment)
		case *payload.AuthenticationBadRequest:
			p.ErrorPage(rw, http.StatusBadRequest, err.Error(), err.(*payload.AuthenticationBadRequest).Description())
		case *identity.RedirectError:
			p.Found(rw, err.(*identity.RedirectError).RedirectURI(), nil, false)
		case *identity.LoginRequiredError:
			p.LoginRequiredPage(rw, req, err.(*identity.LoginRequiredError).SignInURI())
		case *identity.IsHandledError:
			// do nothing
		case *konnectoidc.OAuth2Error:
			err = ar.NewError(err.Error(), err.(*konnectoidc.OAuth2Error).Description())
			p.Found(rw, ar.RedirectURI, err, ar.UseFragment)
		default:
			p.logger.WithFields(utils.ErrorAsFields(err)).Errorln("authorize request failed")
			p.ErrorPage(rw, http.StatusInternalServerError, err.Error(), "well sorry, but there was a problem")
		}

		return
	}

	sessionState, sessionStateErr := p.makeSessionState(req, ar, browserState)
	if sessionStateErr != nil {
		p.logger.WithError(err).Errorln("failed to make session state")
	}

	authorizedScopesList := makeArrayFromBoolMap(authorizedScopes)

	// Successful Authentication Response
	// http://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthResponse
	response := &payload.AuthenticationSuccess{
		State: ar.State,
		Iss:   p.issuerIdentifier,
		Scope: strings.Join(authorizedScopesList, " "),

		SessionState: sessionState,
	}
	if codeString != "" {
		response.Code = codeString
	}
	if accessTokenString != "" {
		response.AccessToken = accessTokenString
		response.TokenType = oidc.TokenTypeBearer
		response.ExpiresIn = int64(p.accessTokenDuration.Seconds())
	}
	if idTokenString != "" {
		response.IDToken = idTokenString
	}

	p.Found(rw, ar.RedirectURI, response, ar.UseFragment)
}

// TokenHandler implements the HTTP token endpoint for OpenID
// Connect 1.0 as specified at http://openid.net/specs/openid-connect-core-1_0.html#TokenEndpoint
func (p *Provider) TokenHandler(rw http.ResponseWriter, req *http.Request) {
	var err error
	var tr *payload.TokenRequest
	var found bool
	var ar *payload.AuthenticationRequest
	var auth identity.AuthRecord
	var session *payload.Session
	var accessTokenString string
	var idTokenString string
	var refreshTokenString string
	var newAccessTokenID string
	var newRefreshTokenID string
	var refreshTokenClaims *konnect.RefreshTokenClaims
	var approvedScopes map[string]bool
	var authorizedScopes map[string]bool
	var clientDetails *clients.Details
	var lookupRedirectURI *url.URL
	signinMethod := p.signingMethodDefault

	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("Pragma", "no-cache")

	ctx := konnect.NewRequestContext(req.Context(), req)

	// Validate request method
	switch req.Method {
	case http.MethodPost:
		// breaks
	default:
		err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidRequest, "request must be sent with POST")
		goto done
	}

	// Token Request Validation
	// http://openid.net/specs/openid-connect-core-1_0.html#TokenRequestValidation
	err = req.ParseForm()
	if err != nil {
		err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidRequest, err.Error())
		goto done
	}
	tr, err = payload.DecodeTokenRequest(req, p.metadata)
	if err != nil {
		err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidRequest, err.Error())
		goto done
	}

	err = tr.Validate(func(token *jwt.Token) (interface{}, error) {
		// Validator for incoming refresh tokens, looks up key.
		return p.validateJWT(token)
	}, &konnect.RefreshTokenClaims{})
	if err != nil {
		goto done
	}

	// Additional validations according to https://tools.ietf.org/html/rfc6749#section-4.1.3
	// The redirect_uri is a parameter of the authorization code grant alone.
	// The refresh token grant of https://tools.ietf.org/html/rfc6749#section-6
	// has none, so a client which sends one anyway must not have it validated
	// against the registration, where it means nothing for that grant. An
	// unparsable value is treated the same way, and fails the comparison with
	// the one of the authorization request below.
	lookupRedirectURI = &url.URL{}
	if tr.GrantType == oidc.GrantTypeAuthorizationCode && tr.RedirectURI != nil {
		lookupRedirectURI = tr.RedirectURI
	}
	clientDetails, err = p.clients.Lookup(ctx, tr.ClientID, tr.ClientSecret, lookupRedirectURI, "", false)
	if err != nil {
		// Unknown client and failed client authentication alike, as defined in
		// https://datatracker.ietf.org/doc/html/rfc6749#section-5.2.
		// access_denied belongs to the authorization endpoint.
		err = konnectoidc.NewOAuth2Error(konnectoidc.ErrorCodeOAuth2InvalidClient, err.Error())
		goto done
	}
	if clientDetails != nil && clientDetails.Registration != nil {
		signinMethod = jwt.GetSigningMethod(clientDetails.Registration.RawIDTokenSignedResponseAlg)
	}

	switch tr.GrantType {
	case oidc.GrantTypeAuthorizationCode:
		codeRecord, codeRecordFound := p.codeManager.Pop(tr.Code)
		if !codeRecordFound {
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidGrant, "code not found")
			goto done
		}

		ar = codeRecord.AuthenticationRequest
		auth = codeRecord.Auth
		session = codeRecord.Session

		authorizedScopes = auth.AuthorizedScopes()

		// Ensure that the authorization code was issued to the client id.
		if ar.ClientID != tr.ClientID {
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidGrant, "client_id mismatch")
			goto done
		}

		// Ensure that the "redirect_uri" parameter is a match.
		if ar.RawRedirectURI != tr.RawRedirectURI {
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidGrant, "redirect_uri mismatch")
			goto done
		}

		// Ensure that the "resource" parameter, when given, is the one the code
		// was issued for. A token can only be bound to a single resource, so
		// the narrowing RFC 8707 section 2 allows is an exact match here.
		if tr.Resource != "" && tr.Resource != ar.Resource {
			err = konnectoidc.NewOAuth2Error(konnectoidc.ErrorCodeOAuth2InvalidTarget, "resource mismatch")
			goto done
		}

		// Validate code challenge according to https://tools.ietf.org/html/rfc7636#section-4.6
		if p.requirePKCE || (clientDetails.Registration != nil && clientDetails.Registration.RequirePKCE) {
			if ar.CodeChallenge == "" {
				err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidGrant, "code_challenge required")
				goto done
			}
		}
		if tr.CodeVerifier != "" || ar.CodeChallenge != "" {
			if codeVerifierErr := oidc.ValidateCodeChallenge(ar.CodeChallenge, ar.CodeChallengeMethod, tr.CodeVerifier); codeVerifierErr != nil {
				err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidGrant, codeVerifierErr.Error())
				goto done
			}
		}

		if _, ok := identity.FromContext(ctx); !ok {
			ctx = identity.NewContext(ctx, auth)
		}

	case oidc.GrantTypeRefreshToken:
		if tr.RefreshToken == nil {
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidGrant, "missing refresh_token")
			goto done
		}

		// Get claims from refresh token.
		claims := tr.RefreshToken.Claims.(*konnect.RefreshTokenClaims)

		// Ensure that the authorization code was issued to the client id.
		if claims.Audience[0] != tr.ClientID {
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidGrant, "client_id mismatch")
			goto done
		}

		// TODO(longsleep): Compare standard claims issuer.

		userID, sessionRef := p.getUserIDAndSessionRefFromClaims(&claims.RegisteredClaims, "", claims.SessionClaims, claims.IdentityClaims)
		if userID == "" {
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidToken, "missing data in kc.identity claim")
			goto done
		}

		ctx = konnect.NewClaimsContext(ctx, claims)

		currentIdentityManager, claimsErr := p.getIdentityManagerFromClaims(claims.IdentityProvider, claims.IdentityClaims)
		if claimsErr != nil {
			err = claimsErr
			goto done
		}

		// Lookup Ref values from backend.
		approvedScopes, err = currentIdentityManager.ApprovedScopes(ctx, claims.Subject, tr.ClientID, claims.Ref)
		if err != nil {
			goto done
		}
		if approvedScopes == nil {
			// Use approvals from token if backend did not say anything.
			approvedScopes = make(map[string]bool)
			for _, scope := range claims.ApprovedScopesList {
				approvedScopes[scope] = true
			}
		}

		// Ensure that the "resource" parameter, when given, is the one the
		// grant was made for. The refresh token carries it as a claim, its own
		// audience is always the client.
		if tr.Resource != "" && tr.Resource != claims.Resource {
			err = konnectoidc.NewOAuth2Error(konnectoidc.ErrorCodeOAuth2InvalidTarget, "resource mismatch")
			goto done
		}

		if len(tr.Scopes) > 0 {
			// Make sure all requested scopes are granted and limit authorized
			// scopes to the requested scopes.
			authorizedScopes = make(map[string]bool)
			for scope := range tr.Scopes {
				if !approvedScopes[scope] {
					err = konnectoidc.NewOAuth2Error(konnectoidc.ErrorCodeOAuth2InvalidScope, "requested scope exceeds the granted scope")
					goto done
				} else {
					authorizedScopes[scope] = true
				}
			}
		} else {
			// Authorize all approved scopes when no scopes are in request.
			authorizedScopes = approvedScopes
		}

		// Pre-generate the IDs of the tokens this exchange issues, so the
		// backend can be told about them with the lookup below. A refresh token
		// is only issued when offline access is authorized.
		newAccessTokenID = rndm.GenerateRandomString(24)
		if authorizedScopes[oidc.ScopeOfflineAccess] {
			newRefreshTokenID = rndm.GenerateRandomString(24)
		}

		// Load user record from identitymanager, without any scopes or claims.
		auth, found, err = currentIdentityManager.Fetch(ctx, userID, sessionRef, nil, nil, authorizedScopes, &identity.LookupContext{
			Audience:          tr.ClientID,
			RefreshTokenID:    claims.ID,
			NewAccessTokenID:  newAccessTokenID,
			NewRefreshTokenID: newRefreshTokenID,
		})
		if !found {
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidGrant, "user not found")
			goto done
		}
		if err != nil {
			goto done
		}
		// Add authorized scopes.
		auth.AuthorizeScopes(authorizedScopes)
		// Add authorized claims from request.
		auth.AuthorizeClaims(claims.ApprovedClaimsRequest)

		// Create fake request for token generation.
		ar = &payload.AuthenticationRequest{
			ClientID: claims.Audience[0],
			Resource: claims.Resource,
		}

		// Remember refresh token claims, for use in access and id token generators later on.
		refreshTokenClaims = claims

	default:
		err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2UnsupportedGrantType, "grant_type value not implemented")
		goto done
	}

	// Create access token.
	accessTokenString, err = p.makeAccessToken(ctx, ar.ClientID, ar.Resource, auth, signinMethod, refreshTokenClaims, newAccessTokenID)
	if err != nil {
		goto done
	}

	switch tr.GrantType {
	case oidc.GrantTypeAuthorizationCode, oidc.GrantTypeRefreshToken:
		// Create ID token when not previously requested amd openid scope is authorized.
		if !ar.ResponseTypes[oidc.ResponseTypeIDToken] && authorizedScopes[oidc.ScopeOpenID] {
			idTokenString, err = p.makeIDToken(ctx, ar, auth, session, accessTokenString, "", signinMethod, refreshTokenClaims)
			if err != nil {
				goto done
			}
		}

		// Create refresh token when granted. The new one carries the scopes of
		// the grant, which approvedScopes holds for a refresh and which the
		// record itself is for a fresh code: a request which asks for less than
		// it was granted narrows its tokens, not the grant behind them, as
		// lico keeps no record of a grant besides its refresh token.
		if authorizedScopes[oidc.ScopeOfflineAccess] {
			refreshTokenString, err = p.makeRefreshToken(ctx, ar.ClientID, ar.Resource, auth, approvedScopes, nil, newRefreshTokenID)
			if err != nil {
				goto done
			}
		}
	}

done:
	if err != nil {
		switch err.(type) {
		case *konnectoidc.OAuth2Error:
			status := http.StatusBadRequest
			if tr != nil && tr.ClientAuthenticationBasic && konnectoidc.IsErrorWithID(err, konnectoidc.ErrorCodeOAuth2InvalidClient) {
				// A client which authenticated with the Authorization header
				// is answered with 401 and a challenge of the same scheme, as
				// https://datatracker.ietf.org/doc/html/rfc6749#section-5.2
				// requires.
				status = http.StatusUnauthorized
				rw.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", p.issuerIdentifier))
			}
			err = utils.WriteJSON(rw, status, err, "")
			if err != nil {
				p.logger.WithError(err).Errorln("token request failed writing response")
				return
			}
		default:
			p.logger.WithFields(utils.ErrorAsFields(err)).Errorln("token request failed")
			p.ErrorPage(rw, http.StatusInternalServerError, err.Error(), "well sorry, but there was a problem")
		}

		return
	}

	// Successful Token Response
	// http://openid.net/specs/openid-connect-core-1_0.html#TokenResponse
	response := &payload.TokenSuccess{}
	if accessTokenString != "" {
		response.AccessToken = accessTokenString
		response.TokenType = oidc.TokenTypeBearer
		response.ExpiresIn = int64(p.accessTokenDuration.Seconds())
		// Report the granted scope, which RFC 6749 section 5.1 requires
		// whenever it differs from the request. A client which cannot see what
		// it got has to guess whenever a scope was dropped, and the request
		// this answers may not have carried a scope at all.
		response.Scope = strings.Join(makeArrayFromBoolMap(authorizedScopes), " ")
	}
	if idTokenString != "" {
		response.IDToken = idTokenString
	}
	if refreshTokenString != "" {
		response.RefreshToken = refreshTokenString
	}

	err = utils.WriteJSON(rw, http.StatusOK, response, "")
	if err != nil {
		p.logger.WithError(err).Errorln("token request failed writing response")
	}
}

// UserInfoHandler implements the HTTP userinfo endpoint for OpenID
// Connect 1.0 as specified at https://openid.net/specs/openid-connect-core-1_0.html#UserInfo
func (p *Provider) UserInfoHandler(rw http.ResponseWriter, req *http.Request) {
	var err error
	addResponseHeaders(rw.Header())

	switch req.Method {
	case http.MethodHead:
		fallthrough
	case http.MethodPost:
		fallthrough
	case http.MethodGet:
		// pass
	default:
		return
	}

	// Parse and validate UserInfo request
	// https://openid.net/specs/openid-connect-core-1_0.html#UserInfoRequest

	claims, err := p.GetAccessTokenClaimsFromRequest(req)
	if err != nil {
		p.logger.WithFields(utils.ErrorAsFields(err)).Debugln("userinfo request unauthorized")
		konnectoidc.WriteWWWAuthenticateError(rw, http.StatusUnauthorized, err)
		return
	}

	var auth identity.AuthRecord
	var found bool
	var requestedClaimsMap []*payload.ClaimsRequestMap
	var authorizedScopes map[string]bool

	// The client is azp for a token bound to a resource, whose audience is the
	// resource rather than the client.
	clientID := claims.AuthorizedPartyOrAudience()

	userID, sessionRef := p.getUserIDAndSessionRefFromClaims(&claims.RegisteredClaims, claims.AuthorizedParty, claims.SessionClaims, claims.IdentityClaims)

	ctx := konnect.NewClaimsContext(konnect.NewRequestContext(req.Context(), req), claims)

	currentIdentityManager, err := p.getIdentityManagerFromClaims(claims.IdentityProvider, claims.IdentityClaims)
	if err != nil {
		goto done
	}

	if userID == "" {
		err = fmt.Errorf("missing data in identity claim")
		goto done
	}

	if claims.AuthorizedClaimsRequest != nil && claims.AuthorizedClaimsRequest.UserInfo != nil {
		requestedClaimsMap = []*payload.ClaimsRequestMap{claims.AuthorizedClaimsRequest.UserInfo}
	}

	authorizedScopes = claims.AuthorizedScopes()

	auth, found, err = currentIdentityManager.Fetch(ctx, userID, sessionRef, authorizedScopes, requestedClaimsMap, authorizedScopes, &identity.LookupContext{
		Audience: clientID,
	})
	if err != nil {
		p.logger.WithFields(utils.ErrorAsFields(err)).Errorln("identity manager fetch failed")
		found = false
	}
	if !found {
		p.logger.WithField("sub", claims.RegisteredClaims.Subject).Debugln("userinfo request user not found")
		p.ErrorPage(rw, http.StatusNotFound, "", "user not found")
		return
	}

done:
	if err != nil {
		p.logger.WithFields(utils.ErrorAsFields(err)).Debugln("userinfo request invalid token")
		konnectoidc.WriteWWWAuthenticateError(rw, http.StatusUnauthorized, konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidToken, err.Error()))
		return
	}

	publicSubject, err := p.PublicSubjectFromAuth(auth)
	if err != nil {
		p.logger.WithFields(utils.ErrorAsFields(err)).Debugln("userinfo request failed to create subject")
		p.ErrorPage(rw, http.StatusInternalServerError, "", err.Error())
		return
	}

	response := &konnect.UserInfoResponse{
		UserInfoResponse: &payload.UserInfoResponse{
			UserInfoClaims: konnectoidc.UserInfoClaims{
				Subject: publicSubject,
			},
			ProfileClaims: konnectoidc.NewProfileClaims(auth.Claims(oidc.ScopeProfile)[0]),
			EmailClaims:   konnectoidc.NewEmailClaims(auth.Claims(oidc.ScopeEmail)[0]),
		},
	}

	// Helper to receive user from auth, but only once.
	withUser := func() func() identity.User {
		var u identity.User
		var fetched bool
		return func() identity.User {
			if !fetched {
				fetched = true
				u = auth.User()
			}
			return u
		}
	}()
	authorizedScopes = auth.AuthorizedScopes()
	var user identity.User

	// Include additional Konnect specific claims when corresponding scopes are authorized.
	if ok, _ := authorizedScopes[konnect.ScopeNumericID]; ok {
		user = withUser()
		if userWithID, ok := user.(identity.UserWithID); ok {
			claims := &konnect.NumericIDClaims{
				NumericID: userWithID.ID(),
			}
			if userWithUsername, ok := user.(identity.UserWithUsername); ok {
				claims.NumericIDUsername = userWithUsername.Username()
			}
			if claims.NumericIDUsername == "" {
				claims.NumericIDUsername = user.Subject()
			}

			response.NumericIDClaims = claims
		}
	}
	if ok, _ := authorizedScopes[konnect.ScopeUniqueUserID]; ok {
		user = withUser()
		if userWithUniqueID, ok := user.(identity.UserWithUniqueID); ok {
			claims := &konnect.UniqueUserIDClaims{
				UniqueUserID: userWithUniqueID.UniqueID(),
			}

			response.UniqueUserIDClaims = claims
		}
	}

	// Create a map so additional user specific claims can be added.
	responseAsMap, err := payload.ToMap(response)
	if err != nil {
		p.logger.WithFields(utils.ErrorAsFields(err)).Debugln("userinfo request failed to encode claims")
		p.ErrorPage(rw, http.StatusInternalServerError, "", err.Error())
		return
	}

	// Inject extra claims.
	extraClaims := auth.Claims(konnect.InternalExtraAccessTokenClaimsClaim)[0]
	if extraClaims != nil {
		if extraClaimsMap, ok := extraClaims.(jwt.MapClaims); ok {
			for claim, value := range extraClaimsMap {
				responseAsMap[claim] = value
			}
		}
	}

	// Support returning signed user info if the registered client requested it
	// as specified in https://openid.net/specs/openid-connect-core-1_0.html#UserInfoResponse and
	// https://openid.net/specs/openid-connect-registration-1_0.html#ClientMetadata
	registration, _ := p.clients.Get(ctx, clientID)
	if registration != nil {
		if registration.RawUserInfoSignedResponseAlg != "" {
			// Get alg.
			alg := jwt.GetSigningMethod(registration.RawUserInfoSignedResponseAlg)
			// Set extra claims.
			responseAsMap[oidc.IssuerIdentifierClaim] = p.issuerIdentifier
			responseAsMap[oidc.AudienceClaim] = registration.ID
			tokenString, err := p.makeJWT(ctx, alg, jwt.MapClaims(responseAsMap))
			if err != nil {
				p.logger.WithFields(utils.ErrorAsFields(err)).Debugln("userinfo request failed to encode jwt")
				p.ErrorPage(rw, http.StatusInternalServerError, "", err.Error())
				return
			}

			rw.Header().Set("Content-Type", "application/jwt")
			rw.Write([]byte(tokenString))
			return
		}
	}

	err = utils.WriteJSON(rw, http.StatusOK, responseAsMap, "")
	if err != nil {
		p.logger.WithError(err).Errorln("userinfo request failed writing response")
	}
}

// EndSessionHandler implements the HTTP endpoint for RP initiated logout with
// OpenID Connect Session Management 1.0 as specified at
// https://openid.net/specs/openid-connect-session-1_0.html#RPLogout
func (p *Provider) EndSessionHandler(rw http.ResponseWriter, req *http.Request) {
	var err error
	var session *payload.Session
	var currentIdentityManager identity.Manager

	addResponseHeaders(rw.Header())

	ctx := konnect.NewRequestContext(req.Context(), req)

	// Validate request.
	err = req.ParseForm()
	if err != nil {
		p.logger.WithError(err).Errorln("endsession request invalid form data")
		p.ErrorPage(rw, http.StatusBadRequest, oidc.ErrorCodeOAuth2InvalidRequest, err.Error())
		return
	}

	esr, err := payload.DecodeEndSessionRequest(req, p.metadata)
	if err != nil {
		p.logger.WithError(err).Errorln("endsession request invalid request data")
		p.ErrorPage(rw, http.StatusBadRequest, oidc.ErrorCodeOAuth2InvalidRequest, err.Error())
		return
	}
	err = esr.Validate(func(token *jwt.Token) (interface{}, error) {
		// Validator for incoming IDToken hints, looks up key.
		return p.validateJWT(token)
	})
	if err != nil {
		goto done
	}

	// Get our session.
	session, err = p.getSession(req)
	if err != nil {
		goto done
	}

	currentIdentityManager, err = p.getIdentityManagerFromSession(session)
	if err != nil {
		goto done
	}

	// Authorization unauthenticates end user.
	err = currentIdentityManager.EndSession(ctx, rw, req, esr)
	if err != nil {
		goto done
	}

done:
	if err != nil {
		switch err.(type) {
		case *payload.AuthenticationBadRequest:
			p.ErrorPage(rw, http.StatusBadRequest, err.Error(), err.(*payload.AuthenticationBadRequest).Description())
		case *identity.RedirectError:
			p.Found(rw, err.(*identity.RedirectError).RedirectURI(), nil, false)
		case *identity.IsHandledError:
			// do nothing
		case *konnectoidc.OAuth2Error:
			err = esr.NewError(err.Error(), err.(*konnectoidc.OAuth2Error).Description())
			uri := esr.MakeRedirectEndSessionRequestURL()
			if uri == nil {
				p.ErrorPage(rw, http.StatusForbidden, err.Error(), "oauth2 error")
			} else {
				p.Found(rw, uri, err, false)
			}
		default:
			p.logger.WithFields(utils.ErrorAsFields(err)).Errorln("endsession request failed")
			p.ErrorPage(rw, http.StatusInternalServerError, err.Error(), "well sorry, but there was a problem")
		}

		return
	}

	// EndSession Response.
	response := &payload.AuthenticationSuccess{
		State: esr.State,
	}

	uri := esr.MakeRedirectEndSessionRequestURL()
	if uri == nil {
		err = utils.WriteJSON(rw, http.StatusOK, response, "")
		if err != nil {
			p.logger.WithError(err).Errorln("endsession request failed writing response")
		}
	} else {
		p.Found(rw, uri, nil, false)
	}
}

// CheckSessionIframeHandler implements the HTTP endpoint for OP iframe with
// OpenID Connect Session Management 1.0 as specified at
// https://openid.net/specs/openid-connect-session-1_0.html#OPiframe
func (p *Provider) CheckSessionIframeHandler(rw http.ResponseWriter, req *http.Request) {
	addResponseHeaders(rw.Header())

	nonce := rndm.GenerateRandomString(32)

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.Header().Set("X-XSS-Protection", "1; mode=block")
	rw.Header().Set("Content-Security-Policy", fmt.Sprintf("default-src 'none'; script-src 'nonce-%s'", nonce))

	data := struct {
		CookieName string
		Nonce      string
	}{
		CookieName: p.browserStateCookieName,
		Nonce:      nonce,
	}
	checkSessionIframeTemplate.Execute(rw, data)
}

// RegistrationHandler implements the HTTP endpoint for client self registration
// with OpenID Connect Registration 1.0 as specified at
// https://openid.net/specs/openid-connect-registration-1_0.html#ClientRegistration
func (p *Provider) RegistrationHandler(rw http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(rw, req.Body, registrationSizeLimit)
	addResponseHeaders(rw.Header())

	crr, err := payload.DecodeClientRegistrationRequest(req)
	if err != nil {
		p.logger.WithError(err).Errorln("client registration request failed to decode request data")

		p.ErrorPage(rw, http.StatusBadRequest, oidc.ErrorCodeOAuth2InvalidRequest, err.Error())
		return
	}

	var cr *clients.ClientRegistration

	// Validate request method
	switch req.Method {
	case http.MethodPost:
		// breaks
	default:
		err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidRequest, "request must be sent with POST")
		goto done
	}

	// Validate request.
	err = crr.Validate()
	if err != nil {
		goto done
	}

	// Get registration record.
	cr, err = crr.ClientRegistration()
	if err != nil {
		goto done
	}

	// Set client to dynamic. This creates the id and client secret.
	err = cr.SetDynamic(clients.NewRegistryContext(req.Context(), p.clients), p.clients.StatelessCreator)
	if err != nil {
		goto done
	}

done:
	if err != nil {
		switch err.(type) {
		case *konnectoidc.OAuth2Error:
			err = utils.WriteJSON(rw, http.StatusBadRequest, err, "")
			if err != nil {
				p.logger.WithError(err).Errorln("client registration request failed writing response")
				return
			}
		default:
			p.logger.WithFields(utils.ErrorAsFields(err)).Errorln("client registration request failed")
			p.ErrorPage(rw, http.StatusInternalServerError, err.Error(), "well sorry, but there was a problem")
		}

		return
	}

	p.logger.WithFields(logrus.Fields{
		"client_id":        cr.ID,
		"name":             cr.Name,
		"application_type": cr.ApplicationType,
		"redirect_uris":    cr.RedirectURIs,
	}).Debugln("registered dynamic client")

	// RFC 7591 section 3.2.1 states a client_secret which does not expire as 0,
	// where the zero time would be stated as the year one.
	var clientSecretExpiresAt int64
	if !cr.SecretExpiresAt.IsZero() {
		clientSecretExpiresAt = cr.SecretExpiresAt.Unix()
	}

	response := &payload.ClientRegistrationResponse{
		ClientID:     cr.ID,
		ClientSecret: cr.Secret,

		ClientIDIssuedAt:      cr.IDIssuedAt.Unix(),
		ClientSecretExpiresAt: clientSecretExpiresAt,

		ClientRegistrationRequest: *crr,
	}

	err = utils.WriteJSON(rw, http.StatusCreated, response, "")
	if err != nil {
		p.logger.WithError(err).Errorln("client registration request failed writing response")
	}
}
