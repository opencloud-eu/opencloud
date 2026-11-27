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

package bootstrap

import (
	"fmt"
	"os"

	"github.com/libregraph/lico/bootstrap"
	"github.com/libregraph/lico/identifier"
	"github.com/libregraph/lico/identifier/backends"
	"github.com/libregraph/lico/identity"
	"github.com/libregraph/lico/identity/managers"

	cs3 "github.com/opencloud-eu/opencloud/services/idp/pkg/backends/cs3/identifier"
)

// Identity managers.
const (
	identityManagerName = "cs3"
)

// BackendWrapper decorates the identifier backend, e.g. to add a second factor to the logon.
type BackendWrapper func(backends.Backend) backends.Backend

// Register adds the CS3 identity manager to the lico bootstrap. If wrap is not nil the
// identifier backend is passed through it.
func Register(wrap BackendWrapper) error {
	return bootstrap.RegisterIdentityManager(identityManagerName, func(bs bootstrap.Bootstrap) (identity.Manager, error) {
		return NewIdentityManager(bs, wrap)
	})
}

// MustRegister adds the CS3 identity manager to the lico bootstrap or panics
func MustRegister(wrap BackendWrapper) {
	if err := Register(wrap); err != nil {
		panic(err)
	}
}

// NewIdentityManager produces a CS3 backed identity manager instance for the idp
func NewIdentityManager(bs bootstrap.Bootstrap, wrap BackendWrapper) (identity.Manager, error) {
	config := bs.Config()

	logger := config.Config.Logger

	if config.AuthorizationEndpointURI.String() != "" {
		return nil, fmt.Errorf("cs3 backend is incompatible with authorization-endpoint-uri parameter")
	}
	config.AuthorizationEndpointURI.Path = bs.MakeURIPath(bootstrap.APITypeSignin, "/identifier/_/authorize")

	if config.EndSessionEndpointURI.String() != "" {
		return nil, fmt.Errorf("cs3 backend is incompatible with endsession-endpoint-uri parameter")
	}
	config.EndSessionEndpointURI.Path = bs.MakeURIPath(bootstrap.APITypeSignin, "/identifier/_/endsession")

	if config.SignInFormURI.EscapedPath() == "" {
		config.SignInFormURI.Path = bs.MakeURIPath(bootstrap.APITypeSignin, "/identifier")
	}

	if config.SignedOutURI.EscapedPath() == "" {
		config.SignedOutURI.Path = bs.MakeURIPath(bootstrap.APITypeSignin, "/goodbye")
	}

	identifierBackend, identifierErr := cs3.NewCS3Backend(
		config.Config,
		config.TLSClientConfig,
		// FIXME add a map[string]interface{} property to the lico config.Config so backends can pass custom config parameters through the bootstrap process
		os.Getenv("CS3_GATEWAY"),
		os.Getenv("CS3_MACHINE_AUTH_API_KEY"),
		config.Settings.Insecure,
	)
	if identifierErr != nil {
		return nil, fmt.Errorf("failed to create identifier backend: %v", identifierErr)
	}

	var activeBackend backends.Backend = identifierBackend
	if wrap != nil {
		activeBackend = wrap(activeBackend)
	}

	fullAuthorizationEndpointURL := bootstrap.WithSchemeAndHost(config.AuthorizationEndpointURI, config.IssuerIdentifierURI)
	fullSignInFormURL := bootstrap.WithSchemeAndHost(config.SignInFormURI, config.IssuerIdentifierURI)
	fullSignedOutEndpointURL := bootstrap.WithSchemeAndHost(config.SignedOutURI, config.IssuerIdentifierURI)

	activeIdentifier, err := identifier.NewIdentifier(&identifier.Config{
		Config: config.Config,

		BaseURI:        config.IssuerIdentifierURI,
		PathPrefix:     bs.MakeURIPath(bootstrap.APITypeSignin, ""),
		StaticFolder:   config.IdentifierClientPath,
		ScopesConf:     config.IdentifierScopesConf,
		WebAppDisabled: config.IdentifierClientDisabled,

		LogonCookieName:     "__Secure-KKT", // Kopano-Konnect-Token
		LogonCookieSameSite: config.CookieSameSite,

		ConsentCookieSameSite: config.CookieSameSite,

		StateCookieSameSite: config.CookieSameSite,

		AuthorizationEndpointURI: fullAuthorizationEndpointURL,
		SignedOutEndpointURI:     fullSignedOutEndpointURL,

		DefaultBannerLogo:       config.IdentifierDefaultBannerLogo,
		DefaultSignInPageText:   config.IdentifierDefaultSignInPageText,
		DefaultUsernameHintText: config.IdentifierDefaultUsernameHintText,
		UILocales:               config.IdentifierUILocales,

		Backend: activeBackend,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create identifier: %v", err)
	}
	err = activeIdentifier.SetKey(config.EncryptionSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid --encryption-secret parameter value for identifier: %v", err)
	}

	// Expose the identifier in the managers registry like the other lico backends do.
	bs.Managers().Set("identifier", activeIdentifier)

	identityManagerConfig := &identity.Config{
		SignInFormURI: fullSignInFormURL,
		SignedOutURI:  fullSignedOutEndpointURL,

		Logger: logger,

		ScopesSupported: config.Config.AllowedScopes,
	}

	identifierIdentityManager := managers.NewIdentifierIdentityManager(identityManagerConfig, activeIdentifier)
	logger.Infoln("using identifier backed identity manager")

	return identifierIdentityManager, nil
}
