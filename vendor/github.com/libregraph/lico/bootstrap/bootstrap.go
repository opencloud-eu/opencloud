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

package bootstrap

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/longsleep/rndm"
	"github.com/sirupsen/logrus"

	"github.com/libregraph/lico/config"
	"github.com/libregraph/lico/encryption"
	"github.com/libregraph/lico/identity"
	"github.com/libregraph/lico/managers"
	oidcProvider "github.com/libregraph/lico/oidc/provider"
	"github.com/libregraph/lico/utils"
)

// API types.
type APIType string

const (
	APITypeKonnect APIType = "konnect"
	APITypeSignin  APIType = "signin"
)

// Defaults.
const (
	DefaultSigningKeyID   = "default"
	DefaultSigningKeyBits = 2048

	DefaultGuestIdentityManagerName = "guest"
	DefaultCookieSameSite           = http.SameSiteNoneMode
)

// Bootstrap is a data structure to hold configuration required to start
// konnectd.
type Bootstrap interface {
	Config() *Config
	Managers() *managers.Managers

	MakeURIPath(api APIType, subpath string) string
}

// Implementation of the bootstrap interface.
type bootstrap struct {
	config *Config

	uriBasePath string

	managers *managers.Managers
}

// Config returns the bootstap configuration.
func (bs *bootstrap) Config() *Config {
	return bs.config
}

// Managers returns bootstrapped identity-managers.
func (bs *bootstrap) Managers() *managers.Managers {
	return bs.managers
}

// Boot is the main entry point to bootstrap the service after validating the
// given configuration. The resulting Bootstrap struct can be used to retrieve
// configured identity-managers and their respective http-handlers and config.
//
// This function should be used by consumers which want to embed this project
// as a library.
func Boot(ctx context.Context, settings *Settings, cfg *config.Config) (Bootstrap, error) {
	// NOTE(longsleep): Ensure to use same salt length as the hash size.
	// See https://www.ietf.org/mail-archive/web/jose/current/msg02901.html for
	// reference and https://github.com/golang-jwt/jwt/v4/issues/285 for
	// the issue in upstream jwt-go.
	for _, alg := range []string{jwt.SigningMethodPS256.Name, jwt.SigningMethodPS384.Name, jwt.SigningMethodPS512.Name} {
		sm := jwt.GetSigningMethod(alg)
		if signingMethodRSAPSS, ok := sm.(*jwt.SigningMethodRSAPSS); ok {
			signingMethodRSAPSS.Options.SaltLength = rsa.PSSSaltLengthEqualsHash
		}
	}

	bs := &bootstrap{
		config: &Config{
			Config:   cfg,
			Settings: settings,
		},
	}

	err := bs.initialize(settings)
	if err != nil {
		return nil, err
	}

	err = bs.setup(ctx, settings)
	if err != nil {
		return nil, err
	}

	return bs, nil
}

// initialize, parsed parameters from commandline with validation and adds them
// to the associated Bootstrap data.
func (bs *bootstrap) initialize(settings *Settings) error {
	logger := bs.config.Config.Logger
	var err error

	if settings.IdentityManager == "" {
		return fmt.Errorf("identity-manager argument missing, use one of kc, ldap, cookie, dummy")
	}

	bs.config.IssuerIdentifierURI, err = url.Parse(settings.Iss)
	if err != nil {
		return fmt.Errorf("invalid iss value, iss is not a valid URL), %v", err)
	} else if settings.Iss == "" {
		return fmt.Errorf("missing iss value, did you provide the --iss parameter?")
	} else if bs.config.IssuerIdentifierURI.Scheme != "https" {
		return fmt.Errorf("invalid iss value, URL must start with https://")
	} else if bs.config.IssuerIdentifierURI.Host == "" {
		return fmt.Errorf("invalid iss value, URL must have a host")
	}

	bs.uriBasePath = settings.URIBasePath

	bs.config.SignInFormURI, err = url.Parse(settings.SignInURI)
	if err != nil {
		return fmt.Errorf("invalid sign-in URI, %v", err)
	}

	bs.config.SignedOutURI, err = url.Parse(settings.SignedOutURI)
	if err != nil {
		return fmt.Errorf("invalid signed-out URI, %v", err)
	}

	bs.config.AuthorizationEndpointURI, err = url.Parse(settings.AuthorizationEndpointURI)
	if err != nil {
		return fmt.Errorf("invalid authorization-endpoint-uri, %v", err)
	}

	bs.config.EndSessionEndpointURI, err = url.Parse(settings.EndsessionEndpointURI)
	if err != nil {
		return fmt.Errorf("invalid endsession-endpoint-uri, %v", err)
	}

	if settings.Insecure {
		// NOTE(longsleep): This disable http2 client support. See https://github.com/golang/go/issues/14275 for reasons.
		bs.config.TLSClientConfig = utils.InsecureSkipVerifyTLSConfig()
		logger.Warnln("insecure mode, TLS client connections are susceptible to man-in-the-middle attacks")
	} else {
		bs.config.TLSClientConfig = utils.DefaultTLSConfig()
	}

	for _, trustedProxy := range settings.TrustedProxy {
		if ip := net.ParseIP(trustedProxy); ip != nil {
			bs.config.Config.TrustedProxyIPs = append(bs.config.Config.TrustedProxyIPs, &ip)
			continue
		}
		if _, ipNet, errParseCIDR := net.ParseCIDR(trustedProxy); errParseCIDR == nil {
			bs.config.Config.TrustedProxyNets = append(bs.config.Config.TrustedProxyNets, ipNet)
			continue
		}
	}
	if len(bs.config.Config.TrustedProxyIPs) > 0 {
		logger.Infoln("trusted proxy IPs", bs.config.Config.TrustedProxyIPs)
	}
	if len(bs.config.Config.TrustedProxyNets) > 0 {
		logger.Infoln("trusted proxy networks", bs.config.Config.TrustedProxyNets)
	}

	if len(settings.AllowScope) > 0 {
		bs.config.Config.AllowedScopes = settings.AllowScope
		logger.Infoln("using custom allowed OAuth 2 scopes", bs.config.Config.AllowedScopes)
	}

	bs.config.Config.AllowClientGuests = settings.AllowClientGuests
	if bs.config.Config.AllowClientGuests {
		logger.Infoln("client controlled guests are enabled")
	}

	bs.config.Config.AllowDynamicClientRegistration = settings.AllowDynamicClientRegistration
	if bs.config.Config.AllowDynamicClientRegistration {
		logger.Infoln("dynamic client registration is enabled")
	}

	encryptionSecretFn := settings.EncryptionSecretFile

	if encryptionSecretFn != "" {
		logger.WithField("file", encryptionSecretFn).Infoln("loading encryption secret from file")
		bs.config.EncryptionSecret, err = ioutil.ReadFile(encryptionSecretFn)
		if err != nil {
			return fmt.Errorf("failed to load encryption secret from file: %v", err)
		}
		if len(bs.config.EncryptionSecret) != encryption.KeySize {
			return fmt.Errorf("invalid encryption secret size - must be %d bytes", encryption.KeySize)
		}
	} else {
		logger.Warnf("missing --encryption-secret parameter, using random encyption secret with %d bytes", encryption.KeySize)
		bs.config.EncryptionSecret = rndm.GenerateRandomBytes(encryption.KeySize)
	}

	bs.config.Config.ListenAddr = settings.Listen

	bs.config.IdentifierClientDisabled = settings.IdentifierClientDisabled
	bs.config.IdentifierClientPath = settings.IdentifierClientPath

	bs.config.IdentifierRegistrationConf = settings.IdentifierRegistrationConf
	if bs.config.IdentifierRegistrationConf != "" {
		bs.config.IdentifierRegistrationConf, _ = filepath.Abs(bs.config.IdentifierRegistrationConf)
		if _, errStat := os.Stat(bs.config.IdentifierRegistrationConf); errStat != nil {
			return fmt.Errorf("identifier-registration-conf file not found or unable to access: %v", errStat)
		}
		bs.config.IdentifierAuthoritiesConf = bs.config.IdentifierRegistrationConf
	}

	bs.config.IdentifierScopesConf = settings.IdentifierScopesConf
	if bs.config.IdentifierScopesConf != "" {
		bs.config.IdentifierScopesConf, _ = filepath.Abs(bs.config.IdentifierScopesConf)
		if _, errStat := os.Stat(bs.config.IdentifierScopesConf); errStat != nil {
			return fmt.Errorf("identifier-scopes-conf file not found or unable to access: %v", errStat)
		}
	}

	if settings.IdentifierDefaultBannerLogo != "" {
		// Load from file.
		b, errRead := ioutil.ReadFile(settings.IdentifierDefaultBannerLogo)
		if errRead != nil {
			return fmt.Errorf("identifier-default-banner-logo failed to open: %w", errRead)
		}
		bs.config.IdentifierDefaultBannerLogo = b
	}
	if settings.IdentifierDefaultSignInPageText != "" {
		bs.config.IdentifierDefaultSignInPageText = &settings.IdentifierDefaultSignInPageText
	}
	if settings.IdentifierDefaultLogoTargetURI != "" {
		bs.config.IdentifierDefaultLogoTargetURI = &settings.IdentifierDefaultLogoTargetURI
	}
	if settings.IdentifierDefaultUsernameHintText != "" {
		bs.config.IdentifierDefaultUsernameHintText = &settings.IdentifierDefaultUsernameHintText
	}
	bs.config.IdentifierUILocales = settings.IdentifierUILocales

	bs.config.SigningKeyID = settings.SigningKid
	bs.config.Signers = make(map[string]crypto.Signer)
	bs.config.Validators = make(map[string]crypto.PublicKey)
	bs.config.Certificates = make(map[string][]*x509.Certificate)

	signingMethodString := settings.SigningMethod
	bs.config.SigningMethod = jwt.GetSigningMethod(signingMethodString)
	if bs.config.SigningMethod == nil {
		return fmt.Errorf("unknown signing method: %s", signingMethodString)
	}

	signingKeyFns := settings.SigningPrivateKeyFiles
	if len(signingKeyFns) > 0 {
		first := true
		for _, signingKeyFn := range signingKeyFns {
			logger.WithField("path", signingKeyFn).Infoln("loading signing key")
			err = addSignerWithIDFromFile(signingKeyFn, "", bs)
			if err != nil {
				return err
			}
			if first {
				// Also add key under the provided id.
				first = false
				err = addSignerWithIDFromFile(signingKeyFn, bs.config.SigningKeyID, bs)
				if err != nil {
					return err
				}
			}
		}
	} else {
		//NOTE(longsleep): remove me - create keypair a random key pair.
		sm := jwt.SigningMethodPS256
		bs.config.SigningMethod = sm
		logger.WithField("alg", sm.Name).Warnf("missing --signing-private-key parameter, using random %d bit signing key", DefaultSigningKeyBits)
		signer, _ := rsa.GenerateKey(rand.Reader, DefaultSigningKeyBits)
		bs.config.Signers[bs.config.SigningKeyID] = signer
	}

	// Ensure we have a signer for the things we need.
	err = validateSigners(bs)
	if err != nil {
		return err
	}

	validationKeysPath := settings.ValidationKeysPath
	if validationKeysPath != "" {
		logger.WithField("path", validationKeysPath).Infoln("loading validation keys")
		err = addValidatorsFromPath(validationKeysPath, bs)
		if err != nil {
			return err
		}
	}

	bs.config.Config.HTTPTransport = utils.HTTPTransportWithTLSClientConfig(bs.config.TLSClientConfig)

	bs.config.AccessTokenDurationSeconds = settings.AccessTokenDurationSeconds
	if bs.config.AccessTokenDurationSeconds == 0 {
		bs.config.AccessTokenDurationSeconds = 60 * 10 // 10 Minutes
	}
	bs.config.IDTokenDurationSeconds = settings.IDTokenDurationSeconds
	if bs.config.IDTokenDurationSeconds == 0 {
		bs.config.IDTokenDurationSeconds = 60 * 60 // 1 Hour
	}
	bs.config.RefreshTokenDurationSeconds = settings.RefreshTokenDurationSeconds
	if bs.config.RefreshTokenDurationSeconds == 0 {
		bs.config.RefreshTokenDurationSeconds = 60 * 60 * 24 * 365 * 3 // 3 Years
	}
	bs.config.DyamicClientSecretDurationSeconds = settings.DyamicClientSecretDurationSeconds

	// add setting to allow setting the same site attribute of the cookies
	bs.config.CookieSameSite = settings.CookieSameSite
	if bs.config.CookieSameSite == 0 {
		bs.config.CookieSameSite = DefaultCookieSameSite
	}

	return nil
}

// setup takes care of setting up the managers based on the associated
// Bootstrap's data.
func (bs *bootstrap) setup(ctx context.Context, settings *Settings) error {
	managers, err := newManagers(ctx, bs)
	if err != nil {
		return err
	}
	bs.managers = managers

	identityManager, err := bs.setupIdentity(ctx, settings)
	if err != nil {
		return err
	}
	managers.Set("identity", identityManager)

	guestManager, err := bs.setupGuest(ctx, identityManager)
	if err != nil {
		return err
	}
	managers.Set("guest", guestManager)

	oidcProvider, err := bs.setupOIDCProvider(ctx)
	if err != nil {
		return err
	}
	managers.Set("oidc", oidcProvider)
	managers.Set("handler", oidcProvider) // Use OIDC provider as default HTTP handler.

	err = managers.Apply()
	if err != nil {
		return fmt.Errorf("failed to apply managers: %v", err)
	}

	// Final steps
	err = oidcProvider.InitializeMetadata()
	if err != nil {
		return fmt.Errorf("failed to initialize provider metadata: %v", err)
	}

	return nil
}

func (bs *bootstrap) MakeURIPath(api APIType, subpath string) string {
	subpath = strings.TrimPrefix(subpath, "/")
	uriPath := ""

	switch api {
	case APITypeKonnect:
		uriPath = fmt.Sprintf("%s/konnect/v1/%s", strings.TrimSuffix(bs.uriBasePath, "/"), subpath)
	case APITypeSignin:
		uriPath = fmt.Sprintf("%s/signin/v1/%s", strings.TrimSuffix(bs.uriBasePath, "/"), subpath)
	default:
		panic("unknown api type")
	}

	if subpath == "" {
		uriPath = strings.TrimSuffix(uriPath, "/")
	}
	return uriPath
}

func (bs *bootstrap) MakeURI(api APIType, subpath string) *url.URL {
	uriPath := bs.MakeURIPath(api, subpath)
	uri, _ := url.Parse(bs.config.IssuerIdentifierURI.String())
	uri.Path = uriPath

	return uri
}

func (bs *bootstrap) setupIdentity(ctx context.Context, settings *Settings) (identity.Manager, error) {
	logger := bs.config.Config.Logger

	if settings.IdentityManager == "" {
		return nil, fmt.Errorf("identity-manager argument missing")
	}

	// Identity manager.
	identityManagerName := settings.IdentityManager
	identityManager, err := getIdentityManagerByName(identityManagerName, bs)
	if err != nil {
		return nil, err
	}
	logger.WithFields(logrus.Fields{
		"name":   identityManagerName,
		"scopes": identityManager.ScopesSupported(nil),
		"claims": identityManager.ClaimsSupported(nil),
	}).Infoln("identity manager set up")

	return identityManager, nil
}

func (bs *bootstrap) setupGuest(ctx context.Context, identityManager identity.Manager) (identity.Manager, error) {
	if !bs.config.Config.AllowClientGuests {
		return nil, nil
	}

	var err error
	logger := bs.config.Config.Logger

	guestManager, err := getIdentityManagerByName(DefaultGuestIdentityManagerName, bs)
	if err != nil {
		return nil, err
	}

	if guestManager != nil {
		logger.Infoln("identity guest manager set up")
	}
	return guestManager, nil
}

func (bs *bootstrap) setupOIDCProvider(ctx context.Context) (*oidcProvider.Provider, error) {
	var err error
	logger := bs.config.Config.Logger

	sessionCookiePath, err := getCommonURLPathPrefix(bs.config.AuthorizationEndpointURI.EscapedPath(), bs.config.EndSessionEndpointURI.EscapedPath())
	if err != nil {
		return nil, fmt.Errorf("failed to find common URL prefix for authorize and endsession: %v", err)
	}

	var registrationPath = ""
	if bs.config.Config.AllowDynamicClientRegistration {
		registrationPath = bs.MakeURIPath(APITypeKonnect, "/register")
	}

	provider, err := oidcProvider.NewProvider(&oidcProvider.Config{
		Config: bs.config.Config,

		IssuerIdentifier:       bs.config.IssuerIdentifierURI.String(),
		WellKnownPath:          "/.well-known/openid-configuration",
		JwksPath:               bs.MakeURIPath(APITypeKonnect, "/jwks.json"),
		AuthorizationPath:      bs.config.AuthorizationEndpointURI.EscapedPath(),
		TokenPath:              bs.MakeURIPath(APITypeKonnect, "/token"),
		UserInfoPath:           bs.MakeURIPath(APITypeKonnect, "/userinfo"),
		EndSessionPath:         bs.config.EndSessionEndpointURI.EscapedPath(),
		CheckSessionIframePath: bs.MakeURIPath(APITypeKonnect, "/session/check-session.html"),
		RegistrationPath:       registrationPath,

		BrowserStateCookiePath:     bs.MakeURIPath(APITypeKonnect, "/session/"),
		BrowserStateCookieName:     "__Secure-KKBS", // Kopano-Konnect-Browser-State
		BrowserStateCookieSameSite: bs.config.CookieSameSite,

		SessionCookiePath:     sessionCookiePath,
		SessionCookieName:     "__Secure-KKCS", // Kopano-Konnect-Client-Session
		SessionCookieSameSite: bs.config.CookieSameSite,

		AccessTokenDuration:  time.Duration(bs.config.AccessTokenDurationSeconds) * time.Second,
		IDTokenDuration:      time.Duration(bs.config.IDTokenDurationSeconds) * time.Second,
		RefreshTokenDuration: time.Duration(bs.config.RefreshTokenDurationSeconds) * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %v", err)
	}
	if bs.config.SigningMethod != nil {
		err = provider.SetSigningMethod(bs.config.SigningMethod)
		if err != nil {
			return nil, fmt.Errorf("failed to set provider signing method: %v", err)
		}
	}

	// All add signers.
	for id, signer := range bs.config.Signers {
		if id == bs.config.SigningKeyID {
			err = provider.SetSigningKey(id, signer)
			// Always set default key.
			if id != DefaultSigningKeyID {
				provider.SetValidationKey(DefaultSigningKeyID, signer.Public())
			}
		} else {
			// Set non default signers as well.
			err = provider.SetSigningKey(id, signer)
		}
		if err != nil {
			return nil, err
		}
	}
	// Add all validators.
	for id, publicKey := range bs.config.Validators {
		err = provider.SetValidationKey(id, publicKey)
		if err != nil {
			return nil, err
		}
	}
	// Add all certificates.
	for id, certificate := range bs.config.Certificates {
		err = provider.SetCertificate(id, certificate)
		if err != nil {
			return nil, err
		}
	}

	sk, ok := provider.GetSigningKey(bs.config.SigningMethod)
	if !ok {
		return nil, fmt.Errorf("no signing key for selected signing method")
	}
	if bs.config.SigningKeyID == "" {
		// Ensure that there is a default signing Key ID even if none was set.
		provider.SetValidationKey(DefaultSigningKeyID, sk.PrivateKey.Public())
	}
	logger.WithFields(logrus.Fields{
		"id":     sk.ID,
		"method": fmt.Sprintf("%T", sk.SigningMethod),
		"alg":    sk.SigningMethod.Alg(),
	}).Infoln("oidc token signing default set up")

	return provider, nil
}
