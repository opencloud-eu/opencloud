// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type jwtClaims struct {
	ShareID string `json:"permissionId"`
	jwt.RegisteredClaims
}

type JwtService struct {
	secret []byte
	ttl    time.Duration
}

func NewJwtService(secret string, ttl time.Duration) *JwtService {
	return &JwtService{secret: []byte(secret), ttl: ttl}
}

// Sign returns a signed jwt token for the given share.
func (m *JwtService) Sign(shareID string) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		ShareID: shareID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}
