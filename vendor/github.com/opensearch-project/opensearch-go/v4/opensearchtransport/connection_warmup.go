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

// startWarmup atomically sets the connection into warmup mode with the given parameters.
// Preserves the current lifecycle bits and only sets the warmup managers.
// If the connection is already warming up (lifecycle managers non-zero), this is a no-op
// to prevent multiple policy pools from resetting an in-progress warmup on the same
// shared Connection during DiscoveryUpdate.
func (c *Connection) startWarmup(maxRounds, maxSkipCount int) {
	lcMgr := packWarmupManager(maxRounds, maxSkipCount)
	rdMgr := packWarmupManager(maxRounds, maxSkipCount)
	for {
		current := c.state.Load()
		cs := connState(current)
		if !cs.lifecycle().has(lcNeedsWarmup) {
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("startWarmup: NO-OP %s (no lcNeedsWarmup, lc=%s)\n", c.URL, cs.lifecycle())
			}
			return // No warmup needed -- connection was proven (e.g. cap demotion)
		}
		if cs.isWarmingUp() {
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("startWarmup: NO-OP %s (already warming, lcMgr=%v rdMgr=%v)\n",
					c.URL, cs.lifecycleManager(), cs.roundManager())
			}
			return // Already warming -- don't reset
		}
		target := packConnState(cs.lifecycle(), lcMgr, rdMgr)
		if c.state.CompareAndSwap(current, int64(target)) {
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("startWarmup: SET %s (rounds=%d, skip=%d)\n", c.URL, maxRounds, maxSkipCount)
			}
			return
		}
		if dl := loadDebugLogger(); dl != nil {
			dl.Logf("startWarmup: CAS race on %p (state=%s during attempt)\n",
				c, ConnState{packed: current}.Hex())
		}
	}
}

// clearWarmup atomically clears warmup by setting managers to zero while preserving lifecycle.
// Uses a CAS loop to avoid clobbering concurrent lifecycle transitions.
func (c *Connection) clearWarmup() {
	for {
		current := c.state.Load()
		lc := connState(current).lifecycle()
		target := newConnState(lc)
		if c.state.CompareAndSwap(current, int64(target)) {
			return
		}
	}
}

// warmupResult represents the outcome of a tryWarmupSkip call.
type warmupResult int

const (
	// warmupSkipped -- request was skipped; try the next connection.
	warmupSkipped warmupResult = iota

	// warmupAccepted -- request was accepted; serve it on this connection.
	warmupAccepted

	// warmupInactive -- no warmup in progress (either completed or never started).
	warmupInactive
)

// smoothstepSkip computes the skip count for a warmup round using a smoothstep
// (Hermite) decay curve: skip = maxSkip * (1 - 3t^2 + 2t^3), where t = d/R.
//
// This produces an S-shaped acceptance ramp: slow initial growth, accelerating
// middle, and steep approach to full traffic at the end -- matching the JVM
// HotSpot JIT compilation profile (interpret -> C1 -> C2 -> steady state).
//
// Unlike the previous linear decay (maxSkip - d), smoothstep guarantees the
// skip count reaches 0 by the final round regardless of the maxSkip/maxRounds
// ratio. Integer arithmetic only; no floating point.
//
// Intermediate values: maxSkip(255) * maxRounds^3(255^3~=16.6M) ~=4.2B -- fits
// comfortably in int (64 bits on all supported platforms).
func smoothstepSkip(maxSkip, maxRounds, delta int) int {
	if maxRounds <= 0 {
		return 0
	}
	// 1 - smoothstep(t) = (R^3 - 3*d^2*R + 2*d^3) / R^3
	r3 := maxRounds * maxRounds * maxRounds
	num := maxSkip * (r3 - 3*delta*delta*maxRounds + 2*delta*delta*delta)
	result := num / r3
	if result < 0 {
		return 0 // Defensive: should not happen for d in [0, R]
	}
	return result
}

// tryWarmupSkip attempts to advance the warmup state via CAS on Connection.state.
//
// The lifecycle manager (lcMgr) is an immutable template that records the
// initial warmup parameters. Only the round manager (rdMgr) is modified:
//
//   - SKIP (remSkip > 0): decrement rdMgr.skipCount, return warmupSkipped.
//   - ACCEPT (remSkip == 0): decrement rdMgr.rounds, compute rdMgr.skipCount
//     for the next round using smoothstep decay, return warmupAccepted.
//   - COMPLETE (newRounds <= 0): clear both managers to zero, return warmupAccepted.
//   - INACTIVE (lcMgr zero): return warmupInactive -- warmup was never started or
//     already completed by another goroutine.
//
// The smoothstep curve (3t^2 - 2t^3) decays the skip count from lcMgr.skipCount
// to 0 across all rounds. For warmupState(16, 32):
//
//	Round 16: skip 32 -> accept -> Round 15: skip 31 -> ... -> Round 1: skip 0 -> accept -> done
//
// This produces an S-shaped acceptance ramp: ~3% initially, accelerating through
// the middle rounds, reaching 50-100% in the final rounds. The skip count always
// reaches 0 by the last round, regardless of the skipCount/rounds ratio.
//
// This method is lock-free and safe for concurrent callers.
func (c *Connection) tryWarmupSkip() warmupResult {
	for {
		raw := c.state.Load()
		current := connState(raw)

		lcMgr := current.lifecycleManager()
		rdMgr := current.roundManager()

		// No warmup in progress (both managers zero).
		if lcMgr.isZero() {
			return warmupInactive
		}

		remSkip := rdMgr.skipCount()
		if remSkip > 0 {
			// SKIP: decrement remaining skip count for this round.
			newRdMgr := rdMgr.withSkipCount(remSkip - 1)
			newState := current.withManagers(lcMgr, newRdMgr)
			if c.state.CompareAndSwap(raw, int64(newState)) {
				if dl := loadDebugLogger(); dl != nil {
					dl.Logf("tryWarmupSkip: SKIP %s (rd: rounds=%d skip=%d->%d)\n",
						c.URL, rdMgr.rounds(), remSkip, remSkip-1)
				}
				return warmupSkipped
			}
			continue // CAS failed, retry
		}

		// remSkip == 0: ACCEPT this request, advance to next round.
		// lcMgr stays unchanged (immutable template).
		newRounds := rdMgr.rounds() - 1

		if newRounds <= 0 {
			// Warmup complete -- clear lcNeedsWarmup, set lcReady.
			lc := (current.lifecycle() &^ lcNeedsWarmup) | lcReady
			if c.state.CompareAndSwap(raw, int64(newConnState(lc))) {
				if dl := loadDebugLogger(); dl != nil {
					dl.Logf("tryWarmupSkip: COMPLETE %s (warmup done)\n", c.URL)
				}
				return warmupAccepted
			}
			continue // CAS failed, retry
		}

		// Compute skip count for the next round using smoothstep decay.
		// delta = how many rounds have elapsed since warmup started.
		// smoothstep guarantees skip reaches 0 by the final round.
		delta := lcMgr.rounds() - newRounds
		newSkip := smoothstepSkip(lcMgr.skipCount(), lcMgr.rounds(), delta)

		newRdMgr := packWarmupManager(newRounds, newSkip)
		newState := current.withManagers(lcMgr, newRdMgr)
		if c.state.CompareAndSwap(raw, int64(newState)) {
			if dl := loadDebugLogger(); dl != nil {
				dl.Logf("tryWarmupSkip: ACCEPT %s (round %d->%d, next skip=%d)\n",
					c.URL, rdMgr.rounds(), newRounds, newSkip)
			}
			return warmupAccepted
		}
		// CAS failed, retry
	}
}
