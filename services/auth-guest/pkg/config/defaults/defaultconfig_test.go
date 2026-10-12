// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigGuard(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, defaultNonceCookieName, cfg.Nonce.CookieName)
	assert.Equal(t, defaultNonceTTL, cfg.Nonce.TTL)
	assert.Equal(t, defaultBackoffMaxAttempts, cfg.BackoffLimiter.MaxAttempts)
	assert.Equal(t, defaultBackoffBase, cfg.BackoffLimiter.BackoffBase)
	assert.Equal(t, defaultBackoffCap, cfg.BackoffLimiter.BackoffCap)
	assert.Equal(t, defaultQuotaCooldown, cfg.QuotaLimiter.Cooldown)
	assert.Equal(t, defaultQuotaWindow, cfg.QuotaLimiter.Window)
	assert.Equal(t, defaultQuotaDailyCap, cfg.QuotaLimiter.DailyCap)
}

func TestEnsureDefaultsFillsGuard(t *testing.T) {
	cfg := &config.Config{}

	EnsureDefaults(cfg)

	assert.Equal(t, defaultNonceCookieName, cfg.Nonce.CookieName)
	assert.Equal(t, defaultNonceTTL, cfg.Nonce.TTL)
	assert.Equal(t, defaultBackoffMaxAttempts, cfg.BackoffLimiter.MaxAttempts)
	assert.Equal(t, defaultBackoffBase, cfg.BackoffLimiter.BackoffBase)
	assert.Equal(t, defaultBackoffCap, cfg.BackoffLimiter.BackoffCap)
	assert.Equal(t, defaultQuotaCooldown, cfg.QuotaLimiter.Cooldown)
	assert.Equal(t, defaultQuotaWindow, cfg.QuotaLimiter.Window)
	assert.Equal(t, defaultQuotaDailyCap, cfg.QuotaLimiter.DailyCap)
}

func TestEnsureDefaultsKeepsCustomGuard(t *testing.T) {
	cfg := &config.Config{
		Nonce: config.Nonce{
			CookieName: "custom-nonce",
			TTL:        2 * time.Minute,
		},
		BackoffLimiter: config.BackoffLimiter{
			MaxAttempts: 3,
			BackoffBase: 2 * time.Second,
			BackoffCap:  10 * time.Second,
		},
		QuotaLimiter: config.QuotaLimiter{
			Cooldown: 30 * time.Second,
			Window:   12 * time.Hour,
			DailyCap: 5,
		},
	}

	EnsureDefaults(cfg)

	assert.Equal(t, "custom-nonce", cfg.Nonce.CookieName)
	assert.Equal(t, 2*time.Minute, cfg.Nonce.TTL)
	assert.Equal(t, 3, cfg.BackoffLimiter.MaxAttempts)
	assert.Equal(t, 2*time.Second, cfg.BackoffLimiter.BackoffBase)
	assert.Equal(t, 10*time.Second, cfg.BackoffLimiter.BackoffCap)
	assert.Equal(t, 30*time.Second, cfg.QuotaLimiter.Cooldown)
	assert.Equal(t, 12*time.Hour, cfg.QuotaLimiter.Window)
	assert.Equal(t, 5, cfg.QuotaLimiter.DailyCap)
}
