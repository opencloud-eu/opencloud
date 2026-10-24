package faiss

/*
#include <faiss/c_api/Index_c.h>
#include <faiss/c_api/IndexIVF_c.h>
#include <faiss/c_api/IndexIVF_c_ex.h>
#include <faiss/c_api/IndexBinaryIVF_c.h>
#include <faiss/c_api/impl/AuxIndexStructures_c.h>
*/
import "C"
import (
	"encoding/json"
	"fmt"
)

type SearchParams struct {
	sp *C.FaissSearchParameters
}

// Delete frees the memory associated with s.
func (s *SearchParams) Delete() {
	if s == nil || s.sp == nil {
		return
	}
	C.faiss_SearchParameters_free(s.sp)
}

type searchParamsIVF struct {
	NprobePct   float32 `json:"ivf_nprobe_pct,omitempty"`
	MaxCodesPct float32 `json:"ivf_max_codes_pct,omitempty"`
}

// IVF Parameters used to override the index-time defaults for a specific query.
// Serve as the 'new' defaults for this query, unless overridden by search-time
// params.
type defaultSearchParamsIVF struct {
	Nprobe int `json:"ivf_nprobe,omitempty"`
	Nlist  int `json:"ivf_nlist,omitempty"`
}

func (s *searchParamsIVF) Validate() error {
	if s.NprobePct < 0 || s.NprobePct > 100 {
		return fmt.Errorf("invalid IVF search params, ivf_nprobe_pct:%v, "+
			"should be in range [0, 100]", s.NprobePct)
	}

	if s.MaxCodesPct < 0 || s.MaxCodesPct > 100 {
		return fmt.Errorf("invalid IVF search params, ivf_max_codes_pct:%v, "+
			"should be in range [0, 100]", s.MaxCodesPct)
	}

	return nil
}

func getNProbeFromSearchParams(params *SearchParams) int32 {
	return int32(C.faiss_SearchParametersIVF_nprobe(params.sp))
}

// Returns a valid SearchParams object, configured according to the provided
// parameters and selector. The returned SearchParams object is allocated,
// thus caller must clean up the object by invoking Delete() method.
func NewSearchParams(idx Index, params json.RawMessage, selector Selector,
	defaultParams *defaultSearchParamsIVF) (*SearchParams, error) {
	// Get the selector C pointer, if any.
	// A nil selector indicates no ID filtering, and it is valid
	// to send a nil pointer to Faiss.
	var sel *C.FaissIDSelector
	if selector != nil {
		sel = selector.Get()
	}

	ivfIdx := C.faiss_IndexIVF_cast(idx.cPtr())
	// if the index is not an IVF index, create a standard SearchParameters object
	if ivfIdx == nil {
		rv := &SearchParams{}
		// Create standard SearchParameters for non-IVF index
		if c := C.faiss_SearchParameters_new(&rv.sp, sel); c != 0 {
			return nil, ErrCreateParamsFailed
		}
		return rv, nil
	}

	nlist := int(C.faiss_IndexIVF_nlist(ivfIdx))
	nprobe := int(C.faiss_IndexIVF_nprobe(ivfIdx))
	nvecs := int(C.faiss_Index_ntotal(idx.cPtr()))

	maxCodes, nprobe, err := resolveSearchParams(params, defaultParams, nlist, nprobe, nvecs)
	if err != nil {
		return nil, err
	}

	if idx.HasRaBitQ() {
		return buildRaBitQSearchParams(maxCodes, nprobe, sel)
	}
	return buildIVFSearchParams(maxCodes, nprobe, sel)
}

func resolveSearchParams(params json.RawMessage, defaultParams *defaultSearchParamsIVF,
	nlist, nprobe, nvecs int) (int, int, error) {
	if defaultParams != nil {
		if defaultParams.Nlist > 0 {
			nlist = defaultParams.Nlist
		}
		if defaultParams.Nprobe > 0 {
			nprobe = defaultParams.Nprobe
		}
	}
	var ivfParams searchParamsIVF
	if len(params) > 0 {
		if err := json.Unmarshal(params, &ivfParams); err != nil {
			return 0, 0, fmt.Errorf("failed to unmarshal IVF search params, "+
				"err:%v", err)
		}
		if err := ivfParams.Validate(); err != nil {
			return 0, 0, err
		}
	}
	if ivfParams.NprobePct > 0 {
		nprobe = max(int(float32(nlist)*(ivfParams.NprobePct/100)), 1)
	}
	var maxCodes int
	if ivfParams.MaxCodesPct > 0 {
		maxCodes = int(float32(nvecs) * (ivfParams.MaxCodesPct / 100))
	} // else, maxCodes will be set to the default value of 0, which means no limit
	return maxCodes, nprobe, nil
}

func buildIVFSearchParams(maxCodes, nprobe int, sel *C.FaissIDSelector) (*SearchParams, error) {
	sp := &SearchParams{}
	if c := C.faiss_SearchParametersIVF_new_with(
		&sp.sp,
		sel,
		C.size_t(nprobe),
		C.size_t(maxCodes),
	); c != 0 {
		return nil, ErrCreateParamsFailed
	}

	return sp, nil
}

func buildRaBitQSearchParams(maxCodes, nprobe int, sel *C.FaissIDSelector) (*SearchParams, error) {
	sp := &SearchParams{}
	if c := C.faiss_SearchParametersRaBitQ_new_with(
		&sp.sp,
		sel,
		C.size_t(nprobe),
		C.size_t(maxCodes),
	); c != 0 {
		return nil, ErrCreateParamsFailed
	}

	return sp, nil
}

// Returns a standard SearchParams object without any special settings with
// the provided selector. The returned SearchParams object is allocated,
// thus caller must clean up the object by invoking Delete() method.
func NewStandardSearchParams(selector Selector) (*SearchParams, error) {
	var sel *C.FaissIDSelector
	if selector != nil {
		sel = selector.Get()
	}
	rv := &SearchParams{}
	if c := C.faiss_SearchParameters_new(&rv.sp, sel); c != 0 {
		return nil, ErrCreateParamsFailed
	}
	return rv, nil
}

func NewBinarySearchParams(idx BinaryIndex, params json.RawMessage, selector Selector,
	defaultParams *defaultSearchParamsIVF) (*SearchParams, error) {
	// Get the selector C pointer, if any.
	// A nil selector indicates no ID filtering, and it is valid
	// to send a nil pointer to Faiss.
	var sel *C.FaissIDSelector
	if selector != nil {
		sel = selector.Get()
	}

	ivfPtrBinary := C.faiss_IndexBinaryIVF_cast(idx.bPtr())

	// if the index is not an IVF index, create a standard SearchParameters object
	if ivfPtrBinary == nil {
		rv := &SearchParams{}
		// Create standard SearchParameters for non-IVF index
		if c := C.faiss_SearchParameters_new(&rv.sp, sel); c != 0 {
			return nil, ErrCreateParamsFailed
		}
		return rv, nil
	}

	nlist := int(C.faiss_IndexBinaryIVF_nlist(ivfPtrBinary))
	nprobe := int(C.faiss_IndexBinaryIVF_nprobe(ivfPtrBinary))
	nvecs := int(C.faiss_IndexBinary_ntotal(idx.bPtr()))

	maxCodes, nprobe, err := resolveSearchParams(params, defaultParams, nlist, nprobe, nvecs)
	if err != nil {
		return nil, err
	}

	return buildIVFSearchParams(maxCodes, nprobe, sel)
}
