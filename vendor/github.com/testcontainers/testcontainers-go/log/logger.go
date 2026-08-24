package log

import (
	"log"
	"os"
	"strings"
	"testing"
)

// Validate our types implement the required interfaces.
var (
	_ Logger = (*log.Logger)(nil)
	_ Logger = (*noopLogger)(nil)
	_ Logger = (*testLogger)(nil)
)

// Logger defines the Logger interface.
type Logger interface {
	Printf(format string, v ...any)
}

// defaultLogger would print available information to stderr
var defaultLogger Logger = log.New(os.Stderr, "", log.LstdFlags)

func NewNoopLogger() Logger {
	return &noopLogger{}
}

func init() {
	// Disable default logger in testing mode if explicitly disabled via -test.v=false.
	if testing.Testing() {
		// Disable logging if explicitly disabled via -test.v=false
		for _, arg := range os.Args {
			if strings.EqualFold(arg, "-test.v=false") {
				defaultLogger = NewNoopLogger()
				break
			}
		}
	}
}

// Default returns the default Logger instance.
func Default() Logger {
	return defaultLogger
}

// SetDefault sets the default Logger instance.
func SetDefault(logger Logger) {
	defaultLogger = logger
}

func Printf(format string, v ...any) {
	defaultLogger.Printf(format, v...)
}

type noopLogger struct{}

// Printf implements Logging.
func (n noopLogger) Printf(_ string, _ ...any) {
	// NOOP
}

// TestLogger returns a Logging implementation for testing.TB
// This way logs from testcontainers are part of the test output of a test suite or test case.
func TestLogger(tb testing.TB) Logger {
	tb.Helper()
	return testLogger{TB: tb}
}

type testLogger struct {
	testing.TB
}

// Printf implements Logging.
func (t testLogger) Printf(format string, v ...any) {
	t.Helper()
	t.Logf(format, v...)
}
