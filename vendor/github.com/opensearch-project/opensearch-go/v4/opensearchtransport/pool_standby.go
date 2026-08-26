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
	"fmt"
	"slices"
	"sync"
	"time"
)

// Sentinel errors for standby rotation failures.
var (
	// ErrRotationHealthCheckFailed indicates a standby candidate was found but its
	// health check failed or its state changed before promotion could occur.
	ErrRotationHealthCheckFailed = errors.New("standby rotation: health check failed")

	// ErrRotationPromotionRace indicates a standby candidate passed health checks
	// but was removed from the standby partition before it could be promoted
	// (e.g., by concurrent discovery or another rotation goroutine).
	ErrRotationPromotionRace = errors.New("standby rotation: candidate removed before promotion")

	// ErrRotationStateChanged indicates the candidate's lifecycle state changed
	// concurrently (e.g., marked dead by another pool) during health checking.
	ErrRotationStateChanged = errors.New("standby rotation: candidate state changed concurrently")

	// ErrRotationNoCandidate indicates no standby candidate was available for rotation.
	ErrRotationNoCandidate = errors.New("standby rotation: no standby candidate available")
)

// demoteOverloaded moves an overloaded connection to the standby partition.
// Unlike OnFailure(), this does NOT increment failures -- the connection isn't broken,
// it's just under too much load. The lcOverloaded metadata flag distinguishes it
// from genuinely failed connections and prevents standby rotation from selecting it.
//
//   - Active connections are moved to standby and the active slot is backfilled.
//   - Standby and dead connections simply have lcOverloaded set.
//
// The stats poller calls promoteFromOverloaded when metrics improve, which
// clears lcOverloaded and makes the connection eligible for promotion again.
func (cp *multiServerPool) demoteOverloaded(c *Connection) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	c.mu.Lock()
	lc := c.loadConnState().lifecycle()

	// Already dead -- just add overloaded flag
	if lc.has(lcUnknown) {
		c.setLifecycleBit(lcOverloaded) //nolint:errcheck // lock held; only errLifecycleNoop possible
		c.storeOverloadedAt(time.Now())
		c.mu.Unlock()
		return
	}

	// Already in standby -- just add overloaded flag
	if lc.has(lcStandby) {
		c.setLifecycleBit(lcOverloaded) //nolint:errcheck // lock held; only errLifecycleNoop possible
		c.storeOverloadedAt(time.Now())
		c.mu.Unlock()
		return
	}

	// Active -> standby with overloaded flag
	if err := c.casLifecycle(c.loadConnState(), 0, lcStandby|lcOverloaded, lcActive); err != nil {
		c.mu.Unlock()
		return // state changed concurrently
	}
	c.storeOverloadedAt(time.Now())
	c.mu.Unlock()
	// Note: do NOT increment c.failures -- this is not a failure

	// Move from active to standby partition
	cp.removeFromReadyWithLock(c)
	cp.appendToReadyStandbyWithLock(c)

	if cp.metrics != nil {
		cp.metrics.standbyDemotions.Add(1)
	}

	if dl := loadDebugLogger(); dl != nil {
		dl.Logf("[%s] Overload-demoted %q to standby (active=%d, standby=%d)\n",
			cp.name, c.URL, cp.mu.activeCount, len(cp.mu.ready)-cp.mu.activeCount)
	}

	if obs := observerFromAtomic(&cp.observer); obs != nil {
		obs.OnOverloadDetected(newConnectionEvent(cp.name, c, cp.countByLifecycleWithLock()))
	}

	// Backfill the active slot from standby
	cp.tryStandbyWithLock()
}

// promoteFromOverloaded clears the lcOverloaded flag from a connection.
// The connection remains in its current partition (standby or dead) -- normal
// standby rotation will promote it to active when a slot is available.
//
// Called by the stats poller when node metrics drop below overload thresholds.
func (cp *multiServerPool) promoteFromOverloaded(c *Connection) {
	c.mu.Lock()
	if !c.loadConnState().lifecycle().has(lcOverloaded) {
		c.mu.Unlock()
		return
	}

	c.clearLifecycleBit(lcOverloaded) //nolint:errcheck // lock held; only errLifecycleNoop possible
	c.storeOverloadedAt(time.Time{})
	c.mu.Unlock()

	if dl := loadDebugLogger(); dl != nil {
		dl.Logf("[%s] Cleared overloaded flag on %q (state=%s)\n",
			cp.name, c.URL, c.loadConnState().lifecycle())
	}

	if obs := observerFromAtomic(&cp.observer); obs != nil {
		cp.mu.RLock()
		obs.OnOverloadCleared(newConnectionEvent(cp.name, c, cp.countByLifecycleWithLock()))
		cp.mu.RUnlock()
	}
}

// enforceActiveCapWithLock trims the active partition by moving overflow
// fully-warmed connections to the standby partition (past activeCount).
//
// The cap applies only to fully-warmed connections -- warming connections are
// always allowed in the active partition alongside the capped warmed ones.
// This ensures proven active connections aren't evicted to make room for a
// warming connection that hasn't yet integrated into the traffic mix.
// When a warming connection finishes warmup (detected in Next()), triggerCapEnforcement
// fires and this function evicts the excess fully-warmed connection.
//
// No-op when activeListCap <= 0 (disabled) or when the fully-warmed active count
// is within cap.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
func (cp *multiServerPool) enforceActiveCapWithLock() {
	if cp.mu.activeListCap <= 0 || cp.mu.activeCount <= cp.mu.activeListCap {
		return
	}

	// Count fully-warmed (non-warming) active connections.
	nonWarmCount := 0
	for i := 0; i < cp.mu.activeCount; i++ {
		if !cp.mu.ready[i].loadConnState().isWarmingUp() {
			nonWarmCount++
		}
	}

	if nonWarmCount <= cp.mu.activeListCap {
		// Not enough fully-warmed connections to enforce the cap.
		// Warming connections need to finish before we can evict.
		return
	}

	// Partition active connections: warming first (kept), non-warming at tail (evicted).
	// This ensures we only demote fully-warmed connections and never disrupt warmup.
	warmCount := 0
	for i := 0; i < cp.mu.activeCount; i++ {
		if cp.mu.ready[i].loadConnState().isWarmingUp() {
			if i != warmCount {
				cp.mu.ready[i], cp.mu.ready[warmCount] = cp.mu.ready[warmCount], cp.mu.ready[i]
			}
			warmCount++
		}
	}

	// New active count = warming connections + capped non-warming connections.
	newActiveCount := warmCount + cp.mu.activeListCap
	overflow := cp.mu.activeCount - newActiveCount

	// Transition overflow connections (non-warming, at tail) from active to standby.
	// Set lcNeedsWarmup so they warm up gradually when re-promoted, ensuring
	// smooth traffic transitions during standby rotation.
	// clearWarmup() zeros the warmup managers -- without this, startWarmup()
	// (called on re-promotion) would see isWarmingUp()==true and no-op,
	// leaving the connection with lcStandby lifecycle in the active partition.
	for i := newActiveCount; i < cp.mu.activeCount; i++ {
		cp.mu.ready[i].mu.Lock()
		//nolint:errcheck // lock held; only errLifecycleNoop possible
		cp.mu.ready[i].casLifecycle(
			cp.mu.ready[i].loadConnState(), 0,
			lcStandby|lcNeedsWarmup,
			lcActive,
		)
		cp.mu.ready[i].clearWarmup()
		cp.mu.ready[i].mu.Unlock()
	}
	cp.mu.activeCount = newActiveCount

	// Re-shuffle active partition to interleave warming and non-warming connections.
	// Without this, all warming connections would be clustered at the front,
	// causing latency spikes from consecutive warmup skips in round-robin selection.
	cp.shuffleActiveWithLock()

	if cp.metrics != nil {
		cp.metrics.standbyDemotions.Add(int64(overflow))
	}

	if dl := loadDebugLogger(); dl != nil {
		dl.Logf("[%s] Enforced active cap=%d: moved %d connections to standby (active=%d, standby=%d)\n",
			cp.name, cp.mu.activeListCap, overflow, cp.mu.activeCount, len(cp.mu.ready)-cp.mu.activeCount)
	}

	if obs := observerFromAtomic(&cp.observer); obs != nil {
		counts := cp.countByLifecycleWithLock()
		for i := cp.mu.activeCount; i < cp.mu.activeCount+overflow; i++ {
			obs.OnStandbyDemote(newConnectionEvent(
				cp.name, cp.mu.ready[i], counts,
			))
		}
	}
}

// tryStandbyWithLock promotes the next standby connection to active by advancing
// the activeCount boundary. The connection is already in position -- no swap needed.
// The prior OnFailure already shrank the active partition, so we're just filling
// the gap.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
//   - Caller should call OnSuccess() on success or OnFailure() on failure
func (cp *multiServerPool) tryStandbyWithLock() *Connection {
	if cp.mu.activeCount >= len(cp.mu.ready) {
		return nil
	}

	c := cp.mu.ready[cp.mu.activeCount]
	// Under duress: all active connections exhausted. Promote immediately
	// without warmup -- strip lcNeedsWarmup to maximize throughput.
	c.mu.Lock()
	c.casLifecycle(c.loadConnState(), 0, lcActive, lcStandby|lcNeedsWarmup) //nolint:errcheck // lock held; only errLifecycleNoop possible
	c.mu.Unlock()
	cp.mu.activeCount++

	if cp.metrics != nil {
		cp.metrics.standbyPromotions.Add(1)
	}

	if dl := loadDebugLogger(); dl != nil {
		dl.Logf("[%s] tryStandby: promoted %q to active (forced, no warmup) (active=%d, standby=%d)\n",
			cp.name, c.URL, cp.mu.activeCount, len(cp.mu.ready)-cp.mu.activeCount)
	}

	return c
}

// promoteStandbyWithLock promotes a specific standby connection to active.
// If the connection is not at the boundary, it is swapped there before advancing
// activeCount -- this happens when un-warmed standby connections sit between the
// boundary and the warmed connection.
//
// Returns false if the connection is no longer in the standby partition (e.g.,
// removed by concurrent discovery while we were health-checking).
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
func (cp *multiServerPool) promoteStandbyWithLock(c *Connection) bool {
	// Search backward -- recently appended items are at the tail
	idx := -1
	for i := len(cp.mu.ready) - 1; i >= cp.mu.activeCount; i-- {
		if cp.mu.ready[i] == c {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false // not in standby
	}

	// Swap to boundary if not already there
	if idx != cp.mu.activeCount {
		cp.mu.ready[idx], cp.mu.ready[cp.mu.activeCount] = cp.mu.ready[cp.mu.activeCount], cp.mu.ready[idx]
	}

	c.mu.Lock()
	c.casLifecycle(c.loadConnState(), 0, lcActive, lcStandby) //nolint:errcheck // lock held; only errLifecycleNoop possible
	c.mu.Unlock()
	rounds, skip := cp.getWarmupParamsWithLock()
	c.startWarmup(rounds, skip)
	cp.mu.activeCount++

	cp.shuffleActiveWithLock()

	if cp.metrics != nil {
		cp.metrics.standbyPromotions.Add(1)
	}

	return true
}

// performStandbyHealthCheck performs multiple consecutive health checks on a
// standby connection before allowing promotion. Returns true only if all
// checks pass. This pre-warms the connection before it handles production traffic.
func (cp *multiServerPool) performStandbyHealthCheck(ctx context.Context, c *Connection) bool {
	cp.mu.RLock()
	hc := cp.mu.healthCheck
	cp.mu.RUnlock()

	if hc == nil {
		return true
	}
	for i := int64(0); i < cp.standbyPromotionChecks; i++ {
		if !cp.performHealthCheck(ctx, c, true) {
			return false
		}
	}
	return true
}

// findActiveCandidate searches the standby partition for the best connection
// to promote to active. Searches backward from the tail to avoid contention
// near the active boundary where promotions and demotions occur.
//
// Search priority (two passes):
//  1. Idle standby -- not overloaded, unknown, or health-checking. Preferred
//     because it is genuinely idle and ready for immediate promotion.
//  2. Health-checking standby -- currently undergoing a health check but
//     otherwise viable. Last resort when no idle standby is available.
//
// Overloaded and unknown connections are never selected here; those are only
// promoted forcefully when there are zero active connections.
//
// For idle candidates without lcNeedsWarmup already set, the function sets it
// to ensure warmup ramp-up after promotion.
//
// Returns nil if no candidate is available.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool lock
func (cp *multiServerPool) findActiveCandidate() *Connection {
	// Pass 1: prefer a genuinely idle standby.
	for i := len(cp.mu.ready) - 1; i >= cp.mu.activeCount; i-- {
		c := cp.mu.ready[i]
		c.mu.Lock()
		lc := c.loadConnState().lifecycle()
		if lc.hasAny(lcOverloaded | lcUnknown | lcHealthChecking) {
			c.mu.Unlock()
			continue
		}
		// Set lcHealthChecking to claim this candidate exclusively -- concurrent
		// goroutines in rotateStandby will skip it in Pass 1.
		// Also ensure lcNeedsWarmup is set for warmup on promotion.
		c.casLifecycle( //nolint:errcheck // lock held; only errLifecycleNoop possible
			c.loadConnState(),
			lcOverloaded|lcUnknown,
			lcHealthChecking|lcNeedsWarmup, 0,
		)
		c.mu.Unlock()
		return c
	}

	// Pass 2: fall back to a standby that is mid-health-check.
	for i := len(cp.mu.ready) - 1; i >= cp.mu.activeCount; i-- {
		c := cp.mu.ready[i]
		c.mu.Lock()
		lc := c.loadConnState().lifecycle()
		if lc.hasAny(lcOverloaded|lcUnknown) || !lc.has(lcHealthChecking) {
			c.mu.Unlock()
			continue
		}
		c.mu.Unlock()
		return c
	}

	return nil
}

// rotateStandby performs up to count standby<->active rotation cycles concurrently.
// Spins up count goroutines, each attempting one rotation via health-check and promotion.
// If a health check fails, the goroutine retries with the next standby candidate.
// Cancels remaining goroutines once the desired number of successful rotations is reached.
// Returns the number of successful rotations and any errors encountered during rotation.
// The error may be non-nil even when rotated == count (partial failures during parallel attempts).
// Dead connections are not considered -- they have their own resurrection schedule and
// shouldn't incur extra client-side health checks.
func (cp *multiServerPool) rotateStandby(ctx context.Context, count int) (int, error) {
	if count <= 0 {
		return 0, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		rotated int
		errs    []error
	)

	wg.Add(count)
	for range count {
		go func() {
			defer wg.Done()
			for {
				if ctx.Err() != nil {
					return
				}

				attempted, success, err := cp.rotateStandbyOnce(ctx)
				if err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
				}
				if !attempted {
					return // no standby candidates available
				}
				if success {
					mu.Lock()
					rotated++
					done := rotated >= count
					mu.Unlock()
					if done {
						cancel() // target reached, signal others to stop
					}
					return
				}
				// Health check failed -- retry with next candidate
			}
		}()
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	return rotated, errors.Join(errs...)
}

// rotateStandbyOnce attempts one standby<->active rotation.
// Picks a non-warming standby, health-checks it, and on success swaps it into
// active. activeCount is unchanged on success (one in, one out via cap enforcement).
//
// Returns (attempted, rotated, err):
//   - (true, true, nil): standby passed health check and was promoted to active
//   - (true, false, err): standby found but not promotable (health check failed, state changed, or removed)
//   - (false, false, nil): no standby candidate available
func (cp *multiServerPool) rotateStandbyOnce(ctx context.Context) (bool, bool, error) {
	candidate, err := cp.healthcheckStart(ctx)
	if candidate == nil {
		if errors.Is(err, ErrRotationNoCandidate) {
			return false, false, nil
		}
		return true, false, err
	}

	// Promote the verified standby to active with warmup.
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if !cp.promoteStandbyWithLock(candidate) {
		return true, false, fmt.Errorf("[%s] %q: %w", cp.name, candidate.URL, ErrRotationPromotionRace)
	}

	if dl := loadDebugLogger(); dl != nil {
		dl.Logf("[%s] rotateStandby: promoted %q (standby->active with warmup) (active=%d, standby=%d)\n",
			cp.name, candidate.URL, cp.mu.activeCount, len(cp.mu.ready)-cp.mu.activeCount)
	}

	if obs := observerFromAtomic(&cp.observer); obs != nil {
		obs.OnStandbyPromote(newConnectionEvent(
			cp.name, candidate, cp.countByLifecycleWithLock()))
	}

	return true, true, nil
}

// healthcheckStart finds an idle standby candidate, health-checks it, and
// fixes any ready-list inconsistencies discovered after re-acquiring the pool
// lock.
//
// Returns (candidate, err):
//   - (conn, nil): candidate is healthy, in standby partition, ready for promotion
//   - (nil, err): candidate was found but not promotable, or no standby candidate available
//
// Fixups performed while holding the pool lock:
//   - lcUnknown connections found in the ready list are moved to dead
//   - Candidates removed by concurrent discovery have their warmup claim cleared
//   - Failed health checks move the candidate to dead and schedule resurrection
func (cp *multiServerPool) healthcheckStart(ctx context.Context) (*Connection, error) {
	cp.mu.Lock()

	candidate := cp.findActiveCandidate()
	if candidate == nil {
		cp.mu.Unlock()
		return nil, ErrRotationNoCandidate
	}
	cp.mu.Unlock()

	// Health check outside lock (network I/O)
	healthy := cp.performStandbyHealthCheck(ctx, candidate)

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Clear the lcHealthChecking claim set by findActiveCandidate, regardless
	// of outcome. All exit paths below either move the candidate to dead
	// (which resets lifecycle) or hand it back for promotion.
	candidate.mu.Lock()
	candidate.clearLifecycleBit(lcHealthChecking) //nolint:errcheck // lock held; only errLifecycleNoop possible
	candidate.mu.Unlock()

	// Verify candidate is still in the ready list. A concurrent goroutine
	// (via Pass 2 in findActiveCandidate) may have already moved it to dead
	// while we were health-checking outside the lock.
	inReady := slices.Contains(cp.mu.ready, candidate)
	if !inReady {
		return nil, fmt.Errorf("[%s] %q: %w: already removed from ready by concurrent goroutine",
			cp.name, candidate.URL, ErrRotationPromotionRace)
	}

	// Fix up: if candidate became lcUnknown concurrently, move to dead
	if candidate.loadConnState().lifecycle().has(lcUnknown) {
		cp.evictUnknownFromReadyWithLock(candidate)
		cp.scheduleResurrect(ctx, candidate)
		return nil, fmt.Errorf("[%s] %q: %w: became lcUnknown during health check", cp.name, candidate.URL, ErrRotationStateChanged)
	}

	if !healthy {
		// Health check failed -- move to dead
		candidate.mu.Lock()
		if err := candidate.casLifecycle(candidate.loadConnState(), 0, lcDead|lcNeedsWarmup, lcReady|lcActive|lcStandby); err != nil {
			candidate.mu.Unlock()
			return nil, fmt.Errorf("[%s] %q: %w: lifecycle CAS failed", cp.name, candidate.URL, ErrRotationStateChanged)
		}
		candidate.markAsDeadWithLock()
		candidate.mu.Unlock()
		cp.removeFromReadyWithLock(candidate)
		cp.appendToDeadWithLock(candidate)

		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("[%s] healthcheckStart: health check failed for %q, moved to dead (active=%d, dead=%d)\n",
				cp.name, candidate.URL, cp.mu.activeCount, len(cp.mu.dead))
		}

		cp.scheduleResurrect(ctx, candidate)
		return nil, fmt.Errorf("[%s] %q: %w: moved to dead", cp.name, candidate.URL, ErrRotationHealthCheckFailed)
	}

	// Verify candidate is still in the standby partition
	found := false
	for i := len(cp.mu.ready) - 1; i >= cp.mu.activeCount; i-- {
		if cp.mu.ready[i] == candidate {
			found = true
			break
		}
	}
	if !found {
		// Removed by concurrent discovery -- clear warmup claim
		candidate.mu.Lock()
		candidate.clearLifecycleBit(lcNeedsWarmup) //nolint:errcheck // lock held; only errLifecycleNoop possible
		candidate.mu.Unlock()
		return nil, fmt.Errorf("[%s] %q: %w: removed by concurrent discovery", cp.name, candidate.URL, ErrRotationPromotionRace)
	}

	return candidate, nil
}

// evictUnknownFromReadyWithLock moves an lcUnknown connection from the ready
// list to the dead list. This fixes inconsistencies where a connection's
// lifecycle was changed to lcUnknown (e.g., by another pool's health check)
// while it was still positioned in this pool's ready list.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
func (cp *multiServerPool) evictUnknownFromReadyWithLock(c *Connection) {
	c.mu.Lock()
	c.markAsDeadWithLock()
	c.mu.Unlock()

	cp.removeFromReadyWithLock(c)
	cp.appendToDeadWithLock(c)

	if dl := loadDebugLogger(); dl != nil {
		dl.Logf("[%s] evictUnknownFromReadyWithLock: moved %q to dead (active=%d, dead=%d)\n",
			cp.name, c.URL, cp.mu.activeCount, len(cp.mu.dead))
	}
}

// asyncPromoteStandby claims a non-warming standby connection, health-checks it,
// and promotes it to active. Called asynchronously after OnFailure to fill the
// gap left by a failed connection (1:1 replacement). Unlike rotateStandby, this
// does not evict an active connection -- it grows the active partition by one.
func (cp *multiServerPool) asyncPromoteStandby(ctx context.Context) {
	candidate, err := cp.healthcheckStart(ctx)
	if candidate == nil {
		if err != nil && !errors.Is(err, ErrRotationNoCandidate) {
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("[%s] asyncPromoteStandby: healthcheckStart: %v\n", cp.name, err)
			}
		}
		return
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Promote: swap to boundary, advance activeCount (no eviction -- filling a gap)
	if !cp.promoteStandbyWithLock(candidate) {
		return
	}

	if dl := loadDebugLogger(); dl != nil {
		dl.Logf("[%s] asyncPromoteStandby: promoted %q to active (active=%d, standby=%d)\n",
			cp.name, candidate.URL, cp.mu.activeCount, len(cp.mu.ready)-cp.mu.activeCount)
	}

	if obs := observerFromAtomic(&cp.observer); obs != nil {
		obs.OnStandbyPromote(newConnectionEvent(
			cp.name, candidate, cp.countByLifecycleWithLock()))
	}
}

// promoteStandbyGracefullyWithLock spawns up to `gap` asyncPromoteStandby
// goroutines to fill active slots lost during discovery removal. Each promotion
// warms the standby connection (health-check) before moving it to active -- no
// forced promotions.
//
// The actual goroutine count is min(gap, standbyCount) so we never spawn more
// than can succeed. No-op when no standby exists or gap <= 0.
//
// Called from policy DiscoveryUpdate paths after filtering shrinks activeCount.
// The spawned goroutines acquire the pool lock independently after the caller
// releases it.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock (reads activeCount and len(ready))
func (cp *multiServerPool) promoteStandbyGracefullyWithLock(ctx context.Context, gap int) {
	standbyCount := len(cp.mu.ready) - cp.mu.activeCount
	if gap <= 0 || standbyCount <= 0 {
		return
	}

	n := min(gap, standbyCount)

	for range n {
		go cp.asyncPromoteStandby(ctx)
	}
}
