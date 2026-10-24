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
	"encoding/json"
	"errors"
	"reflect"

	"github.com/blevesearch/go-faiss"
	seg "github.com/blevesearch/scorch_segment_api/v2"
)

var (
	errNilIndex     error = errors.New("faiss index is nil")
	errNilParams    error = errors.New("faiss index params is nil")
	errNotSupported error = errors.New("operation not supported")
)

var reflectStaticSizeFaissIndexParams uint64

func init() {
	var f faissIndexParams
	reflectStaticSizeFaissIndexParams = uint64(reflect.TypeOf(f).Size())
}

// -------------------------------------------
// Parameters for constructing a Faiss index
// -------------------------------------------
// faissIndexParams holds the construction-time parameters required by a
// faissIndex implementation that cannot be derived from the underlying
// faiss index objects themselves (e.g. dimension and metric type are
// already available on the faiss index).
type faissIndexParams struct {
	// optimization is the index optimization type string (e.g. latency,
	// recall, memory-efficient, BIVF flavours).
	optimization string
	// numVecs is the number of vectors expected to populate the index.
	// It is needed at construction time because the underlying faiss
	// index reports zero vectors until trainAndAdd/add is called, but
	// some construction-time decisions (e.g. whether to clone to GPU)
	// depend on the eventual vector count.
	numVecs int
	// nlist is the number of centroids the index is being built with.
	// zero during query time.
	nlist int
	// ioFlags used to read the index from bytes
	ioFlags int
	stats   *seg.Stats
}

// newFaissIndexParams constructs a faissIndexParams with the given optimization
// type, expected vector count and centroid count.
func newFaissIndexParams(optimization string, numVecs, nlist, ioFlags int) *faissIndexParams {
	return &faissIndexParams{
		optimization: optimization,
		numVecs:      numVecs,
		nlist:        nlist,
		ioFlags:      ioFlags,
	}
}

// numTrainingVecs returns how many of the availableVecs vectors to train on.
func (p *faissIndexParams) numTrainingVecs(availableVecs int) int {
	if p.nlist <= 0 {
		return availableVecs
	}
	return min(p.nlist*trainingPointsPerCentroid, availableVecs)
}

func (p *faissIndexParams) size() uint64 {
	return reflectStaticSizeFaissIndexParams + uint64(len(p.optimization))
}

// -------------------------------------------
// Faiss Index Interface
// -------------------------------------------

// Abstract interface for Faiss vector indices, which are returned by the go-faiss library.
type faissIndex interface {
	// adds the given vectors to the index.
	add(vecs *vectorSet) error
	// closes the index and releases any associated resources.
	close()
	// returns the dimensionality of the vectors in the index.
	dim() int
	// returns the metric type used by the index, which determines how distances between vectors are computed during search.
	metricType() int
	// ntotal returns the total number of vectors currently stored in the index.
	ntotal() int64
	// reconstructBatch reconstructs the original vectors for the given vector IDs in the index.
	reconstructBatch(vecIDs []int64, prealloc []float32) ([]float32, error)
	// performs a search on the index using the provided query vector and and retrieves the top K nearest neighbors.
	// Optional search constraints can be applied using the selector and additional search parameters.
	search(qVector *vectorSet, k int64, selector faiss.Selector, params json.RawMessage) ([]float32, []int64, error)
	// write out the index content into the provide fileWriter using a reusable buffer
	// returns any error encountered during the write process.
	write(buf []byte, w *FileWriter) error
	// returns the size of the index in bytes.
	size() uint64
	// -----------------------------------------------------------------
	// casting methods to access index-specific operations below
	// -----------------------------------------------------------------
	// returns the underlying IVF index if this is an IVF index,
	// and a boolean indicating whether the cast was successful.
	castIVF() faissIndexIVF
}

// Interface for IVF-specific operations on Faiss vector indices.
type faissIndexIVF interface {
	faissIndex
	// returns the count of the selected vector IDs in each
	// cluster of the IVF index, based on the provided selector.
	clusterVectorCounts(sel faiss.Selector, nlist int) ([]int64, error)
	// returns the top K cardinalities (number of vectors) of the centroids in the IVF index.
	centroidCardinalities(limit int, descending bool) ([]uint64, [][]float32, error)
	// returns the IVF index parameters, nprobe and nlist from the ivf index.
	ivfParams() (nprobe, nlist int)
	// performs a search on the flat index quantizer of the IVF index, considering only the
	// clusters selected by the centroidSelector and returns the search results.
	searchQuantizer(qVector *vectorSet, centroidSelector faiss.Selector, centroidCount int64) ([]int64, []float32, error)
	// performs a search on the IVF index by probing the specified clusters and returns the search results.
	// We restrict the search to a caller-supplied set of pre-assigned clusters rather than probing internally.
	searchClusters(eligibleCentroidIDs []int64, centroidDis []float32,
		centroidsToProbe int, qVecSet *vectorSet, k int64, selector faiss.Selector, params json.RawMessage) ([]float32, []int64, error)
	// sets the direct map type for the IVF index. The direct map is essential for
	// reconstructing vectors based on their sequential vector IDs in future merges.
	setDirectMap(directMapType int) error
	// sets the number of probes (nprobe) for the IVF index. nprobe determines how many
	// inverted lists are probed during search, and is a key parameter that controls the
	// trade-off between search accuracy and latency.
	setNProbe(nprobe int32)
	// trains the IVF index on the provided training data and adds the vectors to
	// the trained index. The training step performs k-means clustering to partition
	// the data space, which enables efficient non-exhaustive search during query time.
	// directMap and nprobe must be set after this call (GPU sync clears them).
	trainAndAdd(trainingData *vectorSet, vecsToAdd *vectorSet) error
	// sets the quantizers for the IVF index. The quantizer is a separate
	// IVF index that is trained on the same data and used to assign vectors
	// to clusters in the IVF index.
	setQuantizers(trainedIndex faissIndexIVF) error
	// returns whether the participating index is eligible for fast merge
	isMergeable() bool
	// merged another faiss index into the current IVF index,
	// with an offset to adjust vector IDs from the other index.
	mergeFrom(other faissIndex, offset int64) error
}

// faissIndexGPU is implemented by any index type that can reside in GPU memory.
type faissIndexGPU interface {
	// inGPURam reports whether the index is currently loaded in GPU memory.
	inGPURam() bool
}

// Interface for batched search operations on Faiss vector indices.
type faissQueryBatch interface {
	// performs a batch search on the index using the provided query vector and parameters,
	// and returns the distances and corresponding vector IDs of the top k results.
	// NOTE: only vector search requests with the same `k` are batched together.
	batchSearch(qVector *vectorSet, k int64) ([]float32, []int64, error)
}
