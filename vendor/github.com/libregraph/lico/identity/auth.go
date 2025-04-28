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

package identity

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/libregraph/lico/oidc/payload"
)

// AuthRecord is an interface which provides identity auth information with scopes and claims..
type AuthRecord interface {
	Manager() Manager
	Subject() string
	AuthorizedScopes() map[string]bool
	AuthorizeScopes(map[string]bool)
	AuthorizedClaims() *payload.ClaimsRequest
	AuthorizeClaims(*payload.ClaimsRequest)
	Claims(...string) []jwt.Claims

	User() PublicUser
	SetUser(PublicUser)

	LoggedOn() (bool, time.Time)
	SetAuthTime(time.Time)
}
