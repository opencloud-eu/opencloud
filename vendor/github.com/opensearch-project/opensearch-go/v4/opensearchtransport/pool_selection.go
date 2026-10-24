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

// Next returns a connection from pool, or an error.
//
// Selection priority:
//  1. Active connections (round-robin over ready[:activeCount])
//  2. Standby connections (ready[activeCount:], moved to active on use)
//  3. Dead connections (zombie mode -- rotated for retry)
//
// Warming connections participate in round-robin but are skipped more often.
// The warmup skip count decrements on each selection, gradually increasing
// traffic to the connection. If all active connections are warming and skip,
// the one closest to its next accept is used (starvation prevention).
func (cp *multiServerPool) Next() (*Connection, error) {
	cp.mu.RLock()

	// Return next active connection using round-robin.
	// After selection, verify the connection still has a position bit (lcActive or lcStandby).
	// Another pool's health check may have externally set lcDead on the shared
	// Connection.state -- detect that here with a lock-free read. Connections in
	// lcStandby (from another pool's cap enforcement) are safe to serve -- each
	// pool manages its own active/standby partitions independently.
	if cp.mu.activeCount > 0 { //nolint:nestif // warmup skip/accept/starvation requires nested branching
		var bestWarmingConn *Connection
		bestWarmingRemSkip := int(^uint(0) >> 1) // max int
		var needsCapEnforce bool

		for attempt := range cp.mu.activeCount {
			conn, selectorCapEnforce := cp.getNextActiveConnWithLock()
			needsCapEnforce = needsCapEnforce || selectorCapEnforce

			if conn == nil {
				continue // selector error
			}

			// Dedicated cluster managers stay in the inventory for discovery but
			// do not serve request traffic. Skipping here means a pool of only
			// dedicated cluster managers exhausts its attempts and falls through
			// to the no-connection path below.
			if cp.excludeDCM && conn.Roles.isDedicatedClusterManager() {
				continue
			}
			state := conn.loadConnState()

			if state.lifecycle()&(lcActive|lcStandby) == 0 {
				// Externally killed (dead/overloaded) -- upgrade to write lock for eviction.
				// We only evict for truly dead connections (no position bits).
				// Standby state from another pool's cap enforcement is safe to ignore --
				// this pool manages its own active/standby partitions independently.
				cp.mu.RUnlock()
				return cp.nextWithEviction()
			}

			// Fast path: no warmup in progress (common case).
			if !state.isWarmingUp() {
				cp.mu.RUnlock()
				cp.poolRequests.Add(1)
				if needsCapEnforce {
					cp.triggerCapEnforcement()
				}
				return conn, nil
			}

			// Warmup in progress -- try skip/accept via CAS.
			switch conn.tryWarmupSkip() {
			case warmupAccepted:
				cp.poolWarmupAccepts.Add(1)
				// Check if warmup finished and cap enforcement is needed.
				warmupDone := !conn.loadConnState().isWarmingUp()
				if warmupDone && cp.mu.activeListCap > 0 && cp.mu.activeCount > cp.mu.activeListCap {
					needsCapEnforce = true
					if dl := loadDebugLogger(); dl != nil {
						dl.Logf("[%s] Next: warmup complete for %s, triggering cap enforcement (active=%d, cap=%d)\n",
							cp.name, conn.URL, cp.mu.activeCount, cp.mu.activeListCap)
					}
				} else if warmupDone {
					if dl := loadDebugLogger(); dl != nil {
						dl.Logf("[%s] Next: warmup complete for %s, no cap enforcement (active=%d, cap=%d)\n",
							cp.name, conn.URL, cp.mu.activeCount, cp.mu.activeListCap)
					}
				}

				// Notify observer: warmup accept (State.IsWarmingUp() tells
				// the observer whether warmup is still in progress or just completed).
				if obs := observerFromAtomic(&cp.observer); obs != nil {
					obs.OnWarmupRequest(newConnectionEvent(cp.name, conn, cp.countByLifecycleWithLock()))
				}

				cp.mu.RUnlock()
				cp.poolRequests.Add(1)
				if needsCapEnforce {
					cp.triggerCapEnforcement()
				}
				return conn, nil

			case warmupInactive:
				// Warmup completed between our isWarmingUp() check and tryWarmupSkip() call.
				cp.mu.RUnlock()
				cp.poolRequests.Add(1)
				if needsCapEnforce {
					cp.triggerCapEnforcement()
				}
				return conn, nil

			case warmupSkipped:
				cp.poolWarmupSkips.Add(1)
				// Skipped -- track as fallback for starvation prevention.
				// Use <= so the last-checked connection wins ties, which
				// alternates with round-robin offset for fair distribution.
				remSkip := conn.loadConnState().roundManager().skipCount()
				if remSkip <= bestWarmingRemSkip {
					bestWarmingRemSkip = remSkip
					bestWarmingConn = conn
				}
			}

			_ = attempt
		}

		// Starvation prevention: all active connections are warming and skipped.
		// Return the one closest to its next accept point.
		if bestWarmingConn != nil {
			if obs := observerFromAtomic(&cp.observer); obs != nil {
				obs.OnWarmupRequest(newConnectionEvent(cp.name, bestWarmingConn, cp.countByLifecycleWithLock()))
			}
			cp.mu.RUnlock()
			cp.poolRequests.Add(1)
			if needsCapEnforce {
				cp.triggerCapEnforcement()
			}
			return bestWarmingConn, nil
		}
	}

	// Fast path: nothing anywhere
	if len(cp.mu.ready) == 0 && len(cp.mu.dead) == 0 {
		cp.mu.RUnlock()
		return nil, ErrNoConnections
	}

	// No active connections -- upgrade to write lock for standby or zombie fallback.
	cp.mu.RUnlock()
	return cp.nextFallback()
}

// nextWithEviction acquires a write lock and iterates active connections,
// evicting any that were externally demoted (lifecycle != lcActive) by another
// pool's stats poller. Returns the first healthy connection found, or falls
// through to standby/zombie selection.
//
// The skip count is bounded by the active partition size at entry to prevent
// infinite iteration when all connections have been externally demoted.
func (cp *multiServerPool) nextWithEviction() (*Connection, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Try up to activeCount connections before falling through.
	maxSkips := cp.mu.activeCount
	for range maxSkips {
		if cp.mu.activeCount <= 0 {
			break
		}

		conn, needsCapEnforce := cp.getNextActiveConnWithLock()
		if conn == nil {
			continue
		}
		if cp.excludeDCM && conn.Roles.isDedicatedClusterManager() {
			continue // dedicated cluster managers do not serve request traffic
		}
		state := conn.loadConnState()

		if state.lifecycle()&(lcActive|lcStandby) != 0 {
			if needsCapEnforce {
				cp.enforceActiveCapWithLock()
			}
			cp.poolRequests.Add(1)
			return conn, nil
		}

		// Externally killed (dead/overloaded, no position bits) -- evict from this pool's ready list.
		cp.evictExternallyDemotedWithLock(conn, state)
	}

	// All active connections were evicted; fall through to standby/zombie.
	return cp.nextFallbackWithLock()
}

// nextFallback upgrades to write lock and tries standby, then zombie connections.
func (cp *multiServerPool) nextFallback() (*Connection, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.mu.activeCount <= 0 {
		return cp.nextFallbackWithLock()
	}

	// Double-check active connections after acquiring write lock.
	// Bounded by activeCount at entry to prevent infinite iteration.
	maxSkips := cp.mu.activeCount + 1 // +1: first attempt was already consumed under RLock
	for range maxSkips {
		if cp.mu.activeCount <= 0 {
			break
		}

		conn, needsCapEnforce := cp.getNextActiveConnWithLock()
		if conn == nil {
			continue
		}
		if cp.excludeDCM && conn.Roles.isDedicatedClusterManager() {
			continue // dedicated cluster managers do not serve request traffic
		}
		state := conn.loadConnState()

		if state.lifecycle()&(lcActive|lcStandby) != 0 {
			if needsCapEnforce {
				cp.enforceActiveCapWithLock()
			}
			cp.poolRequests.Add(1)
			return conn, nil
		}

		cp.evictExternallyDemotedWithLock(conn, state)
	}

	return cp.nextFallbackWithLock()
}

// nextFallbackWithLock tries standby then zombie connections.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
func (cp *multiServerPool) nextFallbackWithLock() (*Connection, error) {
	// Try standby before zombie -- standby connections are healthy but idle
	if c := cp.tryStandbyWithLock(); c != nil && !(cp.excludeDCM && c.Roles.isDedicatedClusterManager()) {
		cp.poolRequests.Add(1)
		return c, nil
	}

	// Last resort: a zombie from the dead list, but only one that is
	// availableForRouting (see that method for why). An all-unavailable dead
	// list yields nil, so Next reports ErrNoConnections and the request
	// cascades to the seed-URL fallback rather than dialing a dead zombie.
	if c := cp.tryZombieWithLock(); c != nil && !(cp.excludeDCM && c.Roles.isDedicatedClusterManager()) {
		cp.poolRequests.Add(1)
		return c, nil
	}
	return nil, ErrNoConnections
}

// evictExternallyDemotedWithLock removes a connection from the active partition
// and moves it to the dead list. Called when Next() detects that a connection
// has no position bits set (lcActive or lcStandby) -- meaning it was externally
// killed (e.g., lcDead or lcDead|lcOverloaded set by another pool's health
// check or overload detection).
//
// Connections in lcStandby state (set by another pool's cap enforcement) are
// NOT evicted -- each pool manages its own active/standby partitions
// independently, and standby is a safe position for continued use.
//
// The connection's lifecycle state is preserved as-is (we don't CAS -- the
// external owner of that state transition is responsible). We only update this
// pool's structural lists (ready -> dead) and set deadSince if not already set.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
func (cp *multiServerPool) evictExternallyDemotedWithLock(c *Connection, state connState) {
	cp.removeFromReadyWithLock(c)
	cp.appendToDeadWithLock(c)

	if cp.metrics != nil {
		cp.metrics.connectionsDemoted.Add(1)
	}

	if dl := loadDebugLogger(); dl != nil {
		dl.Logf("[%s] Next: evicted externally-demoted %q (state=%s, active=%d, dead=%d)\n",
			cp.name, c.URL, ConnState{packed: int64(state)}.Hex(), cp.mu.activeCount, len(cp.mu.dead))
	}

	if obs := observerFromAtomic(&cp.observer); obs != nil {
		obs.OnDemote(newConnectionEvent(cp.name, c, cp.countByLifecycleWithLock()))
	}
}

// tryZombieWithLock returns a dead connection for temporary use without moving it to the ready list.
// This allows attempting requests on potentially dead connections when no ready connections are available.
// The connection remains on the dead list and will continue to be subject to periodic health checks.
// Used by Next() when no ready connections are available, providing a way to short-circuit the periodic
// heartbeat timer by attempting requests on dead connections immediately.
//
// Only a connection that is availableForRouting (a user-supplied seed, or a
// discovered node proven reachable at least once) is eligible: a never-verified
// discovered node must not be dialed as a zombie (see availableForRouting).
// Returns nil when the dead list is empty or holds no eligible connection, so
// the caller reports ErrNoConnections and the seed-URL fallback takes over.
//
// The function rotates through dead connections by popping from the front and pushing to the back,
// ensuring fair distribution of retry attempts across all eligible dead connections.
//
// CONCURRENCY NOTE: This function races with OnFailure() over dead list ordering. OnFailure()
// sorts dead connections by failure count while this function rotates the list for fair distribution.
// The design assumes that during failure scenarios, we iterate through the entire dead list faster
// than new connections fail and trigger list resorting in OnFailure(). This ensures fair rotation
// is maintained most of the time, with occasional resorting to prioritize connections with fewer failures.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
//   - Caller should call OnSuccess() if the connection proves to work (which will resurrect it)
//   - Caller should call OnFailure() if the connection fails (which is a no-op since it's already dead)
func (cp *multiServerPool) tryZombieWithLock() *Connection {
	// Rotate through the dead list at most once, returning the first eligible
	// (availableForRouting) connection. Each iteration pops the front and pushes
	// it to the back, preserving fair round-robin distribution across retries.
	for range cp.mu.dead {
		c := cp.mu.dead[0]
		cp.mu.dead = append(cp.mu.dead[1:], c)
		if !c.availableForRouting() {
			continue
		}
		if cp.metrics != nil {
			cp.metrics.zombieConnections.Add(1)
		}
		return c
	}
	return nil
}
