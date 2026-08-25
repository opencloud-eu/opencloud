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
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for casLifecycle.
var (
	// errLifecycleConflict indicates that masked lifecycle bits were mutated by
	// a concurrent transition between the initial load and the CAS attempt.
	errLifecycleConflict = errors.New("lifecycle conflict: masked bits mutated concurrently")

	// errLifecycleNoop indicates that the requested transition would not change
	// the lifecycle state (bits already in the desired configuration).
	errLifecycleNoop = errors.New("lifecycle noop: state already matches")
)

// ---------------------------------------------------------------------------
// Connection lifecycle methods -- operate on *Connection using lifecycle types
// ---------------------------------------------------------------------------

// loadConnState returns the current packed state of the connection.
// Safe to call without holding any lock -- atomics are for "dirty reads"
// for metrics or reporting, but not for state changes.
// For pool-placement decisions, use isReady under c.mu.RLock.
func (c *Connection) loadConnState() connState { return connState(c.state.Load()) }

// isReady reports whether the connection should be eligible to be in the
// ready list or not (lcActive or lcStandby). Returns false for connections
// that belong on the dead list (lcUnknown / lcDead -- no position bit set).
//
// A connection state change does not propagate to all policies and their
// connection pools, so lifecycle bits serve as hints for cleanup when a pool
// is exercised via its respective activities (which may be never if all
// requests hit an upstream policy). A connection could be in one pool's dead
// list but ready-eligible because another pool already promoted it, or in
// the ready list as standby and therefore correctly placed.
//
// Every pool operation that makes placement decisions -- DiscoveryUpdate,
// Next, CheckDead, RotateStandby -- should consult lifecycle bits rather
// than timestamp fields (deadSince). Multiple pools share the same
// *Connection, so timestamps set by one pool's appendToDeadWithLock are
// visible to other pools and do not reflect that pool's own state.
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold c.mu.RLock (or c.mu.Lock).
//
// Usage requires an atomic Load as a dirty read, but changing a connection's
// state requires c.mu.RLock (or c.mu.Lock) and predicate re-evaluation
// before resetting a bit to indicate a state change.
func (c *Connection) isReady() bool {
	lc := c.loadConnState().lifecycle()
	return lc.has(lcActive) || lc.has(lcStandby)
}

// casLifecycle atomically modifies lifecycle bits with conflict detection.
//
// Parameters:
//   - current: pre-loaded connState (avoids redundant atomic Load on first attempt)
//   - conflict: additional bits to monitor for concurrent mutation (beyond
//     those implied by set/clear)
//   - set: bits to OR into the lifecycle byte
//   - clear: bits to AND-NOT out of the lifecycle byte
//
// The effective conflict mask is (conflict | set | clear). Any bits the caller
// is setting, clearing, or explicitly watching are automatically protected:
// if a CAS retry observes that any masked bits differ from their initial
// snapshot value, the loop bails (returns false) rather than overwriting a
// concurrent lifecycle transition.
//
// The operation computes: next = (lifecycle | set) &^ clear
//
// Uses a CAS loop because lock-free warmup decrements (tryWarmupSkip) may
// modify the lower 52 bits concurrently. If the CAS fails and any masked
// bits differ from their initial value on re-load, the loop bails rather
// than retrying -- a mutation in those bits indicates a concurrent lifecycle
// transition (race) that this caller should not overwrite.
//
// Returns nil if the lifecycle was changed. Returns errLifecycleConflict if
// masked bits were mutated concurrently, or errLifecycleNoop if the computed
// next state equals the current state (no change needed).
//
// CALLER RESPONSIBILITIES:
//   - Caller must hold c.mu (Lock or TryLock).
//   - Lock ordering: cp.mu -> c.mu (never the reverse).
func (c *Connection) casLifecycle(
	current connState,
	conflict, set, clr connLifecycle,
) error {
	mask := conflict | set | clr
	raw := int64(current)
	initialMasked := current.lifecycle() & mask
	for {
		lc := connState(raw).lifecycle()
		if lc&mask != initialMasked {
			return errLifecycleConflict
		}
		next := (lc | set) &^ clr
		if next == lc {
			return errLifecycleNoop
		}
		target := connState(raw).withLifecycle(next)
		if c.state.CompareAndSwap(raw, int64(target)) {
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("casLifecycle: %s -> %s on %s\n", lc, next, c.URL)
			}
			return nil
		}
		// CAS failed -- re-load and re-check masked bits before retrying.
		raw = c.state.Load()
	}
}

// setLifecycleBit atomically sets a single metadata bit.
// Returns nil if the bit was newly set, errLifecycleNoop if already set,
// or errLifecycleConflict if a concurrent transition mutated the bit.
func (c *Connection) setLifecycleBit(bit connLifecycle) error {
	return c.casLifecycle(c.loadConnState(), 0, bit, 0)
}

// clearLifecycleBit atomically clears a single metadata bit.
// Returns nil if the bit was cleared, errLifecycleNoop if already clear,
// or errLifecycleConflict if a concurrent transition mutated the bit.
func (c *Connection) clearLifecycleBit(bit connLifecycle) error {
	return c.casLifecycle(c.loadConnState(), 0, 0, bit)
}

// setNeedsCatUpdate marks this connection as needing a /_cat/shards refresh
// before it can participate in shard-aware routing. The connection remains
// available for general routing (round-robin, zombie tryouts) but is
// excluded from rendezvousTopK candidate sets until the flag is cleared
// by a successful shard placement refresh.
func (c *Connection) setNeedsCatUpdate() error {
	return c.setLifecycleBit(lcNeedsCatUpdate)
}

// clearNeedsCatUpdate removes the shard-placement-stale flag, allowing
// the connection to participate in shard-aware routing again. Called after
// a successful /_cat/shards refresh confirms current shard placement.
func (c *Connection) clearNeedsCatUpdate() error {
	return c.clearLifecycleBit(lcNeedsCatUpdate)
}

// needsCatUpdate reports whether this connection has been flagged as needing
// a /_cat/shards refresh. When true, the connection is excluded from shard-aware
// routing candidate sets.
func (c *Connection) needsCatUpdate() bool {
	return c.loadConnState().lifecycle().has(lcNeedsCatUpdate)
}

// hasClusterHealth reports whether this connection has been probed and supports
// /_cluster/health?local=true.
func (c *Connection) hasClusterHealth() bool {
	lc := c.loadConnState().lifecycle()
	return lc.has(lcClusterHealthProbed) && lc.has(lcClusterHealthAvailable)
}

// clusterHealthPending reports whether cluster health has never been probed on
// this connection (neither probed nor available bits set).
func (c *Connection) clusterHealthPending() bool {
	lc := c.loadConnState().lifecycle()
	return !lc.has(lcClusterHealthProbed) && !lc.has(lcClusterHealthAvailable)
}

// clusterHealthUnavailable reports whether cluster health was probed and found
// unavailable (probed is set, available is not).
func (c *Connection) clusterHealthUnavailable() bool {
	lc := c.loadConnState().lifecycle()
	return lc.has(lcClusterHealthProbed) && !lc.has(lcClusterHealthAvailable)
}

// ---------------------------------------------------------------------------
// connLifecycle -- 12-bit packed lifecycle bitfield
// ---------------------------------------------------------------------------

// connLifecycle represents a connection's lifecycle as a 12-bit value packed
// into the top 12 bits of the 64-bit connState word.
//
// All values are power-of-2 bits. Use has() to test any bit or combination.
//
// The bits are organized in four groups:
//
// Readiness (bits 0-1) -- mutually exclusive. Exactly one must be set:
//
//	lcReady   (0x01) -- connection is believed to be functional
//	lcUnknown (0x02) -- connection status is uncertain; needs health check
//
// Position (bits 2-3) -- mutually exclusive. At most one set; neither = dead list:
//
//	lcActive  (0x04) -- in ready[:activeCount], serving requests
//	lcStandby (0x08) -- in ready[activeCount:], idle
//
// When neither position bit is set, the connection is in the dead list.
// "Dead" is not a separate state -- it is an unknown connection with no position.
//
// Metadata (bits 4-7) -- independent flags, freely combinable:
//
//	lcNeedsWarmup    (0x10) -- needs warmup before full traffic
//	lcOverloaded     (0x20) -- node resource overload; parked in standby
//	lcHealthChecking (0x40) -- health check goroutine running
//	lcDraining       (0x80) -- HTTP/2 GOAWAY; no new requests
//
// Extended metadata (bits 8-11) -- independent flags, freely combinable:
//
//	lcNeedsHardware          (0x0100) -- needs (re)fetch of hardware info (/_nodes/_local/http,os)
//	lcNeedsCatUpdate         (0x0200) -- shard placement stale; excluded from shard-aware routing until /_cat/shards refresh
//	lcClusterHealthProbed    (0x0400) -- node has been probed for /_cluster/health capability
//	lcClusterHealthAvailable (0x0800) -- probe succeeded; /_cluster/health?local=true is usable
//	lcViable                 (0x1000) -- ever proven directly reachable; monotonic latch, never cleared by failure
//
// Metadata bits are observability signals for metrics and monitoring.
// They do NOT serve as concurrency guards -- mutexes and actual field
// values (e.g., checkStartedAt, deadSince) are the source of truth.
//
// State changes (readiness, position, metadata) require conn.mu.Lock().
// The warmup managers in the lower 52 bits of connState can be decremented
// via atomic CAS without holding the lock.
type connLifecycle int64

// Readiness bits (mutually exclusive -- exactly one must be set).
const (
	lcReady   connLifecycle = 0x01 // believed functional
	lcUnknown connLifecycle = 0x02 // status uncertain; needs verification
)

// Position bits (mutually exclusive -- at most one set; neither = dead list).
const (
	lcActive  connLifecycle = 0x04 // in active partition, serving requests
	lcStandby connLifecycle = 0x08 // in standby partition, idle
)

// Metadata flags (independent -- freely combinable with readiness and position).
const (
	lcNeedsWarmup    connLifecycle = 0x0010 // needs warmup before full traffic
	lcOverloaded     connLifecycle = 0x0020 // node resource overload; parked in standby
	lcHealthChecking connLifecycle = 0x0040 // health check goroutine running
	lcDraining       connLifecycle = 0x0080 // HTTP/2 GOAWAY; no new requests
	// lcNeedsHardware marks that hardware info (/_nodes/_local/http,os) needs a
	// (re)fetch. It may be re-set on a reset/scaling event and does NOT imply the
	// node was never reachable -- see lcViable for "ever reachable".
	lcNeedsHardware  connLifecycle = 0x0100
	lcNeedsCatUpdate connLifecycle = 0x0200 // shard placement stale; excluded from shard-aware routing until /_cat/shards refresh

	// lcClusterHealthProbed and lcClusterHealthAvailable together encode a
	// tri-state for the one-time /_cluster/health capability probe (see
	// probeClusterHealthLocal), which detects whether the node's credentials
	// carry the cluster:monitor/health privilege the Security plugin requires.
	// Two bits are needed because the state machine has three cases, and a
	// single bit cannot distinguish "not yet asked" from "asked, answer was no":
	//
	//   - pending      (neither set): never probed; probe on the next cycle.
	//   - available    (both set):    probe returned 200; use /_cluster/health?local=true.
	//   - unavailable  (probed only): probe returned 401/403; the privilege is
	//                                 absent, so keep using GET / and retry the
	//                                 probe only after MaxRetryClusterHealth.
	//
	// The bits are cleared together to force a re-probe (e.g. on demotion). They
	// describe an authorization capability, not reachability, so lcViable is a
	// separate bit rather than a reuse of either: a node can be reachable
	// (lcViable) yet lack the health privilege (probed, not available), and the
	// two must be tracked independently.
	lcClusterHealthProbed    connLifecycle = 0x0400 // cluster-health probe has run (see the block comment above)
	lcClusterHealthAvailable connLifecycle = 0x0800 // cluster-health probe returned 200 (see the block comment above)

	// lcViable records that a connection has been proven directly reachable at
	// least once: we set it on the first successful health check or request
	// (markAsReadyWithLock/markAsHealthyWithLock), and on a user-supplied seed at
	// construction. It is monotonic -- once set it is never cleared by failure,
	// so it answers "was this endpoint ever reachable", which lcNeedsHardware
	// (re-set on any failure or scaling event) cannot.
	//
	// availableForRouting gates on lcViable (seeds short-circuit): a connection
	// without it -- e.g. a freshly discovered node whose publish_address may be
	// unroutable (a NAT'd or misconfigured cluster) -- is neither routed to nor
	// served as a last-resort zombie, so a pool holding only such nodes reports
	// ErrNoConnections and the request cascades to the seed-URL fallback.
	lcViable connLifecycle = 0x1000
)

// Compound aliases for common lifecycle combinations.
const (
	// lcDead is an unknown connection with no position (in the dead list).
	// This is a convenience alias -- "dead" means lcUnknown without any position bit.
	lcDead = lcUnknown
)

// has returns true if all specified bits are set.
// Safe for any bits -- readiness, position, and metadata are all power-of-2.
func (lc connLifecycle) has(flags connLifecycle) bool {
	return lc&flags == flags
}

// hasAny reports whether any of the given flags are set.
func (lc connLifecycle) hasAny(flags connLifecycle) bool {
	return lc&flags != 0
}

// Lifecycle state name constants used by connLifecycleBits and test assertions.
const (
	lcNameReady                  = "ready"
	lcNameUnknown                = "unknown"
	lcNameActive                 = "active"
	lcNameStandby                = "standby"
	lcNameNeedsWarmup            = "needsWarmup"
	lcNameOverloaded             = "overloaded"
	lcNameHealthChecking         = "healthChecking"
	lcNameDraining               = "draining"
	lcNameNeedsHardware          = "needsHardware"
	lcNameNeedsCatUpdate         = "needsCatUpdate"
	lcNameClusterHealthProbed    = "clusterHealthProbed"
	lcNameClusterHealthAvailable = "clusterHealthAvailable"
	lcNameViable                 = "viable"
)

// connLifecycleBits maps each bit to its human-readable name.
var connLifecycleBits = [13]struct { //nolint:gochecknoglobals // lookup table, not mutable state
	bit  connLifecycle
	name string
}{
	{lcReady, lcNameReady},
	{lcUnknown, lcNameUnknown},
	{lcActive, lcNameActive},
	{lcStandby, lcNameStandby},
	{lcNeedsWarmup, lcNameNeedsWarmup},
	{lcOverloaded, lcNameOverloaded},
	{lcHealthChecking, lcNameHealthChecking},
	{lcDraining, lcNameDraining},
	{lcNeedsHardware, lcNameNeedsHardware},
	{lcNeedsCatUpdate, lcNameNeedsCatUpdate},
	{lcClusterHealthProbed, lcNameClusterHealthProbed},
	{lcClusterHealthAvailable, lcNameClusterHealthAvailable},
	{lcViable, lcNameViable},
}

// String returns a human-readable name for the lifecycle.
// Format: "flag+flag+... (0000000000101)" -- set bits named, followed by binary.
func (lc connLifecycle) String() string {
	var b strings.Builder
	for _, entry := range connLifecycleBits {
		if lc.has(entry.bit) {
			if b.Len() > 0 {
				b.WriteByte('+')
			}
			b.WriteString(entry.name)
		}
	}
	if b.Len() == 0 {
		b.WriteString("none")
	}
	return fmt.Sprintf("%s (%013b)", b.String(), uint16(lc)) //nolint:gosec // G115: only bottom 13 bits used
}

// ---------------------------------------------------------------------------
// warmupManager -- packed warmup {rounds, skipCount} pair
// ---------------------------------------------------------------------------

// warmupManager represents a {rounds, skipCount} pair packed into the lower
// 16 bits of a 26-bit field. The upper 10 bits are reserved for future use.
//
// Bit layout within the 26-bit field:
//
//	25              16 15     8 7        0
//	+----------------+--------+---------+
//	|   reserved     | rounds |skipCount|
//	+----------------+--------+---------+
//	    10 bits        8 bits    8 bits
//
// rounds (8 bits, 0-255): The number of warmup rounds remaining.
// skipCount (8 bits, 0-255): The number of requests to skip in the current round.
type warmupManager int32

const (
	wmRoundsShift   = 8
	wmRoundsMask    = 0xFF // 8 bits
	wmSkipCountMask = 0xFF // 8 bits
)

// packWarmupManager constructs a warmupManager from individual fields.
func packWarmupManager(rounds, skipCount int) warmupManager {
	return warmupManager(
		(int32(rounds)&wmRoundsMask)<<wmRoundsShift | //nolint:gosec // G115: masked to 8 bits, overflow impossible
			(int32(skipCount) & wmSkipCountMask), //nolint:gosec // G115: masked to 8 bits, overflow impossible
	)
}

// rounds returns the rounds field (8 bits, 0-255).
func (wm warmupManager) rounds() int {
	return int((int32(wm) >> wmRoundsShift) & wmRoundsMask)
}

// skipCount returns the skipCount field (8 bits, 0-255).
func (wm warmupManager) skipCount() int {
	return int(int32(wm) & wmSkipCountMask)
}

// isZero returns true if both rounds and skipCount are zero.
func (wm warmupManager) isZero() bool { return wm == 0 }

// withRounds returns a new warmupManager with the rounds field replaced.
func (wm warmupManager) withRounds(rounds int) warmupManager {
	return warmupManager(
		(int32(rounds)&wmRoundsMask)<<wmRoundsShift | //nolint:gosec // G115: masked to 8 bits, overflow impossible
			(int32(wm) & wmSkipCountMask),
	)
}

// withSkipCount returns a new warmupManager with the skipCount field replaced.
func (wm warmupManager) withSkipCount(skipCount int) warmupManager {
	return warmupManager(
		(int32(wm) >> wmRoundsShift & wmRoundsMask << wmRoundsShift) |
			(int32(skipCount) & wmSkipCountMask), //nolint:gosec // G115: masked to 8 bits, overflow impossible
	)
}

// ---------------------------------------------------------------------------
// Backward-compatible aliases
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// connState -- packed 64-bit connection state word
// ---------------------------------------------------------------------------

// connState encodes a connection's full state as a 64-bit packed word stored
// in an atomic.Int64. It combines the lifecycle with two warmupManager
// values that implement non-linear warm-up.
//
// Bit layout:
//
//	63      51 50           26 25            0
//	+----------+--------------+--------------+
//	|    LC    | warmupConfig | warmupState  |
//	+----------+--------------+--------------+
//	  13 bits      25 bits        26 bits
//
// LC detail (connLifecycle, 13 bits):
//
//	12    11   8   7        4  3     2  1      0
//	+-----+------+-----------+-------+---------+
//	|viable| ext | metadata  |  pos  |readiness|
//	+-----+------+-----------+-------+---------+
//	 1 bit  4 bits   4 bits    2 bits   2 bits
//
// readiness (bits 0-1):  lcReady=0x01, lcUnknown=0x02 (mutually exclusive)
// position  (bits 2-3):  lcActive=0x04, lcStandby=0x08 (at most one; 0=dead)
// metadata  (bits 4-7):  lcNeedsWarmup, lcOverloaded, lcHealthChecking, lcDraining
// extended  (bits 8-11): lcNeedsHardware=0x100, lcNeedsCatUpdate=0x200, lcClusterHealthProbed=0x400, lcClusterHealthAvailable=0x800
// viable    (bit 12):    lcViable=0x1000 (monotonic latch: ever proven directly reachable)
//
// Each warmupManager field uses the lower 16 bits (warmupConfig has 25 bits, warmupState 26):
//
//	25              16 15     8 7        0
//	+----------------+--------+---------+
//	|   reserved     | rounds |skipCount|
//	+----------------+--------+---------+
//	    10 bits        8 bits    8 bits
//
// When both warmupManagers are zero, the connection is fully warmed (or warmup
// was never enabled). This gives a zero-cost fast path: if the lower 52 bits are
// all zero, skip warmup checks entirely.
type connState int64

const (
	csLifecycleShift = 51
	csLifecycleMask  = 0x1FFF // 13 bits

	csLifecycleMgrShift = 26
	csLifecycleMgrMask  = 0x01FFFFFF // 25 bits

	csRoundMgrShift = 0
	csRoundMgrMask  = 0x03FFFFFF // 26 bits

	// csLowerBitsMask clears the top 13 lifecycle bits, keeping the lower 51 bits.
	// Defined as a positive constant to avoid int64 overflow.
	csLowerBitsMask int64 = (1 << csLifecycleShift) - 1 // 0x0007FFFFFFFFFFFF
)

// Default warmup parameters.
const (
	defaultWarmupRounds    = 16
	defaultWarmupSkipCount = 32

	// Dynamic warmup bounds. When activeListCap is known, warmup parameters
	// are scaled to the pool size so small pools warm quickly while large
	// pools ramp gradually.
	minWarmupRounds    = 4                   // floor: at least 4 rounds even for 1-node pools
	maxWarmupRounds    = defaultWarmupRounds // ceiling: matches the static default
	warmupSkipMultiple = 2                   // skipCount = rounds * warmupSkipMultiple
)

// newConnState creates a connState with the given lifecycle and zero managers.
func newConnState(lc connLifecycle) connState {
	return connState(int64(lc&csLifecycleMask) << csLifecycleShift)
}

// packConnState constructs a full connState from lifecycle and both managers.
func packConnState(lc connLifecycle, lcMgr, rdMgr warmupManager) connState {
	return connState(
		(int64(lc&csLifecycleMask) << csLifecycleShift) |
			(int64(lcMgr&warmupManager(csLifecycleMgrMask)) << csLifecycleMgrShift) |
			(int64(rdMgr&warmupManager(csRoundMgrMask)) << csRoundMgrShift),
	)
}

// lifecycle returns the connLifecycle portion (top 12 bits).
func (s connState) lifecycle() connLifecycle {
	return connLifecycle((int64(s) >> csLifecycleShift) & csLifecycleMask)
}

// lifecycleManager returns the lifecycle warmupManager (immutable template values).
func (s connState) lifecycleManager() warmupManager {
	return warmupManager((int64(s) >> csLifecycleMgrShift) & csLifecycleMgrMask)
}

// roundManager returns the round warmupManager (working values).
func (s connState) roundManager() warmupManager {
	return warmupManager((int64(s) >> csRoundMgrShift) & csRoundMgrMask)
}

// withLifecycle returns a new connState with the lifecycle replaced, keeping managers.
func (s connState) withLifecycle(lc connLifecycle) connState {
	return connState(
		(int64(lc&csLifecycleMask) << csLifecycleShift) |
			(int64(s) & csLowerBitsMask),
	)
}

// withManagers returns a new connState with both managers replaced, keeping lifecycle.
func (s connState) withManagers(lcMgr, rdMgr warmupManager) connState {
	return packConnState(s.lifecycle(), lcMgr, rdMgr)
}

// isWarmingUp returns true if the lifecycle manager has non-zero values,
// indicating warmup is in progress.
func (s connState) isWarmingUp() bool {
	return s.lifecycleManager() != 0
}

// warmupState creates a connState with the given lifecycle and warmup managers set.
func warmupState(lc connLifecycle, maxRounds, maxSkipCount int) connState {
	lcMgr := packWarmupManager(maxRounds, maxSkipCount)
	rdMgr := packWarmupManager(maxRounds, maxSkipCount)
	return packConnState(lc, lcMgr, rdMgr)
}

// ---------------------------------------------------------------------------
// ConnState -- exported read-only state snapshot for observers
// ---------------------------------------------------------------------------

// ConnState is a read-only snapshot of a connection's packed state word.
//
// External packages cannot construct ConnState values; they are created
// internally and delivered to observers via ConnectionEvent.State.
type ConnState struct {
	packed int64
}

// IsWarmingUp returns true if the connection is still going through warmup
// rounds (non-linear ramp-up of traffic).
func (s ConnState) IsWarmingUp() bool { return connState(s.packed).isWarmingUp() }

// IsHealthChecking reports whether a health check goroutine is running.
// This is an observability signal -- checkStartedAt under c.mu remains
// the authoritative concurrency guard.
func (s ConnState) IsHealthChecking() bool {
	return connState(s.packed).lifecycle().has(lcHealthChecking)
}

// WarmupRoundsRemaining returns the number of warmup rounds still pending.
// Returns 0 when the connection is fully warmed or warmup was never enabled.
func (s ConnState) WarmupRoundsRemaining() int {
	return connState(s.packed).roundManager().rounds()
}

// WarmupSkipRemaining returns the number of requests to skip before the next
// accept in the current warmup round. Returns 0 when the connection is fully
// warmed or warmup was never enabled.
func (s ConnState) WarmupSkipRemaining() int {
	return connState(s.packed).roundManager().skipCount()
}

// String returns a compact human-readable representation of the state.
func (s ConnState) String() string {
	lc := connState(s.packed).lifecycle()
	if !s.IsWarmingUp() {
		return lc.String()
	}
	return fmt.Sprintf("%s(warmup: rounds=%d, skip=%d)",
		lc, s.WarmupRoundsRemaining(), s.WarmupSkipRemaining())
}

// Hex returns the packed state as a hex string with decoded field annotations.
// Format: "0xLLLLLLLLLLLLLLLL [LC=name cfg(rnds=N,skip=N) rd(rnds=N,skip=N)]"
//
// The full 64-bit layout is:
//
//	bits 63-52: lifecycle (12 bits: 2-bit readiness + 2-bit position + 4-bit metadata + 4-bit extended)
//	bits 51-26: warmupConfig -- template (26 bits; rounds=bits 41-34, skip=bits 33-26)
//	bits 25-0:  warmupState -- working (26 bits; rounds=bits 15-8,  skip=bits 7-0)
func (s ConnState) Hex() string {
	cs := connState(s.packed)
	lc := cs.lifecycle()
	lcMgr := cs.lifecycleManager()
	rdMgr := cs.roundManager()
	return fmt.Sprintf("0x%016X [LC=%s cfg(rnds=%d,skip=%d) rd(rnds=%d,skip=%d)]",
		uint64(s.packed), //nolint:gosec // G115: packed is a bitfield, not a signed magnitude
		lc, lcMgr.rounds(), lcMgr.skipCount(), rdMgr.rounds(), rdMgr.skipCount())
}
