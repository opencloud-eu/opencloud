// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.
//
// Modifications Copyright OpenSearch Contributors. See
// GitHub history for details.

// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package opensearchtransport

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"
)

// Compile-time interface compliance checks.
var (
	_ Policy             = (*PolicyChain)(nil)
	_ policyConfigurable = (*PolicyChain)(nil)
)

// NewPolicy creates a policy that tries sub-policies in order.
// Returns a PolicyChain that implements the Policy interface.
func NewPolicy(policies ...Policy) Policy {
	return &PolicyChain{policies: policies}
}

// Route tries each policy in sequence until one returns a NextHop with a connection.
func (r *PolicyChain) Route(ctx context.Context, req *http.Request) (NextHop, error) {
	for _, policy := range r.policies {
		// Quick check if policy is enabled before evaluation
		if !policy.IsEnabled() {
			continue
		}

		hop, err := policy.Eval(ctx, req)
		switch {
		case err != nil:
			return NextHop{}, err // Error occurred, stop
		case hop.Conn != nil:
			return hop, nil
		default:
			// hop.Conn == nil && err == nil: no match, try next policy
			continue
		}
	}

	return NextHop{}, ErrNoConnections // No policies matched
}

// OnSuccess reports that a request completed without transport errors.
//
// Transport success means the HTTP round-trip completed (regardless of
// status code). This does not reflect thread pool health (managed by
// the stats poller via applyPoolAIMD) or connection-level health checks
// (managed by checkDead).
//
// Marks the connection as healthy atomically. Each pool lazily resurrects
// the connection from its dead list during the next checkDead() cycle.
func (r *PolicyChain) OnSuccess(conn *Connection) {
	// Fast path is lock-free. See c.mu for details.
	if conn.deadSinceIsZero() {
		return
	}

	// Slow path: connection was dead and succeeded --try to mark healthy.
	conn.mu.Lock()

	// Double-check under write lock (another goroutine may have resurrected).
	if conn.deadSinceIsZero() {
		conn.mu.Unlock()
		return
	}

	// Skip draining connections (HTTP/2 stream reset quiescing).
	if conn.drainingQuiescingRemaining.Load() > 0 {
		conn.mu.Unlock()
		return
	}

	// Skip overload-demoted connections (stats poller manages lifecycle).
	if conn.loadConnState().lifecycle().has(lcOverloaded) {
		conn.mu.Unlock()
		return
	}

	conn.markAsHealthyWithLock()
	conn.mu.Unlock()
}

// OnFailure reports that a request suffered a transport-level failure
// (e.g., connection refused, EOF, TLS error, timeout).
//
// This does not handle HTTP-level errors like 429 or 503 --those are
// successful transports with error status codes, handled in Perform.
// Thread pool congestion is managed separately by the stats poller.
//
// Marks the connection as dead atomically. Each pool lazily evicts the
// connection from its ready list when Next() encounters it (handled by
// nextWithEviction -> evictExternallyDemotedWithLock in pool_selection.go).
func (r *PolicyChain) OnFailure(conn *Connection) error {
	// Pre-check without lock: lcUnknown connections are seed URLs that
	// haven't been discovered yet --never mark them dead.
	if conn.loadConnState().lifecycle().has(lcUnknown) {
		return nil
	}

	conn.mu.Lock()

	// Re-check under lock.
	if conn.loadConnState().lifecycle().has(lcUnknown) {
		conn.mu.Unlock()
		return nil
	}

	err := conn.casLifecycle(
		conn.loadConnState(), 0,
		lcDead|lcNeedsWarmup|lcNeedsHardware,
		lcReady|lcActive|lcStandby|lcOverloaded,
	)
	if err != nil {
		// CAS failed (lifecycle noop) --another goroutine already
		// transitioned this connection. Nothing more to do.
		conn.mu.Unlock()
		return nil //nolint:nilerr // intentional: casLifecycle noop is not a caller-visible error
	}
	conn.storeOverloadedAt(time.Time{})
	conn.markAsDeadWithLock()
	conn.mu.Unlock()

	return nil
}

// DiscoveryUpdate notifies all policies that node discovery has occurred.
// Each policy updates its own state and connection pools based on topology changes.
// Logs all errors encountered, but continues updating remaining policies.
// Returns the first error encountered.
func (r *PolicyChain) DiscoveryUpdate(added, removed, unchanged []*Connection) error {
	var firstError error
	hasEnabledPolicy := false

	// Update policies in reverse order (lowest level first) so dependencies work correctly
	for _, policy := range slices.Backward(r.policies) {
		if err := policy.DiscoveryUpdate(added, removed, unchanged); err != nil {
			if firstError == nil {
				firstError = err // Capture first error to return
			}
			// Log error if debug logging is enabled
			if dl := loadDebugLogger(); dl != nil {
				_ = dl.Logf("PolicyChain: policy DiscoveryUpdate failed: %v", err)
			}
		}

		// Cache if any policy is enabled
		if !hasEnabledPolicy && policy.IsEnabled() {
			hasEnabledPolicy = true
		}
	}

	psSetEnabled(&r.policyState, hasEnabledPolicy)
	return firstError
}

// CheckDead triggers health checks across all configured policies.
// The first policy performs actual health checks, subsequent policies sync their pools.
// Logs all errors encountered, but continues checking remaining policies.
// Returns the first error encountered.
func (r *PolicyChain) CheckDead(ctx context.Context, healthCheck HealthCheckFunc) error {
	var firstError error
	for _, policy := range r.policies {
		if err := policy.CheckDead(ctx, healthCheck); err != nil {
			if firstError == nil {
				firstError = err // Capture first error to return
			}
			// Log error if debug logging is enabled
			if dl := loadDebugLogger(); dl != nil {
				_ = dl.Logf("PolicyChain: policy CheckDead failed: %v", err)
			}
		}
	}
	return firstError
}

// RotateStandby delegates to all sub-policies, summing successful rotations.
func (r *PolicyChain) RotateStandby(ctx context.Context, count int) (int, error) {
	var (
		total int
		errs  []error
	)
	for _, policy := range r.policies {
		n, err := policy.RotateStandby(ctx, count)
		total += n
		if err != nil {
			errs = append(errs, err)
		}
	}
	return total, errors.Join(errs...)
}

// IsEnabled uses cached state for O(1) lookup (when PolicyChain is used as Policy).
func (r *PolicyChain) IsEnabled() bool {
	return psIsEnabled(r.policyState.Load())
}

// Eval tries each sub-policy in sequence until one returns a NextHop with a connection.
// Returns (NextHop{}, nil) only if all sub-policies return (NextHop{}, nil).
func (r *PolicyChain) Eval(ctx context.Context, req *http.Request) (NextHop, error) {
	if r.policyState.Load()&psEnvDisabled != 0 {
		return NextHop{}, nil
	}

	for _, policy := range r.policies {
		// Skip not-enabled policies, mirroring Route. Otherwise a nested chain
		// reached via Eval could serve an unverified zombie connection and mask
		// the seed fallback.
		if !policy.IsEnabled() {
			continue
		}

		hop, err := policy.Eval(ctx, req)
		switch {
		case err != nil:
			return NextHop{}, err
		case hop.Conn != nil:
			return hop, nil
		default:
			continue
		}
	}

	return NextHop{}, nil
}

// configurePolicySettings configures pool settings for all sub-policies.
func (r *PolicyChain) configurePolicySettings(config policyConfig) error {
	var firstError error

	// Configure all sub-policies
	for _, policy := range r.policies {
		if configurablePolicy, ok := policy.(policyConfigurable); ok {
			if err := configurablePolicy.configurePolicySettings(config); err != nil && firstError == nil {
				firstError = err
			}
		}
	}

	return firstError
}

// childPolicies returns the sub-policies for tree walking.
func (r *PolicyChain) childPolicies() []Policy {
	return r.policies
}

// smoothedMaxBucketForIndex walks the policy tree to find the index slot cache,
// looks up the slot, and returns the current smoothed max RTT bucket. Returns 0
// if no scoring data exists for the index.
func (r *PolicyChain) smoothedMaxBucketForIndex(indexName string) float64 {
	if indexName == "" {
		return 0
	}
	for _, p := range r.policies {
		if cache := findRouterCache(p); cache != nil {
			if slot := cache.slotFor(indexName); slot != nil {
				return slot.loadSmoothedMaxBucket()
			}
			// Cache found but no slot for this index. Since all scored
			// policies share the same cache, no point checking further.
			return 0
		}
	}
	return 0
}
