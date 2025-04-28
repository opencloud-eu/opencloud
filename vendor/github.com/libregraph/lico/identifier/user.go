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
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	konnect "github.com/libregraph/lico"
	"github.com/libregraph/lico/identifier/backends"
	"github.com/libregraph/lico/identity"
	"github.com/libregraph/lico/identity/authorities"
)

// A IdentifiedUser is a user with meta data.
type IdentifiedUser struct {
	sub string

	backend           backends.Backend
	externalAuthority *authorities.Details

	username      string
	email         string
	emailVerified bool
	displayName   string
	familyName    string
	givenName     string

	id  int64
	uid string

	sessionRef *string
	logonRef   *string
	claims     map[string]interface{}
	scopes     []string

	logonAt      time.Time
	expiresAfter *time.Time

	lockedScopes []string
}

// Subject returns the associated users subject field. The subject is the main
// authentication identifier of the user.
func (u *IdentifiedUser) Subject() string {
	return u.sub
}

// Email returns the associated users email field.
func (u *IdentifiedUser) Email() string {
	return u.email
}

// EmailVerified returns trye if the associated users email field was verified.
func (u *IdentifiedUser) EmailVerified() bool {
	return u.emailVerified
}

// Name returns the associated users name field. This is the display name of
// the accociated user.
func (u *IdentifiedUser) Name() string {
	return u.displayName
}

// FamilyName returns the associated users family name field.
func (u *IdentifiedUser) FamilyName() string {
	return u.familyName
}

// GivenName returns the associated users given name field.
func (u *IdentifiedUser) GivenName() string {
	return u.givenName
}

// ID returns the associated users numeric user id. If it is 0, it means that
// this user does not have a numeric ID. Do not use this field to identify a
// user - always use the subject instead. The numeric ID is kept for compatibility
// with systems which require user identification to be numeric.
func (u *IdentifiedUser) ID() int64 {
	return u.id
}

// UniqueID returns the accociated users unique user id. When empty, then this
// user does not have a unique ID. This field can be used for unique user mapping
// to external systems which use the same authentication source as Konnect. The
// value depends entirely on the identifier backend.
func (u *IdentifiedUser) UniqueID() string {
	return u.uid
}

// Username returns the accociated users username. This might be different or
// the same as the subject, depending on the backend in use. If can also be
// empty, which means that the accociated user does not have a username.
func (u *IdentifiedUser) Username() string {
	return u.username
}

// Claims returns extra claims of the accociated user.
func (u *IdentifiedUser) Claims() jwt.MapClaims {
	claims := make(map[string]interface{})
	claims[konnect.IdentifiedUsernameClaim] = u.Username()
	claims[konnect.IdentifiedDisplayNameClaim] = u.Name()

	for k, v := range u.claims {
		claims[k] = v
	}

	return jwt.MapClaims(claims)
}

// ScopedClaims returns scope bound extra claims of the accociated user.
func (u *IdentifiedUser) ScopedClaims(authorizedScopes map[string]bool) jwt.MapClaims {
	if u.backend == nil {
		return nil
	}

	claims := u.backend.UserClaims(u.Subject(), authorizedScopes)
	return jwt.MapClaims(claims)
}

// Scopes returns the scopes attached to this user.
func (u *IdentifiedUser) Scopes() []string {
	return u.scopes
}

// LoggedOn returns true if the accociated user has a logonAt time set.
func (u *IdentifiedUser) LoggedOn() (bool, time.Time) {
	return !u.logonAt.IsZero(), u.logonAt
}

// SessionRef returns the accociated users underlaying session reference.
func (u *IdentifiedUser) SessionRef() *string {
	return u.sessionRef
}

// UserRef returns the accociated users underlaying logon reference.
func (u *IdentifiedUser) LogonRef() *string {
	return u.logonRef
}

func (u *IdentifiedUser) ExternalAuthorityID() *string {
	if u.externalAuthority == nil {
		return nil
	}
	id := u.externalAuthority.ID
	return &id
}

// BackendName returns the accociated users underlaying backend name.
func (u *IdentifiedUser) BackendName() string {
	return u.backend.Name()
}

func (u *IdentifiedUser) LockedScopes() []string {
	return u.lockedScopes
}

func (i *Identifier) logonUser(ctx context.Context, audience, username, password string) (*IdentifiedUser, error) {
	success, subject, sessionRef, u, err := i.backend.Logon(ctx, audience, username, password)
	if err != nil {
		return nil, err
	}

	if !success || u == nil {
		return nil, nil
	}

	user := &IdentifiedUser{
		sub: *subject,

		username: u.Username(),

		backend: i.backend,

		sessionRef: sessionRef,
		claims:     u.BackendClaims(),

		lockedScopes: u.RequiredScopes(),
	}

	return user, nil
}

func (i *Identifier) resolveUser(ctx context.Context, username string) (*IdentifiedUser, error) {
	u, err := i.backend.ResolveUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if u == nil {
		return nil, nil
	}

	// Construct user from resolved result.
	user := &IdentifiedUser{
		sub: u.Subject(),

		username: u.Username(),

		backend: i.backend,

		claims: u.BackendClaims(),

		lockedScopes: u.RequiredScopes(),
	}

	return user, nil
}

func (i *Identifier) updateUser(ctx context.Context, user *IdentifiedUser, externalAuthority *authorities.Details) error {
	var userID string
	identityClaims := user.Claims()
	if userIDString, ok := identityClaims[konnect.IdentifiedUserIDClaim]; ok {
		userID = userIDString.(string)
	}
	if userID == "" {
		return errors.New("no id claim in user identity claims")
	}

	u, err := i.backend.GetUser(ctx, userID, user.sessionRef, nil)
	if err != nil {
		return err
	}

	if uwp, ok := u.(identity.UserWithProfile); ok {
		user.displayName = uwp.Name()
	}

	user.backend = i.backend
	user.externalAuthority = externalAuthority

	return nil
}
