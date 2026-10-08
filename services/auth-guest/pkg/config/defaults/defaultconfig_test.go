// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package defaults

import (
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigChallengeAndRenew(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, defaultChallengeCookieName, cfg.Challenge.CookieName)
	assert.Equal(t, defaultChallengeTTL, cfg.Challenge.TTL)
	assert.Equal(t, defaultChallengeMaxAttempts, cfg.Challenge.MaxAttempts)
	assert.Equal(t, defaultChallengeBackoffBase, cfg.Challenge.BackoffBase)
	assert.Equal(t, defaultChallengeBackoffCap, cfg.Challenge.BackoffCap)
	assert.Equal(t, defaultRenewCooldown, cfg.Renew.Cooldown)
	assert.Equal(t, defaultRenewDailyCap, cfg.Renew.DailyCap)
}

func TestEnsureDefaultsFillsChallengeAndRenew(t *testing.T) {
	cfg := &config.Config{}

	EnsureDefaults(cfg)

	assert.Equal(t, defaultChallengeCookieName, cfg.Challenge.CookieName)
	assert.Equal(t, defaultChallengeTTL, cfg.Challenge.TTL)
	assert.Equal(t, defaultChallengeMaxAttempts, cfg.Challenge.MaxAttempts)
	assert.Equal(t, defaultChallengeBackoffBase, cfg.Challenge.BackoffBase)
	assert.Equal(t, defaultChallengeBackoffCap, cfg.Challenge.BackoffCap)
	assert.Equal(t, defaultRenewCooldown, cfg.Renew.Cooldown)
	assert.Equal(t, defaultRenewDailyCap, cfg.Renew.DailyCap)
}

func TestEnsureDefaultsKeepsCustomChallengeAndRenew(t *testing.T) {
	cfg := &config.Config{
		Challenge: config.Challenge{
			CookieName:  "custom-challenge",
			TTL:         2 * time.Minute,
			MaxAttempts: 3,
			BackoffBase: 2 * time.Second,
			BackoffCap:  10 * time.Second,
		},
		Renew: config.Renew{
			Cooldown: 30 * time.Second,
			DailyCap: 5,
		},
	}

	EnsureDefaults(cfg)

	assert.Equal(t, "custom-challenge", cfg.Challenge.CookieName)
	assert.Equal(t, 2*time.Minute, cfg.Challenge.TTL)
	assert.Equal(t, 3, cfg.Challenge.MaxAttempts)
	assert.Equal(t, 2*time.Second, cfg.Challenge.BackoffBase)
	assert.Equal(t, 10*time.Second, cfg.Challenge.BackoffCap)
	assert.Equal(t, 30*time.Second, cfg.Renew.Cooldown)
	assert.Equal(t, 5, cfg.Renew.DailyCap)
}
