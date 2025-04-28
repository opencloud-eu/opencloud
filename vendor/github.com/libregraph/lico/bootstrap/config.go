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
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt/v5"

	"github.com/libregraph/lico/config"
)

// Config is a typed application config which represents the active
// bootstrap configuration.
type Config struct {
	Config   *config.Config
	Settings *Settings

	SignInFormURI            *url.URL
	SignedOutURI             *url.URL
	AuthorizationEndpointURI *url.URL
	EndSessionEndpointURI    *url.URL

	TLSClientConfig *tls.Config

	IssuerIdentifierURI *url.URL

	IdentifierClientDisabled          bool
	IdentifierClientPath              string
	IdentifierRegistrationConf        string
	IdentifierAuthoritiesConf         string
	IdentifierScopesConf              string
	IdentifierDefaultBannerLogo       []byte
	IdentifierDefaultSignInPageText   *string
	IdentifierDefaultLogoTargetURI    *string
	IdentifierDefaultUsernameHintText *string
	IdentifierUILocales               []string

	EncryptionSecret []byte
	SigningMethod    jwt.SigningMethod
	SigningKeyID     string
	Signers          map[string]crypto.Signer
	Validators       map[string]crypto.PublicKey
	Certificates     map[string][]*x509.Certificate

	AccessTokenDurationSeconds        uint64
	IDTokenDurationSeconds            uint64
	RefreshTokenDurationSeconds       uint64
	DyamicClientSecretDurationSeconds uint64

	CookieSameSite http.SameSite
}
