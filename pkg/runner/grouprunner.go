package runner

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// GroupRunner represent a group of tasks that need to run together.
// The expectation is that all the tasks will run at the same time, and when
// one of them stops, the rest will also stop.
//
// The GroupRunner is intended to be used to run multiple services, which are
// more or less independent from eachother, but at the same time it doesn't
// make sense to have any of them stopped while the rest are running.
// Basically, either all of them run, or none of them.
// For example, you can have a GRPC and HTTP servers running, each of them
// providing a piece of functionality, however, if any of them fails, the
// feature provided by them would be incomplete or broken.
//
// The interrupt duration for the group can be set through the
// `WithInterruptDuration` option. If the option isn't supplied, the default
// value `DefaultGroupInterruptDuration` will be used.
//
// It's recommended that the timeouts are handled by each runner individually,
// meaning that each runner's timeout should be less than the group runner's
// timeout. This way, we can know which runner timed out.
// If the group timeout is reached, the remaining results will have the
// runner's id as "_unknown_".
//
// Note that, as services, the task aren't expected to stop by default.
// This means that, if a task finishes naturally, the rest of the task will
// asked to stop as well.
type GroupRunner struct {
	runners       sync.Map
	runnersCount  int
	isRunning     bool
	interruptDur  time.Duration
	interrupted   atomic.Bool
	interruptedCh chan time.Duration
	runningMutex  sync.Mutex
}

// NewGroup will create a GroupRunner
func NewGroup(opts ...Option) *GroupRunner {
	options := Options{
		InterruptDuration: DefaultGroupInterruptDuration,
	}

	for _, o := range opts {
		o(&options)
	}

	return &GroupRunner{
		runners:       sync.Map{},
		runningMutex:  sync.Mutex{},
		interruptDur:  options.InterruptDuration,
		interruptedCh: make(chan time.Duration, 1),
	}
}

// Add will add a runner to the group.
//
// It's mandatory that each runner in the group has an unique id, otherwise
// there will be issues
// Adding new runners once the group starts will cause a panic
func (gr *GroupRunner) Add(r *Runner) {
	gr.runningMutex.Lock()
	defer gr.runningMutex.Unlock()

	if gr.isRunning {
		panic("Adding a new runner after the group starts is forbidden")
	}

	// LoadOrStore will try to store the runner
	if _, loaded := gr.runners.LoadOrStore(r.ID, r); loaded {
		// there is already a runner with the same id, which is forbidden
		panic("Trying to add a runner with an existing Id in the group")
	}
	// Only increase the count if a runner is stored.
	// Currently panicking if the runner exists and is loaded
	gr.runnersCount++
}

// Run will execute all the tasks in the group at the same time.
//
// Similarly to the "regular" runner's `Run` method, the execution thread
// will be blocked here until all tasks are completed, and their results
// will be available (each result will have the runner's id so it's easy to
// find which one failed). Note that there is no guarantee about the result's
// order, so the first result in the slice might or might not be the first
// result to be obtained.
//
// When the context is marked as done, the groupRunner will call all the
// stoppers for each runner to notify each task to stop. Note that the tasks
// might still take a while to complete.
//
// If a task finishes naturally (with the context still "alive"), it will also
// cause the groupRunner to call the stoppers of the rest of the tasks. So if
// a task finishes, the rest will also finish.
// Note that it is NOT expected for the finished task's stopper to be called
// in this case.
func (gr *GroupRunner) Run(ctx context.Context) []*Result {
	// Set the flag inside the runningMutex to ensure we don't read the old value
	// in the `Add` method and add a new runner when this method is being executed
	// Note that if multiple `Run` or `RunAsync` happens, the underlying runners
	// will panic
	gr.runningMutex.Lock()
	gr.isRunning = true
	gr.runningMutex.Unlock()

	results := make([]*Result, 0, gr.runnersCount)

	ch := make(chan *Result, gr.runnersCount) // no need to block writing results
	gr.runners.Range(func(_, value any) bool {
		r := value.(*Runner)
		r.RunAsync(ch)
		return true
	})

	var d time.Duration
	// wait for a result or for the context to be done
	select {
	case result := <-ch:
		results = append(results, result)
	case d = <-gr.interruptedCh:
		results = append(results, &Result{
			RunnerID:    "_unknown_",
			RunnerError: NewGroupTimeoutError(d),
		})
	case <-ctx.Done():
		// Do nothing
	}

	// interrupt the rest of the runners
	gr.Interrupt()

	// Having notified that the context has been finished, we still need to
	// wait for the rest of the results
	for i := len(results); i < gr.runnersCount; i++ {
		select {
		case result := <-ch:
			results = append(results, result)
		case d2, ok := <-gr.interruptedCh:
			if ok {
				d = d2
			}
			results = append(results, &Result{
				RunnerID:    "_unknown_",
				RunnerError: NewGroupTimeoutError(d),
			})
		}
	}

	// Even if we reach the group time out and bail out early, tasks might
	// be running and eventually deliver the result through the channel.
	// We'll rely on the buffered channel so the tasks won't block and the
	// data can be eventually garbage-collected along with the unused
	// channel, so we won't close the channel here.
	return results
}

// RunAsync will execute the tasks in the group asynchronously.
// The result of each task will be placed in the provided channel as soon
// as it's available.
// Note that this method will finish as soon as all the tasks are running.
func (gr *GroupRunner) RunAsync(ch chan<- *Result) {
	// Set the flag inside the runningMutex to ensure we don't read the old value
	// in the `Add` method and add a new runner when this method is being executed
	// Note that if multiple `Run` or `RunAsync` happens, the underlying runners
	// will panic
	gr.runningMutex.Lock()
	gr.isRunning = true
	gr.runningMutex.Unlock()

	// we need a secondary channel to receive the first result so we can
	// interrupt the rest of the tasks
	interCh := make(chan *Result, gr.runnersCount)
	gr.runners.Range(func(_, value any) bool {
		r := value.(*Runner)
		r.RunAsync(interCh)
		return true
	})

	go func() {
		var result *Result
		var d time.Duration

		select {
		case result = <-interCh:
			// result already assigned, so do nothing
		case d = <-gr.interruptedCh:
			// we aren't tracking which runners have finished and which are still
			// running, so we'll use "_unknown_" as runner id
			result = &Result{
				RunnerID:    "_unknown_",
				RunnerError: NewGroupTimeoutError(d),
			}
		}
		gr.Interrupt()

		ch <- result
		for i := 1; i < gr.runnersCount; i++ {
			select {
			case result = <-interCh:
				// result already assigned, so do nothing
			case d2, ok := <-gr.interruptedCh:
				// if ok is true, d2 will have a good value; if false, the channel
				// is closed and we get a default value
				if ok {
					d = d2
				}
				result = &Result{
					RunnerID:    "_unknown_",
					RunnerError: NewGroupTimeoutError(d),
				}
			}
			ch <- result
		}
	}()
}

// Interrupt will execute the stopper function of ALL the tasks, which should
// notify the tasks in order for them to finish.
// The stoppers will be called immediately but sequentially. This means that
// the second stopper won't be called until the first one has returned. This
// usually isn't a problem because the service `Stop`'s methods either don't
// take a long time to return, or they run asynchronously in another goroutine.
//
// As said, this will affect ALL the tasks in the group. It isn't possible to
// try to stop just one task.
// If a task has finished, the corresponding stopper won't be called
//
// The interrupt timeout for the group will start after all the runners in the
// group have been notified. Note that, if the task's stopper for a runner
// takes a lot of time to return, it will delay the timeout's start, so it's
// advised that the stopper either returns fast or is run asynchronously.
func (gr *GroupRunner) Interrupt() {
	if gr.interrupted.CompareAndSwap(false, true) {
		gr.runners.Range(func(_, value any) bool {
			r := value.(*Runner)
			select {
			case <-r.Finished():
				// No data should be sent through the channel, so we'd be
				// here only if the channel is closed. This means the task
				// has finished and we don't need to interrupt. We do
				// nothing in this case
			default:
				r.Interrupt()
			}
			return true
		})

		_ = time.AfterFunc(gr.interruptDur, func() {
			// timeout reached -> send it through the channel so our runner
			// can abort
			gr.interruptedCh <- gr.interruptDur
			close(gr.interruptedCh)
		})
	}
}
