//  Copyright (c) 2026 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zap

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/RoaringBitmap/roaring/v2"
	index "github.com/blevesearch/bleve_index_api"
)

type nestedIndexCache struct {
	cache *nestedCacheEntry
}

// newNestedIndexCache creates a new nested index cache
// instance, which contains cached edge list
// for a nested segment
func newNestedIndexCache() *nestedIndexCache {
	return &nestedIndexCache{}
}

// Clear clears the nested index cache, removing the cached edge list
func (nc *nestedIndexCache) Clear() {
	nc.cache = nil
}

func (nc *nestedIndexCache) initialize(numDocs uint64, edgeListOffset uint64, mem []byte) error {
	// pos stores the current read position
	pos := edgeListOffset
	if pos == 0 {
		// no edge list
		return nil
	}
	// read number of edges in the edge list
	numEdges, read := binary.Uvarint(mem[pos : pos+binary.MaxVarintLen64])
	if read <= 0 {
		return fmt.Errorf("error reading number of edges in nested edge list")
	}
	pos += uint64(read)
	// if no documents or edges/nested documents or invalid state, return
	if numDocs == 0 || numEdges == 0 || numDocs <= numEdges {
		return nil
	}
	// create and cache our edge list
	edgeList := newEdgeList(numDocs, numEdges)
	for i := uint64(0); i < numEdges; i++ {
		child, read := binary.Uvarint(mem[pos : pos+binary.MaxVarintLen64])
		if read <= 0 {
			return fmt.Errorf("error reading child doc id in nested edge list")
		}
		pos += uint64(read)
		parent, read := binary.Uvarint(mem[pos : pos+binary.MaxVarintLen64])
		if read <= 0 {
			return fmt.Errorf("error reading parent doc id in nested edge list")
		}
		pos += uint64(read)
		edgeList.addEdge(child, parent)
	}
	// create and cache our descendant store
	numRoots := numDocs - numEdges
	descendantStore := newDescendantStore(numRoots)
	// populate the descendant store using the following invariants:
	// Invariant: child docNums is always > parent docNums
	// Invariant: descendants of root docNum R is always a
	// contiguous range of docNums [R + 1 : R + 1 + N)
	// where N is the number of descendants of R
	roots := make([]uint64, 0, numRoots)
	for docNum := uint64(0); docNum < numDocs; docNum++ {
		if _, ok := edgeList.parent(docNum); !ok {
			roots = append(roots, docNum)
		}
	}
	// descendants of each root are the contiguous range of
	// docNums between it and the next root
	for idx, root := range roots {
		start := root + 1
		end := numDocs
		nextIdx := idx + 1
		if nextIdx < len(roots) {
			end = roots[nextIdx]
		}
		numDescendants := end - start
		if numDescendants > 0 {
			bitmap := roaring.New()
			bitmap.AddRange(start, end)
			descendantStore.add(root, bitmap)
		}
	}

	nc.cache = &nestedCacheEntry{
		el: edgeList,
		ds: descendantStore,
	}
	return nil
}

type nestedCacheEntry struct {
	el edgeList
	ds descendantStore
}

func (nc *nestedIndexCache) ancestry(docNum uint64, prealloc []index.AncestorID) []index.AncestorID {
	cache := nc.cache
	// add self as first ancestor
	prealloc = append(prealloc, index.NewAncestorID(docNum))
	if cache == nil || cache.el == nil {
		return prealloc
	}
	current := docNum
	for {
		parent, ok := cache.el.parent(current)
		if !ok {
			break
		}
		prealloc = append(prealloc, index.NewAncestorID(parent))
		current = parent
	}
	return prealloc
}

func (nc *nestedIndexCache) descendants(root uint64) (*roaring.Bitmap, bool) {
	cache := nc.cache
	if cache == nil || cache.ds == nil {
		return nil, false
	}
	return cache.ds.descendants(root)
}

func (nc *nestedIndexCache) edgeList() edgeList {
	cache := nc.cache
	if cache == nil || cache.el == nil {
		return nil
	}
	return cache.el
}

func (nc *nestedIndexCache) descendantStore() descendantStore {
	cache := nc.cache
	if cache == nil || cache.ds == nil {
		return nil
	}
	return cache.ds
}

func (nc *nestedIndexCache) countNested() uint64 {
	cache := nc.cache
	if cache == nil || cache.el == nil {
		return 0
	}
	return cache.el.count()
}

// countRootDeleted returns the number of root documents in the given bitmap that are deleted
func (nc *nestedIndexCache) countRootDeleted(bm *roaring.Bitmap) uint64 {
	// empty bitmap means no root documents
	if bm == nil || bm.IsEmpty() {
		return 0
	}
	totalDocs := bm.GetCardinality()
	// if no nested documents, all documents in the bitmap are root documents
	if nc.countNested() == 0 {
		return totalDocs
	}
	// get the edgeList for this segment
	el := nc.edgeList()
	if el == nil {
		return totalDocs
	}
	// count nested documents in the bitmap, a nested doc is one that has a parent in the edge list
	var nestedDocCount uint64
	bmItr := bm.Iterator()
	for bmItr.HasNext() {
		docNum := bmItr.Next()
		if _, ok := el.parent(uint64(docNum)); ok {
			nestedDocCount++
		}
	}
	// root docs = total docs - nested docs
	if totalDocs < nestedDocCount {
		// should not happen, but just in case
		return 0
	}
	return totalDocs - nestedDocCount
}

// -------------------------------------------------------

// edgeList provides an interface to access parent of a child document
type edgeList interface {
	// parent returns the parent of the given child document ID,
	// and a boolean indicating if the parent exists.
	parent(child uint64) (uint64, bool)

	// addEdge adds an edge from child to parent in the edge list.
	addEdge(child uint64, parent uint64)

	// count returns the number of edges in the edge list.
	count() uint64

	// iterate iterates over all edges in the edge list, calling the provided function
	// with each child-parent pair. If the function returns false, iteration stops.
	iterate(func(child uint64, parent uint64) bool)
}

type edgeListMap struct {
	edges map[uint64]uint64
}

func newEdgeListMap(numEdges uint64) *edgeListMap {
	return &edgeListMap{
		edges: make(map[uint64]uint64, numEdges),
	}
}

func (elm *edgeListMap) parent(child uint64) (uint64, bool) {
	parent, ok := elm.edges[child]
	return parent, ok
}

func (elm *edgeListMap) addEdge(child uint64, parent uint64) {
	elm.edges[child] = parent
}

func (elm *edgeListMap) count() uint64 {
	return uint64(len(elm.edges))
}

func (elm *edgeListMap) iterate(f func(child uint64, parent uint64) bool) {
	for child, parent := range elm.edges {
		if !f(child, parent) {
			return
		}
	}
}

type edgeListSlice struct {
	numEdges uint64
	sentinel uint64
	edges    []uint64
}

func newEdgeListSlice(numDocs uint64, numEdges uint64) *edgeListSlice {
	var sentinel uint64 = math.MaxUint64
	edges := make([]uint64, numDocs)
	for i := range edges {
		edges[i] = sentinel
	}
	return &edgeListSlice{
		numEdges: numEdges,
		sentinel: sentinel,
		edges:    edges,
	}
}

func (els *edgeListSlice) parent(child uint64) (uint64, bool) {
	if child >= uint64(len(els.edges)) {
		return 0, false
	}
	parent := els.edges[child]
	if parent == els.sentinel {
		return 0, false
	}
	return parent, true
}

func (els *edgeListSlice) addEdge(child uint64, parent uint64) {
	if child >= uint64(len(els.edges)) {
		// out of bounds, ignore as this should not happen
		return
	}
	els.edges[child] = parent
}

func (els *edgeListSlice) count() uint64 {
	return els.numEdges
}

func (els *edgeListSlice) iterate(f func(child uint64, parent uint64) bool) {
	for child, parent := range els.edges {
		if parent != els.sentinel {
			if !f(uint64(child), parent) {
				return
			}
		}
	}
}

// edgeListMapThreshold defines the threshold ratio of nested documents to total documents.
// It is derived using the following reasoning:
//
// Let N = number of nested documents (i.e., edges in the edge list)
// Let T = total number of documents
//
// Memory usage if the edge list is stored as a map[uint64]uint64:
//
//	~30 bytes per entry (key + value + map overhead)
//	Total ≈ 30 * N bytes
//
// Memory usage if the edge list is stored as a []uint64:
//
//	8 bytes per entry
//	Total ≈ 8 * T bytes
//
// We want the threshold at which a map becomes more memory-efficient than a slice:
//
//	30N < 8T
//	N/T < 8/30
//
// Therefore, if the ratio of nested documents to total documents is less than 8/30,
// we use a map for the edge list; otherwise, we use a slice.
var edgeListMapThreshold = 8.0 / 30.0

// newEdgeList creates a new edgeList instance based on the provided
// constants, the total number of documents and the number of nested documents/edges.
func newEdgeList(numDocs uint64, numEdges uint64) edgeList {
	if numDocs == 0 || numEdges == 0 {
		// no edges, return nil
		return nil
	}
	ratio := float64(numEdges) / float64(numDocs)
	if ratio < edgeListMapThreshold {
		// use map representation
		return newEdgeListMap(numEdges)
	}
	// use slice representation
	return newEdgeListSlice(numDocs, numEdges)
}

// -------------------------------------------------------

// descendantStore provides an interface to access precomputed descendant bitmaps for root documents
type descendantStore interface {
	// add a descendant bitmap for a root document
	add(root uint64, descendants *roaring.Bitmap)
	// returns the descendant bitmap for a root document, with an indication of its existence
	descendants(root uint64) (*roaring.Bitmap, bool)
}

type descendantStoreMap struct {
	m map[uint64]*roaring.Bitmap
}

func newDescendantStoreMap(numRoots uint64) *descendantStoreMap {
	return &descendantStoreMap{
		m: make(map[uint64]*roaring.Bitmap, numRoots),
	}
}

func (dsm *descendantStoreMap) add(root uint64, descendants *roaring.Bitmap) {
	dsm.m[root] = descendants
}

func (dsm *descendantStoreMap) descendants(root uint64) (*roaring.Bitmap, bool) {
	bm, ok := dsm.m[root]
	return bm, ok
}

func newDescendantStore(numRoots uint64) descendantStore {
	return newDescendantStoreMap(numRoots)
}
