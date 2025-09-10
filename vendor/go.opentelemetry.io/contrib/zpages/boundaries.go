// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpages // import "go.opentelemetry.io/contrib/zpages"

import (
	"slices"
	"time"
)

const (
	zeroDuration = time.Duration(0)
	maxDuration  = time.Duration(1<<63 - 1)
)

var defaultBoundaries = newBoundaries([]time.Duration{
	10 * time.Microsecond,
	100 * time.Microsecond,
	time.Millisecond,
	10 * time.Millisecond,
	100 * time.Millisecond,
	time.Second,
	10 * time.Second,
	100 * time.Second,
})

// boundaries represents the interval bounds for the latency based samples.
type boundaries struct {
	durations []time.Duration
}

// newBoundaries returns a new boundaries.
func newBoundaries(durations []time.Duration) *boundaries {
	slices.Sort(durations)
	return &boundaries{durations: durations}
}

// numBuckets returns the number of buckets needed for these boundaries.
func (lb boundaries) numBuckets() int {
	return len(lb.durations) + 1
}

// getBucketIndex returns the appropriate bucket index for a given latency.
func (lb boundaries) getBucketIndex(latency time.Duration) int {
	i := 0
	for i < len(lb.durations) && latency >= lb.durations[i] {
		i++
	}
	return i
}
