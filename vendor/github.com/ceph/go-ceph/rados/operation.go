package rados

// #include <stdlib.h>
import "C"

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/ceph/go-ceph/internal/log"
)

// The file operation.go exists to support both read op and write op types that
// have some pretty common behaviors between them. In C/C++ its assumed that
// the buffer types and other pointers will not be freed between passing them
// to the action setup calls (things like rados_write_op_write or
// rados_read_op_omap_get_vals2) and the call to Operate(...).  Since there's
// nothing stopping one from sleeping for hours between these calls, or passing
// the op to other functions and calling Operate there, we want a mechanism
// that's (fairly) simple to understand and won't run afoul of Go's garbage
// collection.  That's one reason the operation type tracks the steps (the
// parts that track complex inputs and outputs) so that as long as the op
// exists it will have a reference to the step, which will have references
// to the C language types.

type opKind string

const (
	readOp  opKind = "read"
	writeOp opKind = "write"
)

// OperationError is an error type that may be returned by an Operate call.
// It captures the error from the operate call itself and any errors from
// steps that can return an error.
type OperationError struct {
	kind       opKind
	OpError    error
	StepErrors map[int]error
}

func (e OperationError) Error() string {
	subErrors := []string{}
	if e.OpError != nil {
		subErrors = append(subErrors,
			fmt.Sprintf("op=%s", e.OpError))
	}
	for idx, es := range e.StepErrors {
		subErrors = append(subErrors,
			fmt.Sprintf("Step#%d=%s", idx, es))
	}
	return fmt.Sprintf(
		"%s operation error: %s",
		e.kind,
		strings.Join(subErrors, ", "))
}

func (e OperationError) Unwrap() []error {
	subErrors := make([]error, 0, len(e.StepErrors)+1)
	if e.OpError != nil {
		subErrors = append(subErrors, e.OpError)
	}
	for _, es := range e.StepErrors {
		subErrors = append(subErrors, es)
	}
	return subErrors
}

// opStep provides an interface for types that are tied to the management of
// data being input or output from write ops and read ops. The steps are
// meant to simplify the internals of the ops themselves and be exportable when
// appropriate. If a step is not being exported it should not be returned
// from an ops action function. If the step is exported it should be
// returned from an ops action function.
//
// Not all types implementing opStep are expected to need all the functions
// in the interface. However, for the sake of simplicity on the op side, we use
// the same interface for all cases and expect those implementing opStep
// just embed the without* types that provide no-op implementation of
// functions that make up this interface.
type opStep interface {
	// update the state of the step after the call to Operate.
	// It can be used to convert values from C and cache them and/or
	// communicate a failure of the action associated with the step.  The
	// update call will only be made once. Implementations are not required to
	// handle this call being made more than once.
	update() error
	// free will be called to free any resources, especially C memory, that
	// the step is managing. The behavior of free should be idempotent and
	// handle being called more than once.
	free()
}

// operation represents some of the shared underlying mechanisms for
// both read and write op types.
type operation struct {
	steps []opStep
}

// free will call the free method of all the steps this operation
// contains.
func (o *operation) free() {
	for i := range o.steps {
		o.steps[i].free()
	}
}

// update the operation and the steps it contains. The top-level result
// of the rados call is passed in as ret and used to construct errors.
// The update call of each step is used to update the contents of each
// step and gather any errors from those steps.
func (o *operation) update(kind opKind, ret C.int) error {
	stepErrors := map[int]error{}
	for i := range o.steps {
		if err := o.steps[i].update(); err != nil {
			stepErrors[i] = err
		}
	}
	if ret == 0 && len(stepErrors) == 0 {
		return nil
	}
	return OperationError{
		kind:       kind,
		OpError:    getError(ret),
		StepErrors: stepErrors,
	}
}

func opStepFinalizer(s opStep) {
	if s != nil {
		log.Warnf("unreachable opStep object found. Cleaning up.")
		s.free()
	}
}

// withoutUpdate can be embedded in a struct to help indicate
// the type implements the opStep interface but has a no-op
// update function.
type withoutUpdate struct{}

func (*withoutUpdate) update() error { return nil }

// withoutFree can be embedded in a struct to help indicate
// the type implements the opStep interface but has a no-op
// free function.
type withoutFree struct{}

func (*withoutFree) free() {}

// withRefs is a embeddable type to help track and free C memory.
type withRefs struct {
	refs []unsafe.Pointer
}

func (w *withRefs) free() {
	for i := range w.refs {
		C.free(w.refs[i])
	}
	w.refs = nil
}

func (w *withRefs) add(ptr unsafe.Pointer) {
	w.refs = append(w.refs, ptr)
}
