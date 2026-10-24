// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package guard

import (
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestBackoffLimiterExponentialBackoff(t *testing.T) {
	now := time.Now()
	limiter := NewBackoffLimiter(config.BackoffLimiter{MaxAttempts: 5, BackoffBase: time.Second, BackoffCap: 30 * time.Second})
	var s BackoffLimiterState
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}

	for i, w := range want {
		exhausted := limiter.Record(&s, now)
		assert.Equal(t, i+1, s.Count, "failure %d", i+1)
		assert.WithinDuration(t, now.Add(w), s.NotBefore, time.Millisecond, "failure %d", i+1)
		assert.Equal(t, i+1 >= 5, exhausted, "failure %d", i+1)
		assert.Equal(t, w, limiter.RetryAfter(&s, now), "retry after %d", i+1)
		assert.Zero(t, limiter.RetryAfter(&s, now.Add(w)), "retry after elapsed %d", i+1)
	}
}

func TestBackoffLimiterCap(t *testing.T) {
	now := time.Now()
	limiter := NewBackoffLimiter(config.BackoffLimiter{MaxAttempts: 10, BackoffBase: time.Second, BackoffCap: 4 * time.Second})
	var s BackoffLimiterState
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 4 * time.Second, 4 * time.Second}

	for i, w := range want {
		limiter.Record(&s, now)
		assert.WithinDuration(t, now.Add(w), s.NotBefore, time.Millisecond, "attempt %d", i+1)
	}
}

func TestBackoffLimiterReset(t *testing.T) {
	limiter := NewBackoffLimiter(config.BackoffLimiter{MaxAttempts: 5, BackoffBase: time.Second, BackoffCap: 30 * time.Second})
	s := BackoffLimiterState{
		Count:     3,
		NotBefore: time.Now().Add(time.Minute),
	}

	limiter.Reset(&s)
	assert.Zero(t, s.Count)
	assert.True(t, s.NotBefore.IsZero())
}
