package errors

import (
	"database/sql"
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// captureStack captures a stack trace with the configured depth.
// Immune to inlining: captures from frame 1 and trims by shifting within
// the same buffer so the pooled slice always retains its full capacity.
func captureStack(skip int) []uintptr {
	buf := stackPool.Get().([]uintptr)
	buf = buf[:cap(buf)]

	// Capture from frame 1 (skipping runtime.Callers itself).
	// captureStack can never be inlined because it calls runtime.Callers,
	// so buf[0] is always captureStack regardless of compiler inlining above.
	n := runtime.Callers(1, buf)
	if n == 0 {
		stackPool.Put(buf)
		return nil
	}

	// Trim leading internal frames in-place using copy, preserving the
	// buffer's full capacity so the pool never fills with shrinking slices.
	// skip+1: +1 for captureStack itself (always buf[0]).
	trimmed := skip + 1
	if trimmed >= n {
		stackPool.Put(buf)
		return nil
	}

	length := n - trimmed
	// Shift the useful frames to the start of buf — same backing array,
	// same capacity, zero allocation.
	copy(buf, buf[trimmed:n])
	return buf[:length]
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// clearMap removes all entries from a map without reallocating it.
func clearMap(m map[string]interface{}) {
	for k := range m {
		delete(m, k)
	}
}

// sqlNull detects if a value represents a SQL NULL type.
func sqlNull(v interface{}) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case sql.NullString:
		return !val.Valid
	case sql.NullTime:
		return !val.Valid
	case sql.NullInt64:
		return !val.Valid
	case sql.NullBool:
		return !val.Valid
	case sql.NullFloat64:
		return !val.Valid
	default:
		return false
	}
}

// getFuncName extracts the function name from an interface value.
// Returns "unknown" if the input is nil or invalid.
func getFuncName(fn interface{}) string {
	if fn == nil {
		return "unknown"
	}
	fullName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	return strings.TrimPrefix(fullName, ".")
}

// isInternalFrame reports whether a stack frame belongs to this library's
// internals and should be filtered from user-visible stack traces.
//
// Rules:
//   - runtime.* and reflect.* are always internal.
//   - _test.go files are NEVER internal: test functions must survive
//     filtering so that assertions like "stack contains testing.tRunner"
//     and "stack contains TestErrorTraceStackContent" can pass.
//   - Source files under github.com/olekukonko/errors/ (errors.go, utils.go,
//     helper.go, retry.go, multi_error.go) are internal.
func isInternalFrame(frame runtime.Frame) bool {
	if strings.HasPrefix(frame.Function, "runtime.") || strings.HasPrefix(frame.Function, "reflect.") {
		return true
	}

	// Exempt test files before the path-prefix check: errors_test.go lives
	// at github.com/olekukonko/errors/errors_test.go which contains the
	// "errors" suffix and would otherwise be incorrectly filtered.
	if strings.HasSuffix(frame.File, "_test.go") {
		return false
	}

	suffixes := []string{
		"errors",
		"utils",
		"helper",
		"retry",
		"multi",
	}
	for _, v := range suffixes {
		if strings.Contains(frame.File, fmt.Sprintf("github.com/olekukonko/errors/%s", v)) {
			return true
		}
	}
	return false
}

// FormatError returns a formatted string representation of an error.
func FormatError(err error) string {
	if err == nil {
		return "<nil>"
	}
	var sb strings.Builder
	if e, ok := err.(*Error); ok {
		sb.WriteString(fmt.Sprintf("Error: %s\n", e.Error()))
		if e.name != "" {
			sb.WriteString(fmt.Sprintf("Name: %s\n", e.name))
		}
		if ctx := e.Context(); len(ctx) > 0 {
			sb.WriteString("Context:\n")
			for k, v := range ctx {
				sb.WriteString(fmt.Sprintf("\t%s: %v\n", k, v))
			}
		}
		if stack := e.Stack(); len(stack) > 0 {
			sb.WriteString("Stack Trace:\n")
			for _, frame := range stack {
				sb.WriteString(fmt.Sprintf("\t%s\n", frame))
			}
		}
		if e.cause != nil {
			sb.WriteString(fmt.Sprintf("Caused by: %s\n", FormatError(e.cause)))
		}
	} else {
		sb.WriteString(fmt.Sprintf("Error: %s\n", err.Error()))
	}
	return sb.String()
}

// Caller returns the file, line, and function name of the caller at skip level.
// Skip=0 returns the caller of this function, 1 returns its caller, etc.
func Caller(skip int) (file string, line int, function string) {
	configMu.RLock()
	defer configMu.RUnlock()
	var pcs [1]uintptr
	n := runtime.Callers(skip+2, pcs[:])
	if n == 0 {
		return "", 0, "unknown"
	}
	frame, _ := runtime.CallersFrames(pcs[:n]).Next()
	return frame.File, frame.Line, frame.Function
}
