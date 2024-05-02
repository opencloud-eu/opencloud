package runner

import (
	"os"
	"strings"
	"syscall"
	"time"
)

var (
	StopSignals = []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT}
)

// Runable represent a task that can be executed by the Runner.
// It expected to be a long running task with an indefinite execution time,
// so it's suitable for servers or services.
// The task can eventually return an error, or nil if the execution finishes
// without errors
type Runable func() error

// Stopper represent a function that will stop the Runable.
// The stopper acts as a notification to the runable to know that the task
// needs to be finished now.
//
// The stopper won't need to crash the runable or force the runable to stop,
// instead, it will let the runable to know it has to stop and let it finish.
// This means that the runable might still run for a while.
//
// It's recommended the stopper to run asynchronously. This means that the
// stopper might need to spawn a goroutine. The intention is avoid blocking
// the running thread.
//
// Usually, the stoppers are the servers's `Shutdown()` or `Close()` methods,
// that will cause the server to start its shutdown procedure. As said, there
// is no need to force the shutdown, so graceful shutdowns are preferred if
// they're available
type Stopper func()

// Result represents the result of a runner.
// The result contains the provided runner's id (for easier identification
// in case of multiple results) and the runner's error, which is the result
// of the Runable function (might be nil if no error happened)
type Result struct {
	RunnerID    string
	RunnerError error
}

// TimeoutError is an error that should be used for timeouts.
// It implements the `error` interface
type TimeoutError struct {
	RunnerID string
	Duration time.Duration
}

// NewTimeoutError creates a new timeout error. Both runnerID and duration
// will be used in the error message
func NewTimeoutError(runnerID string, duration time.Duration) *TimeoutError {
	return &TimeoutError{
		RunnerID: runnerID,
		Duration: duration,
	}
}

// NewGroupTimeoutError creates a new timeout error. This is intended to be
// used for group runners when the timeout of the group is reached.
// The runner id will be set to "_unknown_" because we don't know which is
// the id of the missing runner.
func NewGroupTimeoutError(duration time.Duration) *TimeoutError {
	return &TimeoutError{
		RunnerID: "_unknown_",
		Duration: duration,
	}
}

// Error generates the message for this particular error.
func (te *TimeoutError) Error() string {
	var sb strings.Builder
	sb.WriteString("Runner ")
	sb.WriteString(te.RunnerID)
	sb.WriteString(" timed out after waiting for ")
	sb.WriteString(te.Duration.String())
	return sb.String()
}
