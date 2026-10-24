# errors — production-grade error handling for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/olekukonko/errors.svg)](https://pkg.go.dev/github.com/olekukonko/errors)
[![Go Report Card](https://goreportcard.com/badge/github.com/olekukonko/errors)](https://goreportcard.com/report/github.com/olekukonko/errors)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go 1.21+](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)

A feature-complete error handling library for Go. Fully compatible with `errors.Is`, `errors.As`, and `errors.Unwrap`. Optimised for high-throughput systems with object pooling, hybrid context storage, and inlining-immune stack capture.

---

## Contents

- [Installation](#installation)
- [Package overview](#package-overview)
- [Core — `errors`](#core--errors)
  - [Creating errors](#creating-errors)
  - [Stack traces](#stack-traces)
  - [Context](#context)
  - [Wrapping and chaining](#wrapping-and-chaining)
  - [Sentinel errors](#sentinel-errors)
  - [Type assertions — Is / As](#type-assertions--is--as)
  - [Multi-error aggregation](#multi-error-aggregation)
  - [Retry](#retry)
  - [Chain execution](#chain-execution)
  - [Channel utilities and streaming](#channel-utilities-and-streaming)
  - [HTTP helpers](#http-helpers)
  - [Concurrent group](#concurrent-group)
  - [Inspect](#inspect)
  - [slog integration](#slog-integration)
  - [Pool management](#pool-management)
- [Management — `errmgr`](#management--errmgr)
- [Performance](#performance)
- [Migration guide](#migration-guide)
- [FAQ](#faq)

---

## Installation

```bash
go get github.com/olekukonko/errors@latest
```

Requires Go 1.21 or later.

---

## Package overview

| Package | Purpose |
|---|---|
| `errors` | Core error type, wrapping, context, stack traces, retry, chain, multi-error, channel utilities |
| `errmgr` | Parameterised error templates, occurrence monitoring, threshold alerting |

---

## Core — `errors`

### Creating errors

```go
// Fast — no stack trace, 0 allocations with pooling
err := errors.New("connection failed")

// Formatted — full fmt verb support including %w
err := errors.Newf("user %s not found", "alice")
err := errors.Errorf("query failed: %w", cause) // alias of Newf

// With stack trace
err := errors.Trace("critical issue")
err := errors.Tracef("query %s failed: %w", query, cause)

// Named — useful for sentinel-style matching
err := errors.Named("AuthError")

// Standard library compatible
err := errors.Std("connection failed")   // returns plain error
err := errors.Stdf("error %s", "detail") // formatted plain error
```

### Stack traces

```go
// Capture at creation
err := errors.Trace("critical issue")

// Add to an existing error
err = err.WithStack()

// Read frames
for _, frame := range err.Stack() {
    fmt.Println(frame) // "main.go:42 main.main"
}

// Lightweight version (file:line only, no function names)
for _, frame := range err.FastStack() {
    fmt.Println(frame)
}
```

Stack capture is immune to compiler inlining — frames are collected from
the physical call stack and trimmed by slice arithmetic, not by skip count.

### Context

```go
err := errors.New("processing failed").
    With("user_id", "123").
    With("attempt", 3).
    With("retryable", true)

// Read back
ctx := errors.Context(err) // map[user_id:123 attempt:3 retryable:true]

// Check for a key
if err.HasContextKey("user_id") { ... }

// Variadic bulk attach
err.With("k1", v1, "k2", v2)

// Semantic helpers
err.WithCode(500)
err.WithCategory("network")
err.WithTimeout()
err.WithRetryable()
```

The first four context items are stored in a fixed-size array (no allocation).
Items beyond four spill to a map.

### Wrapping and chaining

```go
lowErr  := errors.New("connection timeout").With("server", "db01")
bizErr  := errors.New("failed to load user").Wrap(lowErr)
apiErr  := errors.Wrapf(bizErr, "request failed: %w", bizErr)

// Traverse
for i, e := range errors.UnwrapAll(apiErr) {
    fmt.Printf("%d. %s\n", i+1, e)
}
// 1. request failed: ...
// 2. failed to load user
// 3. connection timeout
```

### Sentinel errors

`Const` creates a stable, pointer-comparable sentinel safe for package-level variables.

```go
var (
    ErrNotFound  = errors.Const("not_found",  "resource not found")
    ErrForbidden = errors.Const("forbidden",  "access denied")
)

// Match anywhere in a chain
if errors.Is(err, ErrNotFound) { ... }

// Add call-site context without losing the sentinel
err := ErrNotFound.With("user 42 not found")
errors.Is(err, ErrNotFound) // true — sentinel is the cause

// JSON and slog work automatically
b, _ := json.Marshal(ErrNotFound)   // {"error":"resource not found","code":"not_found"}
slog.Error("lookup failed", "err", ErrNotFound)
```

> **`Const` vs `errmgr.Define`**
> `errors.Const` — static comparable value for `errors.Is` matching.
> `errmgr.Define` — parameterised factory that creates new `*Error` instances from a format template.

### Type assertions — Is / As

```go
// Is — checks identity or name match
err := errors.Named("AuthError")
wrapped := errors.Wrapf(err, "login failed")
errors.Is(wrapped, err) // true

// As — extract the first matching *Error from the chain
var target *errors.Error
if errors.As(wrapped, &target) {
    fmt.Println(target.Name()) // "AuthError"
}

// Generic helpers (Go 1.18+)
if e, ok := errors.AsType[*MyError](err); ok { ... }
if errors.IsType[*MyError](err) { ... }

found, ok := errors.FindType(err, func(e *MyError) bool {
    return e.Code() == 404
})

codes := errors.Map(err, func(e *MyError) int { return e.Code() })
errors.Filter[*MyError](err)       // []  *MyError from chain
errors.FirstOfType[*MyError](err)  // first *MyError
```

> **`Is()` string-equality note** — `(*Error).Is` falls back to string comparison as a convenience for matching stdlib errors by message. For strict identity matching use `Const()`.

### Multi-error aggregation

```go
// Basic
m := errors.NewMultiError()
m.Add(errors.New("name required"))
m.Add(errors.New("email invalid"))
fmt.Println(m.Count()) // 2

// With limits and sampling
m := errors.NewMultiError(
    errors.WithLimit(100),
    errors.WithSampling(10), // 10% sample rate
)

// Custom formatter
m := errors.NewMultiError(
    errors.WithFormatter(func(errs []error) string {
        return fmt.Sprintf("%d errors", len(errs))
    }),
)

// Inspect
m.First()   // first error
m.Last()    // last error
m.Errors()  // []error snapshot
m.Has()     // bool
m.Single()  // nil | first error | *MultiError

// Filter
networkErrs := m.Filter(func(e error) bool {
    return strings.Contains(e.Error(), "network")
})

// Merge two MultiErrors
m.Merge(other)

// Join is a convenience that collapses errors to *MultiError or nil
err := errors.Join(err1, err2, err3)
```

### Retry

```go
retry := errors.NewRetry(
    errors.WithMaxAttempts(5),
    errors.WithDelay(200*time.Millisecond),
    errors.WithMaxDelay(2*time.Second),
    errors.WithJitter(true),
    errors.WithBackoff(errors.ExponentialBackoff{}),
    errors.WithRetryIf(errors.IsRetryable),
    errors.WithOnRetry(func(attempt int, err error) {
        log.Printf("attempt %d: %v", attempt, err)
    }),
)

err := retry.Execute(func() error {
    return callExternalService()
})

// Generic version — preserves return value
result, err := errors.ExecuteReply[string](retry, func() (string, error) {
    return fetchData()
})

// Context-aware
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
retry2 := retry.Transform(errors.WithContext(ctx))
err = retry2.Execute(fn)

// Backoff strategies
errors.ConstantBackoff{}
errors.LinearBackoff{}
errors.ExponentialBackoff{}
```

### Chain execution

Sequential steps with per-step retry, timeout, tagging, and optional steps.

```go
chain := errors.NewChain(
    errors.ChainWithTimeout(10*time.Second),
    errors.ChainWithLogHandler(slog.Default().Handler()),
).
    Step(validateInput).Tag("validation").
    Step(verifyKYC).Tag("kyc").
    Step(processPayment).Tag("billing").Code(402).
        Retry(3, 100*time.Millisecond, errors.WithRetryIf(errors.IsRetryable)).
    Step(sendNotification).Tag("notification").Optional()

if err := chain.Run(); err != nil {
    errors.Inspect(err, os.Stderr)
}

// Run all steps, collect every error
if err := chain.RunAll(); err != nil {
    errors.Inspect(err, os.Stderr)
}
```

`StepCtx` passes the chain-level context (with its deadline) to the step, so
blocking calls like HTTP or database queries respect the chain timeout:

```go
chain.StepCtx(func(ctx context.Context) error {
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    _, err := http.DefaultClient.Do(req)
    return err
})
```

### Channel utilities and streaming

#### `<-chan error` utilities

These compose with the standard Go `(chan T, chan error)` idiom rather than replacing it.

```go
// Drain — block until channel closes, collect into *MultiError
err := errors.Drain(errs)

// First — return first non-nil error; ctx for deadline only, caller owns cancel
err := errors.First(ctx, errs)
if err != nil {
    cancel() // caller decides to stop siblings
}

// Collect — bounded sample; wraps ErrLimitReached when n is hit
err := errors.Collect(ctx, errs, 10)
if errors.Is(err, errors.ErrLimitReached) {
    log.Warn("more than 10 errors — some dropped")
}

// Fan — merge multiple error channels; caller must drain or cancel to avoid leak
merged := errors.Fan(ctx, validateErrs, enrichErrs)
for err := range merged {
    log.Println(err)
}
```

#### Stream — concurrent item processing

```go
// Process items concurrently, collect all errors
s := errors.NewStream(ctx, urls, func(url string) error {
    return fetch(url)
}, 8) // 8 workers; omit for len(items) workers

// Option A — block until done
if err := s.Wait(); err != nil {
    errors.Inspect(err, os.Stderr)
}

// Option B — process errors as they arrive
s.Each(func(err error) {
    log.Println(err)
})

// Stop early (drains channel to avoid goroutine leak)
s.Stop()
```

`Wait` and `Each` are mutually exclusive. Calling either a second time panics immediately.

### HTTP helpers

```go
// Resolve HTTP status from an *Error's code
status := errors.HTTPStatusCode(err, http.StatusInternalServerError)

// Write HTTP error response
errors.HTTPError(w, err) // plain text, status from err.Code()

// With options
errors.HTTPError(w, err,
    errors.WithFallbackCode(http.StatusBadGateway),
    errors.WithBody(false),        // header only
    errors.WithBodyFunc(func(e error) string {
        return fmt.Sprintf(`{"error":%q}`, e.Error())
    }),
)
```

### Concurrent group

`Group` collects all errors from concurrent goroutines — unlike `errgroup` which stops at the first.

```go
g := errors.NewGroup()

g.Go(func() error { return validateUser(id) })
g.Go(func() error { return validatePerms(id) })

if err := g.Wait(); err != nil {
    // err is *MultiError containing every failure
    errors.Inspect(err, os.Stderr)
}

// Context-aware
g := errors.NewGroup(
    errors.GroupWithContext(ctx, true), // cancelOnFirst=true
    errors.GroupWithLimit(50),
)

g.GoCtx(func(ctx context.Context) error {
    return longRunningCheck(ctx)
})

_ = g.Wait()
```

### Inspect

```go
// Default — writes to os.Stderr
errors.Inspect(err)

// Targeted output
var buf bytes.Buffer
errors.Inspect(err, &buf)

// Multiple destinations
errors.Inspect(err, os.Stderr, logFile)

// Options
errors.Inspect(err, os.Stderr,
    errors.WithStackFrames(5),
    errors.WithMaxDepth(20),
)

// *Error-specific convenience
errors.InspectError(err, os.Stderr)
```

`Inspect` handles `*Error`, `*MultiError`, and any stdlib error. It writes
to the supplied `io.Writer` values (merged via `io.MultiWriter`) and never
touches stdout.

### slog integration

Both `*Error` and `*Sentinel` implement `slog.LogValuer`:

```go
slog.Error("request failed", "err", err)
// produces structured group: err.message, err.name, err.code, err.category, err.context, err.cause

slog.Error("lookup failed", "err", errors.ErrNotFound)
// produces: err.error="resource not found", err.code="not_found"
```

### Pool management

```go
// Pre-warm (called automatically at init with 100 instances)
errors.WarmPool(1000)
errors.WarmStackPool(500)

// Tune global config
errors.Configure(errors.Config{
    StackDepth:     32,
    ContextSize:    4,
    DisablePooling: false,
    FilterInternal: true,
    AutoFree:       false, // opt-in GC-based pool return
})

// Explicit pool return (preferred)
err := errors.New("temp")
defer err.Free()

// Copy without affecting original
copied := err.Copy().With("extra", "data")

// Transform (non-destructive)
enriched := errors.Transform(err, func(e *errors.Error) {
    e.WithCode(500).With("env", "prod").WithStack()
})
```

---

## Management — `errmgr`

### Parameterised error templates

```go
// Define a reusable template
var ErrDBQuery = errmgr.Define("DBQuery", "database query failed: %s")

// Instantiate with arguments
err := ErrDBQuery("SELECT timed out")
fmt.Println(err)            // "database query failed: SELECT timed out"
fmt.Println(err.Category()) // "database"
```

### Predefined errors

```go
err := errmgr.ErrNotFound
fmt.Println(err.Code()) // 404

err := errmgr.ErrDBQuery("SELECT failed")
```

### Threshold monitoring

```go
netErr := errmgr.Define("NetError", "network issue: %s")
monitor := errmgr.NewMonitor("NetError")
errmgr.SetThreshold("NetError", 3)
defer monitor.Close()

go func() {
    for alert := range monitor.Alerts() {
        fmt.Printf("alert: %s (count: %d)\n", alert, alert.Count())
    }
}()

err := netErr("timeout")
err.Free()
```

---

Key design decisions:

- **Pool** — `New` and `Wrap` reuse `*Error` instances from `sync.Pool` (12 ns/op, 0 allocs).
- **Hybrid context** — up to 4 key-value pairs in a fixed array; overflow to map. Avoids heap allocation for the common case.
- **Stack capture** — `captureStack` is inlining-immune: it always starts from `runtime.Callers` frame 1 and trims by array slicing, so the compiler's inlining decisions never corrupt the skip count.
- **Pool capacity preservation** — the pool buffer is trimmed in-place (`copy(buf, buf[trimmed:n])`), not re-allocated. Prevents progressive capacity shrinkage under repeated `Free()` cycles.
- **`MarshalJSON`** — bytes are copied out of the pool buffer before returning it, eliminating the race between concurrent JSON serialisations.
- **`With()`** — the mutex is acquired once at entry, eliminating the TOCTOU race in the former optimistic read-then-lock path.

---

## Migration guide

### From standard library

```go
// Before
err := fmt.Errorf("user %s not found: %w", username, cause)

// After — same output, plus context, code, and chain traversal
err := errors.Newf("user %s not found: %w", username, cause).
    With("username", username).
    WithCode(404)
```

### From `pkg/errors`

```go
// Before
err := pkgerrors.Wrap(cause, "operation failed")

// After
err := errors.New("operation failed").Wrap(cause).WithStack()
```

### Stdlib `errors.Is` / `errors.As` compatibility

```go
// Fully compatible — no changes needed
if errors.Is(err, io.EOF) { ... }

var target *errors.Error
if errors.As(err, &target) {
    fmt.Println(target.Name())
}
```

---

## FAQ

**When should I use `Const` vs `Named`?**
`Const` — package-level sentinel for `errors.Is` matching. Returns the same pointer every call, so pointer equality works. `Named` — creates a new `*Error` instance each call; useful for structured errors with context but not for `==` comparison.

**When should I use `Const` vs `errmgr.Define`?**
`errors.Const("not_found", "resource not found")` creates a static sentinel. `errmgr.Define("DBQuery", "query failed: %s")` creates a parameterised factory — you call it with arguments to produce a new `*Error` each time.

**When should I call `Free()`?**
In hot paths where the error is short-lived and you want to return it to the pool immediately. For most application code, letting the GC handle it is fine. If `AutoFree` is enabled in `Config`, the GC returns the error automatically — but `defer err.Free()` is more predictable.

**Why does `First` not cancel the context?**
`context.Context` is immutable — only `context.WithCancel` produces a cancellable context. `First` accepts `ctx` for deadline support only. The pattern is: call `First`, then call `cancel()` yourself if you want to stop siblings.

**Why do `Each` and `Wait` on `Stream` panic on second call?**
Consuming the same channel twice silently splits errors between two callers. The panic surfaces the bug immediately rather than letting it produce subtly wrong results in production.

**How do I debug a deep error chain?**
```go
errors.Inspect(err, os.Stderr, errors.WithMaxDepth(30), errors.WithStackFrames(10))
```

**How do I write to both stderr and a log file?**
```go
errors.Inspect(err, os.Stderr, logFile) // io.MultiWriter internally
```

---

## Contributing

Fork → branch → commit → PR. Please include tests for new behaviour and run `go test -count=10 -race ./...` before opening a PR.

## License

MIT — see [LICENSE](LICENSE).