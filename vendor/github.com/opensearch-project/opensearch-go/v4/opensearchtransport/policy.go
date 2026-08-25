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
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

// HealthCheckFunc defines the signature for health check functions.
// The conn parameter provides the Connection being health-checked, allowing the function
// to read connection state (e.g., cluster health lifecycle bits) to choose the appropriate endpoint.
type HealthCheckFunc func(ctx context.Context, conn *Connection, url *url.URL) (*http.Response, error)

// policyConfig holds shared configuration for policy-owned connection pools.
type policyConfig struct {
	name                         string          // Pool identity for metrics/debug
	ctx                          context.Context //nolint:containedctx // Long-lived pool context, not a request context.
	resurrectTimeoutInitial      time.Duration
	resurrectTimeoutMax          time.Duration
	resurrectTimeoutFactorCutoff int
	minimumResurrectTimeout      time.Duration
	jitterScale                  float64
	serverMaxNewConnsPerSec      float64
	clientsPerServer             float64
	healthCheck                  HealthCheckFunc
	observer                     *ConnectionObserver // nil means no observer
	poolInfoReady                *atomic.Bool        // nil-safe; true once thread pool quorum is reached
	clusterSearchCwnd            *atomic.Int32       // nil-safe; cluster-wide search cwnd for MCSR

	// Standby pool configuration
	activeListCap          *int  // nil = auto-scale; non-nil = user-specified cap value
	standbyPromotionChecks int64 // consecutive health checks before standby->ready

	// includeDedicatedClusterManagers admits dedicated cluster managers into the
	// round-robin routing pool. Honored only by RoundRobinPolicy, the sole policy
	// that admits nodes irrespective of role.
	includeDedicatedClusterManagers bool

	// Metrics is always non-nil for a New()-constructed transport. Detailed
	// callback registration below is gated by metrics.detailedEnabled()
	// (EnableMetrics), not by nil-ness. Policies may register metric callbacks
	// during configurePolicySettings.
	metrics *metrics
}

// Policy is a routing strategy that selects a connection for a request.
// See [Eval] for return value semantics.
type Policy interface {
	// DiscoveryUpdate notifies the policy of topology changes from node discovery.
	// Called on every discovery cycle for all configured policies, regardless of
	// IsEnabled state. Parameters:
	//   - added: newly discovered nodes (nil if none)
	//   - removed: nodes that left since the last cycle (nil if none)
	//   - unchanged: nodes present in both the previous and current topology
	DiscoveryUpdate(added, removed, unchanged []*Connection) error

	// CheckDead health-checks dead connections and resurrects those that respond.
	// Leaf policies that own a connection pool perform the actual health check;
	// wrapper policies delegate to their children.
	CheckDead(ctx context.Context, healthCheck HealthCheckFunc) error

	// RotateStandby promotes standby connections to active by health-checking
	// each candidate. Returns the number of successful promotions and any error.
	// Blocks until count rotations complete or no candidates remain.
	RotateStandby(ctx context.Context, count int) (int, error)

	// IsEnabled performs a quick check if this policy can be evaluated.
	// This should use cached state for maximum performance.
	IsEnabled() bool

	// Eval evaluates the policy and returns a NextHop if applicable.
	// A NextHop with Conn != nil means "use this connection".
	// A zero-value NextHop (Conn == nil) with nil error means "no match, try next".
	Eval(ctx context.Context, req *http.Request) (NextHop, error)
}

// NextHop is the result of a routing decision. Returned by Policy.Eval and
// Router.Route with the selected connection and optional routing metadata.
//
// NOTE: This is intentionally a concrete struct rather than an interface.
// As a value type it is returned on the stack with zero heap allocation --
// critical since every HTTP request produces one. The PoolName field is a
// string literal (e.g., "search", "write") which does not allocate.
// Adding fields to the struct is always non-breaking.
type NextHop struct {
	Conn     *Connection
	PoolName string // Thread pool name for in-flight tracking; empty for non-scored routes.

	// MaxConcurrentShardRequests is the adaptive shard fan-out limit derived
	// from the selected connection's search pool congestion window. When > 0,
	// Perform() injects it as the max_concurrent_shard_requests query parameter
	// on search requests routed through a coordinator node. Zero means "not set"
	// (shard-exact routing, non-search request, or feature disabled).
	MaxConcurrentShardRequests int
}

// policyConfigurable is a package-internal interface for policies that need configuration.
// This allows injecting client-specific pool settings (like timeout settings) after policy creation.
type policyConfigurable interface {
	configurePolicySettings(config policyConfig) error
}

// createPoolFromConfig creates a new multiServerPool with the given configuration.
// This is a helper function for leaf policies that manage their own connection pools.
func createPoolFromConfig(config policyConfig) *multiServerPool {
	// Derive effective activeListCap from config pointer.
	var effectiveCap int
	if config.activeListCap != nil {
		effectiveCap = *config.activeListCap
	}

	pool := &multiServerPool{
		name:                         config.name,
		ctx:                          config.ctx,
		resurrectTimeoutInitial:      config.resurrectTimeoutInitial,
		resurrectTimeoutMax:          config.resurrectTimeoutMax,
		resurrectTimeoutFactorCutoff: config.resurrectTimeoutFactorCutoff,
		minimumResurrectTimeout:      config.minimumResurrectTimeout,
		jitterScale:                  config.jitterScale,
		serverMaxNewConnsPerSec:      config.serverMaxNewConnsPerSec,
		clientsPerServer:             config.clientsPerServer,
		activeListCapConfig:          config.activeListCap,
		standbyPromotionChecks:       config.standbyPromotionChecks,
	}
	pool.mu.activeListCap = effectiveCap
	pool.mu.healthCheck = config.healthCheck
	pool.mu.members = make(map[*Connection]struct{}, defaultMembersCapacity)
	if config.observer != nil {
		pool.observer.Store(config.observer)
	}
	return pool
}
