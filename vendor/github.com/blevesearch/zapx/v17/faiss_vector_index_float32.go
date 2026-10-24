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
	"reflect"
	"sync/atomic"

	index "github.com/blevesearch/bleve_index_api"
	faiss "github.com/blevesearch/go-faiss"
)

var reflectStaticSizeFloat32Index uint64

func init() {
	var f faissFloat32Index
	reflectStaticSizeFloat32Index = uint64(reflect.TypeOf(f).Size())
}

// ---------------------------------
// Faiss Float32 Index
// ---------------------------------
type faissFloat32Index struct {
	idx *faiss.IndexImpl
	// idxBytes holds the original serialized index bytes to prevent GC.
	idxBytes []byte
	params   *faissIndexParams
}

func newFaissFloat32Index(idx *faiss.IndexImpl, params *faissIndexParams) (faissIndex, error) {
	if idx == nil {
		return nil, errNilIndex
	}
	if params == nil {
		return nil, errNilParams
	}
	return &faissFloat32Index{
		idx:    idx,
		params: params,
	}, nil
}

func newFaissFloat32IndexFromBytes(idxBytes []byte, params *faissIndexParams) (faissIndex, error) {
	if idxBytes == nil {
		return nil, errNilIndex
	}

	if params == nil {
		return nil, errNilParams
	}

	idx, err := faiss.ReadIndexFromBuffer(idxBytes, params.ioFlags)
	if err != nil {
		return nil, err
	}

	return &faissFloat32Index{
		idx:      idx,
		idxBytes: idxBytes,
		params:   params,
	}, nil
}

func (f *faissFloat32Index) add(vecs *vectorSet) error {
	return f.idx.Add(vecs.floatData)
}

func (f *faissFloat32Index) close() {
	f.idx.Close()
	f.idxBytes = nil
}

func (f *faissFloat32Index) dim() int {
	return f.idx.D()
}

func (f *faissFloat32Index) metricType() int {
	return f.idx.MetricType()
}

func (f *faissFloat32Index) ntotal() int64 {
	return f.idx.Ntotal()
}

func (f *faissFloat32Index) reconstructBatch(vecIDs []int64, prealloc []float32) ([]float32, error) {
	return f.idx.ReconstructBatch(vecIDs, prealloc)
}

func (f *faissFloat32Index) search(qVector *vectorSet, k int64, selector faiss.Selector, params json.RawMessage) ([]float32, []int64, error) {
	return f.idx.SearchWithOptions(qVector.floatData, k, selector, params)
}

func (f *faissFloat32Index) write(buf []byte, w *FileWriter) error {
	idxBytes, err := faiss.WriteIndexIntoBuffer(f.idx)
	if err != nil {
		return err
	}
	idxBytes = w.process(idxBytes)
	atomic.AddUint64(&f.params.stats.TotVecSectionFloatIndexBytesWritten, uint64(len(idxBytes)))
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

func (f *faissFloat32Index) size() uint64 {
	return reflectStaticSizeFloat32Index +
		f.params.size() +
		f.idx.Size()
}

// -----------------------------------------------------------------
// casting methods to access index-specific operations below
// -----------------------------------------------------------------
func (f *faissFloat32Index) castIVF() faissIndexIVF {
	if f.idx.IsIVFIndex() {
		// return f itself, as the IVF interface is implemented by the same
		// struct as the non-IVF interface in go-faiss.
		return f
	}
	// not an IVF index, return nil.
	return nil
}

// -----------------------------------------------------------------
// IVF-Index specific operations
// -----------------------------------------------------------------
func (f *faissFloat32Index) clusterVectorCounts(sel faiss.Selector, nlist int) ([]int64, error) {
	return f.idx.ObtainClusterVectorCountsFromIVFIndex(sel, nlist)
}

func (f *faissFloat32Index) centroidCardinalities(limit int, descending bool) ([]uint64, [][]float32, error) {
	return f.idx.ObtainKCentroidCardinalitiesFromIVFIndex(limit, descending)
}

func (f *faissFloat32Index) ivfParams() (nprobe, nlist int) {
	return f.idx.IVFParams()
}

func (f *faissFloat32Index) searchQuantizer(qVector *vectorSet, centroidSelector faiss.Selector, centroidCount int64) ([]int64, []float32, error) {
	return f.idx.ObtainClustersWithDistancesFromIVFIndex(qVector.floatData, centroidSelector, centroidCount)
}

func (f *faissFloat32Index) searchClusters(eligibleCentroidIDs []int64, centroidDis []float32,
	centroidsToProbe int, qVecSet *vectorSet, k int64, selector faiss.Selector, params json.RawMessage) ([]float32, []int64, error) {
	return f.idx.SearchClustersFromIVFIndex(eligibleCentroidIDs, centroidDis, centroidsToProbe, qVecSet.floatData, k, selector, params)
}

func (f *faissFloat32Index) setDirectMap(directMapType int) error {
	return f.idx.SetDirectMap(directMapType)
}

func (f *faissFloat32Index) setNProbe(nprobe int32) {
	f.idx.SetNProbe(nprobe)
}

func (f *faissFloat32Index) trainAndAdd(trainingData *vectorSet, vecsToAdd *vectorSet) error {
	nvecsToTrain := f.params.numTrainingVecs(trainingData.nvecs)
	err := f.idx.Train(trainingData.floatData[:nvecsToTrain*f.dim()])
	if err != nil {
		return err
	}
	return f.add(vecsToAdd)
}

func (f *faissFloat32Index) setQuantizers(trainedIndex faissIndexIVF) error {
	centroidFaissIndex, ok := trainedIndex.(*faissFloat32Index)
	if !ok {
		// if not a float32 trained index, we cannot set it as the quantizer
		// for the current index, return an error.
		return errNotSupported
	}
	return f.idx.SetQuantizers(centroidFaissIndex.idx)
}

func (f *faissFloat32Index) isMergeable() bool {
	switch f.params.optimization {
	case index.IndexOptimizedForLatency, index.IndexOptimizedForRecall:
		return f.params.numVecs >= ivfSq8Threshold
	case index.IndexOptimizedForMemoryEfficient, index.IndexIVFRaBitQ:
		return f.params.numVecs >= ivfThreshold
	default:
		return false
	}
}

func (f *faissFloat32Index) mergeFrom(other faissIndex, offset int64) error {
	otherFaissIndex, ok := other.(*faissFloat32Index)
	if !ok {
		return errNotSupported
	}

	if otherFaissIndex.isMergeable() {
		return f.idx.MergeFrom(otherFaissIndex.idx, offset)
	}
	return errNotSupported
}
