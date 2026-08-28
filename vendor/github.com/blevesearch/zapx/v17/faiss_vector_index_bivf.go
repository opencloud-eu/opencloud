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

	index "github.com/blevesearch/bleve_index_api"
	faiss "github.com/blevesearch/go-faiss"
)

var reflectStaticSizeBinaryIndex uint64

func init() {
	var b faissBinaryIndex
	reflectStaticSizeBinaryIndex = uint64(reflect.TypeOf(b).Size())
}

// ---------------------------------
// Faiss Binary IVF Index
// ---------------------------------
type faissBinaryIndex struct {
	backing *faiss.IndexImpl
	binary  *faiss.BinaryIndexImpl

	// backingBytes and binaryBytes hold the original serialized index bytes to prevent GC.
	backingBytes []byte
	binaryBytes  []byte

	params *faissIndexParams
}

func newFaissBinaryIndex(binary *faiss.BinaryIndexImpl, backing *faiss.IndexImpl, params *faissIndexParams) (faissIndex, error) {
	// we always create this object only with valid backing and binary indexes
	if binary == nil || backing == nil {
		return nil, errNilIndex
	}
	if params == nil {
		return nil, errNilParams
	}
	return &faissBinaryIndex{
		backing: backing,
		binary:  binary,
		params:  params,
	}, nil
}

func newFaissBinaryIndexFromBytes(binaryIndexBytes, backingIndexBytes []byte,
	params *faissIndexParams) (faissIndex, error) {
	if binaryIndexBytes == nil || backingIndexBytes == nil {
		return nil, errNilIndex
	}

	if params == nil {
		return nil, errNilParams
	}

	backing, err := faiss.ReadIndexFromBuffer(backingIndexBytes, params.ioFlags)
	if err != nil {
		return nil, err
	}

	binary, err := faiss.ReadBinaryIndexFromBuffer(binaryIndexBytes, params.ioFlags)
	if err != nil {
		return nil, err
	}

	return &faissBinaryIndex{
		backing:      backing,
		binary:       binary,
		backingBytes: backingIndexBytes,
		binaryBytes:  binaryIndexBytes,
		params:       params,
	}, nil
}

func (b *faissBinaryIndex) add(vecs *vectorSet) error {
	// add float data to backing index and the binary data to binary index
	err := b.backing.Add(vecs.floatData)
	if err != nil {
		return err
	}
	return b.binary.Add(vecs.binaryData)
}

func (b *faissBinaryIndex) close() {
	b.binary.Close()
	b.backing.Close()
	b.backingBytes = nil
	b.binaryBytes = nil
}

func (b *faissBinaryIndex) dim() int {
	return b.binary.D()
}

func (b *faissBinaryIndex) metricType() int {
	return b.backing.MetricType()
}

func (b *faissBinaryIndex) ntotal() int64 {
	return b.binary.Ntotal()
}

func (b *faissBinaryIndex) reconstructBatch(vecIDs []int64, prealloc []float32) ([]float32, error) {
	// reconstruct vectors from backing index
	return b.backing.ReconstructBatch(vecIDs, prealloc)
}

func (b *faissBinaryIndex) search(qVector *vectorSet, k int64, selector faiss.Selector, params json.RawMessage) ([]float32, []int64, error) {
	// search the binary index with oversampling and then do a re-ranking on the
	// FAISS index to get the top K results
	// first binarize the query vector if not already done
	qVector.binarize()
	// search the binary index with oversampling to get a larger set of candidate binary IDs for re-ranking
	_, binIDs, err := b.binary.SearchWithOptions(qVector.binaryData, binaryOversampleValue*k,
		selector, params)
	if err != nil {
		return nil, nil, err
	}

	// use backing index for re-ranking, compute the distances/scores for the
	// retrieved binary IDs and then get the top K results based on those distances/scores.
	distances, err := b.backing.DistCompute(qVector.floatData, binIDs)
	if err != nil {
		return nil, nil, err
	}
	// quick select algorithm for inplace partial sorting to get top K results
	// based on distances/scores
	scores, labels := topNIDsByDistance(distances, binIDs, int(k))
	return scores, labels, nil
}

func (b *faissBinaryIndex) write(buf []byte, w *FileWriter) error {
	backingBytes, err := faiss.WriteIndexIntoBuffer(b.backing)
	if err != nil {
		return err
	}
	backingBytes = w.process(backingBytes)

	// write the length of the serialized vector index bytes
	n := binary.PutUvarint(buf, uint64(len(backingBytes)))
	_, err = w.Write(buf[:n])
	if err != nil {
		return err
	}

	_, err = w.Write(backingBytes)
	if err != nil {
		return err
	}

	binaryBytes, err := faiss.WriteBinaryIndexIntoBuffer(b.binary)
	if err != nil {
		return err
	}
	binaryBytes = w.process(binaryBytes)

	// write the length of the serialized vector index bytes
	n = binary.PutUvarint(buf, uint64(len(binaryBytes)))
	_, err = w.Write(buf[:n])
	if err != nil {
		return err
	}

	_, err = w.Write(binaryBytes)
	if err != nil {
		return err
	}
	return nil
}

func (b *faissBinaryIndex) size() uint64 {
	return reflectStaticSizeBinaryIndex +
		b.params.size() +
		b.backing.Size() +
		b.binary.Size()
}

// -----------------------------------------------------------------
// casting methods to access index-specific operations below
// -----------------------------------------------------------------
func (b *faissBinaryIndex) castIVF() faissIndexIVF {
	if b.binary.IsIVFIndex() {
		// return b itself, as the IVF interface is implemented by the same
		// struct as the non-IVF interface in go-faiss.
		return b
	}
	// not an IVF index, return nil.
	return nil
}

// -----------------------------------------------------------------
// IVF-Index specific operations
// -----------------------------------------------------------------

func (b *faissBinaryIndex) centroidCardinalities(limit int, descending bool) ([]uint64, [][]float32, error) {
	cardinalites, bCentroids, err := b.binary.ObtainKCentroidCardinalitiesFromIVFIndex(limit, descending)
	if err != nil {
		return nil, nil, err
	}
	centroids := make([][]float32, len(bCentroids))
	for i := range bCentroids {
		centroids[i] = make([]float32, len(bCentroids[i]))
		for j := range bCentroids[i] {
			centroids[i][j] = float32(bCentroids[i][j])
		}
	}
	return cardinalites, centroids, nil
}

func (b *faissBinaryIndex) clusterVectorCounts(sel faiss.Selector, nlist int) ([]int64, error) {
	return b.binary.ObtainClusterVectorCountsFromIVFIndex(sel, nlist)
}

func (b *faissBinaryIndex) ivfParams() (nprobe, nlist int) {
	return b.binary.IVFParams()
}

func (b *faissBinaryIndex) searchQuantizer(qVector *vectorSet, centroidSelector faiss.Selector, centroidCount int64) ([]int64, []float32, error) {
	// binarize the query vector if not already done
	qVector.binarize()
	ids, dis, err := b.binary.ObtainClustersWithDistancesFromIVFIndex(qVector.binaryData, centroidSelector, centroidCount)
	if err != nil {
		return nil, nil, err
	}
	distances := make([]float32, len(dis))
	for i, d := range dis {
		distances[i] = float32(d)
	}
	return ids, distances, nil
}

func (b *faissBinaryIndex) searchClusters(eligibleCentroidIDs []int64, centroidDis []float32,
	centroidsToProbe int, qVector *vectorSet, k int64, selector faiss.Selector, params json.RawMessage) ([]float32, []int64, error) {
	// binarize the query vector if not already done
	qVector.binarize()
	// convert the float distances to binary distances for the binary index search
	binaryCentroidDis := make([]int32, len(centroidDis))
	for i, d := range centroidDis {
		binaryCentroidDis[i] = int32(d)
	}
	// search the binary index without oversampling, since we are already searching a
	// limited number of centroids specified by centroidsToProbe
	_, binIDs, err := b.binary.SearchClustersFromIVFIndex(eligibleCentroidIDs, binaryCentroidDis,
		centroidsToProbe, qVector.binaryData, k, selector, params)
	if err != nil {
		return nil, nil, err
	}

	// use backing index for re-ranking, compute the distances/scores for the
	// retrieved binary IDs and then get the top K results based on those distances/scores.
	// reranking is still necessary since hamming distance has a lot of collisions
	distances, err := b.backing.DistCompute(qVector.floatData, binIDs)
	if err != nil {
		return nil, nil, err
	}
	// quick select algorithm for inplace partial sorting to get top K results
	// based on distances/scores
	scores, labels := topNIDsByDistance(distances, binIDs, int(k))
	return scores, labels, nil
}

func (b *faissBinaryIndex) setDirectMap(directMapType int) error {
	return b.binary.SetDirectMap(directMapType)
}

func (b *faissBinaryIndex) setNProbe(nprobe int32) {
	b.binary.SetNProbe(nprobe)
}

func (b *faissBinaryIndex) trainAndAdd(trainingData *vectorSet, vecsToAdd *vectorSet) error {
	nvecsToTrain := b.params.numTrainingVecs(trainingData.nvecs)
	var err error
	if b.backing.IsSQIndex() {
		err = b.backing.Train(trainingData.floatData[:nvecsToTrain*b.dim()])
		if err != nil {
			return err
		}
	}

	err = b.binary.Train(trainingData.binaryData[:nvecsToTrain*(b.dim()/8)])
	if err != nil {
		return err
	}
	return b.add(vecsToAdd)
}

func (b *faissBinaryIndex) setQuantizers(trainedIndex faissIndexIVF) error {
	if idx, ok := trainedIndex.(*faissBinaryIndex); ok {
		// set quantizers for the binary and the backing index if its an SQ8 index
		var err error
		if idx.backing.IsSQIndex() {
			err = b.backing.SetQuantizers(idx.backing)
			if err != nil {
				return err
			}
		}
		err = b.binary.SetQuantizers(idx.binary)
		if err != nil {
			return err
		}
		return nil
	}
	return errNotSupported
}

func (b *faissBinaryIndex) isMergeable() bool {
	switch b.params.optimization {
	case index.IndexBIVFWithBackingFlat:
		// the flat backing index currently doesn't support merge_from
		return false
	case index.IndexBIVFWithBackingSQ8:
		return b.params.numVecs >= ivfThreshold
	default:
		return false
	}
}

func (b *faissBinaryIndex) mergeFrom(other faissIndex, offset int64) error {
	if idx, ok := other.(*faissBinaryIndex); ok {
		if !idx.isMergeable() {
			return errNotSupported
		}
		// merge the binary and the backing index, both flat and SQ8 indexes support
		// merge_from API underneath the hood. the add_id is kept to 0 since we will
		// be merging the largest set of indexes which will be sequential in the list
		// of segments being merged, so there won't be any ID conflicts.
		err := b.backing.MergeFrom(idx.backing, 0)
		if err != nil {
			return err
		}
		err = b.binary.MergeFrom(idx.binary, offset)
		if err != nil {
			return err
		}

		return nil
	}
	return errNotSupported
}
