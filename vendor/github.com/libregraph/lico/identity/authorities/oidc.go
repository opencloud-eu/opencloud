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

package authorities

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/go-jose/go-jose/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/libregraph/oidc-go"
	"github.com/sirupsen/logrus"

	konnectoidc "github.com/libregraph/lico/oidc"
	"github.com/libregraph/lico/oidc/payload"
	"github.com/libregraph/lico/utils"
)

// Authority default values.
var (
	oidcAuthorityDefaultScopes              = []string{oidc.ScopeOpenID, oidc.ScopeProfile}
	oidcAuthorityDefaultResponseType        = oidc.ResponseTypeCode
	oidcAuthorityDefaultResponseMode        = oidc.ResponseModeQuery
	oidcAuthorityDefaultCodeChallengeMethod = oidc.S256CodeChallengeMethod
	oidcAuthorityDefaultIdentityClaimName   = oidc.PreferredUsernameClaim
)

type oidcAuthorityRegistration struct {
	registry *Registry
	data     *authorityRegistrationData

	discover              bool
	metadataEndpoint      *url.URL
	authorizationEndpoint *url.URL
	tokenEndpoint         *url.URL
	endSessionEndpoint    *url.URL
	userInfoEndpoint      *url.URL

	validationKeys map[string]crypto.PublicKey

	mutex sync.RWMutex
	ready bool

	wellKnown *oidc.WellKnown
}

func newOIDCAuthorityRegistration(registry *Registry, registrationData *authorityRegistrationData) (*oidcAuthorityRegistration, error) {
	ar := &oidcAuthorityRegistration{
		registry: registry,
		data:     registrationData,
	}

	if ar.data.RawMetadataEndpoint != "" {
		if u, err := url.Parse(ar.data.RawMetadataEndpoint); err == nil {
			ar.metadataEndpoint = u
		} else {
			return nil, fmt.Errorf("invalid metadata_endpoint value: %v", err)
		}
	}
	if ar.data.RawAuthorizationEndpoint != "" {
		if u, err := url.Parse(ar.data.RawAuthorizationEndpoint); err == nil {
			if u.Scheme != "https" {
				return nil, errors.New("authorization_endpoint must be https")
			}

			ar.authorizationEndpoint = u
		} else {
			return nil, fmt.Errorf("invalid authorization_endpoint value: %v", err)
		}
	}
	if ar.data.RawTokenEndpoint != "" {
		if u, err := url.Parse(ar.data.RawTokenEndpoint); err == nil {
			if u.Scheme != "https" {
				return nil, errors.New("token_endpoint must be https")
			}

			ar.tokenEndpoint = u
		} else {
			return nil, fmt.Errorf("invalid token_endpoint value: %v", err)
		}
	}
	if ar.data.UserInfoEndpoint != "" {
		if u, err := url.Parse(ar.data.UserInfoEndpoint); err == nil {
			if u.Scheme != "https" {
				return nil, errors.New("userinfo_endpoint must be https")
			}

			ar.userInfoEndpoint = u
		} else {
			return nil, fmt.Errorf("invalid userinfo_endpoint value: %v", err)
		}
	}
	if ar.data.JWKS != nil {
		if err := ar.setValidationKeysFromJWKS(ar.data.JWKS, false); err != nil {
			return nil, err
		}
	}
	if ar.data.Discover != nil {
		ar.discover = *ar.data.Discover
	}

	// Additional behavior.
	if ar.metadataEndpoint == nil && (ar.data.Discover == nil || ar.discover == true) {
		if ar.data.Iss == "" {
			return nil, fmt.Errorf("oidc authority iss is empty")
		}
		if issuer, err := url.Parse(ar.data.Iss); err == nil {
			relativeWellKnownURI, parseErr := url.Parse(strings.TrimRight(issuer.Path, "/") + "/.well-known/openid-configuration")
			if parseErr != nil {
				return nil, parseErr
			}
			ar.metadataEndpoint = issuer.ResolveReference(relativeWellKnownURI)
			ar.discover = true
		} else {
			return nil, fmt.Errorf("invalid iss value: %v", err)
		}
	}

	if !ar.discover {
		if ar.authorizationEndpoint == nil {
			return nil, errors.New("authorization_endpoint is empty")
		}
		if ar.data.JWKS == nil && !ar.data.Insecure {
			return nil, errors.New("jwks is empty")
		}
	}

	return ar, nil
}

func (ar *oidcAuthorityRegistration) ID() string {
	return ar.data.ID
}

func (ar *oidcAuthorityRegistration) Name() string {
	return ar.data.Name
}

func (ar *oidcAuthorityRegistration) AuthorityType() string {
	return ar.data.AuthorityType
}

func (ar *oidcAuthorityRegistration) Authority() *Details {
	details := &Details{
		ID:            ar.data.ID,
		Name:          ar.data.Name,
		AuthorityType: ar.data.AuthorityType,

		ClientID:     ar.data.ClientID,
		ClientSecret: ar.data.ClientSecret,

		Trusted:  ar.data.Trusted,
		Insecure: ar.data.Insecure,

		Scopes:              ar.data.Scopes,
		ResponseType:        ar.data.ResponseType,
		ResponseMode:        ar.data.ResponseMode,
		CodeChallengeMethod: ar.data.CodeChallengeMethod,

		EndSessionEnabled: ar.data.EndSessionEnabled,

		registration: ar,
	}

	ar.mutex.RLock()
	details.ready = ar.ready
	if ar.ready {
		details.validationKeys = ar.validationKeys
	}
	ar.mutex.RUnlock()

	return details
}

func (ar *oidcAuthorityRegistration) Issuer() string {
	return ar.data.Iss
}

func (ar *oidcAuthorityRegistration) setValidationKeysFromJWKS(jwks *jose.JSONWebKeySet, skipInvalid bool) error {
	if jwks == nil || len(jwks.Keys) == 0 {
		ar.validationKeys = nil
		return nil
	}

	ar.validationKeys = make(map[string]crypto.PublicKey)
	skipped := 0
	for _, jwk := range jwks.Keys {
		if jwk.Use == "sig" {
			if key, ok := jwk.Key.(crypto.PublicKey); ok {
				ar.validationKeys[jwk.KeyID] = key
			} else {
				if !skipInvalid {
					return fmt.Errorf("failed to decode public key")
				} else {
					skipped++
				}
			}
		}
	}
	if skipped > 0 {
		return fmt.Errorf("failed to decode %d keys in set", skipped)
	}

	return nil
}

func (ar *oidcAuthorityRegistration) Validate() error {
	if ar.data.ClientID == "" {
		return errors.New("invalid authority client_id")
	}

	// Ensure some defaults.
	if len(ar.data.Scopes) == 0 {
		ar.data.Scopes = oidcAuthorityDefaultScopes
	}
	if ar.data.ResponseType == "" {
		ar.data.ResponseType = oidcAuthorityDefaultResponseType
	}
	if ar.data.ResponseMode == "" {
		ar.data.ResponseMode = oidcAuthorityDefaultResponseMode
	}
	if ar.data.CodeChallengeMethod == "" {
		ar.data.CodeChallengeMethod = oidcAuthorityDefaultCodeChallengeMethod
	}
	if ar.data.IdentityClaimName == "" {
		ar.data.IdentityClaimName = oidcAuthorityDefaultIdentityClaimName
	}

	return nil
}

func (ar *oidcAuthorityRegistration) Initialize(ctx context.Context, registry *Registry) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if ar.authorizationEndpoint != nil && ar.validationKeys != nil {
		ar.ready = true
	}
	if ar.metadataEndpoint == nil {
		return fmt.Errorf("no metadata_endpoint set")
	}

	return initializeOIDC(ctx, registry.logger, ar)
}

func (ar *oidcAuthorityRegistration) IdentityClaimValue(rawToken interface{}) (string, map[string]interface{}, error) {
	idToken, _ := rawToken.(*jwt.Token)
	if idToken == nil {
		return "", nil, errors.New("invalid ID token data")
	}
	claims, _ := idToken.Claims.(jwt.MapClaims)
	if claims == nil {
		return "", nil, errors.New("invalid claims data")
	}

	icn := ar.data.IdentityClaimName
	if icn == "" {
		icn = oidc.PreferredUsernameClaim
	}

	cvr, ok := claims[icn]
	if !ok {
		return "", nil, errors.New("identity claim not found")
	}
	cvs, ok := cvr.(string)
	if !ok {
		return "", nil, errors.New("identify claim has invalid type")
	}

	// Add extra external authority claims, for example SessionIndex.
	extra := make(map[string]interface{})
	extra["RawIDToken"] = idToken.Raw

	// Convert claim value.
	whitelisted := false
	if ar.data.IdentityAliases != nil {
		if alias, ok := ar.data.IdentityAliases[cvs]; ok && alias != "" {
			cvs = alias
			whitelisted = true
		}
	}

	// Check whitelist.
	if ar.data.IdentityAliasRequired && !whitelisted {
		return "", nil, errors.New("identity claim has no alias")
	}

	return cvs, extra, nil
}

func (ar *oidcAuthorityRegistration) MakeRedirectAuthenticationRequestURL(state string) (*url.URL, map[string]interface{}, error) {
	ar.mutex.RLock()
	defer ar.mutex.RUnlock()

	if !ar.ready {
		return nil, nil, errors.New("not ready")
	}

	uri, _ := url.Parse(ar.authorizationEndpoint.String())
	query := make(url.Values)
	query.Add("state", state)
	uri.RawQuery = query.Encode()

	return uri, nil, nil
}

func (ar *oidcAuthorityRegistration) MakeRedirectEndSessionRequestURL(ref interface{}, state string) (*url.URL, map[string]interface{}, error) {
	ar.mutex.RLock()
	defer ar.mutex.RUnlock()

	if !ar.ready {
		return nil, nil, errors.New("not ready")
	}

	logonRef := ref.(*string)
	if logonRef == nil {
		// Do nothing when we cannot provide id token hint.
		return nil, nil, nil
	}

	uri, _ := url.Parse(ar.endSessionEndpoint.String())
	query := make(url.Values)
	query.Add("state", state)
	query.Add("id_token_hint", *logonRef)
	uri.RawQuery = query.Encode()

	return uri, nil, nil
}

func (ar *oidcAuthorityRegistration) MakeRedirectEndSessionResponseURL(req interface{}, state string) (*url.URL, map[string]interface{}, error) {
	return nil, nil, fmt.Errorf("idp end session not implemented")
}

func (ar *oidcAuthorityRegistration) ParseStateResponse(req *http.Request, state string, extra map[string]interface{}) (interface{}, error) {
	if authenticationErrorID := req.Form.Get("error"); authenticationErrorID != "" {
		// Incoming error case.
		return nil, konnectoidc.NewOAuth2Error(authenticationErrorID, req.Form.Get("error_description"))
	}

	// Success case.
	authenticationSuccess := &payload.AuthenticationSuccess{}
	err := utils.DecodeURLSchema(authenticationSuccess, req.Form)
	if err != nil {
		return nil, fmt.Errorf("failed to parse oidc state response: %w", err)
	}
	return authenticationSuccess, nil
}

func (ar *oidcAuthorityRegistration) ValidateIdpEndSessionRequest(req interface{}, state string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (ar *oidcAuthorityRegistration) ValidateIdpEndSessionResponse(res interface{}, state string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (ar *oidcAuthorityRegistration) Metadata() AuthorityMetadata {
	ar.mutex.RLock()
	defer ar.mutex.RUnlock()

	return &oidc.WellKnown{
		Issuer:                ar.data.Iss,
		AuthorizationEndpoint: ar.authorizationEndpoint.String(),
		TokenEndpoint:         ar.tokenEndpoint.String(),
		UserInfoEndpoint:      ar.userInfoEndpoint.String(),
		EndSessionEndpoint:    ar.endSessionEndpoint.String(),
	}
}

type oidcProviderLogger struct {
	logger logrus.FieldLogger
}

func (logger *oidcProviderLogger) Printf(format string, args ...interface{}) {
	logger.logger.Debugf(format, args...)
}

func initializeOIDC(ctx context.Context, logger logrus.FieldLogger, ar *oidcAuthorityRegistration) error {
	providerLogger := logger.WithFields(logrus.Fields{
		"id":   ar.data.ID,
		"type": AuthorityTypeOIDC,
	})
	config := &oidc.ProviderConfig{
		Logger:     &oidcProviderLogger{providerLogger},
		HTTPHeader: http.Header{},
	}
	if ar.data.Insecure {
		config.HTTPClient = utils.InsecureHTTPClient
	} else {
		config.HTTPClient = utils.DefaultHTTPClient
	}
	config.HTTPHeader.Set("User-Agent", utils.DefaultHTTPUserAgent)

	issuer, err := url.Parse(ar.data.Iss)
	if err != nil {
		return fmt.Errorf("failed to parse issuer: %v", err)
	}
	if issuer.Scheme != "https" {
		return fmt.Errorf("issuer scheme is not https")
	}
	if issuer.Host == "" {
		return fmt.Errorf("issuer host is empty")
	}
	provider, err := oidc.NewProvider(issuer, config)
	if err != nil {
		return fmt.Errorf("failed to create oidc provider: %v", err)
	}
	updateCh := make(chan *oidc.ProviderDefinition)
	errorCh := make(chan error)
	err = provider.Initialize(ctx, updateCh, errorCh)
	if err != nil {
		return fmt.Errorf("failed to initialize oidc provider: %v", err)
	}
	go func() {
		// Handle updates and errors of authority meta data.
		var pd *oidc.ProviderDefinition
		var jwks *jose.JSONWebKeySet
		for {
			pd = nil

			select {
			case <-ctx.Done():
				return
			case update := <-updateCh:
				pd = update
			case chErr := <-errorCh:
				providerLogger.Errorf("error while oidc provider update: %v", chErr)
			}

			if pd != nil {
				ar.mutex.Lock()

				if pd.WellKnown != nil && pd.WellKnown.AuthorizationEndpoint != "" {
					if ar.authorizationEndpoint, err = url.Parse(pd.WellKnown.AuthorizationEndpoint); err != nil {
						providerLogger.WithError(err).Errorln("failed to parse oidc provider discover document authorization_endpoint")
					}
				}

				if pd.WellKnown != nil && pd.WellKnown.EndSessionEndpoint != "" {
					if ar.endSessionEndpoint, err = url.Parse(pd.WellKnown.EndSessionEndpoint); err != nil {
						providerLogger.WithError(err).Errorln("failed to parse oidc provider discover document endsession_endpoint")
					}
				}

				if pd.WellKnown != nil && pd.WellKnown.TokenEndpoint != "" {
					if ar.tokenEndpoint, err = url.Parse(pd.WellKnown.TokenEndpoint); err != nil {
						providerLogger.WithError(err).Errorln("failed to parse oidc provider discover document token_endpoint")
					}
				}

				if pd.WellKnown != nil && pd.WellKnown.UserInfoEndpoint != "" {
					if ar.userInfoEndpoint, err = url.Parse(pd.WellKnown.UserInfoEndpoint); err != nil {
						providerLogger.WithError(err).Errorln("failed to parse oidc provider discover document userinfo_endpoint")
					}
				}

				if pd.JWKS != jwks {
					if err := ar.setValidationKeysFromJWKS(pd.JWKS, true); err != nil {
						providerLogger.Errorf("failed to set authority keys from oidc provider jwks: %v", err)
					}
				}

				if pd.WellKnown != nil {
					ar.wellKnown = pd.WellKnown
				}

				ready := ar.ready
				if ar.authorizationEndpoint != nil && ar.validationKeys != nil {
					ar.ready = true
				} else {
					ar.ready = false
				}
				if ready != ar.ready {
					if ar.ready {
						providerLogger.Infoln("authority is now ready")
					} else {
						providerLogger.Warnln("authority is no longer ready")
					}
				} else if !ar.ready {
					providerLogger.Warnln("authority not ready")
				}

				ar.mutex.Unlock()
			}
		}
	}()

	return nil
}
