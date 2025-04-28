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

package provider

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"

	"github.com/libregraph/lico/identity"
	"github.com/libregraph/lico/oidc/payload"
)

func (p *Provider) getIdentityManager(identityProvider string) (identity.Manager, error) {
	if identityProvider == "" {
		// Return default manager when empty (backwards compatibility).
		return p.identityManager, nil
	}

	if identityProvider == p.identityManager.Name() {
		return p.identityManager, nil
	}
	if p.guestManager != nil && identityProvider == p.guestManager.Name() {
		return p.guestManager, nil
	}

	return nil, errors.New("identity provider mismatch")
}

func (p *Provider) getIdentityManagerFromClaims(identityProvider string, identityClaims jwt.MapClaims) (identity.Manager, error) {
	if identityClaims == nil {
		// Return default manager when no claims.
		return p.identityManager, nil
	}

	return p.getIdentityManager(identityProvider)
}

func (p *Provider) getIdentityManagerFromSession(session *payload.Session) (identity.Manager, error) {
	if session == nil {
		// Return default manager when no session.
		return p.identityManager, nil
	}

	return p.getIdentityManager(session.Provider)
}
