package faiss

/*
#include <stdlib.h>
#include <stdint.h>
#include <faiss/c_api/Index_c_ex.h>
#include <faiss/c_api/IndexBinary_c_ex.h>
#include <faiss/c_api/IndexBinaryIVF_c_ex.h>
#include <faiss/c_api/IndexBinaryIVF_c.h>
#include <faiss/c_api/index_factory_c.h>
*/
import "C"
import (
	"encoding/json"
	"reflect"
	"unsafe"
)

var reflectStaticSizeFaissBinaryIndex uint64

func init() {
	var b faissBinaryIndex
	reflectStaticSizeFaissBinaryIndex = uint64(reflect.TypeOf(b).Size())
}

type BinaryIndex interface {
	// D returns the dimension of the indexed vectors.
	D() int

	// MetricType returns the metric type of the index.
	MetricType() int

	// Ntotal returns the total number of vectors currently stored in the index.
	Ntotal() int64

	// set the direct map type for IVF indexes.
	// 0 for No Map
	// 1 for Array
	// 2 for Hash
	SetDirectMap(maptype int) error

	// set the number of probes for IVF indexes
	SetNProbe(nprobe int32)

	// returns true if the underlying index is an IVF index
	IsIVFIndex() bool

	// IVFParams returns the nlist and nprobe parameters for IVF indexes
	IVFParams() (nprobe int, nlist int)

	// trains the index on a representative set of vectors
	Train(xb []uint8) error

	// adds vectors to the index
	Add(xb []uint8) error

	// sets the qunatizers from the source index, supposed to be used only for
	// BIVF indexes and returns error otherwise
	SetQuantizers(srcIndex BinaryIndex) error

	// merges another binary index into this one, currently applicable only for
	// IVF indexes returns an error
	MergeFrom(other BinaryIndex, add_id int64) error

	// queries the index with the vectors in xb
	// returns the IDs of the k nearest neighbors for each query vector and
	// their corresponding distances
	Search(xb []uint8, k int64) (distances []int32, labels []int64, err error)

	// SearchWithOptions performs a search with additional optional constraints.
	// - Selector can be used to restrict the search to a subset of the indexed vectors based on their IDs.
	// - params is a JSON object that can contain additional search parameters specific to the index type, such as IVF search parameters.
	SearchWithOptions(xb []uint8, k int64, sel Selector, params json.RawMessage) (distances []int32, labels []int64, err error)

	// returns a slice where each index corresponds to a cluster in an IVF
	// index, and the value at each index is the count of vectors in that
	// cluster, considering only the vectors specified in the include selector.
	ObtainClusterVectorCountsFromIVFIndex(include Selector, nlist int) (
		[]int64, error)

	// returns the IDs and distances of the closest numCentroids centroids to
	// the query vector xb, considering only the centroids specified in the
	// includedCentroids selector.
	ObtainClustersWithDistancesFromIVFIndex(xb []uint8, includedCentroids Selector,
		numCentroids int64) ([]int64, []int32, error)

	// Applicable only to IVF indexes: Returns the top k centroid cardinalities and
	// their vectors in chosen order (descending or ascending)
	ObtainKCentroidCardinalitiesFromIVFIndex(limit int, descending bool) ([]uint64, [][]uint8, error)

	// searches the specified clusters in an IVF index for the k nearest neighbors
	// of the query vector xb, considering only the vectors specified in the include selector
	// and additional search parameters passed as a JSON object.
	SearchClustersFromIVFIndex(eligibleCentroidIDs []int64, centroidDis []int32,
		centroidsToProbe int, xb []uint8, k int64, include Selector,
		params json.RawMessage) ([]int32, []int64, error)

	// Size estimates the memory footprint of the index in bytes
	// if the underlying faiss index is memory-mapped and not fully loaded into memory.
	Size() uint64

	// frees the memory associated with the index
	Close()

	// bPtr returns a pointer to the underlying C index struct.
	bPtr() *C.FaissIndexBinary

	// CodeSize returns the size of the produced codes in bytes.
	CodeSize() (uint64, error)
}

type faissBinaryIndex struct {
	bIdx *C.FaissIndexBinary
}

func (b *faissBinaryIndex) bPtr() *C.FaissIndexBinary {
	return b.bIdx
}

func (b *faissBinaryIndex) D() int {
	return int(C.faiss_IndexBinary_d(b.bIdx))
}

func (b *faissBinaryIndex) MetricType() int {
	return int(C.faiss_IndexBinary_metric_type(b.bIdx))
}

func (b *faissBinaryIndex) Ntotal() int64 {
	return int64(C.faiss_IndexBinary_ntotal(b.bIdx))
}

func (b *faissBinaryIndex) SetDirectMap(mapType int) (err error) {
	// Applicable only to IVF indexes
	ivfPtrBinary := C.faiss_IndexBinaryIVF_cast(b.bIdx)
	if ivfPtrBinary == nil {
		return ErrNotBIVFIndex
	}
	if c := C.faiss_IndexBinaryIVF_set_direct_map(
		ivfPtrBinary,
		C.int(mapType),
	); c != 0 {
		err = newFaissError(ErrSetParamsFailed, getLastError(), int(c))
	}
	return err
}

func (b *faissBinaryIndex) SetNProbe(nprobe int32) {
	// Applicable only to IVF indexes
	ivfPtrBinary := C.faiss_IndexBinaryIVF_cast(b.bIdx)
	if ivfPtrBinary == nil {
		return
	}
	C.faiss_IndexBinaryIVF_set_nprobe(ivfPtrBinary, C.size_t(nprobe))
}

func (b *faissBinaryIndex) IsIVFIndex() bool {
	ivfPtrBinary := C.faiss_IndexBinaryIVF_cast(b.bIdx)
	return ivfPtrBinary != nil
}

func (b *faissBinaryIndex) IVFParams() (nprobe int, nlist int) {
	// Applicable only to IVF indexes
	ivfPtrBinary := C.faiss_IndexBinaryIVF_cast(b.bIdx)
	if ivfPtrBinary == nil {
		return 0, 0
	}
	nlist = int(C.faiss_IndexBinaryIVF_nlist(ivfPtrBinary))
	nprobe = int(C.faiss_IndexBinaryIVF_nprobe(ivfPtrBinary))
	return nprobe, nlist
}

func (b *faissBinaryIndex) Train(x []uint8) error {
	n := (len(x) * 8) / b.D()
	if c := C.faiss_IndexBinary_train(b.bIdx, C.idx_t(n),
		(*C.uint8_t)(&x[0])); c != 0 {
		return newFaissError(ErrTrainFailed, getLastError(), int(c))
	}
	return nil
}

func (b *faissBinaryIndex) Add(x []uint8) error {
	n := (len(x) * 8) / b.D()
	if c := C.faiss_IndexBinary_add(b.bIdx, C.idx_t(n),
		(*C.uint8_t)(&x[0])); c != 0 {
		return newFaissError(ErrAddFailed, getLastError(), int(c))
	}
	return nil
}

func (b *faissBinaryIndex) Search(xb []uint8, k int64) (
	[]int32, []int64, error) {
	nq := (len(xb) * 8) / b.D()
	distances := make([]int32, int64(nq)*k)
	labels := make([]int64, int64(nq)*k)

	if c := C.faiss_IndexBinary_search(
		b.bIdx,
		C.idx_t(nq),
		(*C.uint8_t)(&xb[0]),
		C.idx_t(k),
		(*C.int32_t)(&distances[0]),
		(*C.idx_t)(&labels[0]),
	); c != 0 {
		return nil, nil, newFaissError(ErrSearchFailed, getLastError(), int(c))
	}
	return distances, labels, nil
}

func (b *faissBinaryIndex) SearchWithOptions(xb []uint8, k int64, sel Selector, params json.RawMessage) ([]int32, []int64, error) {
	if sel == nil && params == nil {
		return b.Search(xb, k)
	}
	return b.searchWithOptions(xb, k, sel, params)
}

func (b *faissBinaryIndex) searchWithOptions(xb []uint8, k int64, selector Selector,
	params json.RawMessage) ([]int32, []int64, error) {
	// Build a binary search params object to contain either the selector, the additional params, or both.
	searchParams, err := NewBinarySearchParams(b, params, selector, nil)
	if err != nil {
		return nil, nil, err
	}
	defer searchParams.Delete()

	nq := (len(xb) * 8) / b.D()
	distances := make([]int32, int64(nq)*k)
	labels := make([]int64, int64(nq)*k)

	if c := C.faiss_IndexBinary_search_with_params(
		b.bIdx,
		C.idx_t(nq),
		(*C.uint8_t)(&xb[0]),
		C.idx_t(k),
		searchParams.sp,
		(*C.int32_t)(&distances[0]),
		(*C.idx_t)(&labels[0]),
	); c != 0 {
		return nil, nil, newFaissError(ErrSearchFailed, getLastError(), int(c))
	}
	return distances, labels, nil
}

func (b *faissBinaryIndex) ObtainClusterVectorCountsFromIVFIndex(includedVectors Selector, nlist int) ([]int64, error) {
	// Applicable only to IVF indexes
	ivfPtrBinary := C.faiss_IndexBinaryIVF_cast(b.bIdx)
	if ivfPtrBinary == nil {
		return nil, ErrNotBIVFIndex
	}
	// Creating a slice to hold the count of vectors per cluster
	// Since we have nlist clusters, we create a slice of size nlist
	// listCount[i] will hold the count of vectors in cluster i
	listCount := make([]int64, nlist)
	// Creating a FAISS selector based on the include bitmap.
	params, err := NewStandardSearchParams(includedVectors)
	if err != nil {
		return nil, err
	}
	defer params.Delete()
	// Calling the C function to populate listCount
	// with the count of vectors per cluster, considering only
	// the vectors specified in the include selector.
	if c := C.faiss_IndexBinaryIVF_list_vector_count(
		ivfPtrBinary,
		(*C.idx_t)(unsafe.Pointer(&listCount[0])),
		C.size_t(nlist),
		params.sp,
	); c != 0 {
		return nil, newFaissError(ErrInspectIndexFailed, getLastError(), int(c))
	}
	return listCount, nil
}

func (b *faissBinaryIndex) ObtainClustersWithDistancesFromIVFIndex(xb []uint8, includedCentroids Selector, numCentroids int64) ([]int64, []int32, error) {
	// Applicable only to IVF indexes
	ivfPtrBinary := C.faiss_IndexBinaryIVF_cast(b.bIdx)
	if ivfPtrBinary == nil {
		return nil, nil, ErrNotBIVFIndex
	}
	params, err := NewStandardSearchParams(includedCentroids)
	if err != nil {
		return nil, nil, err
	}
	defer params.Delete()

	// Populate these with the centroids and their distances.
	centroids := make([]int64, numCentroids)
	centroidDistances := make([]int32, numCentroids)

	n := (len(xb) * 8) / b.D()

	if c := C.faiss_IndexBinaryIVF_search_closest_eligible_centroids(
		ivfPtrBinary,
		(C.idx_t)(n),
		(*C.uint8_t)(&xb[0]),
		(C.idx_t)(numCentroids),
		(*C.int32_t)(&centroidDistances[0]),
		(*C.idx_t)(&centroids[0]),
		params.sp,
	); c != 0 {
		return nil, nil, newFaissError(ErrSearchFailed, getLastError(), int(c))
	}

	return centroids, centroidDistances, nil
}

func (b *faissBinaryIndex) ObtainKCentroidCardinalitiesFromIVFIndex(limit int, descending bool) (
	[]uint64, [][]uint8, error) {
	if limit <= 0 {
		return nil, nil, nil
	}

	// Applicable only to IVF indexes
	ivfPtrBinary := C.faiss_IndexBinaryIVF_cast(b.bIdx)
	if ivfPtrBinary == nil {
		return nil, nil, ErrNotBIVFIndex
	}

	nlist := int(C.faiss_IndexBinaryIVF_nlist(ivfPtrBinary))
	if nlist == 0 {
		return nil, nil, nil
	}

	centroidCardinalities := make([]C.size_t, nlist)

	// Allocate a flat buffer for all centroids, then slice it per centroid
	d := b.D()
	flatCentroids := make([]uint8, nlist*d/8)

	// Call the C function to fill centroid vectors and cardinalities
	c := C.faiss_IndexBinaryIVF_get_centroids_and_cardinality(
		ivfPtrBinary,
		(*C.uint8_t)(&flatCentroids[0]),
		(*C.size_t)(&centroidCardinalities[0]),
		nil,
	)
	if c != 0 {
		return nil, nil, newFaissError(ErrInspectIndexFailed, getLastError(), int(c))
	}

	topIndices := getIndicesOfKCentroidCardinalities(
		centroidCardinalities,
		min(limit, nlist),
		descending)

	rvCardinalities := make([]uint64, len(topIndices))
	rvCentroids := make([][]uint8, len(topIndices))

	for i, idx := range topIndices {
		rvCardinalities[i] = uint64(centroidCardinalities[idx])
		rvCentroids[i] = flatCentroids[idx*d : (idx+1)*d]
	}

	return rvCardinalities, rvCentroids, nil

}

func (b *faissBinaryIndex) SearchClustersFromIVFIndex(eligibleCentroidIDs []int64, centroidDis []int32, centroidsToProbe int,
	xb []uint8, k int64, include Selector, params json.RawMessage) ([]int32, []int64, error) {
	// Applicable only to IVF indexes
	ivfPtrBinary := C.faiss_IndexBinaryIVF_cast(b.bIdx)
	if ivfPtrBinary == nil {
		return nil, nil, ErrNotBIVFIndex
	}
	// If no include selector is provided, we have no results to return.
	// return an error indicating that the SearchClustersFromIVFIndex requires a valid selector.
	if include == nil {
		return nil, nil, ErrSelectorNil
	}
	// create a temporary search params object to set nprobe, this will override
	// the nprobe and the nlist set at index time, this will allow the search to
	// probe only the clusters specified in eligibleCentroidIDs
	tempParams := &defaultSearchParamsIVF{
		// Nlist is set to the number of eligible centroids, which will override
		// the nlist set at index time.
		Nlist: len(eligibleCentroidIDs),
		// Have to override nprobe so that more clusters will be searched for this
		// query, if required.
		Nprobe: centroidsToProbe,
	}
	searchParams, err := NewBinarySearchParams(b, params, include, tempParams)
	if err != nil {
		return nil, nil, err
	}
	defer searchParams.Delete()

	n := (len(xb) * 8) / b.D()

	distances := make([]int32, int64(n)*k)
	labels := make([]int64, int64(n)*k)
	// Adjust the slices to match the effective nprobe set in searchParams, as the input
	// parameters may have different nprobe value, which will be a hard override, over the
	// centroidsToProbe value passed to this function.
	// If the effective nprobe is greater than the length of eligibleCentroidIDs,
	// we limit it to the length of eligibleCentroidIDs.
	effectiveNprobe := min(getNProbeFromSearchParams(searchParams), int32(len(eligibleCentroidIDs)))
	eligibleCentroidIDs = eligibleCentroidIDs[:effectiveNprobe]
	centroidDis = centroidDis[:effectiveNprobe]

	if c := C.faiss_IndexBinaryIVF_search_preassigned_with_params(
		ivfPtrBinary,
		(C.idx_t)(n),
		(*C.uint8_t)(&xb[0]),
		(C.idx_t)(k),
		(*C.idx_t)(&eligibleCentroidIDs[0]),
		(*C.int32_t)(&centroidDis[0]),
		(*C.int32_t)(&distances[0]),
		(*C.idx_t)(&labels[0]),
		(C.int)(0),
		searchParams.sp,
	); c != 0 {
		return nil, nil, newFaissError(ErrSearchFailed, getLastError(), int(c))
	}

	return distances, labels, nil
}

func (b *faissBinaryIndex) Size() uint64 {
	rv := reflectStaticSizeFaissBinaryIndex
	var size C.size_t
	if code := C.faiss_IndexBinary_size(b.bIdx, &size); code == 0 {
		rv += uint64(size)
	}
	return rv
}

func (b *faissBinaryIndex) CodeSize() (uint64, error) {
	var size C.size_t
	if c := C.faiss_IndexBinary_sa_code_size(b.bIdx, &size); c != 0 {
		return 0, newFaissError(ErrInspectIndexFailed, getLastError(), int(c))
	}
	return uint64(size), nil
}

func (idx *faissBinaryIndex) Close() {
	C.faiss_IndexBinary_free(idx.bIdx)
}

type BinaryIndexImpl struct {
	BinaryIndex
}

func BinaryIndexFactory(dims int, description string) (*BinaryIndexImpl, error) {
	var cDescription *C.char
	if description != "" {
		cDescription = C.CString(description)
		defer C.free(unsafe.Pointer(cDescription))
	}
	var idx faissBinaryIndex
	if c := C.faiss_index_binary_factory(&idx.bIdx, C.int(dims), cDescription); c != 0 {
		return nil, newFaissError(ErrCreateIndexFailed, getLastError(), int(c))
	}
	return &BinaryIndexImpl{&idx}, nil
}

func (idx *faissBinaryIndex) SetQuantizers(srcIndex BinaryIndex) error {
	if !(idx.IsIVFIndex() && srcIndex.IsIVFIndex()) {
		return ErrSetQuantizerNotSupported
	}
	c := C.faiss_Set_quantizers_binary(idx.bIdx, srcIndex.bPtr())
	if c != 0 {
		return newFaissError(ErrSetQuantizerFailed, getLastError(), int(c))
	}
	return nil
}

func (idx *faissBinaryIndex) MergeFrom(other BinaryIndex, add_id int64) (err error) {
	if !(idx.IsIVFIndex() && other.IsIVFIndex()) {
		return ErrMergeFromNotSupported
	}
	if c := C.faiss_IndexBinaryIVF_merge_from(
		idx.bPtr(),
		other.bPtr(),
		(C.idx_t)(add_id),
	); c != 0 {
		err = newFaissError(ErrMergeFromFailed, getLastError(), int(c))
	}
	return err
}
