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

package oidc

import (
	"github.com/golang-jwt/jwt/v5"
)

// IDTokenClaims define the claims found in OIDC ID Tokens.
type IDTokenClaims struct {
	jwt.RegisteredClaims

	Nonce           string `json:"nonce,omitempty"`
	AuthTime        int64  `json:"auth_time,omitempty"`
	AccessTokenHash string `json:"at_hash,omitempty"`
	CodeHash        string `json:"c_hash,omitempty"`

	*ProfileClaims
	*EmailClaims

	*SessionClaims
}

// ProfileClaims define the claims for the OIDC profile scope.
// https://openid.net/specs/openid-connect-basic-1_0.html#Scopes
type ProfileClaims struct {
	jwt.RegisteredClaims
	Name              string `json:"name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
	GivenName         string `json:"given_name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
}

// NewProfileClaims return a new ProfileClaims set from the provided
// jwt.Claims or nil.
func NewProfileClaims(claims jwt.Claims) *ProfileClaims {
	if claims == nil {
		return nil
	}

	return claims.(*ProfileClaims)
}

// EmailClaims define the claims for the OIDC email scope.
// https://openid.net/specs/openid-connect-basic-1_0.html#Scopes
type EmailClaims struct {
	jwt.RegisteredClaims
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified"`
}

// NewEmailClaims return a new EmailClaims set from the provided
// jwt.Claims or nil.
func NewEmailClaims(claims jwt.Claims) *EmailClaims {
	if claims == nil {
		return nil
	}

	return claims.(*EmailClaims)
}

// UserInfoClaims define the claims defined by the OIDC UserInfo
// endpoint.
type UserInfoClaims struct {
	Subject string `json:"sub,omitempty"`
}

// SessionClaims define claims related to front end sessions, for example as
// specified by https://openid.net/specs/openid-connect-frontchannel-1_0.html
type SessionClaims struct {
	SessionID string `json:"sid,omitempty"`
}
