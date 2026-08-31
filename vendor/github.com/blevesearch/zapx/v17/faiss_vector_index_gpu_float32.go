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

//go:build vectors
// +build vectors

package zap

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"reflect"
	"sync/atomic"

	index "github.com/blevesearch/bleve_index_api"
	faiss "github.com/blevesearch/go-faiss"
)

var (
	reflectStaticSizeGPUFloat32Index uint64
	reflectStaticSizeGPUState        uint64
)

// maxGPUSearchK is the largest value of k that the Faiss GPU index supports.
// For a top-k search where k exceeds this limit, the GPU implementation is
// not usable and the search must fall back to the CPU index.
const maxGPUSearchK = 2048

func init() {
	var f faissGPUFloat32Index
	reflectStaticSizeGPUFloat32Index = uint64(reflect.TypeOf(f).Size())
	var gs gpuState
	reflectStaticSizeGPUState = uint64(reflect.TypeOf(gs).Size())
}

// gpuState holds all the resources related to gpu vector search,
// the gpu index and the request batcher to the gpu
type gpuState struct {
	idx     *faiss.GPUIndexImpl
	batcher *requestBatcher
}

// batchSearch implements faissIndexBatch directly on gpuState, so the batcher
// holds a reference to the index without going through the atomic pointer.
func (gs *gpuState) batchSearch(qVector *vectorSet, k int64) ([]float32, []int64, error) {
	return gs.idx.Search(qVector.floatData, k)
}

// size returns the amount of memory used by the gpuState.
func (gs *gpuState) size() uint64 {
	return reflectStaticSizeGPUState +
		gs.idx.Size() +
		gs.batcher.size()
}

// ---------------------------------
// Faiss GPU Float32 Index
// ---------------------------------
// faissGPUFloat32Index wraps a CPU float32 index alongside a GPU index.
// The GPU is used for unfiltered searches (no selector), while all
// other operations (filtered searches, IVF cluster searches, SQ/IVF
// operations, serialization, etc.) are delegated to the CPU index.
type faissGPUFloat32Index struct {
	cpuIdx *faiss.IndexImpl

	// idxBytes holds the original serialized index bytes to prevent GC.
	idxBytes []byte
	params   *faissIndexParams

	// doneCh is closed when initGPU completes.
	doneCh chan struct{}

	// gpu holds both the GPU index and its request batcher as a single
	// atomic pointer; a nil load means the GPU is not yet available or has
	// been torn down.
	gpu atomic.Pointer[gpuState]
}

// newFaissGPUFloat32Index creates a GPU-backed float32 index. The GPU clone is
// always performed asynchronously; search falls back to CPU until it
// completes. All other GPU-operating methods block on doneCh before proceeding.
func newFaissGPUFloat32Index(cpuIdx *faiss.IndexImpl, params *faissIndexParams) (faissIndex, error) {
	if cpuIdx == nil {
		return nil, errNilIndex
	}
	if params == nil {
		return nil, errNilParams
	}
	f := &faissGPUFloat32Index{
		cpuIdx: cpuIdx,
		params: params,
		doneCh: make(chan struct{}),
	}
	go f.initGPU()
	return f, nil
}

func newFaissGPUFloat32IndexFromBytes(idxBytes []byte, params *faissIndexParams) (faissIndex, error) {
	if idxBytes == nil {
		return nil, errNilIndex
	}

	if params == nil {
		return nil, errNilParams
	}

	cpuIdx, err := faiss.ReadIndexFromBuffer(idxBytes, params.ioFlags)
	if err != nil {
		return nil, err
	}

	f := &faissGPUFloat32Index{
		cpuIdx:   cpuIdx,
		idxBytes: idxBytes,
		params:   params,
		doneCh:   make(chan struct{}),
	}
	go f.initGPU()
	return f, nil
}

// waitGPU blocks until initGPU has completed
func (f *faissGPUFloat32Index) waitGPU() {
	<-f.doneCh
}

// initGPU clones the CPU index to the GPU and sets up the request batcher.
// It always closes doneCh when it returns, signalling completion to waiters.
func (f *faissGPUFloat32Index) initGPU() {
	// ensure doneCh is closed on exit, even if the GPU clone/setup fails.
	defer close(f.doneCh)
	// Check if we can use the GPU before attempting the clone,
	// to avoid unnecessary overhead when the GPU is not expected to help.
	if !f.canUseGPU() {
		return
	}
	gpuIdx, err := faiss.CloneToGPU(f.cpuIdx)
	if err != nil || gpuIdx == nil {
		return
	}
	gs := &gpuState{idx: gpuIdx}
	gs.batcher = newRequestBatcher(gs)
	f.gpu.Store(gs)
}

// attempt to add the vectors to the GPU index. If it fails,
// fallback to the CPU index
func (f *faissGPUFloat32Index) add(vecs *vectorSet) error {
	f.waitGPU()
	gpuState := f.gpu.Load()
	if gpuState == nil {
		return f.cpuIdx.Add(vecs.floatData)
	}

	err := gpuState.idx.Add(vecs.floatData)
	if err != nil {
		f.teardownGPU()
		return f.cpuIdx.Add(vecs.floatData)
	}

	err = f.syncGPUToCPU()
	if err != nil {
		f.teardownGPU()
		return f.cpuIdx.Add(vecs.floatData)
	}

	return nil
}

func (f *faissGPUFloat32Index) close() {
	f.waitGPU()
	f.teardownGPU()
	f.cpuIdx.Close()
}

// teardownGPU stops the batcher first (while gpuIdx is still live so that
// the final flush can complete on the GPU), then nils and closes the GPU index.
func (f *faissGPUFloat32Index) teardownGPU() {
	f.waitGPU()
	// Swap to nil first so new searches fall through to CPU immediately.
	// The batcher holds a direct reference to gpuState.idx via gpuState.batchSearch,
	// so the final flush completes safely without touching f.gpu.
	gpuState := f.gpu.Swap(nil)
	if gpuState == nil {
		return
	}
	gpuState.batcher.stop()
	gpuState.idx.Close()
}

func (f *faissGPUFloat32Index) dim() int {
	return f.cpuIdx.D()
}

func (f *faissGPUFloat32Index) metricType() int {
	return f.cpuIdx.MetricType()
}

func (f *faissGPUFloat32Index) ntotal() int64 {
	return f.cpuIdx.Ntotal()
}

func (f *faissGPUFloat32Index) reconstructBatch(vecIDs []int64, prealloc []float32) ([]float32, error) {
	return f.cpuIdx.ReconstructBatch(vecIDs, prealloc)
}

func (f *faissGPUFloat32Index) search(qVector *vectorSet, k int64, selector faiss.Selector, params json.RawMessage) ([]float32, []int64, error) {
	// Faiss GPU does not support a top-k search with k greater than
	// maxGPUSearchK; such searches must fall back to the CPU index.
	if selector == nil && len(params) == 0 && k <= maxGPUSearchK {
		if gpuState := f.gpu.Load(); gpuState != nil {
			return gpuState.batcher.search(qVector, k)
		}
	}
	// GPU not ready, filtered search, k too large, or non-empty params —
	// fall back to CPU
	return f.cpuIdx.SearchWithOptions(qVector.floatData, k, selector, params)
}

func (f *faissGPUFloat32Index) write(buf []byte, w *FileWriter) error {
	idxBytes, err := faiss.WriteIndexIntoBuffer(f.cpuIdx)
	if err != nil {
		return err
	}
	idxBytes = w.process(idxBytes)

	// write the length of the serialized vector index bytes
	n := binary.PutUvarint(buf, uint64(len(idxBytes)))
	_, err = w.Write(buf[:n])
	if err != nil {
		return err
	}

	_, err = w.Write(idxBytes)
	if err != nil {
		return err
	}
	return nil
}

func (f *faissGPUFloat32Index) size() uint64 {
	sizeInBytes := reflectStaticSizeGPUFloat32Index +
		f.params.size() +
		f.cpuIdx.Size()
	if gpuState := f.gpu.Load(); gpuState != nil {
		sizeInBytes += gpuState.size()
	}
	return sizeInBytes
}

// inGPURam reports if the index is currently running on the GPU.
// returns false if the async clone is not yet done or the clone fails.
func (f *faissGPUFloat32Index) inGPURam() bool {
	return f.gpu.Load() != nil
}

// -----------------------------------------------------------------
// casting methods to access index-specific operations below
// -----------------------------------------------------------------
func (f *faissGPUFloat32Index) castIVF() faissIndexIVF {
	if f.cpuIdx.IsIVFIndex() {
		return f
	}
	return nil
}

// -----------------------------------------------------------------
// IVF-Index specific operations (delegate to CPU index)
// -----------------------------------------------------------------
func (f *faissGPUFloat32Index) clusterVectorCounts(sel faiss.Selector, nlist int) ([]int64, error) {
	return f.cpuIdx.ObtainClusterVectorCountsFromIVFIndex(sel, nlist)
}

func (f *faissGPUFloat32Index) centroidCardinalities(limit int, descending bool) ([]uint64, [][]float32, error) {
	return f.cpuIdx.ObtainKCentroidCardinalitiesFromIVFIndex(limit, descending)
}

func (f *faissGPUFloat32Index) ivfParams() (nprobe, nlist int) {
	return f.cpuIdx.IVFParams()
}

func (f *faissGPUFloat32Index) searchQuantizer(qVector *vectorSet, centroidSelector faiss.Selector, centroidCount int64) ([]int64, []float32, error) {
	return f.cpuIdx.ObtainClustersWithDistancesFromIVFIndex(qVector.floatData, centroidSelector, centroidCount)
}

func (f *faissGPUFloat32Index) searchClusters(eligibleCentroidIDs []int64, centroidDis []float32,
	centroidsToProbe int, qVecSet *vectorSet, k int64, selector faiss.Selector, params json.RawMessage) ([]float32, []int64, error) {
	return f.cpuIdx.SearchClustersFromIVFIndex(eligibleCentroidIDs, centroidDis, centroidsToProbe, qVecSet.floatData, k, selector, params)
}

func (f *faissGPUFloat32Index) setDirectMap(directMapType int) error {
	return f.cpuIdx.SetDirectMap(directMapType)
}

func (f *faissGPUFloat32Index) setNProbe(nprobe int32) {
	f.cpuIdx.SetNProbe(nprobe)
}

// attempt to train and add the vectors to the GPU index. If it fails,
// fallback to the CPU index
func (f *faissGPUFloat32Index) trainAndAdd(trainingData *vectorSet, vecsToAdd *vectorSet) error {
	f.waitGPU()
	gpuState := f.gpu.Load()
	if gpuState == nil {
		return f.trainAndAddCPU(trainingData, vecsToAdd)
	}

	nvecsToTrain := f.params.numTrainingVecs(trainingData.nvecs)
	err := gpuState.idx.Train(trainingData.floatData[:nvecsToTrain*f.dim()])
	if err != nil {
		f.teardownGPU()
		return f.trainAndAddCPU(trainingData, vecsToAdd)
	}

	err = gpuState.idx.Add(vecsToAdd.floatData)
	if err != nil {
		// Fallback to full CPU train and add if:
		// 1. We get a non-OOM error from the GPU index
		if !errors.Is(err, faiss.ErrGPUOutOfMemory) {
			f.teardownGPU()
			return f.trainAndAddCPU(trainingData, vecsToAdd)
		}
		// 2. We get an OOM error but syncing GPU to CPU fails.
		if f.syncGPUToCPU() != nil {
			f.teardownGPU()
			return f.trainAndAddCPU(trainingData, vecsToAdd)
		}
		// OOM but CPU sync succeeded: both indexes are trained.
		// Teardown GPU to avoid discrepancy, add to CPU instead.
		f.teardownGPU()
		return f.cpuIdx.Add(vecsToAdd.floatData)
	}

	err = f.syncGPUToCPU()
	if err != nil {
		f.teardownGPU()
		return f.trainAndAddCPU(trainingData, vecsToAdd)
	}

	return nil
}

func (f *faissGPUFloat32Index) trainAndAddCPU(trainingData *vectorSet, vecsToAdd *vectorSet) error {
	nvecsToTrain := f.params.numTrainingVecs(trainingData.nvecs)
	err := f.cpuIdx.Train(trainingData.floatData[:nvecsToTrain*f.dim()])
	if err != nil {
		return err
	}
	return f.cpuIdx.Add(vecsToAdd.floatData)
}

func (f *faissGPUFloat32Index) setQuantizers(trainedIndex faissIndexIVF) error {
	return errNotSupported
}

func (f *faissGPUFloat32Index) isMergeable() bool {
	return false
}

func (f *faissGPUFloat32Index) mergeFrom(other faissIndex, offset int64) error {
	return errNotSupported
}

// syncGPUToCPU clones the current GPU index state back to the CPU index,
// replacing the old CPU index.
func (f *faissGPUFloat32Index) syncGPUToCPU() error {
	f.waitGPU()
	gpuState := f.gpu.Load()
	if gpuState == nil {
		return nil
	}

	cpuIdx, err := faiss.CloneToCPU(gpuState.idx)
	if err != nil {
		return err
	}

	oldCPUIdx := f.cpuIdx
	f.cpuIdx = cpuIdx
	oldCPUIdx.Close()
	return nil
}

func (f *faissGPUFloat32Index) canUseGPU() bool {
	switch f.params.optimization {
	case index.IndexOptimizedForLatency, index.IndexOptimizedForRecall, index.IndexOptimizedForMemoryEfficient:
		return f.params.numVecs >= ivfThreshold
	default:
		return false
	}
}
