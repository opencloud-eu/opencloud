// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package guard

import (
	"time"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
)

// QuotaLimiterState is the persisted state of a QuotaLimiter.
type QuotaLimiterState struct {
	Count       int       `json:"count,omitempty"`
	NotBefore   time.Time `json:"notbefore,omitzero"`
	WindowStart time.Time `json:"windowstart,omitzero"`
}

// QuotaLimiter allows a maximum number of events per window, with a minimum
// cooldown between them.
type QuotaLimiter struct {
	max      int
	cooldown time.Duration
	window   time.Duration
}

// NewQuotaLimiter returns a QuotaLimiter configured from cfg.
func NewQuotaLimiter(cfg config.QuotaLimiter) QuotaLimiter {
	return QuotaLimiter{max: cfg.DailyCap, cooldown: cfg.Cooldown, window: cfg.Window}
}

// RetryAfter returns the remaining wait before the next event is allowed, or 0.
func (l QuotaLimiter) RetryAfter(s *QuotaLimiterState, now time.Time) time.Duration {
	var wait time.Duration
	if l.window > 0 && now.Sub(s.WindowStart) < l.window && s.Count >= l.max {
		wait = l.window - now.Sub(s.WindowStart)
	}
	if s.NotBefore.After(now) {
		if w := s.NotBefore.Sub(now); w > wait {
			wait = w
		}
	}
	return wait
}

// Record records an event and reports whether the limit is exhausted.
func (l QuotaLimiter) Record(s *QuotaLimiterState, now time.Time) (exhausted bool) {
	if l.window > 0 && (s.WindowStart.IsZero() || now.Sub(s.WindowStart) >= l.window) {
		s.WindowStart = now
		s.Count = 0
	}
	s.Count++
	s.NotBefore = now.Add(l.cooldown)
	return s.Count >= l.max
}

// Reset clears the persisted state.
func (l QuotaLimiter) Reset(s *QuotaLimiterState) {
	*s = QuotaLimiterState{}
}
