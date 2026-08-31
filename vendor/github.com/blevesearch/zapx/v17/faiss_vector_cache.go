//  Copyright (c) 2024 Couchbase, Inc.
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

//go:build vectors
// +build vectors

package zap

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RoaringBitmap/roaring/v2"
	seg "github.com/blevesearch/scorch_segment_api/v2"
)

// -----------------------------------------------------------------------------

func newVectorIndexCache() *vectorIndexCache {
	return &vectorIndexCache{
		cache:   make(map[uint16]*cacheEntry),
		closeCh: make(chan struct{}),
	}
}

type vectorIndexCache struct {
	closeCh  chan struct{}
	m        sync.RWMutex
	cache    map[uint16]*cacheEntry
	isClosed bool
}

// Clear clears the entire vector index cache.
func (vc *vectorIndexCache) Clear() {
	vc.m.Lock()
	// if already closed, no-op
	if vc.isClosed {
		vc.m.Unlock()
		return
	}
	vc.isClosed = true
	close(vc.closeCh)

	// forcing a close on all indexes to avoid memory leaks.
	for _, entry := range vc.cache {
		entry.close()
	}
	vc.cache = nil
	vc.m.Unlock()
}

// vectorCacheOptions controls what loadOrCreate builds and returns.
type vectorCacheOptions struct {
	mem     []byte
	numDocs uint32
	except  *roaring.Bitmap
	useGPU  bool
	reader  *FileReader
	optStr  string

	skipMapping bool // if true, skip building the idMapping
	stats       *seg.Stats
}

func newVectorCacheOptions(mem []byte, numDocs uint32, except *roaring.Bitmap,
	useGPU bool, reader *FileReader, optStr string, skipMapping bool) *vectorCacheOptions {
	return &vectorCacheOptions{
		mem:         mem,
		numDocs:     numDocs,
		except:      except,
		useGPU:      useGPU,
		reader:      reader,
		optStr:      optStr,
		skipMapping: skipMapping,
	}
}

// loadOrCreate obtains the vector index from the cache or creates it if it's not present.
func (vc *vectorIndexCache) loadOrCreate(fieldID uint16, opts *vectorCacheOptions) (
	index faissIndex, mapping *idMapping, exclude *bitmap, err error) {
	if opts == nil {
		return nil, nil, nil, fmt.Errorf("vectorCacheOptions cannot be nil")
	}
	// first try to read from the cache with a read lock
	vc.m.RLock()
	if vc.isClosed {
		// if cache is closed, no-op
		vc.m.RUnlock()
		return nil, nil, nil, nil
	}
	entry, ok := vc.cache[fieldID]
	if ok {
		vc.m.RUnlock()
		return entry.load(opts.except)
	}
	vc.m.RUnlock()
	// cache miss, rebuild the cache entry under a write lock
	vc.m.Lock()
	defer vc.m.Unlock()
	if vc.isClosed {
		// if cache is closed, no-op
		return nil, nil, nil, nil
	}
	// check again if we have the entry now
	entry, ok = vc.cache[fieldID]
	if ok {
		return entry.load(opts.except)
	}
	// still not present, create and cache it
	return vc.createAndCacheLOCKED(fieldID, opts)
}

func readVectorSectionFromFile(opts *vectorCacheOptions) (index faissIndex,
	mapping *idMapping, err error) {
	if opts == nil {
		return nil, nil, fmt.Errorf("vectorCacheOptions cannot be nil")
	}
	// if the cache doesn't have the entry, construct the vector to doc id map and
	// the vector index out of the mem bytes and update the cache under lock.
	mem := opts.mem
	pos := 0
	numVecs, n := binary.Uvarint(mem[pos : pos+binary.MaxVarintLen64])
	if n <= 0 {
		return nil, nil, fmt.Errorf("could not read numVecs")
	}
	pos += n

	// if no vectors or no documents, return empty cache entry
	if numVecs == 0 || opts.numDocs == 0 {
		return nil, nil, nil
	}

	// read the length of the docID list
	listLen, n := binary.Uvarint(mem[pos : pos+binary.MaxVarintLen64])
	if n <= 0 {
		return nil, nil, fmt.Errorf("could not read docID list length")
	}
	pos += n
	listPos := pos
	pos += int(listLen)

	if !opts.skipMapping {
		// read the entierity of the docID list through the file reader
		buf, err := opts.reader.process(mem[listPos : listPos+int(listLen)])
		if err != nil {
			return nil, nil, fmt.Errorf("could not process docID list: %v", err)
		}

		bufPos := 0
		bufLen := len(buf)
		mapping = newIDMapping(uint32(numVecs), opts.numDocs)
		for vecID := uint32(0); vecID < uint32(numVecs); vecID++ {
			docID, n := binary.Uvarint(buf[bufPos:min(bufPos+binary.MaxVarintLen64, bufLen)])
			if n <= 0 {
				return nil, nil, fmt.Errorf("could not read docID for vecID %d", vecID)
			}
			bufPos += n
			mapping.add(vecID, uint32(docID))
		}
	}

	// read the type of the vector index
	indexType, n := binary.Uvarint(mem[pos : pos+binary.MaxVarintLen64])
	if n <= 0 {
		return nil, nil, fmt.Errorf("could not read faiss index type")
	}
	pos += n

	// read the faiss index size
	indexSize, n := binary.Uvarint(mem[pos : pos+binary.MaxVarintLen64])
	if n <= 0 {
		return nil, nil, fmt.Errorf("could not read faiss index size")
	}
	pos += n

	// read the index bytes through the file reader
	fIndexBytes, err := opts.reader.process(mem[pos : pos+int(indexSize)])
	if err != nil {
		return nil, nil, err
	}
	pos += int(indexSize)

	params := newFaissIndexParams(opts.optStr, int(numVecs), 0, faissIOFlagsReadOnly)
	if faissIndexType(indexType) == faissBIVFIndex {
		// read the faiss binary index size
		binSize, n := binary.Uvarint(mem[pos : pos+binary.MaxVarintLen64])
		pos += n
		// read the index bytes through the file reader
		bIndexBytes, err := opts.reader.process(mem[pos : pos+int(binSize)])
		if err != nil {
			return nil, nil, err
		}
		pos += int(binSize)
		index, err = newFaissBinaryIndexFromBytes(bIndexBytes, fIndexBytes, params)
		if err != nil {
			return nil, nil, fmt.Errorf("faiss binary index creation error: %v", err)
		}
	} else {
		if opts.useGPU {
			index, err = newFaissGPUFloat32IndexFromBytes(fIndexBytes, params)
		} else {
			index, err = newFaissFloat32IndexFromBytes(fIndexBytes, params)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("faiss float32 index creation error: %v", err)
		}
	}

	return index, mapping, nil
}

func (vc *vectorIndexCache) createAndCacheLOCKED(fieldID uint16, opts *vectorCacheOptions) (index faissIndex,
	mapping *idMapping, exclude *bitmap, err error) {
	if opts == nil {
		return nil, nil, nil, fmt.Errorf("vectorCacheOptions cannot be nil")
	}
	index, mapping, err = readVectorSectionFromFile(opts)
	if err != nil {
		return nil, nil, nil, err
	}
	// update the cache with a complete entry for the fieldID, note that while performing
	// fast merge we won't be tracking the id mapping since its not relevant
	vc.insertLOCKED(fieldID, index, mapping)
	return index, mapping, getExcludedVectors(mapping, opts.except), nil
}

func (vc *vectorIndexCache) insertLOCKED(fieldID uint16,
	index faissIndex, mapping *idMapping) {
	// the first time we've hit the cache, try to spawn a monitoring routine
	// which will reconcile the moving averages for all the fields being hit
	if len(vc.cache) == 0 {
		go vc.monitor()
	}
	// initializing the alpha with 0.4 essentially means that we are favoring
	// the history a little bit more relative to the current sample value.
	// this makes the average to be kept above the threshold value for a
	// longer time and thereby the index to be resident in the cache
	// for longer time.
	vc.cache[fieldID] = createCacheEntry(index, mapping, 0.4)
}

func (vc *vectorIndexCache) decRef(fieldID uint16) {
	vc.m.RLock()
	entry, ok := vc.cache[fieldID]
	if ok {
		entry.decRef()
	}
	vc.m.RUnlock()
}

// vectorIndexLocation describes where a cached vector index currently resides.
type vectorIndexLocation uint8

const (
	vectorIndexNotCached vectorIndexLocation = iota // not present in the cache
	vectorIndexInCPU                                // loaded in CPU memory
	vectorIndexInGPU                                // loaded in GPU memory
)

// indexLocation reports where the vector index for fieldID currently resides.
func (vc *vectorIndexCache) indexLocation(fieldID uint16) vectorIndexLocation {
	vc.m.RLock()
	defer vc.m.RUnlock()
	if vc.isClosed {
		return vectorIndexNotCached
	}
	entry, ok := vc.cache[fieldID]
	if !ok {
		return vectorIndexNotCached
	}
	if gpuIdx, ok := entry.index.(faissIndexGPU); ok && gpuIdx.inGPURam() {
		return vectorIndexInGPU
	}
	return vectorIndexInCPU
}

func (vc *vectorIndexCache) cleanup() bool {
	vc.m.Lock()
	cache := vc.cache

	// for every field reconcile the average with the current sample values
	for fieldID, entry := range cache {
		sample := atomic.LoadUint64(&entry.tracker.sample)
		entry.tracker.add(sample)

		refCount := atomic.LoadInt64(&entry.refs)
		// the comparison threshold as of now is (1 - a). mathematically it
		// means that there is only 1 query per second on average as per history.
		// and in the current second, there were no queries performed against
		// this index.
		if entry.tracker.avg <= (1-entry.tracker.alpha) && refCount <= 0 {
			atomic.StoreUint64(&entry.tracker.sample, 0)
			delete(vc.cache, fieldID)
			entry.close()
			continue
		}
		atomic.StoreUint64(&entry.tracker.sample, 0)
	}

	rv := len(vc.cache) == 0
	vc.m.Unlock()
	return rv
}

var monitorFreq = 1 * time.Second

func (vc *vectorIndexCache) monitor() {
	ticker := time.NewTicker(monitorFreq)
	defer ticker.Stop()
	for {
		select {
		case <-vc.closeCh:
			return
		case <-ticker.C:
			exit := vc.cleanup()
			if exit {
				// no entries to be monitored, exit
				return
			}
		}
	}
}

// -----------------------------------------------------------------------------

func createCacheEntry(index faissIndex, mapping *idMapping, alpha float64) *cacheEntry {
	ce := &cacheEntry{
		index:   index,
		mapping: mapping,
		tracker: &ewma{
			alpha:  alpha,
			sample: 1,
		},
		refs: 1,
	}
	return ce
}

type cacheEntry struct {
	tracker *ewma

	// this is used to track the live references to the cache entry,
	// such that while we do a cleanup() and we see that the avg is below a
	// threshold we close/cleanup only if the live refs to the cache entry is 0.
	refs int64

	index   faissIndex
	mapping *idMapping
}

func (ce *cacheEntry) incHit() {
	atomic.AddUint64(&ce.tracker.sample, 1)
}

func (ce *cacheEntry) addRef() {
	atomic.AddInt64(&ce.refs, 1)
}

func (ce *cacheEntry) decRef() {
	atomic.AddInt64(&ce.refs, -1)
}

func (ce *cacheEntry) load(except *roaring.Bitmap) (faissIndex, *idMapping, *bitmap, error) {
	ce.incHit()
	ce.addRef()
	return ce.index, ce.mapping, getExcludedVectors(ce.mapping, except), nil
}

func (ce *cacheEntry) close() {
	go func() {
		if ce.index != nil {
			ce.index.close()
		}
		ce.mapping = nil
	}()
}

// -----------------------------------------------------------------------------

func getExcludedVectors(idMap *idMapping, except *roaring.Bitmap) (exclude *bitmap) {
	if except != nil && !except.IsEmpty() && idMap != nil {
		numVecs := idMap.numVectors()
		// if there are no vectors, nothing to exclude
		if numVecs == 0 {
			return exclude
		}
		// iterate over the docs present in the except bitmap to
		// construct the vector exclude bitmap. we can guarantee that
		// this except bitmap is immutable and derived from the segment
		// snapshot, but the vector exclude bitmap is part of the
		// SegmentBase's cache, because of which it is necessary to create
		// a new vector exclude bitmap per cache load operation
		// get an iterator over the except bitmap
		exceptItr := except.Iterator()
		// as we iterate over the except docIDs, get the vector IDs
		// for those docIDs and set them in our exclude bitmap
		for exceptItr.HasNext() {
			docID := exceptItr.Next()
			vecs, ok := idMap.vecsForDoc(docID)
			if ok && len(vecs) > 0 {
				if exclude == nil {
					exclude = newBitmap(numVecs)
				}
				for _, vecID := range vecs {
					exclude.set(vecID)
				}
			}
		}
	}
	return exclude
}

// -----------------------------------------------------------------------------

// trainedIndexCache is specifically for caching the trained index in the segment
// such that it can be shared and used during fast merge, and avoid putting pressure
// on garbage collection
type trainedIndexCacheEntry struct {
	index faissIndex
}

type trainedIndexCache struct {
	m     sync.RWMutex
	cache map[uint16]*trainedIndexCacheEntry
}

func newTrainedIndexCache() *trainedIndexCache {
	return &trainedIndexCache{
		cache: make(map[uint16]*trainedIndexCacheEntry),
	}
}

func (tc *trainedIndexCache) Clear() {
	tc.m.Lock()
	defer tc.m.Unlock()
	for _, entry := range tc.cache {
		entry.index.close()
	}
	tc.cache = nil
}

func (tc *trainedIndexCache) loadOrCreate(fieldID uint16, opts *vectorCacheOptions) (faissIndex, error) {
	if opts == nil {
		return nil, fmt.Errorf("vectorCacheOptions cannot be nil")
	}
	tc.m.RLock()
	entry, ok := tc.cache[fieldID]
	if ok {
		tc.m.RUnlock()
		return entry.index, nil
	}
	tc.m.RUnlock()

	tc.m.Lock()
	defer tc.m.Unlock()

	index, err := tc.createAndCachedLOCKED(fieldID, opts)
	if err != nil {
		return nil, err
	}

	return index, nil
}

func (tc *trainedIndexCache) createAndCachedLOCKED(fieldID uint16, opts *vectorCacheOptions) (faissIndex, error) {
	index, _, err := readVectorSectionFromFile(opts)
	if err != nil {
		return nil, err
	}
	tc.cache[fieldID] = &trainedIndexCacheEntry{
		index: index,
	}
	return index, nil
}
