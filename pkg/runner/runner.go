package runner

import (
	"context"
	"sync/atomic"
	"time"
)

// Runner represents the one executing a long running task, such as a server
// or a service.
// The ID of the runner is public to make identification easier, and the
// Result that it will generated will contain the same ID, so we can
// know which runner provided which result.
//
// Runners are intended to be used only once. Reusing them isn't possible.
// You'd need to create a new runner if you want to rerun the same task.
type Runner struct {
	ID            string
	interruptDur  time.Duration
	fn            Runable
	interrupt     Stopper
	running       atomic.Bool
	interrupted   atomic.Bool
	interruptedCh chan time.Duration
	finished      chan struct{}
}

// New will create a new runner.
// The runner will be created with the provided id (the id must be unique,
// otherwise undefined behavior might occur), and will run the provided
// runable task, using the "interrupt" function to stop that task if needed.
//
// The interrupt duration, which can be set through the `WithInterruptDuration`
// option, will be used to ensure the runner doesn't block forever. If the
// option isn't supplied, the default value `DefaultInterruptDuration` will be used.
// The interrupt duration will be used to start a timeout when the
// runner gets interrupted (either the context of the `Run` method is done
// or this runner's `Interrupt` method is called). If the timeout is reached,
// a timeout result will be returned instead of whatever result the task should
// be returning.
//
// Note that it's your responsibility to provide a proper stopper for the task.
// The runner will just call that method assuming it will be enough to
// eventually stop the task at some point.
func New(id string, fn Runable, interrupt Stopper, opts ...Option) *Runner {
	options := Options{
		InterruptDuration: DefaultInterruptDuration,
	}

	for _, o := range opts {
		o(&options)
	}

	return &Runner{
		ID:            id,
		interruptDur:  options.InterruptDuration,
		fn:            fn,
		interrupt:     interrupt,
		interruptedCh: make(chan time.Duration, 1),
		finished:      make(chan struct{}),
	}
}

// Run will execute the task associated to this runner in a synchronous way.
// The task will be spawned in a new goroutine, and the current thread will
// wait until the task finishes.
//
// The task will finish "naturally". The stopper will be called in the
// following ways:
// - Manually calling this runner's `Interrupt` method
// - When the provided context is done
// As said, it's expected that calling the provided stopper will be enough to
// make the task to eventually complete.
//
// Once the task finishes, the result will be returned.
// When the context is done, or if the runner is interrupted, a timeout will
// start using the provided "interrupt duration". If this timeout is reached,
// a timeout result will be returned instead of the one from the task. This is
// intended to prevent blocking the main thread indefinitely. A suitable
// duration should be used depending on the task, usually 5, 10 or 30 secs
//
// Some nice things you can do:
// - Use signal.NotifyContext(...) to call the stopper and provide a clean
// shutdown procedure when an OS signal is received
// - Use context.WithDeadline(...) or context.WithTimeout(...) to run the task
// for a limited time
func (r *Runner) Run(ctx context.Context) *Result {
	if !r.running.CompareAndSwap(false, true) {
		// If not swapped, the task is already running.
		// Running the same task multiple times is a bug, so we panic
		panic("Runner with id " + r.ID + " was running twice")
	}

	ch := make(chan *Result)

	go r.doTask(ch, true)

	select {
	case result := <-ch:
		return result
	case <-ctx.Done():
		r.Interrupt()
		return <-ch
	}
}

// RunAsync will execute the task associated to this runner asynchronously.
// The task will be spawned in a new goroutine and this method will finish.
// The task's result will be written in the provided channel when it's
// available, so you can wait for it if needed. It's up to you to decide
// to use a blocking or non-blocking channel, but the task will always finish
// before writing in the channel.
//
// To interrupt the running task, the only option is to call the `Interrupt`
// method at some point.
func (r *Runner) RunAsync(ch chan<- *Result) {
	if !r.running.CompareAndSwap(false, true) {
		// If not swapped, the task is already running.
		// Running the same task multiple times is a bug, so we panic
		panic("Runner with id " + r.ID + " was running twice")
	}

	go r.doTask(ch, false)
}

// Interrupt will execute the stopper function, which should notify the task
// in order for it to finish.
// The stopper will be called immediately, although it's expected the
// consequences to take a while (task might need a while to stop)
// A timeout will start using the provided "interrupt duration". Once that
// timeout is reached, the task must provide a result with a timeout error.
// Note that, even after returning the timeout result, the task could still
// be being executed and consuming resource.
// This method will be called only once. Further calls won't do anything
func (r *Runner) Interrupt() {
	if r.interrupted.CompareAndSwap(false, true) {
		go func() {
			select {
			case <-r.Finished():
				// Task finished -> runner should be delivering the result
			case <-time.After(r.interruptDur):
				// timeout reached -> send it through the channel so our runner
				// can abort
				r.interruptedCh <- r.interruptDur
				close(r.interruptedCh)
			}
		}()
		r.interrupt()
	}
}

// Finished will return a receive-only channel that can be used to know when
// the task has finished but the result hasn't been made available yet. The
// channel will be closed (without sending any message) when the task has finished.
// This can be used specially with the `RunAsync` method when multiple runners
// use the same channel: results could be waiting on your side of the channel
func (r *Runner) Finished() <-chan struct{} {
	return r.finished
}

// doTask will perform this runner's task and write the result in the provided
// channel. The channel will be closed if requested.
// A result will be provided when either the task finishes naturally or we
// reach the timeout after being interrupted
func (r *Runner) doTask(ch chan<- *Result, closeChan bool) {
	tmpCh := make(chan *Result, 1)

	// spawn the task and return the result in a temporary channel
	go func(tmpCh chan *Result) {
		err := r.fn()

		close(r.finished)

		result := &Result{
			RunnerID:    r.ID,
			RunnerError: err,
		}
		tmpCh <- result

		close(tmpCh)
	}(tmpCh)

	// wait for the result in the temporary channel or until we get the
	// interrupted signal
	var result *Result
	select {
	case d := <-r.interruptedCh:
		result = &Result{
			RunnerID:    r.ID,
			RunnerError: NewTimeoutError(r.ID, d),
		}
	case result = <-tmpCh:
		// Just assign the received value, nothing else to do
	}

	// send the result
	ch <- result
	if closeChan {
		close(ch)
	}
}
