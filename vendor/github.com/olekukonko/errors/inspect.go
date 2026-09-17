// Human-readable error inspection. Output is written to caller-supplied
// io.Writer values; this library never owns stdout or stderr.

package errors

import (
	stderrs "errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// inspectConfig holds resolved options for a single Inspect call.
type inspectConfig struct {
	w           io.Writer
	stackFrames int
	maxDepth    int
}

// InspectOption configures an Inspect call.
type InspectOption func(*inspectConfig)

// WithStackFrames sets the maximum number of stack frames printed per error node.
// Default is 3.
func WithStackFrames(n int) InspectOption {
	return func(c *inspectConfig) { c.stackFrames = n }
}

// WithMaxDepth sets the maximum chain depth traversed before output is truncated.
// Default is 10.
func WithMaxDepth(n int) InspectOption {
	return func(c *inspectConfig) { c.maxDepth = n }
}

// Inspect writes a human-readable description of err to each writer in ws.
// If no writers are supplied it defaults to os.Stderr.
// Multiple writers are combined with io.MultiWriter so a single call can
// write to a log file and a buffer simultaneously.
//
// Example — default (stderr):
//
//	errors.Inspect(err)
//
// Example — write to a buffer for testing:
//
//	var buf bytes.Buffer
//	errors.Inspect(err, &buf)
//
// Example — write to both stderr and a file:
//
//	errors.Inspect(err, os.Stderr, logFile)
//
// Example — customise stack depth:
//
//	errors.Inspect(err, os.Stderr, errors.WithStackFrames(5))
//
// Note: InspectOption values must come after all io.Writer values. Any value
// that is neither an io.Writer nor an InspectOption is silently ignored.
func Inspect(err error, targets ...interface{}) {
	cfg := &inspectConfig{
		stackFrames: 3,
		maxDepth:    10,
	}

	var writers []io.Writer
	for _, t := range targets {
		switch v := t.(type) {
		case InspectOption:
			v(cfg)
		case io.Writer:
			writers = append(writers, v)
		}
	}
	if len(writers) == 0 {
		writers = []io.Writer{os.Stderr}
	}
	if len(writers) == 1 {
		cfg.w = writers[0]
	} else {
		cfg.w = io.MultiWriter(writers...)
	}

	writeInspect(cfg, err)
}

// InspectError is a convenience wrapper for *Error that calls Inspect.
// Kept for backwards compatibility; prefer Inspect for new code.
func InspectError(err *Error, targets ...interface{}) {
	Inspect(err, targets...)
}

// writeInspect does the actual formatting.
func writeInspect(cfg *inspectConfig, err error) {
	w := cfg.w
	if err == nil {
		fmt.Fprintln(w, "no error")
		return
	}

	fmt.Fprintf(w, "\n=== error inspection ===\n")
	fmt.Fprintf(w, "type:    %T\n", err)
	fmt.Fprintf(w, "message: %v\n", err)

	switch e := err.(type) {
	case *Error:
		writeChain(cfg, e)
		writeDiagnostics(cfg, err)
	case *MultiError:
		errs := e.Errors()
		fmt.Fprintf(w, "errors:  %d\n", len(errs))
		for i, sub := range errs {
			fmt.Fprintf(w, "\n--- error %d ---\n", i+1)
			writeSingle(cfg, sub, 0)
		}
		writeDiagnostics(cfg, err)
	default:
		writeSingle(cfg, err, 0)
		writeDiagnostics(cfg, err)
	}
	fmt.Fprintf(w, "========================\n\n")
}

// writeChain walks an *Error chain printing each node.
func writeChain(cfg *inspectConfig, e *Error) {
	var current error = e
	depth := 0
	for current != nil && depth <= cfg.maxDepth {
		writeSingle(cfg, current, depth)
		next := stderrs.Unwrap(current)
		if next == current || next == nil {
			break
		}
		current = next
		depth++
	}
	if depth > cfg.maxDepth {
		fmt.Fprintf(cfg.w, "  ... (chain truncated at depth %d)\n", cfg.maxDepth)
	}
}

// writeSingle prints one error node at the given indent depth.
func writeSingle(cfg *inspectConfig, err error, depth int) {
	if err == nil {
		return
	}
	w := cfg.w
	pad := strings.Repeat("  ", depth)
	if depth > 0 {
		fmt.Fprintf(w, "%scause (%T): %v\n", pad, err, err)
	}
	e, ok := err.(*Error)
	if !ok {
		return
	}
	if n := e.Name(); n != "" {
		fmt.Fprintf(w, "%s  name:     %s\n", pad, n)
	}
	if cat := e.Category(); cat != "" {
		fmt.Fprintf(w, "%s  category: %s\n", pad, cat)
	}
	if code := e.Code(); code != 0 {
		fmt.Fprintf(w, "%s  code:     %d\n", pad, code)
	}
	if ctx := e.Context(); len(ctx) > 0 {
		fmt.Fprintf(w, "%s  context:\n", pad)
		for k, v := range ctx {
			fmt.Fprintf(w, "%s    %s: %v\n", pad, k, v)
		}
	}
	if stack := e.Stack(); len(stack) > 0 {
		limit := cfg.stackFrames
		if len(stack) < limit {
			limit = len(stack)
		}
		fmt.Fprintf(w, "%s  stack (top %d):\n", pad, limit)
		for i := 0; i < limit; i++ {
			fmt.Fprintf(w, "%s    %s\n", pad, stack[i])
		}
		if len(stack) > limit {
			fmt.Fprintf(w, "%s    ... (%d more frames)\n", pad, len(stack)-limit)
		}
	}
}

// writeDiagnostics appends a short diagnostic summary.
func writeDiagnostics(cfg *inspectConfig, err error) {
	var parts []string
	if IsRetryable(err) {
		parts = append(parts, "retryable")
	}
	if IsTimeout(err) {
		parts = append(parts, "timeout")
	}
	if code := getErrorCode(err); code != 0 {
		parts = append(parts, fmt.Sprintf("code=%d", code))
	}
	if len(parts) > 0 {
		fmt.Fprintf(cfg.w, "diagnostics: %s\n", strings.Join(parts, ", "))
	}
}

// getErrorCode traverses the error chain to find the first non-zero code.
func getErrorCode(err error) int {
	if e, ok := err.(*Error); ok {
		if c := e.Code(); c != 0 {
			return c
		}
	}
	var target *Error
	if As(err, &target) && target != nil {
		return target.Code()
	}
	return 0
}
