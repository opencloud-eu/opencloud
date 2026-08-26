// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"math"
	"slices"
	"sync/atomic"
	"time"
)

const (
	// defaultRTTRingSize is the ring buffer capacity when no discovery or
	// resurrection intervals are configured.
	defaultRTTRingSize = 12
)

// rttRingSizeFor computes the RTT ring buffer size from discovery and
// resurrection intervals:
//
//	size = 2 * ceil(discoverInterval / resurrectInterval)
//
// The factor of 2 provides a full window of measurements plus one
// window of history. With default intervals (30s / 5s), this yields 12.
// Returns [defaultRTTRingSize] when either interval is zero.
func rttRingSizeFor(discoverInterval, resurrectInterval time.Duration) int {
	if discoverInterval <= 0 || resurrectInterval <= 0 {
		return defaultRTTRingSize
	}
	ratio := float64(discoverInterval) / float64(resurrectInterval)
	size := max(2*int(math.Ceil(ratio)), 2)
	return size
}

// rttRing is a lock-free ring buffer of bucketed RTT measurements.
//
// Every slot starts at [rttBucketUnknown]. As health checks call [add],
// real bucket values overwrite sentinels. [medianBucket] uses the
// monotonic cursor to read only the written portion of the ring, so
// even a single health check produces a usable RTT tier.
//
// Concurrency: single-writer ([add] from the health check goroutine),
// lock-free readers ([medianBucket] copies and sorts).
type rttRing struct {
	buckets []atomic.Int64 // bucketed us values; all init to rttBucketUnknown
	cursor  atomic.Int32   // monotonic write counter; index = cursor % len(buckets)
}

// newRTTRing allocates a ring buffer with every slot set to [rttBucketUnknown].
func newRTTRing(size int) *rttRing {
	if size <= 0 {
		size = 1
	}

	r := &rttRing{
		buckets: make([]atomic.Int64, size),
	}
	for i := range r.buckets {
		r.buckets[i].Store(rttBucketUnknown.Int64())
	}

	return r
}

// add records a health check RTT measurement. The duration is bucketed via
// [durationMicros.Bucket] and stored in the next ring slot. Single-writer only.
func (r *rttRing) add(d time.Duration) {
	if r == nil {
		return
	}

	bucket := durationFromStd(d).Micros().Bucket()
	idx := int(r.cursor.Add(1)-1) % len(r.buckets)
	r.buckets[idx].Store(bucket.Int64())
}

// medianBucket returns the median of measured RTT buckets in the ring.
//
// Lock-free: uses the monotonic cursor to determine how many slots have
// been written (min(cursor, ringSize)), then walks backwards from the
// most recent write collecting exactly that many values. Before the ring
// is full, only the written slots are included -- unwritten sentinels are
// never part of the median. Once the ring wraps, all slots contain real
// data and the full ring is used.
//
// Returns [rttBucketUnknown] only when no writes have occurred (cursor == 0).
// A single health check measurement is enough for a real median.
func (r *rttRing) medianBucket() rttBucket {
	if r == nil {
		return rttBucketUnknown
	}

	n := len(r.buckets)
	cur := int(r.cursor.Load())

	// The cursor is monotonic: cursor == total writes. Before the ring
	// wraps, only cursor slots contain measured data. After wrapping,
	// all n slots are valid.
	written := min(cur, n)
	if written == 0 {
		return rttBucketUnknown
	}

	// Walk backwards from the most recent write, collecting measured values.
	vals := make([]int64, written)
	for i := range written {
		// cursor points one past the last write; start at cursor-1.
		idx := ((cur-1-i)%n + n) % n
		vals[i] = r.buckets[idx].Load()
	}

	slices.Sort(vals)

	return rttBucketFromInt64(vals[written/2])
}

// rttBucketOf returns the [rttBucket] for a measured RTT duration.
// This is a convenience wrapper over the typed conversion chain:
//
//	durationFromStd(d).Micros().Bucket()
func rttBucketOf(d time.Duration) rttBucket {
	return durationFromStd(d).Micros().Bucket()
}
