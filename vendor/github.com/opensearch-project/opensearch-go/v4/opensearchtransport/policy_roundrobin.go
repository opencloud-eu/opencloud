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
	_ Policy             = (*RoundRobinPolicy)(nil)
	_ policyConfigurable = (*RoundRobinPolicy)(nil)
	_ policyTyped        = (*RoundRobinPolicy)(nil)
	_ policyOverrider    = (*RoundRobinPolicy)(nil)
)

// RoundRobinPolicy implements round-robin routing across all available connections.
type RoundRobinPolicy struct {
	pool        *multiServerPool // Embedded connection pool for round-robin selection
	policyState atomic.Int32     // Bitfield: psEnabled|psDisabled|psEnvEnabled|psEnvDisabled

	// includeDCM admits dedicated cluster managers into the round-robin pool.
	// When false (default), dedicated cluster managers are not routed to. This
	// is the only routing policy that admits nodes irrespective of role, so it
	// is the only one that honors this setting; role-scoped policies exclude
	// dedicated cluster managers by their role criteria.
	includeDCM bool
}

func (p *RoundRobinPolicy) policyTypeName() string      { return policyTypeNameRoundRobin }
func (p *RoundRobinPolicy) setEnvOverride(enabled bool) { psSetEnvOverride(&p.policyState, enabled) }

// NewRoundRobinPolicy creates a new round-robin routing policy.
func NewRoundRobinPolicy() Policy {
	return &RoundRobinPolicy{
		pool: nil, // Will be created when policy settings are configured
	}
}

// configurePolicySettings configures pool settings for this policy (leaf policy - no sub-policies).
func (p *RoundRobinPolicy) configurePolicySettings(config policyConfig) error {
	p.includeDCM = config.includeDedicatedClusterManagers
	// Create pool with proper settings if we don't have one yet
	if p.pool == nil {
		config.name = policyTypeNameRoundRobin
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

// CheckDead performs actual health checks on dead connections and resurrects healthy ones.
// As the first policy, RoundRobinPolicy is responsible for actual HTTP health checks.
func (p *RoundRobinPolicy) CheckDead(ctx context.Context, healthCheck HealthCheckFunc) error {
	if p.pool == nil {
		return nil
	}

	return p.pool.checkDead(ctx, healthCheck)
}

// RotateStandby rotates standby connections into active in this policy's pool.
func (p *RoundRobinPolicy) RotateStandby(ctx context.Context, count int) (int, error) {
	if p.pool == nil {
		return 0, nil
	}

	return p.pool.rotateStandby(ctx, count)
}

// DiscoveryUpdate updates the internal connection pool based on cluster topology changes.
// Adds are processed before removes so that the pool is never empty during a topology
// change where old seed URLs are replaced by discovered node addresses.
func (p *RoundRobinPolicy) DiscoveryUpdate(added, removed, unchanged []*Connection) error {
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

	// Recalculate activeListCap and warmup parameters based on projected pool size.
	// Done before adds/removes so startWarmup calls use the correctly-scaled values.
	targetPoolSize := len(p.pool.mu.ready) + len(p.pool.mu.dead) + len(added) - len(removed)
	p.pool.recalculateWarmupParamsWithLock(targetPoolSize)

	// Add new connections based on their health status
	for _, conn := range added {
		// Dedicated cluster managers do not serve request traffic unless
		// explicitly included; they remain in the connection inventory for
		// discovery but are not admitted to the round-robin pool.
		if !p.includeDCM && conn.Roles.isDedicatedClusterManager() {
			continue
		}
		// Guard: skip if already a member of this pool.
		if _, exists := p.pool.mu.members[conn]; exists {
			continue
		}
		p.pool.mu.members[conn] = struct{}{}

		conn.mu.RLock()
		isHealthy := conn.isReady()
		conn.mu.RUnlock()

		if isHealthy {
			conn.mu.Lock()
			conn.casLifecycle(conn.loadConnState(), 0, lcActive, lcUnknown|lcStandby) //nolint:errcheck // lock held; only errLifecycleNoop possible
			conn.mu.Unlock()
			rounds, skip := p.pool.getWarmupParamsWithLock()
			conn.startWarmup(rounds, skip)
			p.pool.appendToReadyActiveWithLock(conn)

			continue
		}

		if err := conn.casLifecycle(conn.loadConnState(), 0, lcDead|lcNeedsWarmup, lcReady|lcActive|lcStandby|lcOverloaded); err != nil {
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("[roundrobin] casLifecycle failed for %s (lc=%s): %v; appending to dead anyway\n",
					conn.URL, conn.loadConnState().lifecycle(), err)
			}
		}
		p.pool.appendToDeadWithLock(conn)
	}
	if added != nil {
		p.pool.shuffleActiveWithLock()
		p.pool.enforceActiveCapWithLock()
	}

	// Remove old connections
	if removed != nil { //nolint:nestif // filtering ready and dead lists requires nested iteration
		removedMap := make(map[string]struct{}, len(removed))
		for _, conn := range removed {
			removedMap[conn.URL.String()] = struct{}{}
		}

		// Filter ready connections (both active and standby partitions)
		if len(removedMap) > 0 {
			activeCountBefore := p.pool.mu.activeCount

			ready := p.pool.mu.ready[:0]
			activeCount := 0
			for i, conn := range p.pool.mu.ready {
				if _, found := removedMap[conn.URL.String()]; !found {
					ready = append(ready, conn)
					if i < p.pool.mu.activeCount {
						activeCount++
					}
				} else {
					delete(p.pool.mu.members, conn)
				}
			}
			p.pool.mu.ready = ready
			p.pool.mu.activeCount = activeCount

			// Filter dead connections
			dead := p.pool.mu.dead[:0]
			for _, conn := range p.pool.mu.dead {
				if _, found := removedMap[conn.URL.String()]; !found {
					dead = append(dead, conn)
				} else {
					delete(p.pool.mu.members, conn)
				}
			}
			p.pool.mu.dead = dead

			// If removal shrunk the active partition and standby exists,
			// schedule graceful (warmed) promotions to fill the gap.
			gap := activeCountBefore - p.pool.mu.activeCount
			p.pool.promoteStandbyGracefullyWithLock(p.pool.poolCtx(), gap)
		}
	}

	// Update cached enabled state. A policy is enabled only when it has a
	// connection worth routing to -- a ready connection, or a dead connection
	// currently confirmed reachable (lcNeedsHardware clear; see
	// availableForRouting). Dead connections not currently confirmed reachable
	// (freshly discovered, or verified-then-failed, possibly unroutable) do not
	// count, so the request cascades to the seed fallback instead of being
	// served as a zombie.
	psSetEnabled(&p.policyState, p.pool.hasAvailableConnsWithLock())

	return nil
}

// IsEnabled uses cached state to quickly determine if connections are available.
func (p *RoundRobinPolicy) IsEnabled() bool {
	return psIsEnabled(p.policyState.Load())
}

// Eval returns a NextHop from the round-robin pool.
func (p *RoundRobinPolicy) Eval(ctx context.Context, req *http.Request) (NextHop, error) {
	if p.policyState.Load()&psEnvDisabled != 0 {
		return NextHop{}, nil
	}

	if p.pool == nil {
		return NextHop{}, nil
	}

	// Respect the enabled bit from DiscoveryUpdate. With nothing available for
	// routing, pool.Next() would hand back a zombie from the dead list -- a
	// transport error, not ErrNoConnections -- masking the seed fallback.
	// Mirrors the gate in CoordinatorPolicy.Eval and RolePolicy.Eval.
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
func (p *RoundRobinPolicy) PolicySnapshot() PolicySnapshot {
	if p.pool == nil {
		return PolicySnapshot{Name: policyTypeNameRoundRobin}
	}
	snap := p.pool.snapshot()
	snap.Enabled = psIsEnabled(p.policyState.Load())
	return snap
}
