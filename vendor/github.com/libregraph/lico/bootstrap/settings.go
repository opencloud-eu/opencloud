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
	"net/http"
)

// Settings is a typed application config which represents the user accessible
// boostrap settings params.
type Settings struct {
	Iss                               string
	IdentityManager                   string
	URIBasePath                       string
	SignInURI                         string
	SignedOutURI                      string
	AuthorizationEndpointURI          string
	EndsessionEndpointURI             string
	Insecure                          bool
	TrustedProxy                      []string
	AllowScope                        []string
	AllowClientGuests                 bool
	AllowDynamicClientRegistration    bool
	EncryptionSecretFile              string
	Listen                            string
	IdentifierClientDisabled          bool
	IdentifierClientPath              string
	IdentifierRegistrationConf        string
	IdentifierScopesConf              string
	IdentifierDefaultBannerLogo       string
	IdentifierDefaultSignInPageText   string
	IdentifierDefaultLogoTargetURI    string
	IdentifierDefaultUsernameHintText string
	IdentifierUILocales               []string
	SigningKid                        string
	SigningMethod                     string
	SigningPrivateKeyFiles            []string
	ValidationKeysPath                string
	CookieBackendURI                  string
	CookieNames                       []string
	CookieSameSite                    http.SameSite
	AccessTokenDurationSeconds        uint64
	IDTokenDurationSeconds            uint64
	RefreshTokenDurationSeconds       uint64
	DyamicClientSecretDurationSeconds uint64
}
