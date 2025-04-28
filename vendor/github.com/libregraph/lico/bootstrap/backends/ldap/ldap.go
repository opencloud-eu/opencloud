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

package bsldap

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/libregraph/lico/bootstrap"
	"github.com/libregraph/lico/identifier"
	"github.com/libregraph/lico/identifier/backends/ldap"
	"github.com/libregraph/lico/identity"
	"github.com/libregraph/lico/identity/managers"
)

// Identity managers.
const (
	identityManagerName = "ldap"
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
		return nil, fmt.Errorf("ldap backend is incompatible with authorization-endpoint-uri parameter")
	}
	config.AuthorizationEndpointURI.Path = bs.MakeURIPath(bootstrap.APITypeSignin, "/identifier/_/authorize")

	if config.EndSessionEndpointURI.String() != "" {
		return nil, fmt.Errorf("ldap backend is incompatible with endsession-endpoint-uri parameter")
	}
	config.EndSessionEndpointURI.Path = bs.MakeURIPath(bootstrap.APITypeSignin, "/identifier/_/endsession")

	if config.SignInFormURI.EscapedPath() == "" {
		config.SignInFormURI.Path = bs.MakeURIPath(bootstrap.APITypeSignin, "/identifier")
	}

	if config.SignedOutURI.EscapedPath() == "" {
		config.SignedOutURI.Path = bs.MakeURIPath(bootstrap.APITypeSignin, "/goodbye")
	}

	// Default LDAP attribute mappings.
	attributeMapping := map[string]string{
		ldap.AttributeLogin:                        os.Getenv("LDAP_LOGIN_ATTRIBUTE"),
		ldap.AttributeEmail:                        os.Getenv("LDAP_EMAIL_ATTRIBUTE"),
		ldap.AttributeName:                         os.Getenv("LDAP_NAME_ATTRIBUTE"),
		ldap.AttributeFamilyName:                   os.Getenv("LDAP_FAMILY_NAME_ATTRIBUTE"),
		ldap.AttributeGivenName:                    os.Getenv("LDAP_GIVEN_NAME_ATTRIBUTE"),
		ldap.AttributeUUID:                         os.Getenv("LDAP_UUID_ATTRIBUTE"),
		fmt.Sprintf("%s_type", ldap.AttributeUUID): os.Getenv("LDAP_UUID_ATTRIBUTE_TYPE"),
	}
	// Add optional LDAP attribute mappings.
	if numericUIDAttribute := os.Getenv("LDAP_UIDNUMBER_ATTRIBUTE"); numericUIDAttribute != "" {
		attributeMapping[ldap.AttributeNumericUID] = numericUIDAttribute
	}
	// Sub from LDAP attribute mappings.
	var subMapping []string
	if subMappingString := os.Getenv("LDAP_SUB_ATTRIBUTES"); subMappingString != "" {
		subMapping = strings.Split(subMappingString, " ")
	}

	// Use a clone here to avoid changing the config of other possible users of the config.
	tlsConfig := config.TLSClientConfig.Clone()
	if caCertFile := os.Getenv("LDAP_TLS_CACERT"); caCertFile != "" {
		if pemBytes, err := ioutil.ReadFile(caCertFile); err == nil {
			rpool, _ := x509.SystemCertPool()
			if rpool.AppendCertsFromPEM(pemBytes) {
				tlsConfig.RootCAs = rpool
			} else {
				return nil, fmt.Errorf("failed to append CA certificate(s) from '%s' to pool", caCertFile)
			}
		} else {
			return nil, fmt.Errorf("failed to read CA certificate(s) from '%s': %w", caCertFile, err)
		}
	}

	identifierBackend, identifierErr := ldap.NewLDAPIdentifierBackend(
		config.Config,
		tlsConfig,
		os.Getenv("LDAP_URI"),
		os.Getenv("LDAP_BINDDN"),
		os.Getenv("LDAP_BINDPW"),
		os.Getenv("LDAP_BASEDN"),
		os.Getenv("LDAP_SCOPE"),
		os.Getenv("LDAP_FILTER"),
		subMapping,
		attributeMapping,
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
