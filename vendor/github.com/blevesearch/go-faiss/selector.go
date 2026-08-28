package faiss

/*
#include <faiss/c_api/impl/AuxIndexStructures_c.h>
*/
import "C"

// Note: currently we have only one implementation, but we keep the interface for future extensibility
type Selector interface {
	ExcludeFilter() bool
	Get() *C.FaissIDSelector
	Delete()
}

// IDSelector represents a set of IDs to remove.
type IDSelector struct {
	exclude bool
	sel     *C.FaissIDSelector
	inner   *C.FaissIDSelector
}

func (s *IDSelector) Get() *C.FaissIDSelector {
	return s.sel
}

func (s *IDSelector) ExcludeFilter() bool {
	return s.exclude
}

// Delete frees the memory associated with s.
func (s *IDSelector) Delete() {
	if s == nil {
		return
	}

	if s.sel != nil {
		C.faiss_IDSelector_free(s.sel)
	}
	if s.inner != nil {
		C.faiss_IDSelector_free(s.inner)
	}
}

// NewIDSelectorRange creates a selector that removes IDs on [imin, imax).
func NewIDSelectorRange(imin, imax int64) (Selector, error) {
	var sel *C.FaissIDSelectorRange
	c := C.faiss_IDSelectorRange_new(&sel, C.idx_t(imin), C.idx_t(imax))
	if c != 0 {
		return nil, newFaissError(ErrCreateSelectorFailed, getLastError(), int(c))
	}
	return &IDSelector{sel: (*C.FaissIDSelector)(sel)}, nil
}

// NewIDSelectorBatch creates a new batch selector.
func NewIDSelectorBatch(indices []int64) (Selector, error) {
	var sel *C.FaissIDSelectorBatch
	if c := C.faiss_IDSelectorBatch_new(
		&sel,
		C.size_t(len(indices)),
		(*C.idx_t)(&indices[0]),
	); c != 0 {
		return nil, newFaissError(ErrCreateSelectorFailed, getLastError(), int(c))
	}
	return &IDSelector{sel: (*C.FaissIDSelector)(sel)}, nil
}

// NewIDSelectorBatchNot creates a new Not selector, wrapped around a
// batch selector, with the IDs in 'exclude'.
func NewIDSelectorBatchNot(exclude []int64) (Selector, error) {
	batchSelector, err := NewIDSelectorBatch(exclude)
	if err != nil {
		return nil, err
	}

	var sel *C.FaissIDSelectorNot
	if c := C.faiss_IDSelectorNot_new(
		&sel,
		batchSelector.Get(),
	); c != 0 {
		batchSelector.Delete()
		return nil, newFaissError(ErrCreateSelectorFailed, getLastError(), int(c))
	}
	return &IDSelector{exclude: true,
		sel:   (*C.FaissIDSelector)(sel),
		inner: batchSelector.Get()}, nil
}

// NewIDSelectorBitmap creates a selector using a bitset, where each bit
// indicates whether the corresponding ID is to be selected.
// NOTE: This function assumes that len(bitmap)*8 covers the full range of IDs
// in the index, and only works when we have vector IDs ranging from 0 to N-1,
// where N is the number of vectors in the index.
// The length of the bitmap should be at least ceil(N/8).
func NewIDSelectorBitmap(bitmap []byte) (Selector, error) {
	var sel *C.FaissIDSelectorBitmap
	if c := C.faiss_IDSelectorBitmap_new(
		&sel,
		C.size_t(len(bitmap)),
		(*C.uint8_t)(&bitmap[0]),
	); c != 0 {
		return nil, newFaissError(ErrCreateSelectorFailed, getLastError(), int(c))
	}
	return &IDSelector{sel: (*C.FaissIDSelector)(sel)}, nil
}

// NewIDSelectorBitmapNot creates a NOT selector using a bitset, where each bit
// indicates whether the corresponding ID is NOT to be selected.
// NOTE: This function assumes that len(bitmap)*8 covers the full range of IDs
// in the index, and only works when we have vector IDs ranging from 0 to N-1,
// where N is the number of vectors in the index.
// The length of the bitmap should be at least ceil(N/8).
func NewIDSelectorBitmapNot(bitmap []byte) (Selector, error) {
	bitmapSelector, err := NewIDSelectorBitmap(bitmap)
	if err != nil {
		return nil, err
	}
	var sel *C.FaissIDSelectorNot
	if c := C.faiss_IDSelectorNot_new(
		&sel,
		bitmapSelector.Get(),
	); c != 0 {
		bitmapSelector.Delete()
		return nil, newFaissError(ErrCreateSelectorFailed, getLastError(), int(c))
	}
	return &IDSelector{exclude: true,
		sel:   (*C.FaissIDSelector)(sel),
		inner: bitmapSelector.Get()}, nil
}
