// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionSecret(t *testing.T) {
	cfg := &Config{TokenManager: &TokenManager{JWTSecret: "reva-secret"}}
	other := &Config{TokenManager: &TokenManager{JWTSecret: "other-secret"}}

	assert.Len(t, cfg.SessionSecret(), 64)
	assert.Equal(t, cfg.SessionSecret(), cfg.SessionSecret(), "derivation must be deterministic")
	assert.NotEqual(t, cfg.TokenManager.JWTSecret, cfg.SessionSecret())
	assert.NotEqual(t, cfg.SessionSecret(), other.SessionSecret())
}

func TestSessionSecretMissingJWTSecret(t *testing.T) {
	assert.Empty(t, (&Config{}).SessionSecret(), "nil token manager")
	assert.Empty(t, (&Config{TokenManager: &TokenManager{}}).SessionSecret(), "empty jwt secret")
}
