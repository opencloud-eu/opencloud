package errors

import (
	"bytes"
	"fmt"
	"regexp"
	"sync"
)

// Constants defining default configuration and context keys.
const (
	ctxTimeout = "[error] timeout" // Context key marking timeout errors.
	ctxRetry   = "[error] retry"   // Context key marking retryable errors.

	contextSize = 4   // Initial size of fixed-size context array for small contexts.
	bufferSize  = 256 // Initial buffer size for JSON marshaling.
	warmUpSize  = 100 // Number of errors to pre-warm the pool for efficiency.
	stackDepth  = 32  // Maximum stack trace depth to prevent excessive memory use.

	DefaultCode = 500 // Default HTTP status code for errors if not specified.
)

// spaceRe is a precompiled regex for normalizing whitespace in error messages.
var spaceRe = regexp.MustCompile(`\s+`)

// jsonBufferPool manages reusable buffers for JSON marshaling to reduce allocations.
var (
	jsonBufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, bufferSize))
		},
	}
)

// ErrorCategory is a string type for categorizing errors (e.g., "network", "validation").
type ErrorCategory string

// ErrorOpts provides options for customizing error creation.
type ErrorOpts struct {
	SkipStack int // Number of stack frames to skip when capturing the stack trace.
}

// Config defines the global configuration for the errors package, controlling
// stack depth, context size, pooling, and frame filtering.
type Config struct {
	StackDepth     int  // Maximum stack trace depth; 0 uses default (32).
	ContextSize    int  // Initial context map size; 0 uses default (4).
	DisablePooling bool // If true, disables object pooling for errors.
	FilterInternal bool // If true, filters internal package frames from stack traces.
	AutoFree       bool // If true, automatically returns errors to pool when GC collects them.
}

// cachedConfig holds the current configuration, updated only by Configure().
// Protected by configMu for thread-safety.
type cachedConfig struct {
	stackDepth     int
	contextSize    int
	disablePooling bool
	filterInternal bool
	autoFree       bool
}

var (
	// currentConfig stores the active configuration, read frequently and updated rarely.
	currentConfig cachedConfig
	// configMu protects updates to currentConfig for thread-safety.
	configMu sync.RWMutex
	// errorPool manages reusable Error instances to reduce allocations.
	errorPool = NewErrorPool()
	// stackPool manages reusable stack trace slices for efficiency.
	stackPool = sync.Pool{
		New: func() interface{} {
			return make([]uintptr, currentConfig.stackDepth)
		},
	}
	// emptyError is a pre-allocated empty error for lightweight reuse.
	emptyError = &Error{
		smallContext: [contextSize]contextItem{},
		msg:          "",
		name:         "",
		template:     "",
		cause:        nil,
	}
)

// contextItem holds a single key-value pair in the smallContext array.
type contextItem struct {
	key   string
	value interface{}
}

// init sets up the package with default configuration and pre-warms the error pool.
func init() {
	currentConfig = cachedConfig{
		stackDepth:     stackDepth,
		contextSize:    contextSize,
		disablePooling: false,
		filterInternal: true,
		autoFree:       false, // opt-in; explicit Free() is the safe default
	}
	WarmPool(warmUpSize) // Pre-allocate errors for performance.
}

// Configure updates the global configuration for the errors package.
// It is thread-safe and should be called early to avoid race conditions.
// Changes apply to all subsequent error operations.
// Example:
//
//	errors.Configure(errors.Config{StackDepth: 16, DisablePooling: true})
func Configure(cfg Config) {
	configMu.Lock()
	defer configMu.Unlock()

	if cfg.StackDepth != 0 {
		currentConfig.stackDepth = cfg.StackDepth
	}
	if cfg.ContextSize != 0 {
		currentConfig.contextSize = cfg.ContextSize
	}
	currentConfig.disablePooling = cfg.DisablePooling
	currentConfig.filterInternal = cfg.FilterInternal
	currentConfig.autoFree = cfg.AutoFree
}

// WarmPool pre-populates the error pool with count instances.
// Improves performance by reducing initial allocations.
// No-op if pooling is disabled.
// Example:
//
//	errors.WarmPool(1000)
func WarmPool(count int) {
	if currentConfig.disablePooling {
		return
	}
	for i := 0; i < count; i++ {
		e := &Error{
			smallContext: [contextSize]contextItem{},
			stack:        nil,
		}
		errorPool.Put(e)
		stackPool.Put(make([]uintptr, 0, currentConfig.stackDepth))
	}
}

// WarmStackPool pre-populates the stack pool with count slices.
// Improves performance for stack-intensive operations.
// No-op if pooling is disabled.
// Example:
//
//	errors.WarmStackPool(500)
func WarmStackPool(count int) {
	if currentConfig.disablePooling {
		return
	}
	for i := 0; i < count; i++ {
		stackPool.Put(make([]uintptr, 0, currentConfig.stackDepth))
	}
}

// FmtErrorCheck safely formats a string using fmt.Sprintf, catching panics.
// Returns the formatted string and any error encountered.
// Internal use by Newf to validate format strings.
// Example:
//
//	result, err := FmtErrorCheck("value: %s", "test")
func FmtErrorCheck(format string, args ...interface{}) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("panic during formatting: %v", r)
			}
		}
	}()
	result = fmt.Sprintf(format, args...)
	return result, nil
}
