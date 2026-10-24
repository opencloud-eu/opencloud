// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"math"
	"sync/atomic"
	"time"
)

// clock provides timestamps for time-weighted counters.
// Production code uses realClock; tests inject a testClock for
// deterministic behavior.
type clock interface {
	Now() time.Time
}

// realClock reads the system clock.
type realClock struct{}

// Now returns the current wall-clock time.
func (realClock) Now() time.Time { return time.Now() }

const (
	// defaultDecayFactor is the per-request multiplicative decay for [decayCounter].
	// At constant request rate r, the counter converges to r/(1-decay).
	// With decay=0.999 and r=1 req/call: steady state ~ 1000.
	defaultDecayFactor = 0.999

	// defaultLoadHalfLife controls how fast the time-weighted load
	// counter decays. A 5-second half-life means:
	//   - At steady state with rate R and cost v: counter ~= v*R / lambda ~= v*R*7.2
	//   - Counter halves in 5s with no new requests
	//   - 99% decay in ~33s
	//   - Responsive enough for sub-second routing adjustments
	defaultLoadHalfLife = 5 * time.Second
)

// loadDecayLambda is the precomputed decay rate for the time-weighted
// load counter: lambda = ln(2) / halfLife.
//
//nolint:gochecknoglobals // Precomputed constant derived from defaultLoadHalfLife.
var loadDecayLambda = math.Ln2 / defaultLoadHalfLife.Seconds()

// decayCounter is an exponentially decaying load accumulator.
//
// On each add:
//
//	counter = counter * decay + value
//
// The counter converges to a steady-state that reflects both request
// rate and per-request cost. When requests stop, the counter decays
// toward zero with each subsequent add call or explicit decay step.
//
// This provides self-stabilizing load tracking without periodic resets:
// old load naturally fades, no sawtooth patterns, no coordination.
//
// The value is stored as float64 bits in an atomic.Uint64 for lock-free
// CAS updates.
type decayCounter struct {
	bits atomic.Uint64
}

// increment applies one decay step and adds 1.0 to the counter.
// Returns the new value. Safe for concurrent use (CAS loop).
// Used by indexSlot.requestDecay for per-index request volume tracking.
func (c *decayCounter) increment(decay float64) float64 {
	for {
		old := c.bits.Load()
		oldVal := math.Float64frombits(old)
		newVal := oldVal*decay + 1.0
		if c.bits.CompareAndSwap(old, math.Float64bits(newVal)) {
			return newVal
		}
	}
}

// add applies one decay step and adds a variable amount to the counter.
// Returns the new value. Safe for concurrent use (CAS loop).
// Used for CPU-load-based routing where each request contributes
// a cost proportional to its server-side processing time.
func (c *decayCounter) add(decay float64, value float64) float64 {
	for {
		old := c.bits.Load()
		oldVal := math.Float64frombits(old)
		newVal := oldVal*decay + value
		if c.bits.CompareAndSwap(old, math.Float64bits(newVal)) {
			return newVal
		}
	}
}

// decay applies one decay step without adding load.
// Used for periodic idle-connection drain and fan-out contraction.
func (c *decayCounter) decay(factor float64) float64 {
	for {
		old := c.bits.Load()
		oldVal := math.Float64frombits(old)
		newVal := oldVal * factor
		if c.bits.CompareAndSwap(old, math.Float64bits(newVal)) {
			return newVal
		}
	}
}

// load returns the current counter value.
func (c *decayCounter) load() float64 {
	return math.Float64frombits(c.bits.Load())
}

// store sets the counter to an exact value.
func (c *decayCounter) store(v float64) {
	c.bits.Store(math.Float64bits(v))
}

// timeWeightedCounter is a time-decaying load accumulator using an
// exponentially weighted moving average (EWMA) tied to wall clock time.
//
// Unlike [decayCounter] where the decay rate depends on request frequency,
// this counter decays continuously based on elapsed time:
//
//	counter = counter * e^(-lambda * dt) + value
//
// Where dt is wall clock seconds since the last update and lambda = ln(2)/halfLife.
// This decouples decay from request rate: idle nodes drain quickly regardless
// of how few requests arrive, and busy nodes reflect actual load per unit time.
//
// At steady state with request rate R and per-request cost v:
//
//	counter ~= v * R / lambda
//
// With halfLife=5s, lambda~=0.1386: counter ~= 7.2 * v * R.
//
// The counter and timestamp are stored in separate atomics. The small race
// window between the CAS on the value and the timestamp store is benign:
// one request may apply slightly too much or too little decay, corrected
// on the next update.
type timeWeightedCounter struct {
	bits     atomic.Uint64 // float64 stored as bits
	nanoTime atomic.Int64  // last update time (UnixNano); 0 = never updated
	clock    clock         // required; realClock{} in production, testClock in tests
}

// now returns the current time via the injected clock.
func (c *timeWeightedCounter) now() time.Time {
	return c.clock.Now()
}

// add applies time-based decay since the last update and adds value.
// Safe for concurrent use (CAS loop).
func (c *timeWeightedCounter) add(value float64) float64 {
	now := c.now().UnixNano()

	for {
		oldBits := c.bits.Load()
		oldNano := c.nanoTime.Load()

		oldVal := math.Float64frombits(oldBits)
		dt := float64(now-oldNano) / 1e9
		if dt < 0 || oldNano == 0 {
			dt = 0 // First update or clock skew: no decay.
		}

		newVal := oldVal*math.Exp(-loadDecayLambda*dt) + value
		if c.bits.CompareAndSwap(oldBits, math.Float64bits(newVal)) {
			c.nanoTime.Store(now)
			return newVal
		}
	}
}

// load returns the current value decayed to the present time.
// Safe for concurrent use (read-only; does not update stored state).
func (c *timeWeightedCounter) load() float64 {
	bits := c.bits.Load()
	nano := c.nanoTime.Load()

	val := math.Float64frombits(bits)
	if nano == 0 {
		return val // Never updated.
	}

	dt := float64(c.now().UnixNano()-nano) / 1e9
	if dt <= 0 {
		return val
	}
	return val * math.Exp(-loadDecayLambda*dt)
}

// store sets the counter to an exact value and records the current time.
// Used in tests.
func (c *timeWeightedCounter) store(v float64) {
	c.bits.Store(math.Float64bits(v))
	c.nanoTime.Store(c.now().UnixNano())
}
