//go:build !go1.24
// +build !go1.24

package errors

import "runtime"

// setupCleanup registers a finalizer that returns e to the pool when the GC
// collects it — only when AutoFree is enabled.
//
// Finalizer limitation: the GC may not collect the object promptly, and
// finalizers run in a separate goroutine. This is acceptable for pool returns
// since Put() is safe to call from any goroutine and Reset() is idempotent.
func (ep *ErrorPool) setupCleanup(e *Error) {
	if !currentConfig.autoFree || currentConfig.disablePooling {
		return
	}
	runtime.SetFinalizer(e, func(target *Error) {
		if !currentConfig.disablePooling {
			ep.Put(target)
		}
	})
}

// clearCleanup removes the finalizer so explicit Free() calls do not race
// with a pending GC-triggered pool return (double-put).
// This is the correct approach for pre-1.24 Go where finalizers can be cleared.
func (ep *ErrorPool) clearCleanup(e *Error) {
	runtime.SetFinalizer(e, nil)
}
