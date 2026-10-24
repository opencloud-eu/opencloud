// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

// weightMapPool pools map[*Connection]uint64 used by rankByHash for
// pre-computed rendezvous hash weights during sort.
//
//nolint:gochecknoglobals // sync.Pool must be package-level
var weightMapPool = sync.Pool{
	New: func() any {
		m := make(map[*Connection]uint64, 16)
		return &m
	},
}

// rendezvousTopK selects the top-k connections for a given key from an
// RTT-pre-sorted connection list, with hard priority for shard-hosting nodes.
//
// The input conns slice MUST be sorted by RTT bucket ascending (nearest
// first). This invariant is maintained externally -- the list is re-sorted
// on health check updates, not per request.
//
// Slot filling uses a hard partition:
//
//  1. Shard-hosting nodes first (data is already warm in OS page cache on
//     those nodes -- going direct avoids a proxy hop through a coordinating
//     node, which would add server-side RTT and resource consumption).
//     Within shard nodes, fill from nearest RTT tier first, using rendezvous
//     hash for consistent assignment within each tier.
//
//  2. Non-shard nodes only if shard nodes don't fill K slots. These nodes
//     will coordinate/proxy the request to a shard node, adding a hop.
//     Same tier-then-hash ordering applies.
//
// When shardNodes is nil (e.g., /_cat/shards unavailable), the partition
// is skipped and all nodes are treated equally -- graceful degradation to
// RTT-only slot selection.
//
// After slot selection, the K-element result is rotated by jitter.Add(1) % K
// to spread per-client preference across the slot set. When jitter is nil,
// no rotation is applied.
//
// When buf is non-nil, it is used as the backing storage for the result
// slice (avoiding a heap allocation). The caller must call putConnSlice(buf)
// after consuming the result.
func rendezvousTopK(
	keyA, keyB string, conns []*Connection, k int, jitter *atomic.Int64,
	shardNodeNames map[string]struct{}, buf *[]*Connection,
) []*Connection {
	if len(conns) == 0 || k <= 0 {
		return nil
	}

	// Exclude connections flagged as needing a /_cat/shards refresh.
	// These connections have stale shard placement data and must not
	// participate in routing until discovery confirms current
	// state. Fast path: scan once for any flagged connection; only
	// allocate a filtered slice on the rare path when flags are set.
	conns = filterNeedsCatUpdate(conns)
	if len(conns) == 0 {
		return nil
	}

	if k > len(conns) {
		k = len(conns)
	}

	// Split into shard-hosting and non-shard partitions, preserving
	// the caller's RTT sort order within each partition.
	var shard, nonShard []*Connection
	var partBuf *[]*Connection // pooled buffer backing shard + nonShard
	if len(shardNodeNames) > 0 {
		partBuf = getConnSlice(len(conns))
		part := (*partBuf)[:0]
		// Append shard nodes first, then non-shard nodes.
		for _, c := range conns {
			if _, ok := shardNodeNames[c.Name]; ok {
				part = append(part, c)
			}
		}
		shardLen := len(part)
		for _, c := range conns {
			if _, ok := shardNodeNames[c.Name]; !ok {
				part = append(part, c)
			}
		}
		shard = part[:shardLen]
		nonShard = part[shardLen:]
		*partBuf = part // keep slice header in sync for putConnSlice
	} else {
		// No shard placement data -- treat all nodes equally.
		shard = conns
	}

	var slots []*Connection
	if buf != nil {
		slots = (*buf)[:0]
	} else {
		slots = make([]*Connection, 0, k)
	}
	remaining := k

	// Phase 1: fill from shard-hosting nodes.
	remaining = fillSlotsFromTiers(keyA, keyB, shard, slots[:0:k], remaining, &slots)

	// Phase 2: if slots remain, fill from non-shard nodes.
	if remaining > 0 && len(nonShard) > 0 {
		_ = fillSlotsFromTiers(keyA, keyB, nonShard, slots, remaining, &slots)
	}

	if partBuf != nil {
		putConnSlice(partBuf)
	}

	// Phase 3: rotate within the slots by the jitter offset (in-place).
	n := len(slots)
	if jitter != nil && n > 1 {
		offset := int((jitter.Add(1) - 1) % int64(n))
		if offset < 0 {
			offset += n
		}
		if offset > 0 {
			rotateConns(slots, offset)
		}
	}

	return slots
}

// fillSlotsFromTiers walks an RTT-pre-sorted connection list tier by tier,
// applying rendezvous hash within each tier, and appends up to 'remaining'
// connections to dst. Returns the new remaining count.
func fillSlotsFromTiers(keyA, keyB string, conns []*Connection, dst []*Connection, remaining int, out *[]*Connection) int {
	i := 0
	for i < len(conns) && remaining > 0 {
		// Find the extent of the current RTT tier.
		tierBucket := conns[i].rttRing.medianBucket()
		tierStart := i
		for i < len(conns) && conns[i].rttRing.medianBucket() == tierBucket {
			i++
		}
		tierConns := conns[tierStart:i]

		if len(tierConns) <= remaining {
			// Entire tier fits -- take all nodes.
			dst = append(dst, tierConns...)
			remaining -= len(tierConns)
		} else {
			// More nodes in this tier than remaining slots -- rank by
			// rendezvous hash for consistent assignment.
			ranked := rankByHash(keyA, keyB, tierConns)
			dst = append(dst, ranked[:remaining]...)
			remaining = 0
		}
	}

	*out = dst
	return remaining
}

// rankByHash sorts connections in-place by rendezvous hash weight (descending)
// for consistent key-to-node assignment within an RTT tier.
func rankByHash(keyA, keyB string, conns []*Connection) []*Connection {
	// Pre-compute weights using a pooled map. The map avoids repeated
	// hashing during sort comparisons.
	mp := weightMapPool.Get().(*map[*Connection]uint64)
	weights := *mp
	clear(weights) // reset entries from previous use

	for _, c := range conns {
		weights[c] = rendezvousWeight(keyA, keyB, c.URLString)
	}

	slices.SortFunc(conns, func(a, b *Connection) int {
		wa, wb := weights[a], weights[b]
		if wa != wb {
			if wa > wb {
				return -1
			}
			return 1
		}
		return strings.Compare(a.URLString, b.URLString)
	})

	clear(weights) // zero refs to avoid retaining *Connection pointers
	weightMapPool.Put(mp)

	return conns
}

// rendezvousWeight computes the FNV-1a hash weight for a (key, node) pair.
//
// The key is specified as two parts: keyA and keyB. When keyB is non-empty,
// the effective key is keyA + "/" + keyB (e.g., index name + doc ID for
// document-level routing). When keyB is empty, the key is just keyA.
// This avoids a string concatenation allocation for compound keys.
//
// This is a deterministic pseudo-random number used for consistent
// key-to-node assignment within an RTT tier. The node with the highest
// weight in its tier wins that key's slot.
//
// Uses inline FNV-1a to avoid the heap allocation from fnv.New64a()
// (which returns a hash.Hash64 interface, forcing the value to escape).
// A null byte separator prevents ambiguous concatenations.
func rendezvousWeight(keyA, keyB string, nodeURL string) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	h := uint64(offset64)
	for i := range len(keyA) {
		h ^= uint64(keyA[i])
		h *= prime64
	}
	if keyB != "" {
		h ^= uint64('/') // separator between key parts
		h *= prime64
		for i := range len(keyB) {
			h ^= uint64(keyB[i])
			h *= prime64
		}
	}
	h ^= 0x00 // null separator before nodeURL
	h *= prime64
	for i := range len(nodeURL) {
		h ^= uint64(nodeURL[i])
		h *= prime64
	}
	return h
}

// rotateConns rotates s left by offset positions in-place using the
// three-reverse algorithm. O(n) time, zero allocations.
func rotateConns(s []*Connection, offset int) {
	slices.Reverse(s[:offset])
	slices.Reverse(s[offset:])
	slices.Reverse(s)
}

// filterNeedsCatUpdate returns conns with any needsCatUpdate-flagged
// connections removed. On the common path (no flags set), returns the
// original slice with zero allocations. On the rare path, allocates a
// new slice excluding flagged connections.
func filterNeedsCatUpdate(conns []*Connection) []*Connection {
	// Fast scan: check if any connection has the flag set.
	hasAny := false
	for _, c := range conns {
		if c.needsCatUpdate() {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return conns
	}

	// Rare path: build filtered list.
	filtered := make([]*Connection, 0, len(conns))
	for _, c := range conns {
		if !c.needsCatUpdate() {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
