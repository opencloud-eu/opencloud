// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import "sync"

// This file groups the two pooled-buffer access styles used on the routing
// hot path. Both are backed by sync.Pool, which clears entries every GC
// cycle so oversized buffers from transient spikes don't persist
// indefinitely. The two styles serve different access patterns:
//
//   - [getConnSlice] / [putConnSlice]: returns a *[]*Connection with
//     length 0 and at least the requested capacity. Use when the caller
//     appends to the buffer (final size unknown), e.g., rendezvous
//     fan-out and partition routines. The caller works with a raw
//     pointer-to-slice and must keep the slice header in sync.
//
//   - [acquireConns] / [pooledConns.Release], [acquireFloats] /
//     [pooledFloats.Release]: returns a wrapper struct over a slice with
//     length n. Use when the caller writes by index (final size known
//     up-front), e.g., per-candidate cost vectors in calcMultiKeyCost
//     and connScoreSelect. The wrapper is zero-value safe (Release is a
//     no-op on the zero value) and exposes Slice/Len/Release methods.
//
// Both styles share the same connSlicePool, so a buffer acquired by
// either style can be reused by the other without allocation churn.

// connSlicePool pools []*Connection buffers shared by both access styles.
//
//nolint:gochecknoglobals // sync.Pool must be package-level
var connSlicePool = sync.Pool{
	New: func() any {
		s := make([]*Connection, 0, 32)
		return &s
	},
}

// floatSlicePool pools []float64 buffers used for per-candidate scoring
// and extra-cost slices.
//
//nolint:gochecknoglobals // sync.Pool must be package-level
var floatSlicePool = sync.Pool{
	New: func() any {
		s := make([]float64, 0, 16)
		return &s
	},
}

// --- Append-style access (length=0, capacity>=n on acquire) -----------------

// getConnSlice returns a pooled []*Connection buffer with length 0 and at
// least the given capacity. Use when the caller will append into the
// buffer; the final length is determined by the caller's loop. Callers
// must call [putConnSlice] when done; misuse on a nil pointer panics.
func getConnSlice(minCap int) *[]*Connection {
	bp := connSlicePool.Get().(*[]*Connection) //nolint:forcetypeassert // pool only stores *[]*Connection
	if cap(*bp) < minCap {
		*bp = make([]*Connection, 0, minCap)
	}
	*bp = (*bp)[:0]
	return bp
}

// clearConns clears all slots up to capacity and truncates the slice to
// length 0. Shared by [putConnSlice] and [pooledConns.Release] so the
// "no retained *Connection across cycles" invariant has a single
// definition that can be tested without exercising the pool.
func clearConns(bp *[]*Connection) {
	s := *bp
	clear(s[:cap(s)])
	*bp = s[:0]
}

// putConnSlice clears pointer references and returns the buffer to the pool.
func putConnSlice(bp *[]*Connection) {
	clearConns(bp)
	connSlicePool.Put(bp)
}

// --- Indexed-style access (length=n on acquire, fixed-size writes) ----------

// pooledConns is a pooled []*Connection buffer with a known length set at
// acquire time. The zero value represents an empty/nil result; calling
// [pooledConns.Release] on it is a no-op.
type pooledConns struct{ p *[]*Connection }

// acquireConns returns a pooled []*Connection of length n (capacity >= n).
// Use when the caller writes per-index values; for append-style growth
// see [getConnSlice].
func acquireConns(n int) pooledConns {
	bp := connSlicePool.Get().(*[]*Connection) //nolint:forcetypeassert // pool only stores *[]*Connection
	if cap(*bp) < n {
		*bp = make([]*Connection, n)
	} else {
		*bp = (*bp)[:n]
	}
	return pooledConns{p: bp}
}

// Slice returns the underlying []*Connection.
func (b pooledConns) Slice() []*Connection {
	if b.p == nil {
		return nil
	}
	return *b.p
}

// Len returns the length of the pooled slice.
func (b pooledConns) Len() int {
	if b.p == nil {
		return 0
	}
	return len(*b.p)
}

// Release clears pointer references and returns the buffer to the pool.
// Safe to call on the zero value.
func (b pooledConns) Release() {
	if b.p == nil {
		return
	}
	clearConns(b.p)
	connSlicePool.Put(b.p)
}

// pooledFloats is a pooled []float64 buffer with a known length set at
// acquire time. The zero value represents an empty/nil result; calling
// [pooledFloats.Release] on it is a no-op.
type pooledFloats struct{ p *[]float64 }

// acquireFloats returns a pooled []float64 of length n (capacity >= n).
func acquireFloats(n int) pooledFloats {
	bp := floatSlicePool.Get().(*[]float64) //nolint:forcetypeassert // pool only stores *[]float64
	if cap(*bp) < n {
		*bp = make([]float64, n)
	} else {
		*bp = (*bp)[:n]
	}
	return pooledFloats{p: bp}
}

// Slice returns the underlying []float64.
func (b pooledFloats) Slice() []float64 {
	if b.p == nil {
		return nil
	}
	return *b.p
}

// Len returns the length of the pooled slice.
func (b pooledFloats) Len() int {
	if b.p == nil {
		return 0
	}
	return len(*b.p)
}

// Release returns the buffer to the pool. Safe to call on the zero value.
// No clear() is performed: float64 has no pointers, so retained values
// cannot keep heap objects alive across GC cycles.
func (b pooledFloats) Release() {
	if b.p == nil {
		return
	}
	*b.p = (*b.p)[:0]
	floatSlicePool.Put(b.p)
}

// --- Node-name set (membership lookup, not append) --------------------------

//nolint:gochecknoglobals // sync.Pool must be package-level
var nodeSetPool = sync.Pool{
	New: func() any {
		m := make(map[string]struct{}, 8)
		return &m
	},
}

// pooledNodeSet is a pooled map[string]struct{} buffer used as a node-name
// lookup set. The zero value represents an empty/nil result; calling
// [pooledNodeSet.Release] on it is a no-op. Add and Contains panic on the
// zero value -- callers must obtain an instance from [acquireNodeSet].
type pooledNodeSet struct{ p *map[string]struct{} }

// acquireNodeSet returns an empty pooled node-name set. The map is cleared
// by [pooledNodeSet.Release] when prior users return it to the pool, so
// every call here observes an empty map.
func acquireNodeSet() pooledNodeSet {
	bp := nodeSetPool.Get().(*map[string]struct{}) //nolint:forcetypeassert // pool only stores *map[string]struct{}
	return pooledNodeSet{p: bp}
}

// Add inserts a node name into the set.
func (b pooledNodeSet) Add(name string) {
	(*b.p)[name] = struct{}{}
}

// Contains returns true if the set contains the given name.
func (b pooledNodeSet) Contains(name string) bool {
	_, ok := (*b.p)[name]
	return ok
}

// Len returns the number of entries in the set.
func (b pooledNodeSet) Len() int {
	if b.p == nil {
		return 0
	}
	return len(*b.p)
}

// Release clears the map and returns it to the pool. Safe to call on the
// zero value.
func (b pooledNodeSet) Release() {
	if b.p == nil {
		return
	}
	clear(*b.p)
	nodeSetPool.Put(b.p)
}
