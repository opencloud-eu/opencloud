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

package bslibregraph

import (
	"fmt"
	"os"
	"strings"

	"github.com/cevaris/ordered_map"

	"github.com/libregraph/lico/bootstrap"
	"github.com/libregraph/lico/identifier"
	"github.com/libregraph/lico/identifier/backends/libregraph"
	"github.com/libregraph/lico/identity"
	identityClients "github.com/libregraph/lico/identity/clients"
	"github.com/libregraph/lico/identity/managers"
)

// Identity managers.
const (
	identityManagerName = "libregraph"
)

func Register() error {
	return bootstrap.RegisterIdentityManager(identityManagerName, NewIdentityManager)
}

func MustRegister() {
	if err := Register(); err != nil {
		panic(err)
	}
}

func NewIdentityManager(bs bootstrap.Bootstrap) (identity.Manager, error) {
	config := bs.Config()

	logger := config.Config.Logger

	if config.AuthorizationEndpointURI.String() != "" {
		return nil, fmt.Errorf("libregraph backend is incompatible with authorization-endpoint-uri parameter")
	}
	config.AuthorizationEndpointURI.Path = bs.MakeURIPath(bootstrap.APITypeSignin, "/identifier/_/authorize")

	if config.EndSessionEndpointURI.String() != "" {
		return nil, fmt.Errorf("libregraph backend is incompatible with endsession-endpoint-uri parameter")
	}
	config.EndSessionEndpointURI.Path = bs.MakeURIPath(bootstrap.APITypeSignin, "/identifier/_/endsession")

	if config.SignInFormURI.EscapedPath() == "" {
		config.SignInFormURI.Path = bs.MakeURIPath(bootstrap.APITypeSignin, "/identifier")
	}

	if config.SignedOutURI.EscapedPath() == "" {
		config.SignedOutURI.Path = bs.MakeURIPath(bootstrap.APITypeSignin, "/goodbye")
	}

	defaultURI := os.Getenv("LIBREGRAPH_URI")

	var scopedURIs *ordered_map.OrderedMap
	if scopedURIsString := os.Getenv("LIBREGRAPH_SCOPED_URIS"); scopedURIsString != "" {
		scopedURIs = ordered_map.NewOrderedMap()
		// Format is <scope>:<url>,<scope>:<url>,...
		for _, v := range strings.Split(scopedURIsString, ",") {
			parts := strings.SplitN(v, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("failed to parse scoped URIs, format invalid")
			}
			scopedURIs.Set(parts[0], parts[1])
		}
	}

	var clients *identityClients.Registry
	if clientsRecord, ok := bs.Managers().Get("clients"); ok {
		clients = clientsRecord.(*identityClients.Registry)
	} else {
		return nil, fmt.Errorf("clients manager not found but is required")
	}

	identifierBackend, identifierErr := libregraph.NewLibreGraphIdentifierBackend(
		config.Config,
		config.TLSClientConfig,
		defaultURI,
		scopedURIs,
		clients,
	)
	if identifierErr != nil {
		return nil, fmt.Errorf("failed to create identifier backend: %v", identifierErr)
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

		DefaultBannerLogo:        config.IdentifierDefaultBannerLogo,
		DefaultSignInPageText:    config.IdentifierDefaultSignInPageText,
		DefaultSignInPageLogoURI: config.IdentifierDefaultLogoTargetURI,
		DefaultUsernameHintText:  config.IdentifierDefaultUsernameHintText,
		UILocales:                config.IdentifierUILocales,

		Backend: identifierBackend,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create identifier: %v", err)
	}
	err = activeIdentifier.SetKey(config.EncryptionSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid --encryption-secret parameter value for identifier: %v", err)
	}

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
