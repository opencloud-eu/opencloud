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
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/libregraph/oidc-go"
	"github.com/longsleep/rndm"

	konnect "github.com/libregraph/lico"
	"github.com/libregraph/lico/identity"
	konnectoidc "github.com/libregraph/lico/oidc"
	"github.com/libregraph/lico/oidc/payload"
	"github.com/libregraph/lico/utils"
)

// MakeAccessToken implements the oidc.AccessTokenProvider interface.
func (p *Provider) MakeAccessToken(ctx context.Context, audience string, auth identity.AuthRecord) (string, error) {
	return p.makeAccessToken(ctx, audience, auth, nil, nil)
}

func (p *Provider) makeAccessToken(ctx context.Context, audience string, auth identity.AuthRecord, signingMethod jwt.SigningMethod, refreshTokenClaims *konnect.RefreshTokenClaims) (string, error) {
	sk, ok := p.getSigningKey(signingMethod)
	if !ok {
		return "", fmt.Errorf("no signing key")
	}

	authorizedScopes := auth.AuthorizedScopes()
	authorizedScopesList := payload.ScopesValue(makeArrayFromBoolMap(authorizedScopes))

	accessTokenClaims := konnect.AccessTokenClaims{
		TokenType:               konnect.TokenTypeAccessToken,
		AuthorizedScopesList:    authorizedScopesList,
		AuthorizedClaimsRequest: auth.AuthorizedClaims(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    p.issuerIdentifier,
			Subject:   auth.Subject(),
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(p.accessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        rndm.GenerateRandomString(24),
		},
	}

	user := auth.User()
	if user != nil {
		if userWithClaims, ok := user.(identity.UserWithClaims); ok {
			accessTokenClaims.IdentityClaims = userWithClaims.Claims()
		}
		accessTokenClaims.IdentityProvider = auth.Manager().Name()
		if accessTokenClaims.IdentityClaims != nil && refreshTokenClaims != nil && refreshTokenClaims.IdentityClaims != nil {
			if refreshTokenClaims.IdentityProvider != accessTokenClaims.IdentityProvider {
				return "", fmt.Errorf("refresh token claims provider mismatch")
			}
			for k, v := range refreshTokenClaims.IdentityClaims {
				// Force to use refresh token identity claim values. This also locks all
				// the extra claims for id and access tokens to the ones provided from
				// the refresh token claims (which currently includes the session id).
				accessTokenClaims.IdentityClaims[k] = v
			}
		}
	}

	// Support additional custom user specific claims.
	var finalAccessTokenClaims jwt.Claims = accessTokenClaims
	if accessTokenClaims.IdentityClaims != nil {
		accessTokenClaimsMap, err := payload.ToMap(accessTokenClaims)
		if err != nil {
			return "", err
		}

		delete(accessTokenClaimsMap[konnect.IdentityClaim].(map[string]interface{}), konnect.InternalExtraIDTokenClaimsClaim)
		delete(accessTokenClaimsMap[konnect.IdentityClaim].(map[string]interface{}), konnect.InternalExtraAccessTokenClaimsClaim)

		// Look for special internal key, if its a map all claims in there are
		// elevated to top level.
		extraClaimsMap, _ := accessTokenClaims.IdentityClaims[konnect.InternalExtraAccessTokenClaimsClaim].(map[string]interface{})
		if extraClaimsMap != nil {
			// Inject extra claims.
			for claim, value := range extraClaimsMap {
				switch claim {
				case konnect.ScopesClaim:
					// Support to extend the scopes.
					extraScopesList, _ := value.(string)
					if extraScopesList != "" {
						authorizedScopesList = append(authorizedScopesList, extraScopesList)
						value = authorizedScopesList
					}
				default:
					if _, ok := accessTokenClaimsMap[claim]; ok {
						// Prevent override of existing claims, only allow new claims.
						continue
					}
				}
				accessTokenClaimsMap[claim] = value
			}
		}

		finalAccessTokenClaims = jwt.MapClaims(accessTokenClaimsMap)
	}

	accessToken := jwt.NewWithClaims(sk.SigningMethod, finalAccessTokenClaims)
	accessToken.Header[oidc.JWTHeaderKeyID] = sk.ID

	return accessToken.SignedString(sk.PrivateKey)
}

func (p *Provider) makeIDToken(ctx context.Context, ar *payload.AuthenticationRequest, auth identity.AuthRecord, session *payload.Session, accessTokenString string, codeString string, signingMethod jwt.SigningMethod, refreshTokenClaims *konnect.RefreshTokenClaims) (string, error) {
	sk, ok := p.getSigningKey(signingMethod)
	if !ok {
		return "", fmt.Errorf("no signing key")
	}

	publicSubject, err := p.PublicSubjectFromAuth(auth)
	if err != nil {
		return "", err
	}

	idTokenClaims := &konnectoidc.IDTokenClaims{
		Nonce: ar.Nonce,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    p.issuerIdentifier,
			Subject:   publicSubject,
			Audience:  jwt.ClaimStrings{ar.ClientID},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(p.idTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessTokenClaims := konnect.AccessTokenClaims{}

	if session != nil {
		// Include session data in ID token.
		idTokenClaims.SessionClaims = &konnectoidc.SessionClaims{
			SessionID: session.ID,
		}
	}

	// Include requested scope data in ID token when no access token is
	// generated.
	authorizedClaimsRequest := auth.AuthorizedClaims()

	withAccessToken := accessTokenString != ""
	withCode := codeString != ""
	withAuthTime := ar.MaxAge > 0
	withIDTokenClaimsRequest := authorizedClaimsRequest != nil && authorizedClaimsRequest.IDToken != nil

	user := auth.User()
	if user == nil {
		return "", fmt.Errorf("no user")
	}
	if userWithClaims, ok := user.(identity.UserWithClaims); ok {
		accessTokenClaims.IdentityClaims = userWithClaims.Claims()
	}
	accessTokenClaims.IdentityProvider = auth.Manager().Name()
	if accessTokenClaims.IdentityClaims != nil && refreshTokenClaims != nil && refreshTokenClaims.IdentityClaims != nil {
		if refreshTokenClaims.IdentityProvider != accessTokenClaims.IdentityProvider {
			return "", fmt.Errorf("refresh token claims provider mismatch")
		}
		for k, v := range refreshTokenClaims.IdentityClaims {
			// Force to use refresh token identity claim values. This also locks all
			// the extra claims for id and access tokens to the ones provided from
			// the refresh token claims (which currently includes the session id).
			accessTokenClaims.IdentityClaims[k] = v
		}
	}

	if withIDTokenClaimsRequest {
		// Apply additional information from ID token claims request.
		if _, ok := authorizedClaimsRequest.IDToken.Get(oidc.AuthTimeClaim); !withAuthTime && ok {
			// Return auth time claim if requested and not already requested by other means.
			withAuthTime = true
		}
	}

	if !withAccessToken || withIDTokenClaimsRequest {
		var userID string
		if accessTokenClaims.IdentityClaims != nil {
			if userIDString, ok := accessTokenClaims.IdentityClaims[konnect.IdentifiedUserIDClaim]; ok {
				userID = userIDString.(string)
			}
		}
		if userID == "" {
			return "", fmt.Errorf("no id claim in user identity claims")
		}

		var sessionRef *string
		if userWithSessionRef, ok := user.(identity.UserWithSessionRef); ok {
			sessionRef = userWithSessionRef.SessionRef()
		}

		var requestedClaimsMap []*payload.ClaimsRequestMap
		var requestedScopesMap map[string]bool
		if withIDTokenClaimsRequest {
			requestedClaimsMap = []*payload.ClaimsRequestMap{authorizedClaimsRequest.IDToken}
			requestedScopesMap = authorizedClaimsRequest.IDToken.ScopesMap(nil)
		}

		authorizedScopes := auth.AuthorizedScopes()

		freshAuth, found, fetchErr := auth.Manager().Fetch(ctx, userID, sessionRef, authorizedScopes, requestedClaimsMap, authorizedScopes)
		if fetchErr != nil {
			p.logger.WithFields(utils.ErrorAsFields(fetchErr)).Errorln("identity manager fetch failed")
			found = false
		}
		if !found {
			return "", fmt.Errorf("user not found")
		}

		if (!withAccessToken && ar.Scopes[oidc.ScopeProfile]) || requestedScopesMap[oidc.ScopeProfile] {
			idTokenClaims.ProfileClaims = konnectoidc.NewProfileClaims(freshAuth.Claims(oidc.ScopeProfile)[0])
		}
		if (!withAccessToken && ar.Scopes[oidc.ScopeEmail]) || requestedScopesMap[oidc.ScopeEmail] {
			idTokenClaims.EmailClaims = konnectoidc.NewEmailClaims(freshAuth.Claims(oidc.ScopeEmail)[0])
		}

		auth = freshAuth
	}
	if withAccessToken {
		// Add left-most hash of access token.
		// http://openid.net/specs/openid-connect-core-1_0.html#ImplicitIDToken
		hash, hashErr := oidc.HashFromSigningMethod(sk.SigningMethod.Alg())
		if hashErr != nil {
			return "", hashErr
		}

		idTokenClaims.AccessTokenHash = oidc.LeftmostHash([]byte(accessTokenString), hash).String()
	}
	if withCode {
		// Add left-most hash of code.
		// http://openid.net/specs/openid-connect-core-1_0.html#HybridIDToken
		hash, hashErr := oidc.HashFromSigningMethod(sk.SigningMethod.Alg())
		if hashErr != nil {
			return "", hashErr
		}

		idTokenClaims.CodeHash = oidc.LeftmostHash([]byte(codeString), hash).String()
	}
	if withAuthTime {
		// Add AuthTime.
		if loggedOn, logonAt := auth.LoggedOn(); loggedOn {
			idTokenClaims.AuthTime = logonAt.Unix()
		} else {
			// NOTE(longsleep): Return current time to be spec compliant.
			idTokenClaims.AuthTime = time.Now().Unix()
		}
	}

	// To support extra non-standard claims in ID token, convert claim set to
	// map.
	idTokenClaimsMap, err := payload.ToMap(idTokenClaims)
	if err != nil {
		return "", err
	}

	if accessTokenClaims.IdentityClaims != nil {
		// Inject available extra ID token claims.
		extraClaimsMap, _ := accessTokenClaims.IdentityClaims[konnect.InternalExtraIDTokenClaimsClaim].(map[string]interface{})
		if extraClaimsMap != nil {
			for claim, value := range extraClaimsMap {
				idTokenClaimsMap[claim] = value
			}
		}
	}

	if !withAccessToken && accessTokenClaims.IdentityClaims != nil {
		// Include requested scope data in ID token when no access token is
		// generated - additional custom user specific claims.

		// Inject extra claims.
		extraClaimsMap, _ := accessTokenClaims.IdentityClaims[konnect.InternalExtraAccessTokenClaimsClaim].(map[string]interface{})
		if extraClaimsMap != nil {
			for claim, value := range extraClaimsMap {
				idTokenClaimsMap[claim] = value
			}
		}
	}

	// Create signed token.
	idToken := jwt.NewWithClaims(sk.SigningMethod, jwt.MapClaims(idTokenClaimsMap))
	idToken.Header[oidc.JWTHeaderKeyID] = sk.ID

	return idToken.SignedString(sk.PrivateKey)
}

func (p *Provider) makeRefreshToken(ctx context.Context, audience string, auth identity.AuthRecord, signingMethod jwt.SigningMethod) (string, error) {
	sk, ok := p.getSigningKey(signingMethod)
	if !ok {
		return "", fmt.Errorf("no signing key")
	}

	approvedScopesList := []string{}
	approvedScopes := make(map[string]bool)
	for scope, granted := range auth.AuthorizedScopes() {
		if granted {
			approvedScopesList = append(approvedScopesList, scope)
			approvedScopes[scope] = true
		}
	}

	ref, err := auth.Manager().ApproveScopes(ctx, auth.Subject(), audience, approvedScopes)
	if err != nil {
		return "", err
	}

	refreshTokenClaims := &konnect.RefreshTokenClaims{
		TokenType:             konnect.TokenTypeRefreshToken,
		ApprovedScopesList:    approvedScopesList,
		ApprovedClaimsRequest: auth.AuthorizedClaims(),
		Ref:                   ref,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    p.issuerIdentifier,
			Subject:   auth.Subject(),
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(p.refreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        rndm.GenerateRandomString(24),
		},
	}

	user := auth.User()
	if user != nil {
		if userWithClaims, ok := user.(identity.UserWithClaims); ok {
			refreshTokenClaims.IdentityClaims = userWithClaims.Claims()
		}
		refreshTokenClaims.IdentityProvider = auth.Manager().Name()
	}

	refreshToken := jwt.NewWithClaims(sk.SigningMethod, refreshTokenClaims)
	refreshToken.Header[oidc.JWTHeaderKeyID] = sk.ID

	return refreshToken.SignedString(sk.PrivateKey)
}

func (p *Provider) makeJWT(ctx context.Context, signingMethod jwt.SigningMethod, claims jwt.Claims) (string, error) {
	sk, ok := p.getSigningKey(signingMethod)
	if !ok {
		return "", fmt.Errorf("no signing key")
	}

	token := jwt.NewWithClaims(sk.SigningMethod, claims)
	token.Header[oidc.JWTHeaderKeyID] = sk.ID

	return token.SignedString(sk.PrivateKey)
}

func (p *Provider) validateJWT(token *jwt.Token) (interface{}, error) {
	rawAlg, ok := token.Header[oidc.JWTHeaderAlg]
	if !ok {
		return nil, fmt.Errorf("No alg header")
	}
	alg, ok := rawAlg.(string)
	if !ok {
		return nil, fmt.Errorf("Invalid alg value")
	}
	switch jwt.GetSigningMethod(alg).(type) {
	case *jwt.SigningMethodRSA:
	case *jwt.SigningMethodECDSA:
	case *jwt.SigningMethodRSAPSS:
	default:
		return nil, fmt.Errorf("Unexpected alg value")
	}
	rawKid, ok := token.Header[oidc.JWTHeaderKeyID]
	if !ok {
		return nil, fmt.Errorf("No kid header")
	}
	kid, ok := rawKid.(string)
	if !ok {
		return nil, fmt.Errorf("Invalid kid value")
	}
	key, ok := p.getValidationKey(kid)
	if !ok {
		return nil, fmt.Errorf("Unknown kid")
	}
	return key, nil
}
