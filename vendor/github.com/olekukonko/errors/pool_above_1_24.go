//go:build go1.24
// +build go1.24

package errors

import "runtime"

// setupCleanup registers a runtime.AddCleanup callback that returns e to the
// pool when the GC determines e is unreachable — only when AutoFree is enabled.
//
// IMPORTANT: the cleanup argument must be e itself (passed as the third arg),
// NOT captured in the closure. Capturing e in the closure creates a strong
// reference from the cleanup to e, which prevents e from ever becoming
// unreachable and defeats the purpose of the cleanup entirely.
func (ep *ErrorPool) setupCleanup(e *Error) {
	if !currentConfig.autoFree || currentConfig.disablePooling {
		return
	}
	runtime.AddCleanup(e, func(target *Error) {
		if !currentConfig.disablePooling {
			ep.Put(target)
		}
	}, e)
}

// clearCleanup is a no-op for Go 1.24+.
// runtime.AddCleanup does not support cancellation; the double-put risk is
// mitigated by Free() resetting the error before Put, making a second Put
// of an already-reset error safe (it just returns a clean object to the pool).
func (ep *ErrorPool) clearCleanup(_ *Error) {}
