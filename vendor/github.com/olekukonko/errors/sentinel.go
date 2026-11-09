// Comparable, immutable sentinel errors for package-level error variables.
//
// Relationship to errmgr.Define

// The errmgr subpackage provides a PARAMETERISED error factory:
//
//   var ErrDefined = errmgr.Define("ErrTimeout", "operation timed out after %s: %s")
//   err := ErrDefined.New("5s", "dial failed")   // produces a formatted *Error each call
//
// That is for creating new error instances from a template at call sites.
//
// errors.Const (this file) creates a STATIC SENTINEL — a single stable pointer
// stored once as a package-level variable and compared with errors.Is:
//
//   var ErrNotFound = errors.Const("not_found", "resource not found")
//
//   if errors.Is(err, ErrNotFound) { ... }   // pointer equality, always correct
//
// Use errmgr.Define when you need to produce many errors from a format template.
// Use errors.Const when you need a fixed comparable value for Is/switch matching.

package errors

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// Sentinel is a comparable, immutable error value safe to store as a
// package-level variable and match with errors.Is or a type switch.
//
// Unlike Named(), which returns a new *Error instance on every call (making
// pointer equality unreliable), each call to Const() returns a unique stable
// pointer. Two sentinels with identical name/msg are still distinct values
// unless they are the same pointer — intentional, to avoid accidental aliasing.
type Sentinel struct {
	name string
	msg  string
}

// Error implements the error interface.
func (s *Sentinel) Error() string { return s.msg }

// Is reports whether target is the same sentinel (pointer equality).
// This satisfies the errors.Is contract.
func (s *Sentinel) Is(target error) bool {
	t, ok := target.(*Sentinel)
	return ok && s == t
}

// As attempts to assign the sentinel to target if target is **Sentinel.
// Returns true if the assignment was made.
func (s *Sentinel) As(target any) bool {
	if tp, ok := target.(**Sentinel); ok {
		*tp = s
		return true
	}
	return false
}

// Unwrap returns nil — sentinels are root errors with no cause chain.
// Satisfies the errors.Unwrap contract.
func (s *Sentinel) Unwrap() error { return nil }

// Name returns the sentinel's name, useful for logging and diagnostics.
func (s *Sentinel) Name() string { return s.name }

// String returns a debug-friendly representation.
func (s *Sentinel) String() string {
	return fmt.Sprintf("Sentinel(%s: %s)", s.name, s.msg)
}

// LogValue implements slog.LogValuer so a Sentinel can be passed directly
// to any slog logging call and rendered as a structured group.
//
// Example:
//
//	slog.Error("lookup failed", "err", ErrNotFound)
//	// => err.error="resource not found", err.code="not_found"
func (s *Sentinel) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("error", s.msg),
		slog.String("code", s.name),
	)
}

// MarshalJSON serialises the sentinel to {"error":"...","code":"..."}.
func (s *Sentinel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}{
		Error: s.msg,
		Code:  s.name,
	})
}

// With returns a new *Error that wraps this sentinel as its cause and carries
// the additional message msg. Use this to add call-site context to a sentinel
// without losing the ability to match the original with errors.Is.
//
// Example:
//
//	var ErrNotFound = errors.Const("not_found", "resource not found")
//
//	// At call site:
//	err := ErrNotFound.With("user 42 not found")
//	errors.Is(err, ErrNotFound) // true — sentinel is in the cause chain
func (s *Sentinel) With(msg string) *Error {
	e := New(msg)
	e.cause = s
	return e
}

// Const creates a new sentinel error with the given name and message.
// Store the result as a package-level var; never call Const in a hot path.
//
// Example:
//
//	var (
//	    ErrNotFound   = errors.Const("not_found",   "resource not found")
//	    ErrForbidden  = errors.Const("forbidden",   "access denied")
//	    ErrBadRequest = errors.Const("bad_request", "invalid input")
//	)
//
//	func handle(err error) {
//	    switch {
//	    case errors.Is(err, ErrNotFound):   // 404
//	    case errors.Is(err, ErrForbidden):  // 403
//	    }
//	}
func Const(name, msg string) *Sentinel {
	return &Sentinel{name: name, msg: msg}
}
