//  Copyright (c) 2025 Couchbase, Inc.
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
	"fmt"
	"math"
	"math/bits"
	"reflect"
	"slices"

	"github.com/RoaringBitmap/roaring/v2/roaring64"
	index "github.com/blevesearch/bleve_index_api"
	faiss "github.com/blevesearch/go-faiss"
	segment "github.com/blevesearch/scorch_segment_api/v2"
)

var (
	reflectStaticSizeIndexWrapper uint64
	reflectStaticSizeIDMapping    uint64
	reflectStaticSizeBitmap       uint64
)

func init() {
	var w vectorIndexWrapper
	reflectStaticSizeIndexWrapper = uint64(reflect.TypeOf(w).Size())
	var m idMapping
	reflectStaticSizeIDMapping = uint64(reflect.TypeOf(m).Size())
	var b bitmap
	reflectStaticSizeBitmap = uint64(reflect.TypeOf(b).Size())
}

const (
	// maxMultiVectorDocSearchRetries limits repeated searches when deduplicating
	// multi-vector documents. Each retry excludes previously seen vectors to find
	// new unique documents. Acts as a safeguard against pathological data distributions.
	maxMultiVectorDocSearchRetries = 100

	// Pre-Filtered IVF Index search: Threshold for when to start increasing: after 2 iterations without
	// finding enough documents, we start increasing up to the number of centroidsToProbe
	// up to the total number of eligible centroids available
	nprobeIncreaseThreshold = 2

	// binaryOversampleValue is the multiplier used to determine how many additional vectors to retrieve
	// from the binary index as an oversampling strategy to improve recall.
	binaryOversampleValue = 4
)

// vectorIndexWrapper conforms to scorch_segment_api's VectorIndex interface
type vectorIndexWrapper struct {
	index   faissIndex
	mapping *idMapping
	exclude *bitmap
	fieldID uint16

	// nestedMode indicates if the vector index is operating in nested document mode.
	// if so we have a reusable ancestry slice to help with docID lookups
	nestedMode bool
	ancestry   []index.AncestorID

	sb *SegmentBase
}

func (v *vectorIndexWrapper) Search(qVector []float32, k int64, params json.RawMessage) (segment.VecPostingsList, error) {
	if v.index == nil {
		// vector index not found, so return empty postings list
		return emptyVecPostingsList, nil
	}
	if v.index.dim() != len(qVector) {
		// dimensionality mismatch, so return empty postings list
		return emptyVecPostingsList, nil
	}
	// check if number of docs or number of vectors is zero
	if v.mapping == nil || v.mapping.numVectors() == 0 || v.mapping.numDocuments() == 0 {
		// no vectors or no documents indexed, so return empty postings list
		return emptyVecPostingsList, nil
	}
	// check if all the vectors are excluded
	if v.exclude != nil && v.exclude.cardinality() == v.mapping.numVectors() {
		// all vectors excluded, so return empty postings list
		return emptyVecPostingsList, nil
	}
	// create a vector set using the query vector
	qVecSet, err := newVectorSet(len(qVector), qVector)
	if err != nil {
		return nil, err
	}
	rs, err := v.searchWithoutIDs(qVecSet, k, v.exclude, params)
	if err != nil {
		return nil, err
	}
	// populate the postings list from the result set
	return getPostingsList(rs), nil
}

func (v *vectorIndexWrapper) SearchWithFilter(qVector []float32, k int64,
	eligibleList index.EligibleDocumentList, params json.RawMessage) (
	segment.VecPostingsList, error) {
	// if no eligible documents, return empty postings list
	if eligibleList == nil || eligibleList.Count() == 0 {
		return emptyVecPostingsList, nil
	}
	if v.index == nil {
		// vector index not found, so return empty postings list
		return emptyVecPostingsList, nil
	}
	if v.index.dim() != len(qVector) {
		// dimensionality mismatch, so return empty postings list
		return emptyVecPostingsList, nil
	}
	// check if number of docs or number of vectors is zero
	if v.mapping == nil || v.mapping.numVectors() == 0 || v.mapping.numDocuments() == 0 {
		// no vectors or no documents indexed, so return empty postings list
		return emptyVecPostingsList, nil
	}
	// if all documents are eligible, do a normal search
	if eligibleList.Count() == uint64(v.mapping.numDocuments()) {
		return v.Search(qVector, k, params)
	}
	// get the eligible document iterator
	eligibleIterator := eligibleList.Iterator()
	// vector IDs corresponding to the local doc numbers to be
	// considered for the search
	// create a bitmap for the vector IDs to include in the search
	includeBM := newBitmap(v.mapping.numVectors())
	includeCardinality := 0
	for {
		// get the next eligible document ID
		id, ok := eligibleIterator.Next()
		if !ok {
			// exhausted all eligible document IDs
			break
		}
		// get the vector IDs for this document ID
		vecIDs, exists := v.mapping.vecsForDoc(uint32(id))
		if !exists {
			continue
		}
		// since a vector can never belong to multiple documents, we calculate
		// the cardinality by simply adding the number of vectors for each document
		// we include, without worrying about duplicates and avoiding a potential
		// costly population count on the bitmap at the end
		includeCardinality += len(vecIDs)
		for _, vecID := range vecIDs {
			// add all vector IDs for this document to the inclusion bitmap
			includeBM.set(vecID)
		}
	}
	// In case a doc has invalid vector fields but valid non-vector fields,
	// filter hit IDs may be ineligible for the kNN since the document does
	// not have any/valid vectors. Also can happen if no documents have vectors
	numSelected := uint32(includeCardinality)
	if numSelected == 0 {
		return emptyVecPostingsList, nil
	}
	// if we have included all vectors, then we can do a normal search
	// with full selectivity (no filtering)
	if numSelected == v.mapping.numVectors() {
		return v.Search(qVector, k, params)
	}
	// get a vector set using the query vector
	qVecSet, err := newVectorSet(len(qVector), qVector)
	if err != nil {
		return nil, err
	}
	// try to cast the index to an IVF index
	ivfPtr := v.index.castIVF()
	if ivfPtr == nil {
		// perform search with included IDs in the bitmap, since
		// this is not an IVF index
		rs, err := v.searchWithIDs(qVecSet, k, includeBM, params)
		if err != nil {
			return nil, err
		}
		// populate the postings list from the result set
		return getPostingsList(rs), nil
	}
	// Getting the IVF index parameters, nprobe and nlist, set at index time.
	nprobe, nlist := ivfPtr.ivfParams()
	// Create a FAISS selector based on the include bitmap.
	includeSelector, err := getIncludeSelector(includeBM)
	if err != nil {
		return nil, err
	}
	// Ensure the selector is deleted after use, this does NOT free the inner includeBM bitmap.
	// We control its lifecycle in GO.
	defer includeSelector.Delete()
	// Determining which clusters, identified by centroid ID,
	// have at least one eligible vector and hence, ought to be
	// probed.
	clusterVectorCounts, err := ivfPtr.clusterVectorCounts(includeSelector, nlist)
	if err != nil {
		return nil, err
	}
	// Create a bitmap for the eligible centroids to be considered for probing.
	centroidBM := newBitmap(uint32(nlist))
	centroidCount := 0
	for centroidID, vectorCount := range clusterVectorCounts {
		// Only centroids with at least one eligible vector are considered.
		if vectorCount > 0 {
			// since we are adding only unique centroid IDs, this is simply an increment
			// and we can avoid a population count at the end
			centroidCount++
			centroidBM.set(uint32(centroidID))
		}
	}
	if centroidCount == 0 {
		// No centroids have any eligible vectors, so return empty postings list.
		return emptyVecPostingsList, nil
	}
	// create a FAISS selector based on the centroid bitmap
	centroidSelector, err := getIncludeSelector(centroidBM)
	if err != nil {
		return nil, err
	}
	defer centroidSelector.Delete()
	// Search the coarse quantizer to order the centroids based on proximity
	// to the query vector.
	eligibleCentroidIDs, centroidDistances, err := ivfPtr.searchQuantizer(qVecSet, centroidSelector, int64(centroidCount))
	if err != nil {
		return nil, err
	}
	// Determining the minimum number of centroids to be probed
	// to ensure that at least 'k' vectors are collected while
	// examining at least 'nprobe' centroids.
	// centroidsToProbe range: [nprobe, number of eligible centroids]
	var eligibleVecsTillNow int64
	var eligibleCentroidsTillNow int
	centroidsToProbe := len(eligibleCentroidIDs)
	for i, centroidID := range eligibleCentroidIDs {
		// if we get a -1 somehow here, it means no more centroids
		// need to reslice the eligibleCentroidIDs and distances
		// accordingly, just a safeguard check as this does not
		// really happen. FAISS can pad with -1s if there are not enough
		// eligible centroids, but we have already counted the cardinality so
		// we should not see -1s here.
		if centroidID == -1 {
			centroidsToProbe = i
			// reslice to only valid centroids
			eligibleCentroidIDs = eligibleCentroidIDs[:centroidsToProbe]
			centroidDistances = centroidDistances[:centroidsToProbe]
			break
		}
		eligibleVecsTillNow += clusterVectorCounts[centroidID]
		eligibleCentroidsTillNow = i + 1
		// Stop once we've examined at least 'nprobe' centroids and
		// collected at least 'k' vectors.
		if eligibleVecsTillNow >= k && eligibleCentroidsTillNow >= nprobe {
			centroidsToProbe = eligibleCentroidsTillNow
			break
		}
	}
	// Search the clusters specified by 'eligibleCentroidIDs' for
	// vectors whose IDs are present in the includeBM bitmap.
	// This is done while probing only 'centroidsToProbe' clusters.
	// unless overridden dynamically, either by the search parameters
	// or by the deduplication logic in searchClustersFromIVFIndex.
	rs, err := v.searchClustersFromIVFIndex(
		eligibleCentroidIDs, centroidDistances, centroidsToProbe,
		qVecSet, k, includeBM, params)
	if err != nil {
		return nil, err
	}
	// populate the postings list from the result set
	return getPostingsList(rs), nil
}
func (v *vectorIndexWrapper) Close() {
	// skipping the closing because the index is cached and it's being
	// deferred to a later point of time.
	v.sb.vecIndexCache.decRef(v.fieldID)
}

func (v *vectorIndexWrapper) Size() uint64 {
	sizeInBytes := reflectStaticSizeIndexWrapper
	if v.index != nil {
		sizeInBytes += v.index.size()
	}
	if v.mapping != nil {
		sizeInBytes += v.mapping.size()
	}
	if v.exclude != nil {
		sizeInBytes += v.exclude.size()
	}
	return sizeInBytes
}

func (v *vectorIndexWrapper) ObtainKCentroidCardinalitiesFromIVFIndex(limit int, descending bool) (
	[]index.CentroidCardinality, error) {
	if v.index == nil {
		return nil, nil
	}
	var ivfIdx faissIndexIVF
	if ivfIdx = v.index.castIVF(); ivfIdx == nil {
		return nil, nil
	}
	cardinalities, centroids, err := ivfIdx.centroidCardinalities(limit, descending)
	if err != nil {
		return nil, err
	}
	centroidCardinalities := make([]index.CentroidCardinality, len(cardinalities))
	for i, cardinality := range cardinalities {
		centroidCardinalities[i] = index.CentroidCardinality{
			Centroid:    centroids[i],
			Cardinality: cardinality,
		}
	}
	return centroidCardinalities, nil
}

// docSearch performs a search on the vector index to retrieve
// top k documents based on the provided search function.
// It handles deduplication of documents that may have multiple
// vectors associated with them.
// The prepareNextIter function is used to set up the state
// for the next iteration, if more searches are needed to find
// k unique documents. The callback recieves the number of iterations
// done so far and the vector ids retrieved in the last search. While preparing
// the next iteration, if its decided that no further searches are needed,
// the prepareNextIter function can decide whether to continue searching or not
func (v *vectorIndexWrapper) docSearch(k int64, numDocs uint64,
	search func() (scores []float32, labels []int64, err error),
	prepareNextIter func(numIter int, labels []int64) bool) (resultSet, error) {
	// create a result set to hold top K docIDs and their scores
	rs := newResultSet(k, numDocs)
	// flag to indicate if we have exhausted the vector index
	var exhausted bool
	// keep track of number of iterations done, we execute the loop more than once only when
	// we have multi-vector documents leading to duplicates in docIDs retrieved
	numIter := 0
	// get the metric type of the index to help with deduplication logic
	metricType := v.index.metricType()
	// we keep searching until we have k unique docIDs or we have exhausted the vector index
	// or we have reached the maximum number of deduplication iterations allowed
	for numIter < maxMultiVectorDocSearchRetries && rs.size() < k && !exhausted {
		// search the vector index
		numIter++
		scores, labels, err := search()
		if err != nil {
			return nil, err
		}
		// process the retrieved ids and scores, getting the corresponding docIDs
		// for each vector id retrieved, and storing the best score for each unique docID
		for i, vecID := range labels {
			// a vecID of -1 indicates that all valid vectors in the index have been exhausted,
			// so we set the flag to prevent further iterations. However, the current iteration
			// may still contain valid results, so we process them before stopping.
			if vecID == -1 {
				exhausted = true
				continue
			}
			docID, exists := v.getDocIDForVectorID(vecID)
			if !exists {
				continue
			}
			score := scores[i]
			prevScore, exists := rs.get(docID)
			if !exists {
				// first time seeing this docID, so just store it
				rs.put(docID, score)
				continue
			}
			// we have seen this docID before, so we must compare scores
			// check the index metric type first to check how we compare distances/scores
			// and store the best score for the docID accordingly
			// for inner product, higher the score, better the match
			// for euclidean distance, lower the score/distance, better the match
			// so we invert the comparison accordingly
			switch metricType {
			case faiss.MetricInnerProduct: // similarity metrics like dot product => higher is better
				if score > prevScore {
					rs.put(docID, score)
				}
			case faiss.MetricL2:
				fallthrough
			default: // distance metrics like euclidean distance => lower is better
				if score < prevScore {
					rs.put(docID, score)
				}
			}
		}
		// if we still have less than k unique docIDs, prepare for the next iteration, provided
		// we have not exhausted the index
		if rs.size() < k && !exhausted {
			// prepare state for next iteration
			shouldContinue := prepareNextIter(numIter, labels)
			if !shouldContinue {
				break
			}
		}
	}
	// at this point we either have k unique docIDs or we have exhausted
	// the vector index or we have reached the maximum number of deduplication iterations allowed
	// or the prepareNextIter function decided to break out of the loop
	return rs, nil
}

// searchWithoutIDs performs a search on the vector index to retrieve the top K documents
// while excluding any vector IDs specified in the exclude bitmap.
func (v *vectorIndexWrapper) searchWithoutIDs(qVector *vectorSet, k int64,
	exclude *bitmap, params json.RawMessage) (resultSet, error) {
	return v.docSearch(k, v.sb.numDocs,
		func() ([]float32, []int64, error) {
			// build the FAISS selector based on the exclude bitmap, if any.
			// The exclude bitmap can be nil, indicating no exclusions, in that
			// case we can pass a nil selector to FAISS.
			// NOTE: The bitmap selector is just a wrapper over the exclude bitmap
			// which is shared across the CGO layer.
			sel, err := getExcludeSelector(exclude)
			if err != nil {
				return nil, nil, err
			}
			// NOTE: the selector being freed does NOT free the inner bitmap, as we control
			// its lifecycle in GO, to reuse the bitmap across iterations, if needed, for
			// multi-vector document retrieval.
			if sel != nil {
				// The selector can be nil here as we may not be excluding any vectors
				// in which case we can just pass a nil selector to FAISS.
				defer sel.Delete()
			}
			return v.index.search(qVector, k, sel, params)
		},
		func(numIter int, labels []int64) bool {
			// if this is the first loop iteration and we have < k unique docIDs,
			// we must clone the existing exclude bitmap before modifying it
			// to avoid modifying the original bitmap passed in by the caller
			if numIter == 1 {
				// if we do not have an exclude bitmap yet, create a new one
				if exclude == nil {
					exclude = newBitmap(v.mapping.numVectors())
				} else {
					// clone the existing exclude bitmap
					exclude = exclude.clone()
				}
			}
			// prepare the exclude list for the next iteration by adding
			// the vector ids retrieved in this iteration
			for _, vecID := range labels {
				// should not happen, but just a safeguard, as we catch -1
				// in the main loop
				if vecID == -1 {
					continue
				}
				exclude.set(uint32(vecID))
			}
			// with exclude bitmap updated, we can proceed to the next iteration
			// fast check if the exclude bitmap has all vectors excluded, in which case
			// we can stop searching further
			return exclude.cardinality() != v.mapping.numVectors()
		})
}

// searchWithIDs performs a search on the vector index to retrieve the top K documents while only
// considering the vector IDs specified in the include bitmap.
// NOTE: The include bitmap must NOT be nil and must have at least one vector ID set.
func (v *vectorIndexWrapper) searchWithIDs(vecSet *vectorSet, k int64, include *bitmap, params json.RawMessage) (resultSet, error) {
	return v.docSearch(k, v.sb.numDocs,
		func() ([]float32, []int64, error) {
			// build the FAISS selector based on the include bitmap.
			// NOTE: The bitmap selector is just a wrapper over the include bitmap
			// which is shared across the CGO layer.
			sel, err := getIncludeSelector(include)
			if err != nil {
				return nil, nil, err
			}
			// NOTE: the selector being freed does NOT free the inner bitmap, as we control
			// its lifecycle in GO, to reuse the bitmap across iterations, if needed, for
			// multi-vector document retrieval.
			defer sel.Delete()
			return v.index.search(vecSet, k, sel, params)
		},
		func(numIter int, labels []int64) bool {
			// if this is the first loop iteration and we have < k unique docIDs,
			// we clone the existing include slice before modifying it
			if numIter == 1 {
				if include == nil {
					// should not happen, but just a safeguard
					include = newBitmap(v.mapping.numVectors())
				} else {
					// clone the existing include bitmap
					include = include.clone()
				}
			}
			// removing the vector ids retrieved in this iteration
			// from the include set
			for _, vecID := range labels {
				// should not happen, but just a safeguard, as we catch -1
				// in the main loop
				if vecID == -1 {
					continue
				}
				include.clear(uint32(vecID))
			}
			// only continue searching if we still have vector ids to include
			return !include.isEmpty()
		})
}

// searchClustersFromIVFIndex performs a search on the IVF vector index to retrieve the top K documents
// while including only the vectors present in the includeBM bitmap.
// It takes into account the eligible centroid IDs and ensures that at least centroidsToProbe are probed.
// If after a few iterations we haven't found enough documents, it dynamically increases the number of
// clusters searched (up to the number of eligible centroids) to ensure we can find k unique documents.
func (v *vectorIndexWrapper) searchClustersFromIVFIndex(eligibleCentroidIDs []int64, centroidDis []float32,
	centroidsToProbe int, qVecSet *vectorSet, k int64, include *bitmap, params json.RawMessage) (
	resultSet, error) {
	// get ivf index pointer, should not be nil at this point since this method is only called after confirming its an ivf index
	ivfPtr := v.index.castIVF()
	var totalEligibleCentroids = len(eligibleCentroidIDs)
	return v.docSearch(k, v.sb.numDocs,
		func() ([]float32, []int64, error) {
			// build the FAISS selector based on the include bitmap.
			// NOTE: The bitmap selector is just a wrapper over the include bitmap
			// which is shared across the CGO layer.
			sel, err := getIncludeSelector(include)
			if err != nil {
				return nil, nil, err
			}
			// NOTE: the selector being freed does NOT free the inner bitmap, as we control
			// its lifecycle in GO, to reuse the bitmap across iterations, if needed, for
			// multi-vector document retrieval.
			if sel != nil {
				defer sel.Delete()
			}
			return ivfPtr.searchClusters(eligibleCentroidIDs, centroidDis, centroidsToProbe,
				qVecSet, k, sel, params)
		},
		func(numIter int, labels []int64) bool {
			// if this is the first loop iteration and we have < k unique docIDs,
			// we must clone the existing ids slice before modifying it to avoid
			// modifying the original slice passed in by the caller
			if numIter == 1 {
				if include == nil {
					// should not happen, but just a safeguard
					include = newBitmap(v.mapping.numVectors())
				} else {
					// clone the existing include bitmap
					include = include.clone()
				}
			}
			// if we have iterated atleast nprobeIncreaseThreshold times
			// and still have not found enough unique docIDs, we increase
			// the number of centroids to probe for the next iteration
			// to try and find more vectors/documents
			if numIter >= nprobeIncreaseThreshold && centroidsToProbe < totalEligibleCentroids {
				// Calculate how much to increase: increase by 50% of the remaining centroids to probe,
				// but at least by 1 to ensure progress.
				increaseAmount := max((totalEligibleCentroids-centroidsToProbe)/2, 1)
				// Update centroidsToProbe, ensuring it does not exceed the total eligible centroids
				centroidsToProbe = min(centroidsToProbe+increaseAmount, totalEligibleCentroids)
			}
			// removing the vector ids retrieved in this iteration
			// from the include set
			for _, vecID := range labels {
				// should not happen, but just a safeguard, as we catch -1
				// in the main loop
				if vecID == -1 {
					continue
				}
				include.clear(uint32(vecID))
			}
			// only continue searching if we still have vector ids to include
			return !include.isEmpty()
		})
}

// Utility function to get the docID for a given vectorID, used for the
// deduplication logic, to map vectorIDs back to their corresponding docIDs
// if we are in nested mode, this method returns the root docID instead of
// the nested docID, by consulting the edge list. This ensures that kNN searches
// return unique root documents when nested documents are involved.
func (v *vectorIndexWrapper) getDocIDForVectorID(vecID int64) (uint32, bool) {
	docID, exists := v.mapping.docForVec(uint32(vecID))
	if !v.nestedMode || !exists {
		// either not in nested mode, or docID does not exist
		//for the vectorID, so just return the docID as is
		return docID, exists
	}
	// in nested mode and docID exists, so we must get the root docID from the edge list
	// reuse the wrapper's ancestry slice to avoid allocations
	v.ancestry = v.sb.Ancestors(uint64(docID), v.ancestry[:0])
	if len(v.ancestry) == 0 {
		// should not happen, but just in case, return the docID as is
		return docID, exists
	}
	// return the root docID, which is the last element in the ancestry slice
	// in case the docID is a root doc, the ancestry slice would have
	// just one element, which is the docID itself
	return uint32(v.ancestry[len(v.ancestry)-1]), true
}

// ------------------------------------------------------------------------------
// Utility functions not tied to vector index wrapper
// ------------------------------------------------------------------------------

// Utility function to get a faiss.BitmapSelector to include the IDs specified in the bitmap
// The caller must ensure to free the selector by calling selector.Delete() when done using it.
func getIncludeSelector(bm *bitmap) (selector faiss.Selector, err error) {
	if bm == nil {
		// no bitmap provided, so return an error as we expect at least one ID to include
		return nil, fmt.Errorf("include bitmap is nil or empty")
	}
	// create a bitmap inclusion selector
	selector, err = faiss.NewIDSelectorBitmap(bm.bytes())
	if err != nil {
		return nil, err
	}
	return selector, nil
}

// Utility function to get a faiss.BitmapSelector to exclude the IDs specified in the bitmap
// The caller must ensure to free the selector by calling selector.Delete() when done using it.
func getExcludeSelector(bm *bitmap) (selector faiss.Selector, err error) {
	if bm == nil {
		// no bitmap provided, so return nil selector indicating no exclusions
		return nil, nil
	}
	// create a bitmap exclusion selector
	selector, err = faiss.NewIDSelectorBitmapNot(bm.bytes())
	if err != nil {
		return nil, err
	}
	return selector, nil
}

// Utility function to create a vector postings list from the corresponding docID and scores for each
// unique docID retrieved from the vector index
func getPostingsList(rs resultSet) segment.VecPostingsList {
	// 1. returned postings list (of type PostingsList) has two types of information - docNum and its score.
	// 2. both the values can be represented using roaring bitmaps.
	// 3. the Iterator (of type VecPostingsIterator) returned would operate in terms of VecPostings.
	// 4. VecPostings would just have the docNum and the score. Every call of Next()
	//    and just returns the next VecPostings. The caller would do a vp.Number()
	//    and the Score() to get the corresponding values
	rv := &VecPostingsList{
		postings: roaring64.New(),
	}
	rs.iterate(func(docID uint32, score float32) {
		// transform the docID and score to vector code format
		code := getVectorCode(docID, score)
		// add to postings list, this ensures ordered storage
		// based on the docID since it occupies the upper 32 bits
		rv.postings.Add(code)
	})
	return rv
}

// ------------------------------------------------------------------------------
// ResultSet
// ------------------------------------------------------------------------------

// resultSet is a data structure to hold (docID, score) pairs while ensuring
// that each docID is unique. It supports efficient insertion, retrieval,
// and iteration over the stored pairs.
type resultSet interface {
	// Add a (docID, score) pair to the result set.
	put(docID uint32, score float32)
	// Get the score for a given docID. Returns false if docID not present.
	get(docID uint32) (float32, bool)
	// Iterate over all (docID, score) pairs in the result set.
	iterate(func(docID uint32, score float32))
	// Get the size of the result set.
	size() int64
}

// resultSetSliceThreshold defines the threshold ratio of k to total documents
// in the index, below which a map-based resultSet is used, and above which
// a slice-based resultSet is used.
// It is derived using the following reasoning:
//
// Let N = total number of documents
// Let K = number of top K documents to retrieve
//
// Memory usage if the Result Set uses a map[uint32]float32 of size K underneath:
//
//	~20 bytes per entry (key + value + map overhead)
//	Total ≈ 20 * K bytes
//
// Memory usage if the Result Set uses a slice of float32 of size N underneath:
//
//	4 bytes per entry
//	Total ≈ 4 * N bytes
//
// We want the threshold below which a map is more memory-efficient than a slice:
//
//	20K < 4N
//	K/N < 4/20
//
// Therefore, if the ratio of K to N is less than 0.2 (4/20), we use a map-based resultSet.
const resultSetSliceThreshold float64 = 0.2

// newResultSet creates a new resultSet
func newResultSet(k int64, numDocs uint64) resultSet {
	// if numDocs is zero (empty index), just use map-based resultSet as its a no-op
	// else decide based the percent of documents being retrieved. If we require
	// greater than 20% of total documents, use slice-based resultSet for better memory efficiency
	// else use map-based resultSet
	if numDocs == 0 || float64(k)/float64(numDocs) < resultSetSliceThreshold {
		return newResultSetMap(k)
	}
	return newResultSetSlice(numDocs)
}

type resultSetMap struct {
	data map[uint32]float32
}

func newResultSetMap(k int64) resultSet {
	return &resultSetMap{
		data: make(map[uint32]float32, k),
	}
}

func (rs *resultSetMap) put(docID uint32, score float32) {
	rs.data[docID] = score
}

func (rs *resultSetMap) get(docID uint32) (float32, bool) {
	score, exists := rs.data[docID]
	return score, exists
}

func (rs *resultSetMap) iterate(f func(docID uint32, score float32)) {
	for docID, score := range rs.data {
		f(docID, score)
	}
}

func (rs *resultSetMap) size() int64 {
	return int64(len(rs.data))
}

type resultSetSlice struct {
	count int64
	data  []float32
}

func newResultSetSlice(numDocs uint64) resultSet {
	data := make([]float32, numDocs)
	// scores can be negative, so initialize to a sentinel value which is NaN
	sentinel := float32(math.NaN())
	for i := range data {
		data[i] = sentinel
	}
	return &resultSetSlice{
		count: 0,
		data:  data,
	}
}

func (rs *resultSetSlice) put(docID uint32, score float32) {
	// only increment count if this docID was not already present
	if math.IsNaN(float64(rs.data[docID])) {
		rs.count++
	}
	rs.data[docID] = score
}

func (rs *resultSetSlice) get(docID uint32) (float32, bool) {
	score := rs.data[docID]
	if math.IsNaN(float64(score)) {
		return 0, false
	}
	return score, true
}

func (rs *resultSetSlice) iterate(f func(docID uint32, score float32)) {
	for docID, score := range rs.data {
		if !math.IsNaN(float64(score)) {
			f(uint32(docID), score)
		}
	}
}

func (rs *resultSetSlice) size() int64 {
	return rs.count
}

// -----------------------------------------------------------------------------
// Bitmap
// -----------------------------------------------------------------------------

// bitmap is a simple, fixed-size bitmap.
type bitmap struct {
	bits    []byte
	numBits uint32
}

// newBitmap creates a new bitmap with the given number of bits
func newBitmap(numBits uint32) *bitmap {
	bitsetSize := (numBits + 7) / 8
	return &bitmap{
		bits:    make([]byte, bitsetSize),
		numBits: numBits,
	}
}

// set the bit at the given position
func (b *bitmap) set(pos uint32) {
	if pos >= b.numBits {
		return
	}
	// set the bit in the byte slice
	// the byte index is pos / 8, which is equivalent to pos >> 3
	// the bit index within that byte is pos % 8, which is equivalent to pos & 7
	// and is from the LSB side of the byte
	b.bits[pos>>3] |= 1 << (pos & 7)
}

// clear the bit at the given position
func (b *bitmap) clear(pos uint32) {
	if pos >= b.numBits {
		return
	}
	// clear the bit in the byte slice
	// the byte index is pos / 8, which is equivalent to pos >> 3
	// the bit index within that byte is pos % 8, which is equivalent to pos & 7
	// and is from the LSB side of the byte
	b.bits[pos>>3] &^= 1 << (pos & 7)
}

// test if the bit at the given position is set
func (b *bitmap) test(pos uint32) bool {
	if pos >= b.numBits {
		return false
	}
	return (b.bits[pos>>3]>>(pos&7))&1 != 0
}

// return the underlying byte slice
func (b *bitmap) bytes() []byte {
	return b.bits
}

// returns the number of bits currently set
func (b *bitmap) cardinality() uint32 {
	var count int
	for _, byteVal := range b.bits {
		// count the number of set bits in the byte
		count += bits.OnesCount8(byteVal)
	}
	return uint32(count)
}

// isEmpty checks if the bitmap has no bits set
// or if the cardinality (population count) is zero
func (b *bitmap) isEmpty() bool {
	for _, byteVal := range b.bits {
		if byteVal != 0 {
			return false
		}
	}
	return true
}

// size returns the memory size of the bitmap in bytes
func (b *bitmap) size() uint64 {
	return reflectStaticSizeBitmap + uint64(len(b.bits))
}

// creates a clone of the bitmap
func (b *bitmap) clone() *bitmap {
	newB := &bitmap{}
	newB.bits = slices.Clone(b.bits)
	newB.numBits = b.numBits
	return newB
}

// -----------------------------------------------------------------------------
// ID Mapping
// -----------------------------------------------------------------------------

// idMapping maintains a bidirectional mapping between vector IDs and document IDs.
// It allows efficient retrieval of document IDs for given vector IDs and vice versa.
// The mapping assumes that vector IDs and document IDs are ordered sequentially starting from 0
// up to numVecs-1 and numDocs-1 respectively.
type idMapping struct {
	vecToDoc []uint32   //  vector ID -> document ID (size = numVecs)
	docToVec [][]uint32 //  document ID -> vector IDs (size = numDocs)

	// keep track of sizes for convenience
	numVecs uint32
	numDocs uint32
}

// newIDMapping creates a new idMapping with the specified sizes
// numVecs: number of vectors (for vecToDoc mapping)
// numDocs: number of documents (for docToVec mapping)
func newIDMapping(numVecs, numDocs uint32) *idMapping {
	return &idMapping{
		vecToDoc: make([]uint32, numVecs),
		docToVec: make([][]uint32, numDocs),
		numVecs:  numVecs,
		numDocs:  numDocs,
	}
}

// add a mapping from vector ID to document ID and vice versa
func (m *idMapping) add(vecID uint32, docID uint32) {
	// safety check to avoid out of bounds access
	if vecID >= m.numVecs || docID >= m.numDocs {
		return
	}
	m.vecToDoc[vecID] = docID
	m.docToVec[docID] = append(m.docToVec[docID], vecID)
}

// return the number of vectors in the mapping
func (m *idMapping) numVectors() uint32 {
	return m.numVecs
}

// return the number of documents in the mapping
func (m *idMapping) numDocuments() uint32 {
	return m.numDocs
}

// retrieve the document ID for a given vector ID
func (m *idMapping) docForVec(vecID uint32) (uint32, bool) {
	if vecID >= m.numVecs {
		return 0, false
	}
	return m.vecToDoc[vecID], true
}

// retrieve the vector IDs for a given document ID
func (m *idMapping) vecsForDoc(docID uint32) ([]uint32, bool) {
	if docID >= m.numDocs {
		return nil, false
	}
	return m.docToVec[docID], true
}

func (m *idMapping) size() uint64 {
	sizeInBytes := reflectStaticSizeIDMapping +
		(uint64(len(m.vecToDoc)) * uint64(SizeOfUint32)) +
		(uint64(len(m.docToVec)) * uint64(SizeOfSlice))
	for _, vecs := range m.docToVec {
		sizeInBytes += (uint64(cap(vecs)) * uint64(SizeOfUint32))
	}
	return sizeInBytes
}

// ------------------------------------------------------------------------------
// Quick Select
// ------------------------------------------------------------------------------

// topNIDsByDistance performs an in-place Quickselect on the dist slice (while
// keeping ids aligned with their corresponding distances) to find the N largest
// distances without fully sorting the data. It partitions the array such that
// the element at index len(dist)-n is the pivot separating the top-N largest
// values from the rest, and then returns the last N elements of both dist and
// ids (unordered)
func topNIDsByDistance(dist []float32, ids []int64, n int) ([]float32, []int64) {
	if n <= 0 || n > len(dist) {
		return nil, nil
	}

	// We want the N largest distances
	target := len(dist) - n

	left := 0
	right := len(dist) - 1
	for left < right {
		pivotVal := dist[right]
		store := left

		for i := left; i < right; i++ {
			// We want largest distances ⇒ partition small ones left
			if dist[i] < pivotVal {
				dist[i], dist[store] = dist[store], dist[i]
				ids[i], ids[store] = ids[store], ids[i]
				store++
			}
		}

		dist[store], dist[right] = dist[right], dist[store]
		ids[store], ids[right] = ids[right], ids[store]
		if store == target {
			break
		} else if store < target {
			left = store + 1
		} else {
			right = store - 1
		}
	}

	// Return top-N IDs (unordered)
	return dist[target:], ids[target:]
}

// -----------------------------------------------------------------------------
// vectorSet
// -----------------------------------------------------------------------------
type vectorSet struct {
	// dimensionality of each vector
	dim int
	// number of vectors represented
	nvecs int
	// float vectors stored in row-major format,
	// i.e. for N vectors of D dimensions,
	// the length of this slice is N*D,
	floatData []float32
	// row-major binary representation of the float vectors,
	// where each bit represents the sign bit
	// of the corresponding float value.
	binaryData []uint8
}

func newVectorSet(dim int, data []float32) (*vectorSet, error) {
	if len(data) == 0 || dim <= 0 || len(data)%dim != 0 {
		return nil, fmt.Errorf("invalid vector data: dims %d, data length %d", dim, len(data))
	}
	nvecs := len(data) / dim
	return &vectorSet{
		dim:       dim,
		nvecs:     nvecs,
		floatData: data,
	}, nil
}

func (v *vectorSet) binarize() {
	// if binaryData is already populated, no need to convert again
	if v.binaryData != nil {
		return
	}
	// convert the floatData to binary format and store in binaryData
	v.binaryData = faiss.RealToBinary(v.floatData, v.dim)
}

func (v *vectorSet) clone() *vectorSet {
	// create a new vectorSet with the same dimensions and number of vectors
	clone := &vectorSet{
		dim:        v.dim,
		nvecs:      v.nvecs,
		floatData:  slices.Clone(v.floatData),
		binaryData: slices.Clone(v.binaryData),
	}
	return clone
}

func (v *vectorSet) mergeWith(other *vectorSet) {
	// sanity check to ensure the two vector sets are compatible for merging
	if v.dim != other.dim {
		return
	}
	// merge the float data
	v.floatData = append(v.floatData, other.floatData...)
	v.nvecs += other.nvecs
	// invalidate the binary data as the float data has changed
	v.binaryData = nil
}
