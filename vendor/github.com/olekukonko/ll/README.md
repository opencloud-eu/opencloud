# ll - A Modern Structured Logging Library for Go

`ll` is a production-ready logging library designed for Go applications requiring:
- **Hierarchical namespaces** for fine-grained log control
- **Structured logging** with rich metadata
- **Middleware pipeline** for customizable log processing
- **Conditional logging** to optimize performance
- **Multiple output formats** (text, colorized, JSON, slog)

## Installation

```bash
go get github.com/olekukonko/ll
```

## Getting Started

`ll` provides a simple yet powerful API for logging. Below is a basic example to get started:

```go
package main

import (
    "github.com/olekukonko/ll"
    "github.com/olekukonko/ll/lh"
    "os"
)

func main() {
    // Create a logger with namespace "app"
    logger := ll.New("app").Enable().Handler(lh.NewTextHandler(os.Stdout))

    // Log a simple message
    logger.Info("Application started") // Output: [app] INFO: Application started

    // Add structured fields
    logger.Fields("user", "alice").Info("User logged in") // Output: [app] INFO: User logged in [user=alice]

    // Conditional logging
    debugMode := false
    logger.If(debugMode).Debug("Debug info") // No output (debugMode is false)
}
```

## Core Features

### 1. Hierarchical Namespaces

Namespaces allow you to organize logs hierarchically, enabling precise control over which parts of your application produce logs. This is ideal for large systems with multiple components.

**Advantages of Namespaces**:
- **Granular Control**: Enable or disable logging for specific subsystems (e.g., "app/db" vs. "app/api").
- **Hierarchical Organization**: Group related logs under parent namespaces (e.g., "app/db/queries").
- **Scalability**: Manage log volume in complex applications by selectively enabling namespaces.
- **Readability**: Clear namespace paths improve log traceability.

**Basic Example**:
```go
logger := ll.New("app").Enable().Handler(lh.NewTextHandler(os.Stdout))

// Create child loggers
dbLogger := logger.Namespace("db")
apiLogger := logger.Namespace("api").Style(lx.NestedPath)

// Enable specific namespaces
logger.NamespaceEnable("app/db")    // Enable DB logs
logger.NamespaceDisable("app/api")  // Disable API logs

dbLogger.Info("Query executed")     // Output: [app/db] INFO: Query executed
apiLogger.Info("Request received")  // No output
```

### 2. Middleware Pipeline

The middleware pipeline allows you to process logs before they are output, using an error-based rejection mechanism (non-nil errors stop logging). This enables custom filtering, transformation, or enrichment.

**Basic Example**:
```go
logger := ll.New("app").Enable().Handler(lh.NewTextHandler(os.Stdout))

// Add middleware to enrich logs
logger.Use(ll.Middle(func(e *lx.Entry) error {
    if e.Fields == nil {
        e.Fields = make(map[string]interface{})
    }
    e.Fields["app"] = "myapp"
    return nil
}))

// Filter low-level logs
logger.Use(ll.Middle(func(e *lx.Entry) error {
    if e.Level < lx.LevelWarn {
        return fmt.Errorf("level too low")
    }
    return nil
}))

logger.Info("Ignored") // No output (filtered by middleware)
logger.Warn("Warning") // Output: [app] WARN: Warning [app=myapp]
```

### 3. Conditional Logging

Conditional logging skips expensive operations when conditions are false, improving performance in production environments.

**Basic Example**:
```go
logger := ll.New("app").Enable().Handler(lh.NewTextHandler(os.Stdout))

// Log only if condition is true
featureEnabled := true
logger.If(featureEnabled).Fields("action", "update").Info("Feature used") // Output: [app] INFO: Feature used [action=update]
logger.If(false).Info("Ignored") // No output, no processing
```

### 4. Structured Logging

Structured logging adds key-value metadata to logs, making them machine-readable and easier to query.

**Basic Example**:
```go
logger := ll.New("app").Enable().Handler(lh.NewTextHandler(os.Stdout))

// Add fields with variadic pairs
logger.Fields("user", "bob", "status", 200).Info("Request completed") // Output: [app] INFO: Request completed [user=bob status=200]

// Add fields from a map
logger.Field(map[string]interface{}{"method": "GET"}).Info("Request") // Output: [app] INFO: Request [method=GET]
```

### 5. Multiple Output Formats

`ll` supports various output formats, including human-readable text, colorized logs, JSON, and compatibility with Go’s `slog`.

**Basic Example**:
```go
logger := ll.New("app").Enable()

// Text output
logger.Handler(lh.NewTextHandler(os.Stdout))
logger.Info("Text log") // Output: [app] INFO: Text log

// JSON output
logger.Handler(lh.NewJSONHandler(os.Stdout, time.RFC3339Nano))
logger.Info("JSON log") // Output: {"timestamp":"...","level":"INFO","message":"JSON log","namespace":"app"}
```


Here's a clear, well-structured documentation section for the debugging features that matches your package's context:

---

### 6. Logging for Debugging

The `ll` package provides specialized debugging utilities that go beyond standard logging:

#### Core Debugging Methods

1. **`Dbg()` - Contextual Inspection**
```go
package main

import (
  "github.com/olekukonko/ll"
)

func main() {
  x := 42
  user := struct{ Name string }{"Alice"}

  ll.Dbg(x)    // [file.go:123] x = 42
  ll.Dbg(user) // [file.go:124] user = {Name:Alice}
}
```
- Shows file/line context
- Preserves variable names
- Handles all Go types

2. **`Dump()` - Binary Inspection**
```go
package main

import (
  "github.com/olekukonko/ll"
  "github.com/olekukonko/ll/lh"
  "os"
)

func main() {
  ll.Handler(lh.NewColorizedHandler(os.Stdout))

  ll.Dump("hello\nworld")

  f, _ := os.Open(os.Args[1])
  ll.Dump(f)
}
```
- Hex/ASCII view like `hexdump -C`
- Optimized for strings/bytes
- Fallback to JSON for complex types

![dump](_example/dump.png "dump")


3. **`Stack()` - Stack Inspection**
```go
package main

import (
  "github.com/olekukonko/ll"
  "github.com/olekukonko/ll/lh"
  "os"
)

func main() {
  ll.Handler(lh.NewColorizedHandler(os.Stdout))
  ll.Stack("hello")
}

```
![stack](_example/stack.png "stack")



#### Advanced Features

3. **Performance Tracking**
```go
defer ll.Measure("database query")() // Logs duration on return

// Or explicitly:
start := ll.Now()
ll.Benchmark(start, "operation") // Logs elapsed time
```

##### Performance Notes

- `Dbg()` calls are compile-time disabled when not enabled
- `Dump()` has optimized paths for:
    - Primitive types (direct binary encoding)
    - Strings/bytes (zero-copy)
    - Structs (JSON fallback)



## Real-world Use Case

Here’s a practical example of using `ll` in a web server:

```go
package main

import (
    "github.com/olekukonko/ll"
    "github.com/olekukonko/ll/lh"
    "net/http"
    "os"
    "time"
)

func main() {
    logger := ll.New("server").Enable().Handler(lh.NewTextHandler(os.Stdout))

    // Create a child logger for HTTP requests
    httpLogger := logger.Namespace("http").Style(lx.NestedPath)

    // Add middleware to include request ID
    httpLogger.Use(ll.Middle(func(e *lx.Entry) error {
        if e.Fields == nil {
            e.Fields = make(map[string]interface{})
        }
        e.Fields["request_id"] = "req-" + time.Now().String()
        return nil
    }))

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        httpLogger.Fields("method", r.Method, "path", r.URL.Path).Info("Request received")
        w.Write([]byte("Hello, world!"))
        httpLogger.Fields("duration_ms", time.Since(start).Milliseconds()).Info("Request completed")
    })

    logger.Info("Starting server on :8080")
    http.ListenAndServe(":8080", nil)
}
```

**Sample Output**:
```
[server] INFO: Starting server on :8080
[server]→[http]: INFO: Request received [method=GET path=/ request_id=req-...]
[server]→[http]: INFO: Request completed [duration_ms=1 request_id=req-...]
```

## Why Choose `ll`?

1. **Granular Namespace Control**: Enable or disable specific subsystems (e.g., "app/db") for precise log management.
2. **Performance Optimization**: Conditional logging (`If`) skips expensive computations when disabled.
3. **Extensible Middleware**: Transform or filter logs with error-based rejection (non-nil errors stop logging).
4. **Structured Logging**: Add key-value metadata for machine-readable logs.
5. **Flexible Outputs**: Support for text, JSON, colorized, and slog handlers.
6. **Thread-safe**: Built for concurrent use with mutex-protected state.
7. **Robust Testing**: Comprehensive test suite with recent fixes (e.g., sampling reliability) and extensive documentation.

## Advantages of Namespaces

Namespaces are a standout feature of `ll`, offering:
- **Selective Logging**: Enable logs for specific components (e.g., "app/db") while disabling others (e.g., "app/api"), reducing noise in large systems.
- **Hierarchical Filtering**: Control entire subsystems (e.g., disable "app" to suppress all child logs like "app/db" and "app/api").
- **Improved Debugging**: Trace logs to specific parts of the application (e.g., "app/db/queries") for faster issue identification.
- **Production Scalability**: Disable verbose namespaces in production to manage log volume without code changes.

## Benchmarks

Compared to standard library `log` and `slog`:
- 30% faster than `slog` for disabled logs due to efficient conditional checks.
- 2x faster than `log` for structured logging with minimal allocations.
- Optimized namespace caching and middleware processing reduce overhead.

See `ll_bench_test.go` for detailed benchmarks on namespace creation, cloning, and field building.

## Testing and Stability

The library includes a robust test suite (`ll_test.go`) covering:
- Logger configuration, namespaces, and conditional logging.
- Middleware, rate limiting, and sampling (fixed for reliable behavior at 0.0 and 1.0 rates).
- Handler output formats and error handling.

Recent improvements include:
- Fixed sampling middleware to ensure correct filtering (issue resolved in `TestSampling`).
- Enhanced documentation with detailed comments in `conditional.go`, `field.go`, `global.go`, `ll.go`, `lx.go`, and `ns.go`.
