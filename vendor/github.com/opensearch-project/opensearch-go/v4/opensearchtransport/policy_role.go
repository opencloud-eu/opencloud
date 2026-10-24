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
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
)

// Compile-time interface compliance checks
var (
	_ Policy             = (*RolePolicy)(nil)
	_ policyConfigurable = (*RolePolicy)(nil)
	_ policyTyped        = (*RolePolicy)(nil)
	_ policyOverrider    = (*RolePolicy)(nil)
)

const (
	// RoleSeparator is used to join multiple roles into a single key
	RoleSeparator = ","
)

// InvalidRoleError indicates a role name contains invalid characters.
type InvalidRoleError struct {
	Role      string
	Separator string
}

func (e InvalidRoleError) Error() string {
	return fmt.Sprintf("role name cannot contain %q: %q", e.Separator, e.Role)
}

// NormalizeRoles deduplicates, sorts, and joins roles into a canonical key.
// This ensures consistent key generation across selectors and discovery.
// Validates that role names don't contain the separator to ensure clean key space.
func NormalizeRoles(roles []string) (string, error) {
	if len(roles) == 0 {
		return "", nil
	}

	// Deduplicate roles using a map and validate no separator chars
	uniqueRoles := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		if strings.Contains(role, RoleSeparator) {
			// Invalid role name - contains our separator
			return "", InvalidRoleError{Role: role, Separator: RoleSeparator}
		}
		uniqueRoles[role] = struct{}{}
	}

	if len(uniqueRoles) == 0 {
		return "", nil
	}

	// Convert to sorted slice
	sortedRoles := make([]string, 0, len(uniqueRoles))
	for role := range uniqueRoles {
		sortedRoles = append(sortedRoles, role)
	}
	slices.Sort(sortedRoles)

	// Join with separator for readable keys
	return strings.Join(sortedRoles, RoleSeparator), nil
}

// RolePolicy implements routing based on required node roles.
type RolePolicy struct {
	requiredRoleKey string           // Normalized role key for this policy
	pool            *multiServerPool // Single pool for connections matching required roles
	policyState     atomic.Int32     // Bitfield: psEnabled|psDisabled|psEnvEnabled|psEnvDisabled
}

func (p *RolePolicy) policyTypeName() string      { return policyTypeNameRole }
func (p *RolePolicy) setEnvOverride(enabled bool) { psSetEnvOverride(&p.policyState, enabled) }

// NewRolePolicy creates a new role-based routing policy.
// Routes requests only to nodes that have ALL of the specified roles.
//
// Common role combinations:
//   - "data" - nodes that can store and search data
//   - "ingest" - nodes that can process documents before indexing
//   - "cluster_manager" - nodes that can manage cluster state (avoid for client requests)
//   - RoleCoordinatingOnly - dedicated client/coordinating nodes (empty roles)
//
// Multiple roles require nodes to have ALL specified roles:
//
//	NewRolePolicy("data", "ingest") // nodes with BOTH data AND ingest roles
//
// Returns InvalidRoleError if role names contain the RoleSeparator character.
// Use with NewRouter() for policy chaining and fallback behavior.
func NewRolePolicy(roles ...string) (Policy, error) {
	if len(roles) == 0 {
		return nil, InvalidRoleError{Role: "<empty>", Separator: "no roles specified"}
	}

	// Normalize roles (deduplicate, sort, validate)
	roleKey, err := NormalizeRoles(roles)
	if err != nil {
		return nil, err
	}

	return &RolePolicy{
		requiredRoleKey: roleKey,
		pool:            nil, // Will be created when policy settings are configured
	}, nil
}

// RequiredRoleKey returns the normalized role key for this policy.
func (p *RolePolicy) RequiredRoleKey() string {
	return p.requiredRoleKey
}

// configurePolicySettings configures pool settings for this policy (leaf policy - no sub-policies).
func (p *RolePolicy) configurePolicySettings(config policyConfig) error {
	// Create pool with proper settings if we don't have one yet
	if p.pool == nil {
		config.name = "role:" + p.requiredRoleKey
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

// DiscoveryUpdate updates the role-based connection pools based on cluster topology changes.
// Adds are processed before removes so that the pool is never empty during a topology
// change where old seed URLs are replaced by discovered node addresses.
func (p *RolePolicy) DiscoveryUpdate(added, removed, unchanged []*Connection) error {
	if p.policyState.Load()&psEnvDisabled != 0 {
		return nil
	}

	// Compute projected pool size for warmup/activeListCap scaling and
	// recalculate the warmup parameters under the pool write lock. The lock is
	// required because recalculateWarmupParamsWithLock writes the pool's warmupRounds,
	// warmupSkipCount, and activeListCap fields, which getWarmupParamsWithLock and the
	// other DiscoveryUpdate callers (roundrobin, cluster_coordinator) read and
	// write under the same lock. Done before the mutations below so startWarmup
	// calls during discoveryUpdateAdd use the new values. The lock is released
	// before discoveryUpdateAdd/Remove, which acquire it per-connection.
	p.pool.Lock()
	// Only count connections that match this policy's required roles.
	targetPoolSize := len(p.pool.mu.ready) + len(p.pool.mu.dead)
	for _, conn := range added {
		if p.connectionMatchesRoles(conn) {
			targetPoolSize++
		}
	}
	for _, conn := range removed {
		if p.connectionMatchesRoles(conn) {
			targetPoolSize--
		}
	}
	p.pool.recalculateWarmupParamsWithLock(targetPoolSize)
	p.pool.Unlock()

	if added != nil {
		p.discoveryUpdateAdd(added)
	}

	if removed != nil {
		p.discoveryUpdateRemove(removed)
	}

	// Recompute the cached enabled bit unconditionally, mirroring RoundRobinPolicy
	// and CoordinatorPolicy. The discoveryUpdateAdd/Remove helpers only recompute
	// when they touch the pool, so an unchanged-only cycle (the steady state)
	// would otherwise leave the bit stale after a health check flips a
	// connection's routability (dead+lcNeedsHardware -> ready) off the discovery
	// path. Without this, a freshly verified node never becomes routable and
	// traffic pins to the seed fallback.
	p.pool.Lock()
	psSetEnabled(&p.policyState, p.pool.hasAvailableConnsWithLock())
	p.pool.Unlock()

	// unchanged connections don't need any action
	return nil
}

// discoveryUpdateAdd handles adding new connections that match this policy's required roles.
func (p *RolePolicy) discoveryUpdateAdd(added []*Connection) {
	for _, conn := range added {
		if p.connectionMatchesRoles(conn) {
			// Establish consistent lock ordering: Pool -> Connection
			p.pool.Lock()

			// Guard: skip if this connection is already a member of this pool.
			// Multiple poolRouters sharing the same inner RolePolicy
			// each propagate DiscoveryUpdate independently, so the same *Connection
			// can arrive here N times per discovery cycle.
			if _, exists := p.pool.mu.members[conn]; exists {
				p.pool.Unlock()
				continue
			}

			conn.mu.RLock()

			// Determine health from lifecycle bits, not timestamps.
			// Multiple pools share the same *Connection, so deadSince is
			// unreliable -- one pool's appendToDeadWithLock may set it before
			// another pool processes the same connection.
			isHealthy := conn.isReady()

			// Release conn.mu before pool operations that may write-lock connections
			// in the ready list (enforceActiveCapWithLock). Since conn is about to be
			// appended to pool.mu.ready, holding conn.mu.RLock here while
			// enforceActiveCapWithLock iterates pool.mu.ready and calls Lock() on each
			// entry would self-deadlock if conn is the eviction target.
			conn.mu.RUnlock()

			// Record membership before appending to pool lists.
			p.pool.mu.members[conn] = struct{}{}

			if isHealthy {
				// Add healthy connection to active partition with warmup
				conn.mu.Lock()
				conn.casLifecycle(conn.loadConnState(), 0, lcActive, lcUnknown|lcStandby) //nolint:errcheck // lock held; only errLifecycleNoop possible
				conn.mu.Unlock()
				rounds, skip := p.pool.getWarmupParamsWithLock()
				conn.startWarmup(rounds, skip)
				p.pool.appendToReadyActiveWithLock(conn)
				p.pool.shuffleActiveWithLock()
				p.pool.enforceActiveCapWithLock()
			} else {
				// Add unhealthy connection to dead list - will be health checked later
				//nolint:errcheck // pool lock held; only errLifecycleNoop possible
				conn.casLifecycle(
					conn.loadConnState(), 0,
					lcDead|lcNeedsWarmup,
					lcReady|lcActive|lcStandby|lcOverloaded,
				)
				p.pool.appendToDeadWithLock(conn)
			}

			p.pool.Unlock()
		}
	}
}

// discoveryUpdateRemove handles removing connections from this policy's pool.
func (p *RolePolicy) discoveryUpdateRemove(removed []*Connection) {
	// Build map of removed connection URLs for O(1) lookup
	removedConns := make(map[string]struct{}, len(removed))
	for _, node := range removed {
		removedConns[node.URL.String()] = struct{}{}
	}

	// First check if we have any connections to remove (using RLock)
	p.pool.RLock()
	hasConnectionsToRemove := false
	for _, conn := range p.pool.mu.ready {
		if _, found := removedConns[conn.URL.String()]; found {
			hasConnectionsToRemove = true
			break
		}
	}
	if !hasConnectionsToRemove {
		for _, conn := range p.pool.mu.dead {
			if _, found := removedConns[conn.URL.String()]; found {
				hasConnectionsToRemove = true
				break
			}
		}
	}
	p.pool.RUnlock()

	// If no connections to remove, exit early
	if !hasConnectionsToRemove {
		return
	}

	// Now acquire write lock and actually remove connections
	p.pool.Lock()
	beforeReadyCount := len(p.pool.mu.ready)
	beforeDeadCount := len(p.pool.mu.dead)
	activeCountBefore := p.pool.mu.activeCount

	// Filter ready connections, tracking active count
	{
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
	}

	// Filter dead connections
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

	afterReadyCount := len(p.pool.mu.ready)
	afterDeadCount := len(p.pool.mu.dead)

	if beforeReadyCount != afterReadyCount || beforeDeadCount != afterDeadCount {
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("RolePolicy[%s]: Removed connections (ready: %d->%d, dead: %d->%d)\n",
				p.requiredRoleKey, beforeReadyCount, afterReadyCount, beforeDeadCount, afterDeadCount)
		}
	}

	// If removal shrunk the active partition and standby exists,
	// schedule graceful (warmed) promotions to fill the gap.
	gap := activeCountBefore - p.pool.mu.activeCount
	p.pool.promoteStandbyGracefullyWithLock(p.pool.poolCtx(), gap)

	p.pool.Unlock()
}

// connectionMatchesRoles checks if a connection matches this policy's required roles.
func (p *RolePolicy) connectionMatchesRoles(conn *Connection) bool {
	if p.requiredRoleKey == "" {
		return false
	}

	// Handle coordinating-only nodes - check both empty roles and explicit coordinating_only role
	if p.requiredRoleKey == RoleCoordinatingOnly {
		return len(conn.Roles) == 0 || conn.Roles.has(RoleCoordinatingOnly)
	}

	// For regular roles, check if connection has all required roles
	requiredRoles := strings.SplitSeq(p.requiredRoleKey, RoleSeparator)
	for requiredRole := range requiredRoles {
		if !conn.Roles.has(requiredRole) {
			return false
		}
	}
	return true
}

// IsEnabled uses cached state to quickly determine if matching roles are available.
func (p *RolePolicy) IsEnabled() bool {
	return psIsEnabled(p.policyState.Load())
}

// Eval returns a NextHop from this role-based policy's pool.
// Returns (NextHop{}, nil) if no matching connections are found (allows fallthrough).
func (p *RolePolicy) Eval(ctx context.Context, req *http.Request) (NextHop, error) {
	if p.policyState.Load()&psEnvDisabled != 0 {
		return NextHop{}, nil
	}

	if p.policyState.Load()&psEnabled != 0 {
		conn, err := p.pool.Next()
		if err != nil {
			return NextHop{}, err
		}
		return NextHop{Conn: conn}, nil
	}
	return NextHop{}, nil
}

// CheckDead syncs the pool based on Connection.mu.isDead state.
func (p *RolePolicy) CheckDead(ctx context.Context, healthCheck HealthCheckFunc) error {
	return p.pool.checkDead(ctx, healthCheck)
}

// RotateStandby rotates standby connections into active in this policy's pool.
func (p *RolePolicy) RotateStandby(ctx context.Context, count int) (int, error) {
	return p.pool.rotateStandby(ctx, count)
}

// PolicySnapshot returns a point-in-time snapshot of this policy's pool.
func (p *RolePolicy) PolicySnapshot() PolicySnapshot {
	if p.pool == nil {
		return PolicySnapshot{Name: "role:" + p.requiredRoleKey}
	}
	snap := p.pool.snapshot()
	snap.Enabled = psIsEnabled(p.policyState.Load())
	return snap
}

// mustRolePolicy creates a new role-based policy or panics if creation fails.
// This is a helper function for creating built-in policies with known-valid roles.
func mustRolePolicy(role string) Policy {
	policy, err := NewRolePolicy(role)
	if err != nil {
		panic(fmt.Sprintf("failed to create role policy for %q: %v", role, err))
	}
	return policy
}
