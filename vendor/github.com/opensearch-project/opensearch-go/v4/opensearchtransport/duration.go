// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"math/bits"
	"time"
)

// ----------------------------------------------------------------------------
// durationNanos
// ----------------------------------------------------------------------------

// durationNanos is an integer nanosecond count that prevents accidental
// unit conversions. Use [durationFromStd] to convert from [time.Duration],
// then call [durationNanos.Micros] to move to microsecond precision.
//
// Unlike a bare int64, arithmetic on durationNanos is unit-safe: you can
// subtract two values, divide by a processor count (yielding another
// durationNanos), and convert to microseconds--all without a float64 cast.
type durationNanos int64

// durationFromStd converts a standard library [time.Duration] (which is
// already int64 nanoseconds) into a [durationNanos]. This is the only
// entry point from untyped durations.
func durationFromStd(d time.Duration) durationNanos {
	return durationNanos(d)
}

// Micros converts nanoseconds to microseconds via integer division,
// truncating toward zero. This matches the behavior of
// [time.Duration.Microseconds].
func (n durationNanos) Micros() durationMicros {
	return durationMicros(n / 1000)
}

// Duration converts back to a standard library [time.Duration].
func (n durationNanos) Duration() time.Duration {
	return time.Duration(n)
}

// IsPositive returns true when the duration is strictly greater than zero.
func (n durationNanos) IsPositive() bool {
	return n > 0
}

// ----------------------------------------------------------------------------
// durationMicros
// ----------------------------------------------------------------------------

// durationMicros is an integer microsecond count. The RTT ring, bucket
// calculation, and CPU-time accumulator all operate in this unit to keep
// values in the same order of magnitude as RTT buckets.
type durationMicros int64

// Bucket returns the [rttBucket] for this microsecond duration using
// power-of-two bucketing:
//
//	bucket = max(rttBucketFloor, floor(log2(microseconds)))
//
// Implemented via [bits.Len] for a branchless, allocation-free result.
// Sub-256us RTTs clamp to [rttBucketFloor] (bucket 8), preventing
// measurement noise from creating meaningless bucket distinctions.
//
// Non-positive durations return [rttBucketUnknown].
func (m durationMicros) Bucket() rttBucket {
	if m <= 0 {
		return rttBucketUnknown
	}
	bucket := rttBucket(bits.Len(uint(m)) - 1)
	if bucket < rttBucketFloor {
		return rttBucketFloor
	}
	return bucket
}

// Nanos converts microseconds back to nanoseconds.
func (m durationMicros) Nanos() durationNanos {
	return durationNanos(m * 1000)
}

// Duration converts to a standard library [time.Duration].
func (m durationMicros) Duration() time.Duration {
	return time.Duration(m) * time.Microsecond
}

// IsPositive returns true when the duration is strictly greater than zero.
func (m durationMicros) IsPositive() bool {
	return m > 0
}

// ----------------------------------------------------------------------------
// rttBucket
// ----------------------------------------------------------------------------

// rttBucket is a quantized RTT tier using power-of-two bucketing:
//
//	bucket = max(rttBucketFloor, floor(log2(microseconds)))
//
// Each bucket spans a power-of-two range of microsecond values, giving
// ~15 meaningful buckets across the entire realistic RTT spectrum.
// The type prevents accidentally mixing raw microsecond values with
// bucket indices in scoring math.
//
// Notable values:
//
//	rttBucketFloor -- 256 us, floor for any measured RTT
//	8              -- [256, 511] us, typical same-AZ latency
//	9-11           -- [512, 4095] us, typical cross-AZ latency
//	13-16          -- [8192, 131071] us, typical cross-region latency
//	rttBucketUnknown -- sentinel for unmeasured connections
type rttBucket int

const (
	// rttBucketFloor is the minimum bucket for any measured RTT. Sub-256us
	// latencies clamp to this value, preventing measurement noise from
	// creating meaningless bucket distinctions. Adjustable: 7 = 128us,
	// 6 = 64us.
	rttBucketFloor rttBucket = 8 // floor(log2(256us))

	// rttBucketUnknown is the sentinel bucket for connections with no RTT
	// data. It sorts after all realistic buckets, so measured nodes are
	// always preferred over unmeasured nodes.
	//
	// All ring buffer slots are initialized to this value. As health checks
	// overwrite slots with real measurements, the median naturally decreases
	// from unknown to a real bucket once more than half the ring contains
	// measured data -- a built-in warmup gate.
	rttBucketUnknown rttBucket = 1 << 30
)

// IsUnknown returns true when the bucket is the unmeasured sentinel.
func (b rttBucket) IsUnknown() bool {
	return b == rttBucketUnknown
}

// Micros reconstructs a [durationMicros] from a bucket value by reversing
// the power-of-two bucketing. This is the inverse of
// [durationMicros.Bucket]: it returns the lower bound of the bucket's
// power-of-two range. Exact round-trips only hold for values that are
// exact powers of two.
func (b rttBucket) Micros() durationMicros {
	if b <= 0 {
		return 0
	}
	return durationMicros(1 << uint(b))
}

// Int64 returns the bucket as int64 for use with atomic.Int64 storage and
// external APIs that haven't adopted the typed bucket yet.
func (b rttBucket) Int64() int64 {
	return int64(b)
}

// rttBucketFromInt64 converts an int64 (e.g., from atomic.Int64.Load) to
// an [rttBucket]. This is the only entry point from untyped int64 values
// and should appear only at the atomic storage boundary in [rttRing].
func rttBucketFromInt64(v int64) rttBucket {
	return rttBucket(v)
}
