// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"cmp"
	"context"
	"math/rand/v2"
	"net/url"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// multiServerPool manages a partitioned connection pool: ready[:activeCount]
// is the active partition (selected by the pool's selector), ready[activeCount:]
// is standby, and dead holds connections awaiting resurrection.
//
// The pool's selection strategy is pluggable via the [poolSelector] interface.
// By default ([poolRoundRobin]), selection is an atomic counter with modulo.
type multiServerPool struct {
	name string // Pool identity for metrics/debug (e.g. "roundrobin", "role:data")

	// excludeDCM skips dedicated cluster managers during selection. Set only on
	// the allConns inventory pool, which serves requests as a last resort when no
	// router is configured; the inventory holds dedicated cluster managers for
	// discovery reuse, but they must not receive request traffic by default.
	excludeDCM bool

	// ctx is the lifecycle context for async pool operations (health checks,
	// resurrection, RTT probes). Derived from Client.ctx during pool construction
	// so that Client.Close() cancels all background goroutines. cancel is the
	// pool's own cancel func for early cleanup on multi-to-single demotion; see
	// [Client.demoteConnectionPoolWithLock].
	//nolint:containedctx // Long-lived context required for background worker lifecycle
	ctx    context.Context
	cancel context.CancelFunc

	mu struct {
		sync.RWMutex
		ready       []*Connection            // Partitioned: ready[:activeCount] are active, ready[activeCount:] are standby
		dead        []*Connection            // List of dead connections
		activeCount int                      // Number of active connections; elements past this index are standby
		members     map[*Connection]struct{} // O(1) containment check; tracks all conns in ready+dead

		// Standby pool configuration.
		// When activeListCap > 0, discovery overflow and resurrected connections go to standby
		// instead of active when the ready list's active partition is at capacity.
		activeListCap int // 0 = disabled (all connections go to active)

		// Dynamic warmup parameters, scaled by recalculateWarmupParamsWithLock().
		// Small pools get lighter warmup (fewer rounds, fewer skips) so connections
		// ramp up quickly. Large pools get heavier warmup to avoid traffic spikes.
		warmupRounds    int // 0 = use defaultWarmupRounds
		warmupSkipCount int // 0 = use defaultWarmupSkipCount

		// Health check function - returns HTTP response on success, error on failure.
		// Rewritten by updateConnectionPool on pool reuse, so reads must hold the lock.
		healthCheck HealthCheckFunc
	}

	// selector determines how the pool picks the next connection from the
	// active partition. When nil, the pool uses the legacy nextReady counter
	// for round-robin selection (backward compatible).
	selector poolSelector

	nextReady atomic.Int64 // Legacy round-robin counter (used when selector is nil)

	resurrectTimeoutInitial      time.Duration
	resurrectTimeoutMax          time.Duration
	resurrectTimeoutFactorCutoff int
	minimumResurrectTimeout      time.Duration
	jitterScale                  float64
	serverMaxNewConnsPerSec      float64 // target max new health check conns a server accepts/sec from all clients
	clientsPerServer             float64 // estimated client instances per server

	// Standby pool configuration.
	standbyPromotionChecks int64 // consecutive health checks before standby->ready

	// activeListCapConfig preserves the user's original intent:
	//   nil = auto-scale activeListCap with cluster size during discovery
	//   non-nil = user-specified value (activeListCap is fixed)
	activeListCapConfig *int

	// Per-pool request counters (atomic, lock-free)
	poolRequests      atomic.Int64 // Connections returned by Next()
	poolSuccesses     atomic.Int64 // Resurrections via OnSuccess()
	poolFailures      atomic.Int64 // Demotions via OnFailure()
	poolWarmupSkips   atomic.Int64 // Requests skipped during warmup
	poolWarmupAccepts atomic.Int64 // Requests accepted during warmup

	metrics  *metrics
	observer atomic.Pointer[ConnectionObserver]
}

// Compile-time checks to ensure interface compliance.
var (
	_ ConnectionPool = (*multiServerPool)(nil)
	_ rwLocker       = (*multiServerPool)(nil)
)

// poolCtx returns the pool's lifecycle context, falling back to
// context.Background() when none has been set (e.g. in tests).
func (cp *multiServerPool) poolCtx() context.Context {
	if cp.ctx != nil {
		return cp.ctx
	}
	return context.Background()
}

// getNextActiveConnWithLock returns the next active connection using the pool's
// selector strategy and a boolean indicating whether cap enforcement is needed.
// Falls back to the legacy round-robin counter when no selector is configured.
//
// When needsCapEnforce is true, the caller must arrange for cap enforcement
// after releasing the read lock (triggerCapEnforcement uses TryLock, which
// fails while any read lock is held). Callers that already hold the write lock
// can call enforceActiveCapWithLock directly.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool read or write lock
//   - Caller must ensure cp.mu.activeCount > 0 before calling
func (cp *multiServerPool) getNextActiveConnWithLock() (*Connection, bool) {
	if cp.selector != nil {
		conn, activeCap, _, err := cp.selector.selectNext(cp.mu.ready, cp.mu.activeCount)
		if err != nil {
			return nil, false
		}
		// Handle cap adjustment signals from the selector.
		// capGrow (standby promotion) stays async -- adding a connection is safe.
		// capShrink is signaled back to the caller, which must call
		// triggerCapEnforcement after releasing the read lock.
		switch activeCap {
		case capGrow:
			go cp.deferredStandbyPromotion()
		case capShrink:
			return conn, true
		}
		return conn, false
	}

	// Legacy round-robin fallback.
	next := cp.nextReady.Add(1)
	idx := int(next-1) % cp.mu.activeCount
	return cp.mu.ready[idx], false
}

// deferredStandbyPromotion acquires the pool write lock and promotes one
// connection from standby to active. Called asynchronously when the selector
// signals that the active partition is saturated.
func (cp *multiServerPool) deferredStandbyPromotion() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.mu.activeCount >= len(cp.mu.ready) {
		return // no standby connections available
	}

	c := cp.mu.ready[cp.mu.activeCount]
	c.mu.Lock()
	c.casLifecycle(c.loadConnState(), 0, lcActive, lcStandby|lcNeedsWarmup) //nolint:errcheck // lock held; only errLifecycleNoop possible
	c.mu.Unlock()
	cp.mu.activeCount++

	// Also grow the cap so the promoted connection stays active.
	if cp.mu.activeListCap > 0 && cp.mu.activeCount > cp.mu.activeListCap {
		cp.mu.activeListCap = cp.mu.activeCount
	}

	if cp.metrics != nil {
		cp.metrics.standbyPromotions.Add(1)
	}

	if dl := loadDebugLogger(); dl != nil {
		dl.Logf("[%s] deferredStandbyPromotion: promoted %q to active (active=%d, standby=%d, cap=%d)\n",
			cp.name, c.URL, cp.mu.activeCount, len(cp.mu.ready)-cp.mu.activeCount, cp.mu.activeListCap)
	}
}

// triggerCapEnforcement attempts to start asynchronous cap enforcement.
//
// Cap enforcement trims the active partition when activeCount > activeListCap
// by demoting fully-warmed connections to standby. Triggered by warmup
// completion in Next() and selector capShrink signals.
//
// TryLock keeps enforcement off the Next() hot path: if the write lock is
// available, we acquire it and launch a goroutine that runs enforcement and
// releases the lock on exit. If the lock is held (by another enforcement
// goroutine, discovery, etc.), the call is a no-op. Self-heals because
// Next() re-checks activeCount > cap on every warmup completion and
// capShrink signal.
func (cp *multiServerPool) triggerCapEnforcement() {
	if !cp.mu.TryLock() {
		return
	}

	go func() {
		defer cp.mu.Unlock()
		cp.enforceActiveCapWithLock()
	}()
}

// snapshot returns a point-in-time PolicySnapshot of this pool's partitions and counters.
func (cp *multiServerPool) snapshot() PolicySnapshot {
	cp.mu.RLock()
	counts := cp.countByLifecycleWithLock()
	activeListCap := cp.mu.activeListCap
	cp.mu.RUnlock()

	return PolicySnapshot{
		Name:                cp.name,
		ActiveCount:         counts.active,
		StandbyCount:        counts.standby,
		DeadCount:           counts.dead,
		ActiveListCap:       activeListCap,
		WarmingCount:        counts.warming,
		HealthCheckingCount: counts.healthCheck,
		Requests:            cp.poolRequests.Load(),
		Successes:           cp.poolSuccesses.Load(),
		Failures:            cp.poolFailures.Load(),
		WarmupSkips:         cp.poolWarmupSkips.Load(),
		WarmupAccepts:       cp.poolWarmupAccepts.Load(),
	}
}

// lifecycleCounts holds connection counts derived from lifecycle bits.
// These reflect actual connection state rather than structural list positions,
// which may diverge temporarily until lazy cleanup runs.
//
// Debug log messages intentionally continue to use structural list positions
// (cp.mu.activeCount, len(cp.mu.ready), len(cp.mu.dead)) since those are
// useful for diagnosing pool corruption.
type lifecycleCounts struct {
	active      int
	standby     int
	dead        int
	warming     int
	healthCheck int
	overloaded  int
}

// countByLifecycleWithLock scans all connections (ready + dead) and counts
// by lifecycle bits. This is the authoritative source for pool statistics.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool read or write lock.
func (cp *multiServerPool) countByLifecycleWithLock() lifecycleCounts {
	var counts lifecycleCounts
	for _, c := range cp.mu.ready {
		cs := c.loadConnState()
		lc := cs.lifecycle()
		switch {
		case lc.has(lcActive):
			counts.active++
		case lc.has(lcStandby):
			counts.standby++
		default:
			counts.dead++ // misplaced in ready list, will be lazily cleaned up
		}
		if cs.isWarmingUp() {
			counts.warming++
		}
		if lc.has(lcHealthChecking) {
			counts.healthCheck++
		}
		if lc.has(lcOverloaded) {
			counts.overloaded++
		}
	}
	for _, c := range cp.mu.dead {
		cs := c.loadConnState()
		lc := cs.lifecycle()
		switch {
		case lc.has(lcActive):
			counts.active++ // misplaced in dead list, will be lazily cleaned up
		case lc.has(lcStandby):
			counts.standby++ // misplaced in dead list
		default:
			counts.dead++
		}
		if lc.has(lcHealthChecking) {
			counts.healthCheck++
		}
		if lc.has(lcOverloaded) {
			counts.overloaded++
		}
	}
	return counts
}

// hasAvailableConnsWithLock reports whether the pool holds any connection that
// is available for routing (see Connection.availableForRouting): any ready
// connection, or any dead connection that is either a user-supplied seed or
// currently confirmed reachable.
//
// A dead connection not currently confirmed reachable does not count. Counting
// it as available is what lets an unreachable discovered node mask the seed-URL
// fallback: the policy reports "enabled", Route hands the request to the pool,
// and the pool serves the unverified connection as a zombie instead of
// returning ErrNoConnections. Excluding it here makes the policy report
// "not enabled" so the request cascades to the seed fallback until a discovered
// connection actually proves reachable.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold the pool lock (read or write).
func (cp *multiServerPool) hasAvailableConnsWithLock() bool {
	if len(cp.mu.ready) > 0 {
		return true
	}
	for _, c := range cp.mu.dead {
		if c.availableForRouting() {
			return true
		}
	}
	return false
}

// recalculateWarmupParamsWithLock recalculates activeListCap (when auto-scaling) and sets
// warmupRounds/warmupSkipCount based on effective pool size.
//
// poolSize is the projected total number of connections in the pool (ready + dead)
// after the current DiscoveryUpdate completes.
//
// Warmup formula:
//
//	n = min(activeListCap, poolSize) when activeListCap > 0
//	n = poolSize                   when activeListCap <= 0
//	rounds = clamp(n, minWarmupRounds, maxWarmupRounds)
//	skipCount = rounds * warmupSkipMultiple
//
// Caller must hold the pool write lock: it reads and writes the mu-guarded
// activeListCap, warmupRounds, and warmupSkipCount fields.
func (cp *multiServerPool) recalculateWarmupParamsWithLock(poolSize int) {
	// Auto-scale activeListCap when the user didn't specify an explicit value.
	if cp.activeListCapConfig == nil && poolSize > 0 {
		cp.mu.activeListCap = poolSize
	}

	n := poolSize
	if cp.mu.activeListCap > 0 && cp.mu.activeListCap < n {
		n = cp.mu.activeListCap
	}
	if n <= 0 {
		n = minWarmupRounds
	}

	rounds := max(min(n, maxWarmupRounds), minWarmupRounds)
	cp.mu.warmupRounds = rounds
	cp.mu.warmupSkipCount = rounds * warmupSkipMultiple
}

// getWarmupParamsWithLock returns the effective warmup parameters for this pool.
// Returns pool-specific values if set, otherwise falls back to defaults.
//
// Caller must hold the pool lock: it reads the mu-guarded warmupRounds and
// warmupSkipCount fields.
func (cp *multiServerPool) getWarmupParamsWithLock() (int, int) {
	rounds := cp.mu.warmupRounds
	if rounds <= 0 {
		rounds = defaultWarmupRounds
	}
	skipCount := cp.mu.warmupSkipCount
	if skipCount <= 0 {
		skipCount = defaultWarmupSkipCount
	}
	return rounds, skipCount
}

// NewConnectionPool creates and returns a default connection pool.
// The selector parameter is accepted for backward compatibility but is not
// used by the internal pool selection logic; the pool uses its own round-robin
// counter. Pass nil for default behavior.
//
//nolint:unparam // public API; selector kept for backward compatibility
func NewConnectionPool(conns []*Connection, selector Selector) ConnectionPool {
	if len(conns) == 1 {
		return newSingleServerPool(conns[0], nil)
	}

	pool := &multiServerPool{
		resurrectTimeoutInitial:      defaultResurrectTimeoutInitial,
		resurrectTimeoutMax:          defaultResurrectTimeoutMax,
		resurrectTimeoutFactorCutoff: defaultResurrectTimeoutFactorCutoff,
		minimumResurrectTimeout:      defaultMinimumResurrectTimeout,
		jitterScale:                  defaultJitterScale,
		serverMaxNewConnsPerSec:      float64(defaultServerCoreCount) * serverMaxNewConnsPerSecMultiplier,
		clientsPerServer:             float64(defaultServerCoreCount),
	}
	pool.mu.ready = conns
	pool.mu.activeCount = len(conns)
	pool.mu.dead = []*Connection{}
	pool.mu.members = make(map[*Connection]struct{}, max(len(conns), defaultMembersCapacity))
	for _, c := range conns {
		pool.mu.members[c] = struct{}{}
	}

	return pool
}

// OnSuccess marks the connection as successful.
func (cp *multiServerPool) OnSuccess(c *Connection) {
	// Establish consistent lock ordering: Pool -> Connection
	cp.mu.Lock()
	defer cp.mu.Unlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Only resurrect if the connection is currently dead
	if c.deadSinceIsZero() {
		return
	}

	// Check if connection is draining (e.g., HTTP/2 GOAWAY received)
	if cp.shouldSkipDraining(c) {
		return
	}

	// Check if connection is overload-demoted
	if cp.shouldSkipOverloaded(c) {
		return
	}

	if dl := loadDebugLogger(); dl != nil {
		dl.Logf("[%s] OnSuccess: %s transitioning from dead to ready\n", cp.name, c.URL)
	}
	c.markAsHealthyWithLock()
	cp.resurrectWithLock(c)
	cp.poolSuccesses.Add(1)

	if obs := observerFromAtomic(&cp.observer); obs != nil {
		obs.OnPromote(newConnectionEvent(cp.name, c, cp.countByLifecycleWithLock()))
	}
}

// shouldSkipDraining returns true if connection is draining and should not be resurrected.
func (cp *multiServerPool) shouldSkipDraining(c *Connection) bool {
	if c.drainingQuiescingRemaining.Load() > 0 {
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("[%s] OnSuccess: %s is draining (quiescing remaining=%d), skipping resurrection\n",
				cp.name, c.URL, c.drainingQuiescingRemaining.Load())
		}
		return true
	}
	return false
}

// shouldSkipOverloaded returns true if connection is overload-demoted and should not be resurrected.
func (cp *multiServerPool) shouldSkipOverloaded(c *Connection) bool {
	if c.loadConnState().lifecycle().has(lcOverloaded) {
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("[%s] OnSuccess: %s is overload-demoted, skipping resurrection (stats poller manages lifecycle)\n", cp.name, c.URL)
		}
		return true
	}
	return false
}

// OnFailure marks the connection as failed.
func (cp *multiServerPool) OnFailure(c *Connection) error {
	cp.mu.Lock()
	holdingCPLock := true
	defer func() {
		if holdingCPLock {
			cp.mu.Unlock()
		}
	}()

	// Transition to dead -- handles Active, Standby, or Overloaded states.
	// A real failure supersedes any previous state.
	c.mu.Lock()
	if c.loadConnState().lifecycle().has(lcUnknown) {
		c.mu.Unlock()
		return nil
	}

	if err := c.casLifecycle(c.loadConnState(), 0, lcDead|lcNeedsWarmup|lcNeedsHardware, lcReady|lcActive|lcStandby|lcOverloaded); err != nil {
		c.mu.Unlock()
		return nil
	}
	c.storeOverloadedAt(time.Time{})
	c.markAsDeadWithLock()
	c.mu.Unlock()

	cp.removeFromReadyWithLock(c)
	cp.appendToDeadWithLock(c)
	cp.poolFailures.Add(1)

	if cp.metrics != nil {
		cp.metrics.connectionsDemoted.Add(1)
	}

	// Sort by failure count for resurrection prioritization.
	slices.SortFunc(cp.mu.dead, func(a, b *Connection) int {
		return cmp.Compare(b.failures.Load(), a.failures.Load())
	})

	// Snapshot standby availability while we hold the lock -- used after unlock
	// to decide whether to spawn async standby promotion (1:1 failure replacement).
	hasStandby := len(cp.mu.ready) > cp.mu.activeCount

	// Build observer event while pool counts are still valid under lock
	var demoteEvent ConnectionEvent
	obs := observerFromAtomic(&cp.observer)
	if obs != nil {
		demoteEvent = newConnectionEvent(cp.name, c, cp.countByLifecycleWithLock())
	}

	// MUST release lock before scheduleResurrect to avoid deadlock:
	// scheduleResurrect needs cp.mu.RLock(), which blocks if we hold cp.mu.Lock()
	holdingCPLock = false
	cp.mu.Unlock()

	if obs != nil {
		obs.OnDemote(demoteEvent)
	}

	// Schedule resurrection after connection has been moved to dead list
	cp.scheduleResurrect(cp.poolCtx(), c)

	// If standby connections are available, asynchronously promote one to fill the gap
	// left by the failed connection (1:1 replacement).
	if hasStandby {
		go cp.asyncPromoteStandby(cp.poolCtx())
	}

	return nil
}

// URLs returns the list of URLs of available connections.
func (cp *multiServerPool) URLs() []*url.URL {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	urls := make([]*url.URL, 0, len(cp.mu.ready)+len(cp.mu.dead))
	for _, c := range cp.mu.ready {
		urls = append(urls, c.URL)
	}
	for _, c := range cp.mu.dead {
		urls = append(urls, c.URL)
	}

	return urls
}

func (cp *multiServerPool) connections() []*Connection {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	conns := make([]*Connection, 0, len(cp.mu.ready)+len(cp.mu.dead))
	conns = append(conns, cp.mu.ready...) // includes both active and standby partitions
	conns = append(conns, cp.mu.dead...)

	return conns
}

// connectionsByState returns snapshots of the ready and dead lists.
func (cp *multiServerPool) connectionsByState() ([]*Connection, []*Connection) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	ready := make([]*Connection, len(cp.mu.ready))
	copy(ready, cp.mu.ready)

	dead := make([]*Connection, len(cp.mu.dead))
	copy(dead, cp.mu.dead)

	return ready, dead
}

// RLock acquires a read lock on the connection pool.
func (cp *multiServerPool) RLock() { cp.mu.RLock() }

// RUnlock releases the read lock on the connection pool.
func (cp *multiServerPool) RUnlock() { cp.mu.RUnlock() }

// Lock acquires a write lock on the connection pool.
func (cp *multiServerPool) Lock() { cp.mu.Lock() }

// Unlock releases the write lock on the connection pool.
func (cp *multiServerPool) Unlock() { cp.mu.Unlock() }

// resurrectWithLock unconditionally moves a connection from dead to the ready list.
// When the active partition is at capacity, the connection lands in the standby
// portion (past activeCount) automatically.
//
// CALLER RESPONSIBILITIES:
//   - Caller must verify connection health before calling this method
//   - Caller must hold both pool lock and connection lock
//   - Connection should exist in the dead list
func (cp *multiServerPool) resurrectWithLock(c *Connection) {
	if dl := loadDebugLogger(); dl != nil {
		dl.Logf("[%s] Resurrecting %q\n", cp.name, c.URL)
	}

	// Clear overloaded state -- node just came back from dead, stats poller will re-evaluate.
	c.storeOverloadedAt(time.Time{})

	c.markAsReadyWithLock()

	// Remove from dead list
	cp.removeFromDeadWithLock(c)
	if cp.metrics != nil {
		cp.metrics.connectionsPromoted.Add(1)
	}

	// Add to ready list. If below cap (or cap disabled), promote to active with warmup.
	// Otherwise the connection lands in the standby portion (warmup deferred to promotion).
	if cp.mu.activeListCap <= 0 || cp.mu.activeCount < cp.mu.activeListCap {
		// Transition state: dead -> active with warmup (lcNeedsWarmup preserved if set)
		c.casLifecycle(c.loadConnState(), 0, lcActive, lcUnknown|lcStandby) //nolint:errcheck // lock held; only errLifecycleNoop possible
		rounds, skip := cp.getWarmupParamsWithLock()
		c.startWarmup(rounds, skip)
		cp.appendToReadyActiveWithLock(c)
		cp.shuffleActiveWithLock()
	} else {
		// Transition state: dead -> standby (warmup deferred to promotion, lcNeedsWarmup preserved)
		c.casLifecycle(c.loadConnState(), 0, lcStandby, lcUnknown|lcActive) //nolint:errcheck // lock held; only errLifecycleNoop possible
		cp.appendToReadyStandbyWithLock(c)
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("[%s] Resurrected %q to standby (active at cap=%d, standby=%d)\n",
				cp.name, c.URL, cp.mu.activeListCap, len(cp.mu.ready)-cp.mu.activeCount)
		}
	}
}

// removeFromReadyWithLock removes ALL entries of a connection from the ready slice.
// A connection may have multiple entries due to weighted round-robin (c.weight > 1).
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
func (cp *multiServerPool) removeFromReadyWithLock(c *Connection) {
	activeRemoved := 0
	writeIdx := 0
	for readIdx := 0; readIdx < len(cp.mu.ready); readIdx++ {
		if cp.mu.ready[readIdx] == c {
			if readIdx < cp.mu.activeCount {
				activeRemoved++
			}
			continue // skip this entry (remove it)
		}
		cp.mu.ready[writeIdx] = cp.mu.ready[readIdx]
		writeIdx++
	}
	if writeIdx == len(cp.mu.ready) {
		return // nothing was removed
	}

	// Clear trailing slots for GC
	for i := writeIdx; i < len(cp.mu.ready); i++ {
		cp.mu.ready[i] = nil
	}
	cp.mu.ready = cp.mu.ready[:writeIdx]
	cp.mu.activeCount -= activeRemoved
}

// removeFromDeadWithLock removes a connection from the dead slice.
// Swap-with-last for O(1) removal once found.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
func (cp *multiServerPool) removeFromDeadWithLock(c *Connection) {
	idx := -1
	// Search backward -- recently appended items are at the tail
	for i, v := range slices.Backward(cp.mu.dead) {
		if v == c {
			idx = i
			break
		}
	}
	if idx < 0 {
		return // not in this dead list
	}

	last := len(cp.mu.dead) - 1
	if idx != last {
		cp.mu.dead[idx] = cp.mu.dead[last]
	}
	cp.mu.dead[last] = nil
	cp.mu.dead = cp.mu.dead[:last]
}

// appendToReadyActiveWithLock appends a connection to the ready slice in the active partition.
// Inserts c.effectiveWeight() copies for weighted round-robin selection.
//
// Enforces ready-list invariants:
//   - Schedules an async RTT probe if the rttRing has no measurements
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
//   - Caller must ensure deadSince is zero (connection is alive)
func (cp *multiServerPool) appendToReadyActiveWithLock(c *Connection) {
	// If RTT is unknown and a health check function is configured, schedule
	// an async one-shot health check to populate the rttRing. This handles
	// connections reused by nodeDiscovery() that were never health-checked.
	if c.rttRing != nil && c.rttRing.medianBucket().IsUnknown() && cp.mu.healthCheck != nil {
		go cp.scheduleRTTProbe(c)
	}

	w := c.effectiveWeight()
	for range w {
		tailIdx := len(cp.mu.ready)
		cp.mu.ready = append(cp.mu.ready, c)
		// Swap into active partition at the boundary
		cp.mu.ready[tailIdx], cp.mu.ready[cp.mu.activeCount] = cp.mu.ready[cp.mu.activeCount], cp.mu.ready[tailIdx]
		cp.mu.activeCount++
	}
}

// appendToReadyStandbyWithLock appends a connection to the standby portion of the ready slice.
//
// Enforces ready-list invariants:
//   - Schedules an async RTT probe if the rttRing has no measurements
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
//   - Caller must ensure deadSince is zero (connection is alive)
func (cp *multiServerPool) appendToReadyStandbyWithLock(c *Connection) {
	// If RTT is unknown and a health check function is configured, schedule
	// an async one-shot health check to populate the rttRing.
	if c.rttRing != nil && c.rttRing.medianBucket().IsUnknown() && cp.mu.healthCheck != nil {
		go cp.scheduleRTTProbe(c)
	}

	w := c.effectiveWeight()
	for range w {
		cp.mu.ready = append(cp.mu.ready, c)
	}
}

// appendToDeadWithLock appends a connection to the dead slice and enforces
// dead-list invariants: deadSince must be set and lifecycle must include lcUnknown.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
func (cp *multiServerPool) appendToDeadWithLock(c *Connection) {
	// Enforce invariants: dead connections must have a timestamp and lcUnknown.
	// Both operations under c.mu to satisfy the casLifecycle contract.
	c.mu.Lock()
	if c.deadSinceIsZero() {
		c.storeDeadSince(time.Now().UTC())
	}
	if !c.loadConnState().lifecycle().has(lcUnknown) {
		c.setLifecycleBit(lcUnknown) //nolint:errcheck // lock held; only errLifecycleNoop possible
	}
	c.mu.Unlock()
	cp.mu.dead = append(cp.mu.dead, c)
}

// scheduleRTTProbe fires an async health check solely to populate the rttRing
// for a connection that has no RTT measurements. Uses proactiveCheck.mu as a
// rate-limiter -- if another probe is already running, this is a no-op.
//
// The first health check on an idle TLS connection includes handshake overhead,
// so we run two: the first populates TLS state, the second gives a clean RTT.
func (cp *multiServerPool) scheduleRTTProbe(c *Connection) {
	// Rate-limit: bail if another probe is in-flight.
	if !c.proactiveCheck.mu.TryLock() {
		return
	}
	defer c.proactiveCheck.mu.Unlock()

	ctx := cp.poolCtx()

	// First probe: amortizes TLS handshake cost -- don't record RTT.
	cp.performHealthCheck(ctx, c, false)

	// Second probe: clean RTT measurement.
	cp.performHealthCheck(ctx, c, true)
}

// shuffleActiveWithLock randomizes the order of active connections to prevent hot-spotting.
// Only shuffles ready[:activeCount] to preserve the active/standby partition.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold pool write lock
func (cp *multiServerPool) shuffleActiveWithLock() {
	if cp.mu.activeCount <= 1 {
		return
	}
	rand.Shuffle(cp.mu.activeCount, func(i, j int) {
		cp.mu.ready[i], cp.mu.ready[j] = cp.mu.ready[j], cp.mu.ready[i]
	})
}
