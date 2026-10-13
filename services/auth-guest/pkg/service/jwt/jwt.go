// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidSession = errors.New("invalid session")

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

// Verify returns the share id of a session token, ignoring its expiry so an
// expired session can still be used to renew the guest link.
func (m *JwtService) Verify(tokenString string) (string, error) {
	claims := &jwtClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithoutClaimsValidation())
	if err != nil || !token.Valid || claims.ShareID == "" {
		return "", ErrInvalidSession
	}

	return claims.ShareID, nil
}
