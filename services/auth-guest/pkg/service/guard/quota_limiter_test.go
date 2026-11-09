// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package guard

import (
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestQuotaLimiterCooldownAndWindow(t *testing.T) {
	now := time.Now()
	limiter := NewQuotaLimiter(config.QuotaLimiter{DailyCap: 2, Cooldown: time.Minute, Window: time.Hour})
	var s QuotaLimiterState

	assert.Zero(t, limiter.RetryAfter(&s, now))

	limiter.Record(&s, now)
	assert.Equal(t, time.Minute, limiter.RetryAfter(&s, now))
	assert.Zero(t, limiter.RetryAfter(&s, now.Add(time.Minute)))

	limiter.Record(&s, now)
	assert.Equal(t, time.Hour, limiter.RetryAfter(&s, now))

	assert.Zero(t, limiter.RetryAfter(&s, now.Add(2*time.Hour)))
	limiter.Record(&s, now.Add(2*time.Hour))
	assert.Equal(t, 1, s.Count)
}

func TestQuotaLimiterCooldownOutlastsWindow(t *testing.T) {
	now := time.Now()
	limiter := NewQuotaLimiter(config.QuotaLimiter{DailyCap: 2, Cooldown: 2 * time.Hour, Window: time.Hour})
	var s QuotaLimiterState

	limiter.Record(&s, now)
	limiter.Record(&s, now)

	assert.Equal(t, 2*time.Hour, limiter.RetryAfter(&s, now))
}

func TestQuotaLimiterReset(t *testing.T) {
	limiter := NewQuotaLimiter(config.QuotaLimiter{DailyCap: 2, Cooldown: time.Minute, Window: time.Hour})
	s := QuotaLimiterState{
		Count:       3,
		NotBefore:   time.Now().Add(time.Minute),
		WindowStart: time.Now(),
	}

	limiter.Reset(&s)
	assert.Zero(t, s.Count)
	assert.True(t, s.NotBefore.IsZero())
	assert.True(t, s.WindowStart.IsZero())
}
