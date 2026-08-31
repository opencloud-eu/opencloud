package faiss

/*
#include <faiss/c_api/error_c.h>
*/
import "C"
import (
	"errors"
	"fmt"
)

// faissError wraps an error returned by a faiss C API call,
// including the error type and the error code returned by the C API.
type faissError struct {
	errType error
	err     error
	errCode int
}

func (e *faissError) Error() string {
	return fmt.Sprintf("faiss %s: %s (code %d)", e.errType, e.err, e.errCode)
}

// returns the error type which can allow usage of
// errors.Is and errors.As for error handling.
func (e *faissError) Unwrap() error {
	return e.errType
}

// create a new faissError with the given error type, underlying error, and error code.
func newFaissError(errType, err error, errCode int) error {
	return &faissError{
		errType: errType,
		err:     err,
		errCode: errCode,
	}
}

// FAISS error types for categorizing errors returned by the C API.
var (
	// ---- Construction ----

	ErrCreateIndexFailed    = errors.New("create index failed")
	ErrCreateSelectorFailed = errors.New("create selector failed")

	// ---- Configuration ----

	ErrCreateParamsFailed = errors.New("create search params failed")
	ErrSetParamsFailed    = errors.New("set index params failed")

	// ---- Vector ops ----

	ErrAddFailed          = errors.New("add vectors failed")
	ErrTrainFailed        = errors.New("train index failed")
	ErrSearchFailed       = errors.New("search index failed")
	ErrReconstructFailed  = errors.New("reconstruct vector failed")
	ErrResetIndexFailed   = errors.New("reset index failed")
	ErrSetQuantizerFailed = errors.New("set quantizer failed")
	ErrMergeFromFailed    = errors.New("merge from index failed")
	ErrRemoveIDsFailed    = errors.New("remove IDs failed")

	// ---- Read-only index introspection ----

	ErrInspectIndexFailed = errors.New("inspect index failed")

	// ---- I/O ----

	ErrWriteIndexFailed = errors.New("write index failed")
	ErrReadIndexFailed  = errors.New("read index failed")

	// ---- GPU ----

	ErrNoUsableGPUDevices = errors.New("no GPU usable devices available")
	ErrGPUCloneFailed     = errors.New("GPU clone failed")
	ErrGPUSetupFailed     = errors.New("GPU setup failed")
	ErrGPUContextFailed   = errors.New("GPU context init failed")
	ErrGPUOutOfMemory     = errors.New("GPU out of memory")

	// ---- State / pre-condition errors ----

	ErrIndexNil      = errors.New("index is nil")
	ErrSelectorNil   = errors.New("selector is nil")
	ErrNotIDMapIndex = errors.New("index is not an IDMap index")
	ErrNotIVFIndex   = errors.New("index is not an IVF index")
	ErrNotBIVFIndex  = errors.New("index is not a binary IVF index")

	// ---- Unsupported operations ----

	ErrMergeFromNotSupported    = errors.New("merge from is not supported for this index type")
	ErrSetQuantizerNotSupported = errors.New("set quantizer not supported for this index type")
)

// getLastError returns the last error message set by the FAISS C API.
//
// The underlying C variable is thread-local / global and can be clobbered
// by concurrent FAISS calls or by goroutine rescheduling across OS threads,
// so this string is best-effort diagnostic context only. Always use the
// errType sentinel (with errors.Is / errors.As) to identify the error.
func getLastError() error {
	return errors.New(C.GoString(C.faiss_get_last_error()))
}
