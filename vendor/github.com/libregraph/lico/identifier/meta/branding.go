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

package meta

// Branding is a container to hold identifier branding meta data.
type Branding struct {
	BannerLogo        *string  `json:"bannerLogo,omitempty"`
	SignInPageText    *string  `json:"signinPageText,omitempty"`
	UsernameHintText  *string  `json:"usernameHintText,omitempty"`
	SignInPageLogoURI *string  `json:"signinPageLogoURI,omitempty"`
	Locales           []string `json:"locales,omitempty"`
}
