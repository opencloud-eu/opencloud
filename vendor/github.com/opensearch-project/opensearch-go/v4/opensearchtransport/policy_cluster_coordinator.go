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
	"sync/atomic"
)

// Compile-time interface compliance checks
var (
	_ Policy             = (*CoordinatorPolicy)(nil)
	_ policyConfigurable = (*CoordinatorPolicy)(nil)
	_ policyTyped        = (*CoordinatorPolicy)(nil)
	_ policyOverrider    = (*CoordinatorPolicy)(nil)
)

// CoordinatorPolicy implements routing to coordinating-only nodes.
type CoordinatorPolicy struct {
	pool        *multiServerPool // Pool of coordinating-only connections
	policyState atomic.Int32     // Bitfield: psEnabled|psDisabled|psEnvEnabled|psEnvDisabled
}

func (p *CoordinatorPolicy) policyTypeName() string      { return policyTypeNameCoordinator }
func (p *CoordinatorPolicy) setEnvOverride(enabled bool) { psSetEnvOverride(&p.policyState, enabled) }

// NewCoordinatorPolicy creates a policy that routes to coordinating-only nodes.
func NewCoordinatorPolicy() Policy {
	return &CoordinatorPolicy{
		pool: nil, // Will be created when policy settings are configured
	}
}

// configurePolicySettings configures pool settings for this policy (leaf policy - no sub-policies).
func (p *CoordinatorPolicy) configurePolicySettings(config policyConfig) error {
	// Create pool with proper settings if we don't have one yet
	if p.pool == nil {
		config.name = policyTypeNameCoordinator
		p.pool = createPoolFromConfig(config)
	}
	if config.metrics.detailedEnabled() {
		config.metrics.policyCallbacks = append(config.metrics.policyCallbacks,
			func() (PolicySnapshot, error) {
				return p.PolicySnapshot(), nil
			})
	}
	return nil
}

// CheckDead syncs the pool based on Connection.mu.isDead state.
// Subsequent policies just sync their pools without doing actual health checks.
func (p *CoordinatorPolicy) CheckDead(ctx context.Context, healthCheck HealthCheckFunc) error {
	if p.pool == nil {
		return nil
	}

	return p.pool.checkDead(ctx, healthCheck)
}

// RotateStandby rotates standby connections into active in this policy's pool.
func (p *CoordinatorPolicy) RotateStandby(ctx context.Context, count int) (int, error) {
	if p.pool == nil {
		return 0, nil
	}

	return p.pool.rotateStandby(ctx, count)
}

// isCoordinatingOnly returns true if the connection is a coordinating-only node.
// Matches both nodes with an explicit "coordinating_only" role and nodes with
// an empty role set (which OpenSearch uses for coordinating-only nodes).
func isCoordinatingOnly(conn *Connection) bool {
	return len(conn.Roles) == 0 || conn.Roles.has(RoleCoordinatingOnly)
}

// DiscoveryUpdate updates the coordinating-only connection pool based on cluster topology changes.
// Adds are processed before removes so that the pool is never empty during a topology
// change where old seed URLs are replaced by discovered node addresses.
func (p *CoordinatorPolicy) DiscoveryUpdate(added, removed, unchanged []*Connection) error {
	if p.policyState.Load()&psEnvDisabled != 0 {
		return nil
	}

	if p.pool == nil {
		return nil
	}

	// Short-circuit if no changes
	if added == nil && removed == nil {
		return nil
	}

	p.pool.Lock()
	defer p.pool.Unlock()

	// Calculate projected pool size and recalculate warmup parameters.
	targetPoolSize := len(p.pool.mu.ready) + len(p.pool.mu.dead)
	for _, conn := range added {
		if isCoordinatingOnly(conn) {
			targetPoolSize++
		}
	}
	for _, conn := range removed {
		if isCoordinatingOnly(conn) {
			targetPoolSize--
		}
	}
	p.pool.recalculateWarmupParamsWithLock(targetPoolSize)

	// Add new coordinating-only connections
	for _, newConn := range added {
		if !isCoordinatingOnly(newConn) {
			continue
		}

		// Guard: skip if already a member of this pool.
		if _, exists := p.pool.mu.members[newConn]; exists {
			continue
		}
		p.pool.mu.members[newConn] = struct{}{}

		newConn.mu.RLock()
		isHealthy := newConn.isReady()
		newConn.mu.RUnlock()

		if isHealthy {
			newConn.mu.Lock()
			newConn.casLifecycle(newConn.loadConnState(), 0, lcActive, lcUnknown|lcStandby) //nolint:errcheck,lll // lock held; only errLifecycleNoop possible
			newConn.mu.Unlock()
			rounds, skip := p.pool.getWarmupParamsWithLock()
			newConn.startWarmup(rounds, skip)
			p.pool.appendToReadyActiveWithLock(newConn)
		} else {
			//nolint:errcheck // pool lock held; only errLifecycleNoop possible
			newConn.casLifecycle(
				newConn.loadConnState(), 0,
				lcDead|lcNeedsWarmup,
				lcReady|lcActive|lcStandby|lcOverloaded,
			)
			p.pool.appendToDeadWithLock(newConn)
		}
	}
	if added != nil {
		p.pool.shuffleActiveWithLock()
		p.pool.enforceActiveCapWithLock()
	}

	// Remove old connections
	if removed != nil { //nolint:nestif // filtering ready and dead lists requires nested iteration
		removedConns := make(map[string]struct{}, len(removed))
		for _, node := range removed {
			removedConns[node.URL.String()] = struct{}{}
		}

		// Remove connections from ready list, tracking active count
		{
			activeCountBefore := p.pool.mu.activeCount

			filtered := p.pool.mu.ready[:0]
			activeCount := 0
			for i, conn := range p.pool.mu.ready {
				if _, found := removedConns[conn.URL.String()]; !found {
					filtered = append(filtered, conn)
					if i < p.pool.mu.activeCount {
						activeCount++
					}
				} else {
					delete(p.pool.mu.members, conn)
				}
			}
			p.pool.mu.ready = filtered
			p.pool.mu.activeCount = activeCount

			// If removal shrunk the active partition and standby exists,
			// schedule graceful (warmed) promotions to fill the gap.
			gap := activeCountBefore - p.pool.mu.activeCount
			p.pool.promoteStandbyGracefullyWithLock(p.pool.poolCtx(), gap)
		}

		// Remove connections from dead list
		{
			filtered := p.pool.mu.dead[:0]
			for _, conn := range p.pool.mu.dead {
				if _, found := removedConns[conn.URL.String()]; !found {
					filtered = append(filtered, conn)
				} else {
					delete(p.pool.mu.members, conn)
				}
			}
			p.pool.mu.dead = filtered
		}
	}

	// Update cached state. Dead-but-never-verified connections do not count as
	// available so the request cascades to the seed fallback rather than being
	// served as a zombie.
	psSetEnabled(&p.policyState, p.pool.hasAvailableConnsWithLock())

	return nil
}

// IsEnabled uses cached state to quickly determine if coordinating-only nodes are available.
func (p *CoordinatorPolicy) IsEnabled() bool {
	return psIsEnabled(p.policyState.Load())
}

// Eval attempts to route to coordinating-only nodes.
// Returns (NextHop{}, nil) if no coordinating-only nodes are available (allows fallthrough).
func (p *CoordinatorPolicy) Eval(ctx context.Context, req *http.Request) (NextHop, error) {
	if p.policyState.Load()&psEnvDisabled != 0 {
		return NextHop{}, nil
	}

	if p.policyState.Load()&psEnabled == 0 {
		return NextHop{}, nil
	}

	conn, err := p.pool.Next()
	if err != nil {
		return NextHop{}, err
	}
	return NextHop{Conn: conn}, nil
}

// PolicySnapshot returns a point-in-time snapshot of this policy's pool.
func (p *CoordinatorPolicy) PolicySnapshot() PolicySnapshot {
	if p.pool == nil {
		return PolicySnapshot{Name: policyTypeNameCoordinator}
	}
	snap := p.pool.snapshot()
	snap.Enabled = psIsEnabled(p.policyState.Load())
	return snap
}
