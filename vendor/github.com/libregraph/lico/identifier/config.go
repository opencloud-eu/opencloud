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

package identifier

import (
	"net/http"
	"net/url"

	"github.com/libregraph/lico/config"
	"github.com/libregraph/lico/identifier/backends"
)

// Config defines a Server's configuration settings.
type Config struct {
	Config *config.Config

	BaseURI    *url.URL
	ScopesConf string

	LogonCookieName     string
	LogonCookieSameSite http.SameSite

	ConsentCookieSameSite http.SameSite

	StateCookieSameSite http.SameSite

	PathPrefix     string
	StaticFolder   string
	WebAppDisabled bool

	AuthorizationEndpointURI *url.URL
	SignedOutEndpointURI     *url.URL

	DefaultBannerLogo        []byte
	DefaultSignInPageText    *string
	DefaultSignInPageLogoURI *string
	DefaultUsernameHintText  *string
	UILocales                []string

	Backend backends.Backend
}
