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
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RoaringBitmap/roaring/v2"
)

type geoIndexCache struct {
	m     sync.RWMutex
	cache map[uint16]*geoCacheEntry

	closeCh  chan struct{}
	isClosed bool
}

var geoMonitorFreq = 1 * time.Second

func newGeoIndexCache() *geoIndexCache {
	return &geoIndexCache{
		cache:   make(map[uint16]*geoCacheEntry),
		closeCh: make(chan struct{}),
	}
}

func (gc *geoIndexCache) Clear() {
	gc.m.Lock()
	if gc.isClosed {
		gc.m.Unlock()
		return
	}
	gc.isClosed = true
	close(gc.closeCh)

	for _, entry := range gc.cache {
		entry.Close()
	}
	gc.cache = nil
	gc.m.Unlock()
}

func (gc *geoIndexCache) loadOrCreate(field uint16, mem []byte, except *roaring.Bitmap, r *FileReader) (*geoData, error) {
	gc.m.RLock()
	if gc.isClosed {
		gc.m.RUnlock()
		return nil, nil
	}

	entry, ok := gc.cache[field]
	if ok {
		gc.m.RUnlock()
		return entry.load(except), nil
	}
	gc.m.RUnlock()

	gc.m.Lock()
	defer gc.m.Unlock()
	if gc.isClosed {
		return nil, nil
	}

	entry, ok = gc.cache[field]
	if ok {
		return entry.load(except), nil
	}

	return gc.createAndCacheLocked(field, mem, except, r)

}

func (gc *geoIndexCache) createAndCacheLocked(field uint16, mem []byte,
	except *roaring.Bitmap, r *FileReader) (*geoData, error) {
	var pos uint64
	// Load Num Docs
	numDocs, n := binary.Uvarint(mem[pos : pos+binary.MaxVarintLen64])
	pos += uint64(n)
	if numDocs == 0 {
		return nil, fmt.Errorf("no geo docs found")
	}

	// Load Doc ID to Doc Num mapping
	docNums, docNumsMem, shift, err := r.ReadUint32Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load the Document Scores Inner
	docScoresInner, docScoresInnerMem, shift, err := r.ReadUint64Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load the Document Scores Cross
	docScoresCross, docScoresCrossMem, shift, err := r.ReadUint64Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load Inner Cells
	innerCells, innerCellsMem, shift, err := r.ReadUint64Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load Inner Cell Doc IDs
	innerDocIDs, innerDocIDsMem, shift, err := r.ReadUint32Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load Cross Cells
	crossCells, crossCellsMem, shift, err := r.ReadUint64Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load Cross Cell Doc IDs
	crossDocIDs, crossDocIDsMem, shift, err := r.ReadUint32Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	// Load BBox Metadata without expanding the BBox data
	bBoxesOffsets, bBoxesOffsetsMem, shift, err := r.ReadUint64Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	bBoxesLen, n := binary.Uvarint(mem[pos : pos+binary.MaxVarintLen64])
	pos += uint64(n)

	bboxMem := mem[pos : pos+bBoxesLen]
	pos += bBoxesLen

	// Load Shape Metadata without expanding the Shape data
	shapeOffsets, shapeOffsetsMem, shift, err := r.ReadUint64Array(mem[pos:])
	if err != nil {
		return nil, err
	}
	pos += shift

	shapeLen, n := binary.Uvarint(mem[pos : pos+binary.MaxVarintLen64])
	pos += uint64(n)

	shapeMem := mem[pos : pos+shapeLen]
	pos += shapeLen

	rv := &geoCacheEntry{
		innerCells:    innerCells,
		innerCellsMem: innerCellsMem,

		innerDocIDs:    innerDocIDs,
		innerDocIDsMem: innerDocIDsMem,

		crossCells:    crossCells,
		crossCellsMem: crossCellsMem,

		crossDocIDs:    crossDocIDs,
		crossDocIDsMem: crossDocIDsMem,

		bboxOffsets:    bBoxesOffsets,
		bboxOffsetsMem: bBoxesOffsetsMem,
		bboxMem:        bboxMem,

		shapeOffsets:    shapeOffsets,
		shapeOffsetsMem: shapeOffsetsMem,
		shapeMem:        shapeMem,

		numDocs: numDocs,

		docNums:    docNums,
		docNumsMem: docNumsMem,

		docScoresInner:    docScoresInner,
		docScoresInnerMem: docScoresInnerMem,
		docScoresCross:    docScoresCross,
		docScoresCrossMem: docScoresCrossMem,

		tracker: &ewma{
			alpha:  0.4,
			sample: 1,
		},
		refs: 1,

		fileReader: r,

		scoresPool: sync.Pool{
			New: func() interface{} {
				var scores map[uint32]uint64
				if numDocs > 100 {
					scores = make(map[uint32]uint64, 100)
				} else {
					scores = make(map[uint32]uint64, numDocs)
				}
				return &scores
			},
		},
	}

	gc.insertLOCKED(field, rv)

	return &geoData{
		geoCacheEntry: rv,
		except:        createNewExcludeBitmap(except, rv.docNums),
	}, nil
}

// createNewExcludeBitmap translates an exclusion bitmap from segment doc
// number space into geo docID space
func createNewExcludeBitmap(except *roaring.Bitmap, docNums []uint32) *roaring.Bitmap {
	if except == nil || except.IsEmpty() || len(docNums) == 0 {
		return nil
	}

	// docNums is sorted ascending, so a disjoint range means nothing to exclude
	lo, hi := docNums[0], docNums[len(docNums)-1]
	if except.Maximum() < lo || except.Minimum() > hi {
		return nil
	}

	var newExcept *roaring.Bitmap
	it := except.Iterator()
	it.AdvanceIfNeeded(lo)
	for i := 0; it.HasNext() && i < len(docNums); {
		docNum := it.Next()
		if docNum > hi {
			break
		}
		// search only the unscanned suffix - i advances monotonically
		i += sort.Search(len(docNums)-i, func(k int) bool { return docNums[i+k] >= docNum })
		// a doc with a multi-valued geo field owns a run of geo docIDs
		for ; i < len(docNums) && docNums[i] == docNum; i++ {
			if newExcept == nil {
				newExcept = roaring.New()
			}
			newExcept.Add(uint32(i))
		}
	}
	return newExcept
}

func (gc *geoIndexCache) insertLOCKED(field uint16, entry *geoCacheEntry) {
	if len(gc.cache) == 0 {
		go gc.monitor()
	}

	gc.cache[field] = entry
}

func (gc *geoIndexCache) monitor() {
	ticker := time.NewTicker(geoMonitorFreq)
	defer ticker.Stop()

	for {
		select {
		case <-gc.closeCh:
			return
		case <-ticker.C:
			exit := gc.cleanup()
			if exit {
				return
			}
		}
	}
}

// cleanup runs one eviction pass over the cache. For each entry it folds the
// hit count accumulated since the last pass into the entry's moving average
// (see ewma.add), then evicts entries that have no outstanding references and
// whose average traffic has decayed to at most (1 - alpha)
// Returns true when the cache is empty, signalling the monitor goroutine to exit.
func (gc *geoIndexCache) cleanup() bool {
	gc.m.Lock()

	for field, entry := range gc.cache {
		sample := atomic.LoadUint64(&entry.tracker.sample)
		entry.tracker.add(sample)

		refCount := atomic.LoadInt64(&entry.refs)

		if refCount <= 0 && entry.tracker.avg <= (1-entry.tracker.alpha) {
			atomic.StoreUint64(&entry.tracker.sample, 0)
			delete(gc.cache, field)
			entry.Close()
			continue
		}
		atomic.StoreUint64(&entry.tracker.sample, 0)
	}

	rv := len(gc.cache) == 0
	gc.m.Unlock()
	return rv
}

// geoCacheEntry represents a cached entry for a specific field in the geo index.
// Each entry holds a copy of the memory to prevent the underlying memory from being
// released while the entry is still in use.
// Includes a pool for reusing score arrays to reduce memory allocations during queries.
type geoCacheEntry struct {
	innerCells    []uint64
	innerCellsMem []byte

	innerDocIDs    []uint32
	innerDocIDsMem []byte

	crossCells    []uint64
	crossCellsMem []byte

	crossDocIDs    []uint32
	crossDocIDsMem []byte

	bboxOffsets    []uint64
	bboxOffsetsMem []byte
	bboxMem        []byte

	shapeOffsets    []uint64
	shapeOffsetsMem []byte
	shapeMem        []byte

	numDocs    uint64
	docNums    []uint32
	docNumsMem []byte

	docScoresInner    []uint64
	docScoresInnerMem []byte

	docScoresCross    []uint64
	docScoresCrossMem []byte

	tracker *ewma
	refs    int64

	fileReader *FileReader

	scoresPool sync.Pool
}

// geoData is a per-load view over a shared geoCacheEntry.
// The exclude bitmap is derived fresh per load from the
// snapshot's except bitmap.
type geoData struct {
	*geoCacheEntry
	except *roaring.Bitmap
}

func (g *geoData) Excluded() *roaring.Bitmap {
	return g.except
}

func (gce *geoCacheEntry) GetScoreMap() map[uint32]uint64 {
	return *gce.scoresPool.Get().(*map[uint32]uint64)
}

func (gce *geoCacheEntry) PutScoreMap(scores map[uint32]uint64) {
	if scores != nil {
		clear(scores)
		gce.scoresPool.Put(&scores)
	}
}

func (gce *geoCacheEntry) Close() {
	gce.decRef()
}

func (gce *geoCacheEntry) load(except *roaring.Bitmap) *geoData {
	gce.incHits()
	gce.incRef()

	return &geoData{
		geoCacheEntry: gce,
		except:        createNewExcludeBitmap(except, gce.docNums),
	}
}

func (gce *geoCacheEntry) incHits() {
	atomic.AddUint64(&gce.tracker.sample, 1)
}

func (gce *geoCacheEntry) incRef() {
	atomic.AddInt64(&gce.refs, 1)
}

func (gce *geoCacheEntry) decRef() {
	atomic.AddInt64(&gce.refs, -1)
}

func (gce *geoCacheEntry) InnerCells() []uint64 {
	return gce.innerCells
}

func (gce *geoCacheEntry) InnerDocIDs() []uint32 {
	return gce.innerDocIDs
}

func (gce *geoCacheEntry) CrossCells() []uint64 {
	return gce.crossCells
}

func (gce *geoCacheEntry) CrossDocIDs() []uint32 {
	return gce.crossDocIDs
}

func (gce *geoCacheEntry) BoundingBox(geoDocID uint32) ([]byte, error) {
	if uint64(geoDocID) >= gce.numDocs {
		return nil, fmt.Errorf("geo docID out of range")
	}

	var offsetStart uint64
	if geoDocID != 0 {
		offsetStart = gce.bboxOffsets[geoDocID-1]
	}
	offsetEnd := gce.bboxOffsets[geoDocID]
	if offsetEnd == offsetStart {
		return nil, fmt.Errorf("no bounding box for geo docID %d", geoDocID)
	}

	buf, err := gce.fileReader.process(gce.bboxMem[offsetStart:offsetEnd])
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (gce *geoCacheEntry) Shape(geoDocID uint32) ([]byte, error) {
	if uint64(geoDocID) >= gce.numDocs {
		return nil, fmt.Errorf("geo docID out of range")
	}

	var offsetStart uint64
	if geoDocID != 0 {
		offsetStart = gce.shapeOffsets[geoDocID-1]
	}
	offsetEnd := gce.shapeOffsets[geoDocID]
	if offsetEnd == offsetStart {
		return nil, fmt.Errorf("no shape for geo docID %d", geoDocID)
	}

	buf, err := gce.fileReader.process(gce.shapeMem[offsetStart:offsetEnd])
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (gce *geoCacheEntry) DocNums() []uint32 {
	return gce.docNums
}

func (gce *geoCacheEntry) NumDocs() uint64 {
	return gce.numDocs
}

func (gce *geoCacheEntry) DocScores() ([]uint64, []uint64) {
	return gce.docScoresInner, gce.docScoresCross
}
