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

package provider

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/libregraph/oidc-go"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ed25519"

	konnect "github.com/libregraph/lico"
	"github.com/libregraph/lico/identity"
	"github.com/libregraph/lico/identity/clients"
	identityManagers "github.com/libregraph/lico/identity/managers"
	"github.com/libregraph/lico/managers"
	konnectoidc "github.com/libregraph/lico/oidc"
	"github.com/libregraph/lico/oidc/code"
	"github.com/libregraph/lico/utils"
)

// Provider defines an OIDC provider with the handlers for the OIDC endpoints.
type Provider struct {
	Config *Config

	issuerIdentifier string
	metadata         *oidc.WellKnown

	wellKnownPath          string
	jwksPath               string
	authorizationPath      string
	tokenPath              string
	userInfoPath           string
	endSessionPath         string
	checkSessionIframePath string
	registrationPath       string

	identityManager   identity.Manager
	guestManager      identity.Manager
	codeManager       code.Manager
	encryptionManager *identityManagers.EncryptionManager
	clients           *clients.Registry

	signingKeys          map[jwt.SigningMethod]*SigningKey
	signingMethodDefault jwt.SigningMethod
	validationKeys       map[string]crypto.PublicKey
	certificates         map[string][]*x509.Certificate

	browserStateCookiePath     string
	browserStateCookieName     string
	browserStateCookieSameSite http.SameSite

	sessionCookiePath     string
	sessionCookieName     string
	sessionCookieSameSite http.SameSite

	accessTokenDuration  time.Duration
	idTokenDuration      time.Duration
	refreshTokenDuration time.Duration

	logger logrus.FieldLogger
}

// NewProvider returns a new Provider.
func NewProvider(c *Config) (*Provider, error) {
	p := &Provider{
		Config: c,

		issuerIdentifier:       c.IssuerIdentifier,
		wellKnownPath:          c.WellKnownPath,
		jwksPath:               c.JwksPath,
		authorizationPath:      c.AuthorizationPath,
		tokenPath:              c.TokenPath,
		userInfoPath:           c.UserInfoPath,
		endSessionPath:         c.EndSessionPath,
		checkSessionIframePath: c.CheckSessionIframePath,
		registrationPath:       c.RegistrationPath,

		signingKeys:    make(map[jwt.SigningMethod]*SigningKey),
		validationKeys: make(map[string]crypto.PublicKey),
		certificates:   make(map[string][]*x509.Certificate),

		browserStateCookiePath:     c.BrowserStateCookiePath,
		browserStateCookieName:     c.BrowserStateCookieName,
		browserStateCookieSameSite: c.BrowserStateCookieSameSite,

		sessionCookiePath:     c.SessionCookiePath,
		sessionCookieName:     c.SessionCookieName,
		sessionCookieSameSite: c.SessionCookieSameSite,

		accessTokenDuration:  c.AccessTokenDuration,
		idTokenDuration:      c.IDTokenDuration,
		refreshTokenDuration: c.RefreshTokenDuration,

		logger: c.Config.Logger,
	}

	return p, nil
}

// RegisterManagers registers the provided managers from the
func (p *Provider) RegisterManagers(mgrs *managers.Managers) error {
	p.identityManager = mgrs.Must("identity").(identity.Manager)
	p.codeManager = mgrs.Must("code").(code.Manager)
	p.encryptionManager = mgrs.Must("encryption").(*identityManagers.EncryptionManager)
	p.clients = mgrs.Must("clients").(*clients.Registry)

	// Register callback to cleanup our cookie whenever the identity is unset or
	// set.
	onSetLogon := func(ctx context.Context, rw http.ResponseWriter, user identity.User) error {
		// NOTE(longsleep): This leaves room for optionmization. In theory it
		// should be possible to set the new browser state here directly since
		// the relevant information should be available in the identity manager
		// and thus avoiding a potentially unset refresh and client side
		// redirects whenever the same user signs in again.
		return p.removeBrowserStateCookie(rw)
	}
	onUnsetLogon := func(ctx context.Context, rw http.ResponseWriter) error {
		var err error

		// Remove browser state cookie.
		if errBsc := p.removeBrowserStateCookie(rw); errBsc != nil {
			err = errBsc
		}
		// Remove OIDC client session cookie.
		if errSc := p.removeSessionCookie(rw); errSc != nil {
			err = errSc
		}

		return err
	}
	p.identityManager.OnSetLogon(onSetLogon)
	p.identityManager.OnUnsetLogon(onUnsetLogon)

	// Add guest manager if any can be found.
	if guestManager, _ := mgrs.Get("guest"); guestManager != nil {
		p.guestManager = guestManager.(identity.Manager)
		p.guestManager.OnSetLogon(onSetLogon)
		p.guestManager.OnUnsetLogon(onUnsetLogon)
	}

	if p.Config.RegistrationPath != "" {
		// NOTE(longsleep): This is hackish. Find a better way to propagate our
		// provides JWT stuff to the client registry.
		p.clients.StatelessCreator = p.makeJWT
		p.clients.StatelessValidator = p.validateJWT
	}

	return nil
}

func (p *Provider) makeIssURL(path string) string {
	if path == "" {
		return ""
	}
	u, _ := url.Parse(p.issuerIdentifier)
	u.Path = "" // Strip path from issuer, whatever prefix must already be applied.

	return fmt.Sprintf("%s%s", u.String(), path)
}

// SetSigningMethod sets the provided signing method as default signing method
// of the associated provider.
func (p *Provider) SetSigningMethod(signingMethod jwt.SigningMethod) error {
	p.logger.WithField("alg", signingMethod.Alg()).Infoln("set provider signing alg")
	p.signingMethodDefault = signingMethod

	return nil
}

// SetSigningKey sets the provided signer as key for token signing with the
// provided id as key id. The public key of the provided signer is also added
// as validation key with the same key id.
func (p *Provider) SetSigningKey(id string, key crypto.Signer) error {
	var signingMethod jwt.SigningMethod

	// Auto select signingMethod based on the signer.
	switch s := key.(type) {
	case *rsa.PrivateKey:
		signingMethod = jwt.SigningMethodPS256
	case *ecdsa.PrivateKey:
		signingMethod = jwt.SigningMethodES256
	case ed25519.PrivateKey:
		signingMethod = jwt.SigningMethodEdDSA
	default:
		return fmt.Errorf("unsupported signer type: %v", s)
	}

	if p.signingMethodDefault == nil {
		if err := p.SetSigningMethod(signingMethod); err != nil {
			return err
		}
	}

	p.logger.WithFields(logrus.Fields{
		"type":   fmt.Sprintf("%T", key),
		"id":     id,
		"method": fmt.Sprintf("%T", signingMethod),
	}).Infoln("set provider signing key")

	switch signingMethod.(type) {
	case *jwt.SigningMethodECDSA:
		// Add all other supported ECDSA signing methods as well.
		p.signingKeys[jwt.SigningMethodES256] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodES256,
		}
		p.signingKeys[jwt.SigningMethodES384] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodES384,
		}
		p.signingKeys[jwt.SigningMethodES512] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodES512,
		}
	case *jwt.SigningMethodRSA:
		// Add all supported RSA and RSAPSS signing methods as well.
		p.signingKeys[jwt.SigningMethodRS256] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodRS256,
		}
		p.signingKeys[jwt.SigningMethodRS384] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodRS384,
		}
		p.signingKeys[jwt.SigningMethodRS512] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodRS512,
		}
		p.signingKeys[jwt.SigningMethodPS256] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodPS256,
		}
		p.signingKeys[jwt.SigningMethodPS384] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodPS384,
		}
		p.signingKeys[jwt.SigningMethodPS512] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodPS512,
		}
	case *jwt.SigningMethodRSAPSS:
		// Add all supported RSA and RSAPSS signing methods as well.
		p.signingKeys[jwt.SigningMethodRS256] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodRS256,
		}
		p.signingKeys[jwt.SigningMethodRS384] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodRS384,
		}
		p.signingKeys[jwt.SigningMethodRS512] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodRS512,
		}
		p.signingKeys[jwt.SigningMethodPS256] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodPS256,
		}
		p.signingKeys[jwt.SigningMethodPS384] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodPS384,
		}
		p.signingKeys[jwt.SigningMethodPS512] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: jwt.SigningMethodPS512,
		}
	case *jwt.SigningMethodEd25519:
		p.signingKeys[signingMethod] = &SigningKey{
			ID:            id,
			PrivateKey:    key,
			SigningMethod: signingMethod,
		}
	default:
		return fmt.Errorf("unsupported signing method type")
	}

	if _, ok := p.signingKeys[signingMethod]; !ok {
		return fmt.Errorf("unsupported signing method")
	}

	p.SetValidationKey(id, key.Public())

	return nil
}

// GetSigningKey returns a matching signing key for the provided signing method.
func (p *Provider) GetSigningKey(signingMethod jwt.SigningMethod) (*SigningKey, bool) {
	return p.getSigningKey(signingMethod)
}

func (p *Provider) getSigningKey(signingMethod jwt.SigningMethod) (*SigningKey, bool) {
	if signingMethod == nil {
		// Use default signign method if none given.
		signingMethod = p.signingMethodDefault
	}

	sk, ok := p.signingKeys[signingMethod]
	return sk, ok
}

// SetValidationKey sets the provider public key as validation key for token
// validation for tokens with the provided key.
func (p *Provider) SetValidationKey(id string, key crypto.PublicKey) error {
	p.logger.WithFields(logrus.Fields{
		"type": fmt.Sprintf("%T", key),
		"id":   id,
	}).Infoln("set provider validation key")

	p.validationKeys[id] = key

	return nil
}

// GetValidationKey returns the validation key for the provided id.
func (p *Provider) GetValidationKey(id string) (crypto.PublicKey, bool) {
	return p.getValidationKey(id)
}

func (p *Provider) getValidationKey(id string) (crypto.PublicKey, bool) {
	vk, ok := p.validationKeys[id]
	return vk, ok
}

// SetCertificate sets the provider certificate chain for a particular key.
func (p *Provider) SetCertificate(id string, certificates []*x509.Certificate) error {
	p.logger.WithFields(logrus.Fields{
		"chain_size": len(certificates),
		"id":         id,
	}).Infoln("set provider certificate")

	p.certificates[id] = certificates

	return nil
}

// InitializeMetadata creates the accociated providers meta data document. Call
// this once all other settings at the provider have been done.
func (p *Provider) InitializeMetadata() error {
	// Create well-known document.
	p.metadata = &oidc.WellKnown{
		Issuer:                p.issuerIdentifier,
		AuthorizationEndpoint: p.makeIssURL(p.authorizationPath),
		TokenEndpoint:         p.makeIssURL(p.tokenPath),
		UserInfoEndpoint:      p.makeIssURL(p.userInfoPath),
		EndSessionEndpoint:    p.makeIssURL(p.endSessionPath),
		CheckSessionIframe:    p.makeIssURL(p.checkSessionIframePath),
		JwksURI:               p.makeIssURL(p.jwksPath),
		RegistrationEndpoint:  p.makeIssURL(p.registrationPath),
		ScopesSupported: uniqueStrings(append([]string{
			oidc.ScopeOpenID,
		}, p.identityManager.ScopesSupported(nil)...)),
		ResponseTypesSupported: []string{
			oidc.ResponseTypeIDTokenToken,
			oidc.ResponseTypeIDToken,
			oidc.ResponseTypeCodeIDToken,
			oidc.ResponseTypeCodeIDTokenToken,
		},
		SubjectTypesSupported: []string{
			oidc.SubjectIDPublic,
		},
		ClaimsParameterSupported: true,
		ClaimsSupported: uniqueStrings(append([]string{
			oidc.IssuerIdentifierClaim,
			oidc.SubjectIdentifierClaim,
			oidc.AudienceClaim,
			oidc.ExpirationClaim,
			oidc.IssuedAtClaim,
		}, p.identityManager.ClaimsSupported(nil)...)),
		RequestParameterSupported:    true,
		RequestURIParameterSupported: false,
	}

	p.metadata.IDTokenSigningAlgValuesSupported = make([]string, 0)
	for alg := range p.signingKeys {
		p.metadata.IDTokenSigningAlgValuesSupported = append(p.metadata.IDTokenSigningAlgValuesSupported, alg.Alg())
	}
	p.metadata.UserInfoSigningAlgValuesSupported = p.metadata.IDTokenSigningAlgValuesSupported
	p.metadata.RequestObjectSigningAlgValuesSupported = []string{
		jwt.SigningMethodES256.Alg(),
		jwt.SigningMethodES384.Alg(),
		jwt.SigningMethodES512.Alg(),
		jwt.SigningMethodRS256.Alg(),
		jwt.SigningMethodRS384.Alg(),
		jwt.SigningMethodRS512.Alg(),
		jwt.SigningMethodPS256.Alg(),
		jwt.SigningMethodPS384.Alg(),
		jwt.SigningMethodPS512.Alg(),
		jwt.SigningMethodNone.Alg(),
		jwt.SigningMethodEdDSA.Alg(),
	}
	p.metadata.TokenEndpointAuthMethodsSupported = []string{
		oidc.AuthMethodClientSecretBasic,
		oidc.AuthMethodNone,
	}
	p.metadata.TokenEndpointAuthSigningAlgValuesSupported = p.metadata.IDTokenSigningAlgValuesSupported

	return nil
}

// ServerHTTP implements the http.HandlerFunc interface.
func (p *Provider) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	switch path := req.URL.Path; {
	case path == p.wellKnownPath:
		cors.Default().ServeHTTP(rw, req, p.WellKnownHandler)
	case path == p.jwksPath:
		cors.Default().ServeHTTP(rw, req, p.JwksHandler)
	case path == p.authorizationPath:
		p.AuthorizeHandler(rw, req)
	case path == p.tokenPath:
		cors.Default().ServeHTTP(rw, req, p.TokenHandler)
	case path == p.userInfoPath:
		// TODO(longsleep): Use more strict CORS.
		cors.AllowAll().ServeHTTP(rw, req, p.UserInfoHandler)
	case path == p.endSessionPath:
		p.EndSessionHandler(rw, req)
	case path == p.checkSessionIframePath:
		p.CheckSessionIframeHandler(rw, req)
	case path == p.registrationPath:
		p.RegistrationHandler(rw, req)
	default:
		http.NotFound(rw, req)
	}
}

// ErrorPage writes a HTML error page to the provided ResponseWriter.
func (p *Provider) ErrorPage(rw http.ResponseWriter, code int, title string, message string) {
	utils.WriteErrorPage(rw, code, title, message)
}

// Found writes a HTTP 302 to the provided ResponseWriter with the appropriate
// Location header creates from the other parameters.
func (p *Provider) Found(rw http.ResponseWriter, uri *url.URL, params interface{}, asFragment bool) {
	err := utils.WriteRedirect(rw, http.StatusFound, uri, params, asFragment)
	if err != nil {
		p.logger.WithError(err).Debugln("failed to write to response")
		p.ErrorPage(rw, http.StatusInternalServerError, "", err.Error())
		return
	}
}

// LoginRequiredPage writes a HTTP 30 to the provided ResponseWrite with the
// URL of the provided request (set to the scheme and host of issuer) as
// continue parameter.
func (p *Provider) LoginRequiredPage(rw http.ResponseWriter, req *http.Request, uri *url.URL) {
	issURI, _ := url.Parse(p.issuerIdentifier)

	trusted, _ := utils.IsRequestFromTrustedSource(req, p.Config.Config.TrustedProxyIPs, p.Config.Config.TrustedProxyNets)

	continueURI := getRequestURL(req, trusted)
	continueURI.Scheme = issURI.Scheme
	continueURI.Host = issURI.Host

	uri, err := url.Parse(fmt.Sprintf("%s?continue=%s&oauth=1", uri.String(), url.QueryEscape(continueURI.String())))
	if err != nil {
		p.logger.WithError(err).Debugln("failed to parse sign-in URL")
		p.ErrorPage(rw, http.StatusInternalServerError, "", err.Error())
		return
	}

	p.Found(rw, uri, nil, false)
}

// GetAccessTokenClaimsFromRequest reads incoming request, validates the
// access token and returns the validated claims.
func (p *Provider) GetAccessTokenClaimsFromRequest(req *http.Request) (*konnect.AccessTokenClaims, error) {
	var err error
	var claims *konnect.AccessTokenClaims

	auth := strings.SplitN(req.Header.Get("Authorization"), " ", 2)
	switch auth[0] {
	case oidc.TokenTypeBearer:
		if len(auth) != 2 {
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidRequest, "Invalid Bearer authorization header format")
			break
		}
		claims = &konnect.AccessTokenClaims{}
		_, err = jwt.ParseWithClaims(auth[1], claims, func(token *jwt.Token) (interface{}, error) {
			// Validator for incoming access tokens, looks up key.
			return p.validateJWT(token)
		})
		if err != nil {
			// Wrap as OAuth2 error.
			err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidToken, err.Error())
		}

	default:
		err = konnectoidc.NewOAuth2Error(oidc.ErrorCodeOAuth2InvalidRequest, "Bearer authorization required")
	}

	return claims, err
}
