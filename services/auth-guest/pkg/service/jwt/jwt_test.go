// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseClaims(t *testing.T, m *JwtService, tokenString string) (*jwtClaims, error) {
	t.Helper()
	c := &jwtClaims{}
	_, err := jwt.ParseWithClaims(tokenString, c, func(*jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	return c, err
}

func TestSignAndParse(t *testing.T) {
	m := NewJwtService("test-secret", time.Hour)

	tok, err := m.Sign("share-id")
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	claims, err := parseClaims(t, m, tok)
	require.NoError(t, err)
	assert.Equal(t, "share-id", claims.ShareID)
}

func TestParseExpired(t *testing.T) {
	m := NewJwtService("test-secret", -time.Minute)

	tok, err := m.Sign("share-id")
	require.NoError(t, err)

	_, err = parseClaims(t, m, tok)
	assert.Error(t, err)
}
