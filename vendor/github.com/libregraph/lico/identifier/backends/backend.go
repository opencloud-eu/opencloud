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

package backends

import (
	"context"

	"github.com/libregraph/lico/identifier/meta/scopes"
	"github.com/libregraph/lico/identity"
)

// A Backend is an identifier Backend providing functionality to logon and to
// fetch user meta data.
type Backend interface {
	RunWithContext(context.Context) error

	Logon(ctx context.Context, audience string, username string, password string) (success bool, userID *string, sessionRef *string, user UserFromBackend, err error)
	GetUser(ctx context.Context, userID string, sessionRef *string, requestedScopes map[string]bool, lookup *identity.LookupContext) (user UserFromBackend, err error)

	ResolveUserByUsername(ctx context.Context, username string) (user UserFromBackend, err error)

	RefreshSession(ctx context.Context, userID string, sessionRef *string, claims map[string]interface{}) error
	DestroySession(ctx context.Context, sessionRef *string) error

	UserClaims(userID string, authorizedScopes map[string]bool) map[string]interface{}
	ScopesSupported() []string
	ScopesMeta() *scopes.Scopes

	Name() string
}

// UserFromBackend are users as provided by backends which can have additional
// claims together with a user name.
type UserFromBackend interface {
	identity.UserWithUsername
	BackendClaims() map[string]interface{}
	BackendScopes() []string
	RequiredScopes() []string
}

// UserFromBackendWithSessionRef are users as provided by backends which know
// the session of the backend they belong to.
type UserFromBackendWithSessionRef interface {
	UserFromBackend
	SessionRef() *string
}
