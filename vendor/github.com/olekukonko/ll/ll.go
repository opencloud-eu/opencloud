package ll

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/olekukonko/ll/lh"
	"github.com/olekukonko/ll/lx"
	"io"
	"math"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// defaultStore is the global namespace store for enable/disable states.
// It is shared across all Logger instances to manage namespace hierarchy consistently.
// Thread-safe via the lx.Namespace struct’s sync.Map.
var defaultStore = &lx.Namespace{}

// systemActive indicates if the global logging system is active.
// Defaults to true, meaning logging is active unless explicitly shut down.
// Or, default to false and require an explicit ll.Start(). Let's default to true for less surprise.
var systemActive int32 = 1 // 1 for true, 0 for false (for atomic operations)

// Logger is the core structure for logging, managing configuration and behavior.
// It encapsulates all logging state, including enablement, log level, namespaces,
// context fields, output style, handler, middleware, and formatting options.
// Thread-safe with a read-write mutex to protect concurrent access to fields.
type Logger struct {
	mu              sync.RWMutex           // Protects concurrent access to fields
	enabled         bool                   // Whether logging is enabled
	level           lx.LevelType           // Minimum level for logging (Debug, Info, Warn, Error)
	namespaces      *lx.Namespace          // Stores namespace enable/disable states
	currentPath     string                 // Current namespace path (e.g., "parent/child")
	context         map[string]interface{} // Contextual fields added to all logs
	style           lx.StyleType           // Namespace formatting style (FlatPath or NestedPath)
	handler         lx.Handler             // Output handler for logs (e.g., text, JSON)
	middleware      []Middleware           // Middleware functions to process log entries
	prefix          string                 // Prefix prepended to log messages
	indent          int                    // Number of double spaces to indent messages
	stackBufferSize int                    // Buffer size for stack trace capture
	separator       string
	entries         atomic.Int64
}

// New creates a new logger instance with the specified namespace and optional configurations.
// It initializes the logger with default settings: disabled, Debug level, flat namespace style,
// a text handler writing to os.Stdout, and an empty middleware chain. Options can override
// defaults (e.g., WithHandler, WithLevel). Thread-safe via mutex-protected methods.
// Example:
//
//	logger := New("app", WithHandler(lh.NewTextHandler(os.Stdout))).Enable()
//	logger.Info("Starting application") // Output: [app] INFO: Starting application
//
//	logger := New("test", WithHandler(NewMemoryHandler())).Enable()
//	logger.Info("Test message")
//	entries := logger.handler.(*MemoryHandler).Entries()
func New(namespace string, opts ...Option) *Logger {
	logger := &Logger{
		enabled:         lx.DefaultEnabled,
		level:           lx.LevelDebug,
		namespaces:      defaultStore,
		currentPath:     namespace,
		context:         make(map[string]interface{}),
		style:           lx.FlatPath,
		handler:         lh.NewTextHandler(os.Stdout),
		middleware:      make([]Middleware, 0),
		stackBufferSize: 4096,
		separator:       lx.Slash,
	}

	// Apply options
	for _, opt := range opts {
		opt(logger)
	}

	return logger
}

// Clone creates a new logger with the same configuration and namespace as the parent.
// It copies all settings (enabled, level, namespaces, etc.) but provides a fresh context map
// to allow independent field additions without affecting the parent. Thread-safe with read lock.
// Example:
//
//	logger := New("app").Enable().Context(map[string]interface{}{"k": "v"})
//	clone := logger.Clone()
//	clone.Info("Cloned") // Output: [app] INFO: Cloned [k=v]
func (l *Logger) Clone() *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return &Logger{
		enabled:         l.enabled,
		level:           l.level,
		namespaces:      l.namespaces,
		currentPath:     l.currentPath,
		context:         make(map[string]interface{}),
		style:           l.style,
		handler:         l.handler,
		middleware:      l.middleware,
		prefix:          l.prefix,
		indent:          l.indent,
		stackBufferSize: l.stackBufferSize,
		separator:       lx.Slash,
	}
}

// Context creates a new logger with additional contextual fields.
// It preserves existing context fields and adds new ones, returning a new logger instance
// to avoid mutating the parent. Thread-safe with write lock. Useful for adding persistent
// metadata to all logs.
// Example:
//
//	logger := New("app").Enable()
//	logger = logger.Context(map[string]interface{}{"user": "alice"})
//	logger.Info("Action") // Output: [app] INFO: Action [user=alice]
func (l *Logger) Context(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Create a new logger with the same configuration
	newLogger := &Logger{
		enabled:         l.enabled,
		level:           l.level,
		namespaces:      l.namespaces,
		currentPath:     l.currentPath,
		context:         make(map[string]interface{}),
		style:           l.style,
		handler:         l.handler,
		middleware:      l.middleware,
		prefix:          l.prefix,
		indent:          l.indent,
		stackBufferSize: l.stackBufferSize,
		separator:       lx.Slash,
	}

	// Copy parent’s context fields
	for k, v := range l.context {
		newLogger.context[k] = v
	}
	// Add new fields
	for k, v := range fields {
		newLogger.context[k] = v
	}

	return newLogger
}

// AddContext adds a key-value pair to the logger's context, modifying the existing logger.
// Unlike Context, it mutates the logger’s context directly. Thread-safe with write lock.
// Example:
//
//	logger := New("app").Enable()
//	logger.AddContext("user", "alice")
//	logger.Info("Action") // Output: [app] INFO: Action [user=alice]
func (l *Logger) AddContext(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Initialize context if nil
	if l.context == nil {
		l.context = make(map[string]interface{})
	}
	l.context[key] = value
	return l
}

// Enabled returns whether the logger is enabled for logging.
// It provides thread-safe read access to the enabled field using a read lock.
// Example:
//
//	logger := New("app").Enable()
//	if logger.Enabled() {
//	    logger.Info("Logging is enabled") // Output: [app] INFO: Logging is enabled
//	}
func (l *Logger) Enabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.enabled
}

// GetLevel returns the minimum log level for the logger.
// It provides thread-safe read access to the level field using a read lock.
// Example:
//
//	logger := New("app").Level(lx.LevelWarn)
//	if logger.GetLevel() == lx.LevelWarn {
//	    logger.Warn("Warning level set") // Output: [app] WARN: Warning level set
//	}
func (l *Logger) GetLevel() lx.LevelType {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// Prefix sets a prefix to be prepended to all log messages of the current logger.
// The prefix is applied before the message in the log output. Thread-safe with write lock.
// Returns the logger for method chaining.
// Example:
//
//	logger := New("app").Enable().Prefix("APP: ")
//	logger.Info("Started") // Output: [app] INFO: APP: Started
func (l *Logger) Prefix(prefix string) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
	return l
}

// Indent sets the indentation level for all log messages of the current logger.
// Each level adds two spaces to the log message, useful for hierarchical output.
// Thread-safe with write lock. Returns the logger for method chaining.
// Example:
//
//	logger := New("app").Enable().Indent(2)
//	logger.Info("Indented") // Output: [app] INFO:     Indented
func (l *Logger) Indent(depth int) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.indent = depth
	return l
}

// Handler sets the handler for processing log entries.
// It configures the output destination and format (e.g., text, JSON) for logs.
// Thread-safe with write lock. Returns the logger for method chaining.
// Example:
//
//	logger := New("app").Enable().Handler(lh.NewTextHandler(os.Stdout))
//	logger.Info("Log") // Output: [app] INFO: Log
func (l *Logger) Handler(handler lx.Handler) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handler = handler
	return l
}

// Level sets the minimum log level required for logging.
// Messages below the specified level are ignored. Thread-safe with write lock.
// Returns the logger for method chaining.
// Example:
//
//	logger := New("app").Enable().Level(lx.LevelWarn)
//	logger.Info("Ignored") // No output
//	logger.Warn("Logged") // Output: [app] WARN: Logged
func (l *Logger) Level(level lx.LevelType) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
	return l
}

// Enable activates logging for the current logger.
// It allows logs to be emitted if other conditions (level, namespace) are met.
// Thread-safe with write lock. Returns the logger for method chaining.
// Example:
//
//	logger := New("app").Enable()
//	logger.Info("Started") // Output: [app] INFO: Started
func (l *Logger) Enable() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = true
	return l
}

// Disable deactivates logging for the current logger.
// It suppresses all logs, regardless of level or namespace. Thread-safe with write lock.
// Returns the logger for method chaining.
// Example:
//
//	logger := New("app").Enable().Disable()
//	logger.Info("Ignored") // No output
func (l *Logger) Disable() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = false
	return l
}

// Style sets the namespace formatting style (FlatPath or NestedPath).
// FlatPath formats namespaces as [parent/child], while NestedPath uses [parent]→[child].
// Thread-safe with write lock. Returns the logger for method chaining.
// Example:
//
//	logger := New("parent/child").Enable().Style(lx.NestedPath)
//	logger.Info("Log") // Output: [parent]→[child]: INFO: Log
func (l *Logger) Style(style lx.StyleType) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.style = style
	return l
}

// Namespace creates a child logger with a sub-namespace appended to the current path.
// The child inherits the parent’s configuration but has an independent context.
// Thread-safe with read lock. Returns the new logger for further configuration or logging.
// Example:
//
//	parent := New("parent").Enable()
//	child := parent.Namespace("child")
//	child.Info("Child log") // Output: [parent/child] INFO: Child log
func (l *Logger) Namespace(name string) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Construct full namespace path
	fullPath := name
	if l.currentPath != "" {
		fullPath = l.currentPath + l.separator + name
	}

	// Create child logger with inherited configuration
	return &Logger{
		enabled:         l.enabled,
		level:           l.level,
		namespaces:      l.namespaces,
		currentPath:     fullPath,
		context:         make(map[string]interface{}),
		style:           l.style,
		handler:         l.handler,
		middleware:      l.middleware,
		prefix:          l.prefix,
		indent:          l.indent,
		stackBufferSize: l.stackBufferSize,
		separator:       l.separator,
	}
}

// NamespaceEnable enables logging for a namespace and its children.
// It sets the specified namespace path to enabled, invalidating the namespace cache to ensure
// updated state. Thread-safe via the lx.Namespace’s sync.Map. Returns the logger for chaining.
// Example:
//
//	logger := New("parent").Enable().NamespaceEnable("parent/child")
//	logger.Namespace("child").Info("Log") // Output: [parent/child] INFO: Log
func (l *Logger) NamespaceEnable(relativePath string) *Logger {
	l.mu.RLock()
	fullPath := l.joinPath(l.currentPath, relativePath)
	l.mu.RUnlock()

	// fmt.Printf("[DEBUG] NamespaceEnable: logger.currentPath=%q, relativePath=%q, fullPath SETTING to true: %q\n", l.currentPath, relativePath, fullPath) // DEBUG
	l.namespaces.Set(fullPath, true)
	return l
}

// NamespaceDisable disables logging for a namespace and its children.
// It sets the specified namespace path to disabled, invalidating the namespace cache.
// Thread-safe via the lx.Namespace’s sync.Map. Returns the logger for chaining.
// Example:
//
//	logger := New("parent").Enable().NamespaceDisable("parent/child")
//	logger.Namespace("child").Info("Ignored") // No output
func (l *Logger) NamespaceDisable(relativePath string) *Logger {
	l.mu.RLock()
	fullPath := l.joinPath(l.currentPath, relativePath)
	l.mu.RUnlock()

	// fmt.Printf("[DEBUG] NamespaceDisable: logger.currentPath=%q, relativePath=%q, fullPath SETTING to false: %q\n", l.currentPath, relativePath, fullPath) // DEBUG
	l.namespaces.Set(fullPath, false)
	return l
}

// NamespaceEnabled returns true if the specified namespace is enabled.
// It checks the namespace hierarchy, considering parent namespaces, and caches the result
// for performance. Thread-safe with read lock.
// Example:
//
//	logger := New("parent").Enable().NamespaceDisable("parent/child")
//	enabled := logger.NamespaceEnabled("parent/child") // false
func (l *Logger) NamespaceEnabled(relativePath string) bool {
	l.mu.RLock()
	fullPath := l.joinPath(l.currentPath, relativePath)
	separator := l.separator
	if separator == "" {
		separator = lx.Slash
	}
	instanceEnabled := l.enabled
	l.mu.RUnlock()

	// fmt.Printf("[DEBUG] NamespaceEnabled CHECK: logger.currentPath=%q, relativePath=%q, fullPath CHECKING: %q, instanceEnabled: %v\n", l.currentPath, relativePath, fullPath, instanceEnabled) // DEBUG
	if fullPath == "" && relativePath == "" {
		return instanceEnabled
	}

	if fullPath != "" {
		isEnabledByNSRule, isDisabledByNSRule := l.namespaces.Enabled(fullPath, separator)
		// fmt.Printf("[DEBUG] Enabled(%q) -> isEnabled:%v, isDisabled:%v\n", fullPath, isEnabledByNSRule, isDisabledByNSRule) // DEBUG

		if isDisabledByNSRule {
			return false
		}
		if isEnabledByNSRule {
			return true
		}
	}
	return instanceEnabled
}

// Use adds a middleware function to process log entries before they are handled.
// It registers the middleware and returns a Middleware handle for removal. Middleware functions
// return a non-nil error to stop the log from being emitted. Thread-safe with write lock.
// Example:
//
//	logger := New("app").Enable()
//	mw := logger.Use(ll.FuncMiddleware(func(e *lx.Entry) error {
//	    if e.Level < lx.LevelWarn {
//	        return fmt.Errorf("level too low")
//	    }
//	    return nil
//	}))
//	logger.Info("Ignored") // No output
//	mw.Remove()
//	logger.Info("Now logged") // Output: [app] INFO: Now logged
func (l *Logger) Use(fn lx.Handler) *Middleware {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Assign a unique ID to the middleware
	id := len(l.middleware) + 1
	// Append middleware to the chain
	l.middleware = append(l.middleware, Middleware{id: id, fn: fn})

	return &Middleware{
		logger: l,
		id:     id,
	}
}

// Separator is use to group namespaces and log entries.
func (l *Logger) Separator(separator string) *Logger {
	l.mu.Lock()
	l.mu.Unlock()
	l.separator = separator
	return l
}

// Remove removes middleware by the reference returned from Use.
// It delegates to the Middleware’s Remove method for thread-safe removal.
func (l *Logger) Remove(m *Middleware) {
	m.Remove()
}

// Clear removes all middleware functions.
// It resets the middleware chain to empty, ensuring no middleware is applied.
// Thread-safe with write lock. Returns the logger for chaining.
// Example:
//
//	logger := New("app").Enable().Use(someMiddleware)
//	logger.Clear()
//	logger.Info("No middleware") // Output: [app] INFO: No middleware
func (l *Logger) Clear() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.middleware = nil
	return l
}

// StackSize sets the buffer size for stack trace capture.
// It configures the maximum size of the stack trace buffer for Stack, Fatal, and Panic methods.
// Thread-safe with write lock. Returns the logger for chaining.
// Example:
//
//	logger := New("app").Enable().StackSize(65536)
//	logger.Stack("Error") // Captures up to 64KB stack trace
func (l *Logger) StackSize(size int) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	if size > 0 {
		l.stackBufferSize = size
	}
	return l
}

// Fields starts a fluent chain for adding fields using variadic key-value pairs.
// It creates a FieldBuilder to attach fields, handling non-string keys or uneven pairs by
// adding an error field. Thread-safe via the FieldBuilder’s logger.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fields("user", "alice").Info("Action") // Output: [app] INFO: Action [user=alice]
func (l *Logger) Fields(pairs ...any) *FieldBuilder {
	fb := &FieldBuilder{logger: l, fields: make(map[string]interface{})}
	for i := 0; i < len(pairs)-1; i += 2 {
		if key, ok := pairs[i].(string); ok {
			fb.fields[key] = pairs[i+1]
		} else {
			fb.fields["error"] = fmt.Errorf("non-string key in Fields: %v", pairs[i])
		}
	}
	if len(pairs)%2 != 0 {
		fb.fields["error"] = fmt.Errorf("uneven key-value pairs in Fields: [%v]", pairs[len(pairs)-1])
	}
	return fb
}

// Field starts a fluent chain for adding fields from a map.
// It creates a FieldBuilder to attach fields from a map, supporting type-safe field addition.
// Thread-safe via the FieldBuilder’s logger.
// Example:
//
//	logger := New("app").Enable()
//	logger.Field(map[string]interface{}{"user": "alice"}).Info("Action") // Output: [app] INFO: Action [user=alice]
func (l *Logger) Field(fields map[string]interface{}) *FieldBuilder {
	fb := &FieldBuilder{logger: l, fields: make(map[string]interface{})}
	for k, v := range fields {
		fb.fields[k] = v
	}
	return fb
}

// CanLog returns true if a log at the given level would be emitted.
// It considers enablement, log level, namespaces, sampling, and rate limits.
// Thread-safe via the shouldLog method.
// Example:
//
//	logger := New("app").Enable().Level(lx.LevelWarn)
//	canLog := logger.CanLog(lx.LevelInfo) // false
func (l *Logger) CanLog(level lx.LevelType) bool {
	return l.shouldLog(level)
}

// Print logs a message at Info level without format specifiers, minimizing allocations.
// It concatenates variadic arguments with spaces and delegates to the internal log method.
// Thread-safe via the log method.
// Example:
//
//	logger := New("app").Enable()
//	logger.Print("message", "value") // Output: [app] INFO: message value
func (l *Logger) Print(args ...any) {
	var builder strings.Builder
	for i, arg := range args {
		if i > 0 {
			builder.WriteString(lx.Space)
		}
		builder.WriteString(fmt.Sprint(arg))
	}

	builder.WriteString(lx.Newline)
	l.log(lx.LevelNone, lx.ClassRaw, builder.String(), nil, false)
}

// Info logs a message at Info level.
// It formats the message using the provided format string and arguments, then delegates
// to the internal log method. Thread-safe.
// Example:
//
//	logger := New("app").Enable().Style(lx.NestedPath)
//	logger.Info("Started") // Output: [app]: INFO: Started
func (l *Logger) Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	l.log(lx.LevelInfo, lx.ClassText, msg, nil, false)
}

// Measure is a benchmarking helper that measures and returns the duration of a function’s execution.
// It logs the duration at Info level with a "duration" field. Thread-safe via the Fields and log methods.
// Example:
//
//	logger := New("app").Enable()
//	duration := logger.Measure(func() { time.Sleep(time.Millisecond) })
//	// Output: [app] INFO: function executed [duration=~1ms]
func (l *Logger) Measure(fns ...func()) time.Duration {
	start := time.Now()
	for _, fn := range fns {
		fn()
	}
	duration := time.Since(start)
	l.Fields("duration", duration).Info("function executed")
	return duration
}

// Benchmark logs the duration since a start time at Info level.
// It calculates the elapsed time and logs it with "start", "end", and "duration" fields.
// Thread-safe via the Fields and log methods.
// Example:
//
//	logger := New("app").Enable()
//	start := time.Now()
//	logger.Benchmark(start) // Output: [app] INFO: benchmark [start=... end=... duration=...]
func (l *Logger) Benchmark(start time.Time) time.Duration {
	duration := time.Since(start)
	l.Fields("start", start, "end", time.Now(), "duration", duration).Info("benchmark")
	return duration
}

// Debug logs a message at Debug level.
// It formats the message and delegates to the internal log method. Thread-safe.
// Example:
//
//	logger := New("app").Enable().Level(lx.LevelDebug)
//	logger.Debug("Debugging") // Output: [app] DEBUG: Debugging
func (l *Logger) Debug(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	l.log(lx.LevelDebug, lx.ClassText, msg, nil, false)
}

// Warn logs a message at Warn level.
// It formats the message and delegates to the internal log method. Thread-safe.
// Example:
//
//	logger := New("app").Enable()
//	logger.Warn("Warning") // Output: [app] WARN: Warning
func (l *Logger) Warn(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	l.log(lx.LevelWarn, lx.ClassText, msg, nil, false)
}

// Error logs a message at Error level.
// It formats the message and delegates to the internal log method. Thread-safe.
// Example:
//
//	logger := New("app").Enable()
//	logger.Error("Error occurred") // Output: [app] ERROR: Error occurred
func (l *Logger) Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	l.log(lx.LevelError, lx.ClassText, msg, nil, false)
}

// GetHandler returns the logger's current handler implementation.
// This provides access to the underlying logging handler for customization
// or inspection. The returned handler should not be modified concurrently
// with logger operations.
//
// Example:
//
//	logger := New("app")
//	logger.GetHandler
func (l *Logger) GetHandler() lx.Handler {
	return l.handler
}

// Err adds one or more errors to the logger’s context and logs them.
// It stores non-nil errors in the "error" context field: a single error if only one is non-nil,
// or a slice of errors if multiple are non-nil. It logs the concatenated string representations
// of non-nil errors (e.g., "failed 1; failed 2") at the Error level. Returns the Logger for chaining,
// allowing further configuration or logging. Thread-safe via the logger’s mutex.
// Example:
//
//	logger := New("app").Enable()
//	err1 := errors.New("failed 1")
//	err2 := errors.New("failed 2")
//	logger.Err(err1, err2).Info("Error occurred")
//	// Output: [app] ERROR: failed 1; failed 2
//	//         [app] INFO: Error occurred [error=[failed 1 failed 2]]
func (l *Logger) Err(errs ...error) *Logger {
	l.mu.Lock()
	// Initialize context if nil
	if l.context == nil {
		l.context = make(map[string]interface{})
	}

	// Collect non-nil errors and build log message
	var nonNilErrors []error
	var builder strings.Builder
	count := 0
	for i, err := range errs {
		if err != nil {
			if i > 0 && count > 0 {
				builder.WriteString("; ")
			}
			builder.WriteString(err.Error())
			nonNilErrors = append(nonNilErrors, err)
			count++
		}
	}

	// Set error context field and log if there are non-nil errors
	if count > 0 {
		if count == 1 {
			// Store single error directly
			l.context["error"] = nonNilErrors[0]
		} else {
			// Store slice of errors
			l.context["error"] = nonNilErrors
		}
		// Log concatenated error messages at Error level
		l.log(lx.LevelError, lx.ClassText, builder.String(), nil, false)
	}
	l.mu.Unlock()

	// Return Logger for chaining
	return l
}

// Stack logs a message at Error level with a stack trace.
// It formats the message and delegates to the internal log method with a stack trace.
// Thread-safe.
// Example:
//
//	logger := New("app").Enable()
//	logger.Stack("Critical error") // Output: [app] ERROR: Critical error [stack=...]
func (l *Logger) Stack(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	l.log(lx.LevelError, lx.ClassText, msg, nil, true)
}

// Len counts the total number of values sent to handler
func (l *Logger) Len() int64 {
	return l.entries.Load()
}

// Fatal logs a message at Error level with a stack trace and exits the program.
// It constructs the message from variadic arguments, logs it with a stack trace, and
// terminates with exit code 1. Thread-safe.
// Example:
//
//	logger := New("app").Enable()
//	logger.Fatal("Fatal error") // Output: [app] ERROR: Fatal error [stack=...], then exits
func (l *Logger) Fatal(args ...any) {
	var builder strings.Builder
	for i, arg := range args {
		if i > 0 {
			builder.WriteString(lx.Space)
		}
		builder.WriteString(fmt.Sprint(arg))
	}
	l.log(lx.LevelError, lx.ClassText, builder.String(), nil, true)
	os.Exit(1)
}

// Panic logs a message at Error level with a stack trace and panics.
// It constructs the message from variadic arguments, logs it with a stack trace, and
// triggers a panic. Thread-safe.
// Example:
//
//	logger := New("app").Enable()
//	logger.Panic("Panic error") // Output: [app] ERROR: Panic error [stack=...], then panics
func (l *Logger) Panic(args ...any) {
	var builder strings.Builder
	for i, arg := range args {
		if i > 0 {
			builder.WriteString(lx.Space)
		}
		builder.WriteString(fmt.Sprint(arg))
	}
	msg := builder.String()
	l.log(lx.LevelError, lx.ClassText, msg, nil, true)
	panic(msg)
}

// Dbg logs debug information including the source file, line number, and expression value.
// It captures the calling line of code and displays both the expression and its value.
// Useful for debugging without adding temporary print statements.
// Example:
//
//	x := 42
//	logger.Dbg(x) // Output: [file.go:123] x = 42
func (l *Logger) Dbg(values ...interface{}) {
	l.dbg(2, values...)
}

func (l *Logger) dbg(skip int, values ...interface{}) {
	for _, exp := range values {
		_, file, line, ok := runtime.Caller(skip)
		if !ok {
			l.log(lx.LevelError, lx.ClassText, "Dbg: Unable to parse runtime caller", nil, false)
			return
		}

		f, err := os.Open(file)
		if err != nil {
			l.log(lx.LevelError, lx.ClassText, "Dbg: Unable to open expected file", nil, false)
			return
		}

		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanLines)
		var out string
		i := 1
		for scanner.Scan() {
			if i == line {
				v := scanner.Text()[strings.Index(scanner.Text(), "(")+1 : len(scanner.Text())-strings.Index(reverseString(scanner.Text()), ")")-1]
				out = fmt.Sprintf("[%s:%d] %s = %+v", file[len(file)-strings.Index(reverseString(file), "/"):], line, v, exp)
				break
			}
			i++
		}
		if err := scanner.Err(); err != nil {
			l.log(lx.LevelError, lx.ClassText, err.Error(), nil, false)
			return
		}
		switch exp.(type) {
		case error:
			l.log(lx.LevelError, lx.ClassText, out, nil, false)
		default:
			l.log(lx.LevelInfo, lx.ClassText, out, nil, false)
		}

		f.Close()
	}
}

// Dump displays a hex and ASCII representation of any value's binary form.
// It serializes the value using gob encoding and shows a hex/ASCII dump similar to hexdump -C.
// Useful for inspecting binary data structures.
// Example:
//
//	type Data struct { X int; Y string }
//	logger.Dump(Data{42, "test"})
func (l *Logger) Dump(values ...interface{}) {
	// Convert any value to bytes
	for _, value := range values {
		l.Info("Dumping %v (%T)", value, value)
		var by []byte
		var err error

		switch v := value.(type) {
		case []byte:
			by = v
		case string:
			by = []byte(v)
		case float32:
			buf := make([]byte, 4)
			binary.BigEndian.PutUint32(buf, math.Float32bits(v))
			by = buf
		case float64:
			buf := make([]byte, 8)
			binary.BigEndian.PutUint64(buf, math.Float64bits(v))
			by = buf
		case int, int8, int16, int32, int64:
			by = make([]byte, 8)
			binary.BigEndian.PutUint64(by, uint64(reflect.ValueOf(v).Int()))
		case uint, uint8, uint16, uint32, uint64:
			by = make([]byte, 8)
			binary.BigEndian.PutUint64(by, reflect.ValueOf(v).Uint())
		case io.Reader:
			by, err = io.ReadAll(v)
		default:
			by, err = json.Marshal(v) // Fallback to JSON
		}

		if err != nil {
			l.Error("Dump error: %v", err)
			continue
		}

		// Now dump the bytes as before
		n := len(by)
		rowcount := 0
		stop := (n / 8) * 8
		k := 0
		s := strings.Builder{}
		for i := 0; i <= stop; i += 8 {
			k++
			if i+8 < n {
				rowcount = 8
			} else {
				rowcount = min(k*8, n) % 8
			}
			s.WriteString(fmt.Sprintf("pos %02d  hex:  ", i))

			for j := 0; j < rowcount; j++ {
				s.WriteString(fmt.Sprintf("%02x  ", by[i+j]))
			}
			for j := rowcount; j < 8; j++ {
				s.WriteString(fmt.Sprintf("    "))
			}
			s.WriteString(fmt.Sprintf("  '%s'\n", viewString(by[i:(i+rowcount)])))

		}
		l.log(lx.LevelNone, lx.ClassDump, s.String(), nil, false)
	}
}

// log is the internal method for processing a log entry.
// It applies rate limiting, sampling, middleware, and context before passing to the handler.
// If a middleware returns a non-nil error, the log is stopped, ensuring precise control.
// Thread-safe with read lock for reading configuration and write lock for stack trace buffer.
// Example (internal usage):
//
//	logger := New("app").Enable()
//	logger.Info("Test") // Calls log(lx.LevelInfo, "Test", nil, false)
func (l *Logger) log(level lx.LevelType, class lx.ClassType, msg string, fields map[string]interface{}, withStack bool) {
	// Check if the log should be emitted
	if !l.shouldLog(level) {
		return
	}

	var stack []byte

	// Capture stack trace if requested
	if withStack {
		l.mu.RLock()
		buf := make([]byte, l.stackBufferSize)
		l.mu.RUnlock()
		n := runtime.Stack(buf, false)
		if fields == nil {
			fields = make(map[string]interface{})
		}
		stack = buf[:n]
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	// Apply prefix and indentation to the message
	var builder strings.Builder
	if l.indent > 0 {
		builder.WriteString(strings.Repeat(lx.DoubleSpace, l.indent))
	}
	if l.prefix != "" {
		builder.WriteString(l.prefix)
	}
	builder.WriteString(msg)
	finalMsg := builder.String()

	// Create log entry
	entry := &lx.Entry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   finalMsg,
		Namespace: l.currentPath,
		Fields:    fields,
		Style:     l.style,
		Class:     class,
		Stack:     stack,
	}

	// Add context fields, avoiding overwrites
	if len(l.context) > 0 {
		if entry.Fields == nil {
			entry.Fields = make(map[string]interface{})
		}
		for k, v := range l.context {
			if _, exists := entry.Fields[k]; !exists {
				entry.Fields[k] = v
			}
		}
	}

	// Apply middleware, stopping if any returns a non-nil error
	for _, mw := range l.middleware {
		if err := mw.fn.Handle(entry); err != nil {
			// TODO: Consider logging middleware errors for debugging
			return
		}
	}

	// Pass to handler if set
	if l.handler != nil {
		_ = l.handler.Handle(entry)
		l.entries.Add(1)
	}
}

// shouldLog determines if a log should be emitted based on enabled state, level, namespaces, sampling, and rate limits.
// It checks logger enablement, log level, and namespace hierarchy, caching namespace results for performance.
// Thread-safe with read lock.
// Example (internal usage):
//
//	logger := New("app").Enable().Level(lx.LevelWarn)
//	if logger.shouldLog(lx.LevelInfo) { // false
//	    // Log would be skipped
//	}
func (l *Logger) shouldLog(level lx.LevelType) bool {

	if !Active() { // Assuming Active is in the 'll' package or imported
		return false
	}

	// 1. Check log level (cheap filter)
	if level < l.level {
		return false
	}

	// 2. Check namespace rules from the global store
	// Use the logger's configured separator, defaulting to lx.Slash if not set
	separator := l.separator
	if separator == "" {
		separator = lx.Slash
	}

	if l.currentPath != "" { // Only check namespace rules if a path is set
		isEnabledByNSRule, isDisabledByNSRule := l.namespaces.Enabled(l.currentPath, separator)

		if isDisabledByNSRule {
			return false // Explicitly disabled by a namespace rule
		}
		if isEnabledByNSRule {
			return true // Explicitly enabled by a namespace rule (overrides logger.enabled if it was false)
		}
		// If neither isEnabledByNSRule nor isDisabledByNSRule is true,
		// it means no explicit namespace rule applies directly to this path or its parents.
		// In this case, we fall through to check the logger's instance 'enabled' flag.
	}

	// 3. If no overriding namespace rule was found (or no path),
	//    check the logger instance's 'enabled' flag.
	if !l.enabled {
		return false
	}

	// If we reach here:
	// - Level is sufficient.
	// - EITHER:
	//    - Namespace path is explicitly enabled by a rule.
	//    - OR Namespace path has no overriding rule, and l.enabled is true.
	return true
}

// joinPath joins a base path and a relative path using the logger's separator.
// Handles cases where base or relative path might be empty.
func (l *Logger) joinPath(base, relative string) string {
	if base == "" {
		return relative
	}
	if relative == "" {
		return base
	}
	separator := l.separator
	if separator == "" {
		separator = lx.Slash // Default separator
	}
	return base + separator + relative
}

// Option defines a functional option for configuring a Logger.
type Option func(*Logger)

// WithHandler sets the handler for the logger.
func WithHandler(handler lx.Handler) Option {
	return func(l *Logger) {
		l.handler = handler
	}
}

// WithLevel sets the minimum log level for the logger.
func WithLevel(level lx.LevelType) Option {
	return func(l *Logger) {
		l.level = level
	}
}

// WithStyle sets the namespace formatting style for the logger.
func WithStyle(style lx.StyleType) Option {
	return func(l *Logger) {
		l.style = style
	}
}

func reverseString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func viewString(b []byte) string {
	r := []rune(string(b))
	for i := range r {
		if r[i] < 32 || r[i] > 126 {
			r[i] = '.'
		}
	}
	return string(r)
}
