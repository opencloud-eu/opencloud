// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import "sync/atomic"

// poolRoundRobin implements [poolSelector] with an atomic counter.
// This is the default selection strategy for [multiServerPool]: increment
// a counter, modulo the active partition size, return the connection.
// Never signals cap changes.
type poolRoundRobin struct {
	counter atomic.Int64
}

func (s *poolRoundRobin) selectNext(ready []*Connection, activeCount int) (*Connection, int, int, error) {
	next := s.counter.Add(1)
	idx := int(next-1) % activeCount
	return ready[idx], capRemain, capRemain, nil
}
