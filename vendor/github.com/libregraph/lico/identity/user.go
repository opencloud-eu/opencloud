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
	"github.com/golang-jwt/jwt/v5"
)

// User defines a most simple user with an id defined as subject.
type User interface {
	Subject() string
}

// UserWithEmail is a User with Email.
type UserWithEmail interface {
	User
	Email() string
	EmailVerified() bool
}

// UserWithProfile is a User with Name.
type UserWithProfile interface {
	User
	Name() string
	FamilyName() string
	GivenName() string
}

// UserWithID is a User with a locally unique numeric id.
type UserWithID interface {
	User
	ID() int64
}

// UserWithUniqueID is a User with a unique string id.
type UserWithUniqueID interface {
	User
	UniqueID() string
}

// UserWithUsername is a User with an username different from subject.
type UserWithUsername interface {
	User
	Username() string
}

// UserWithClaims is a User with jwt claims.
type UserWithClaims interface {
	User
	Claims() jwt.MapClaims
}

// UserWithScopedClaims is a user with jwt claims bound to provided scopes.
type UserWithScopedClaims interface {
	User
	ScopedClaims(authorizedScopes map[string]bool) jwt.MapClaims
}

// UserWithSessionRef is a user which supports an underlaying session reference.
type UserWithSessionRef interface {
	User
	SessionRef() *string
}

// PublicUser is a user with a public Subject and a raw id.
type PublicUser interface {
	Subject() string
	Raw() string
}
