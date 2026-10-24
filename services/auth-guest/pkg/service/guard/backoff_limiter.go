// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

// Package guard provides generic building blocks for restricting access to
// endpoints: a browser-bound nonce, a backoff limiter and a quota limiter. The
// state is designed to be persisted per guarded resource; the policies come
// from the service config package, so guard depends on it.
package guard

import (
	"time"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
)

const backoffFactor = 2

// BackoffLimiterState is the persisted state of a BackoffLimiter.
type BackoffLimiterState struct {
	Count     int       `json:"count,omitempty"`
	NotBefore time.Time `json:"notbefore,omitzero"`
}

// BackoffLimiter counts events, blocks once the configured maximum is reached
// and grows the wait between events exponentially.
type BackoffLimiter struct {
	maxAttempts int
	base        time.Duration
	cap         time.Duration
}

// NewBackoffLimiter returns a BackoffLimiter configured from cfg.
func NewBackoffLimiter(cfg config.BackoffLimiter) BackoffLimiter {
	return BackoffLimiter{maxAttempts: cfg.MaxAttempts, base: cfg.BackoffBase, cap: cfg.BackoffCap}
}

// RetryAfter returns the remaining wait before the next event is allowed, or 0.
func (l BackoffLimiter) RetryAfter(s *BackoffLimiterState, now time.Time) time.Duration {
	if s.NotBefore.After(now) {
		return s.NotBefore.Sub(now)
	}
	return 0
}

// Record records an event and reports whether the limit is exhausted.
func (l BackoffLimiter) Record(s *BackoffLimiterState, now time.Time) (exhausted bool) {
	s.Count++
	s.NotBefore = now.Add(delay(l.base, l.cap, s.Count))
	return s.Count >= l.maxAttempts
}

// Reset clears the persisted state.
func (l BackoffLimiter) Reset(s *BackoffLimiterState) {
	*s = BackoffLimiterState{}
}

// delay returns the wait before the next attempt after the given number of
// consecutive failures: min(base*2^(failures-1), maxWait).
func delay(base, maxWait time.Duration, failures int) time.Duration {
	wait := min(base, maxWait)
	for i := 1; i < failures; i++ {
		wait = min(wait*backoffFactor, maxWait)
	}
	return wait
}
