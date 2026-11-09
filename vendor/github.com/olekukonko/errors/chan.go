// Channel-based error utilities and streaming error collection.
// All functions compose with the standard (chan T, chan error) idiom.

package errors

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// ErrLimitReached is returned by Collect when n errors have been gathered
// before the channel closed or the context was done.
// Callers can use errors.Is(err, ErrLimitReached) to distinguish this case.
var ErrLimitReached = Const("limit_reached", "error collection limit reached")

// Drain reads all errors from ch until it is closed and returns them as a
// *MultiError. Returns nil if every received value was nil.
// Blocks until ch is closed.
//
// Example:
//
//	results, errs := processItems(ctx, items)
//	if err := errors.Drain(errs); err != nil {
//	    log.Println(err)
//	}
func Drain(ch <-chan error) error {
	m := NewMultiError()
	for err := range ch {
		if err != nil {
			m.Add(err)
		}
	}
	return m.Single()
}

// First returns the first non-nil error received from ch, then returns.
// Uses ctx for deadline/cancellation only — it does NOT call any cancel
// function. The caller is responsible for cancelling sibling work after
// First returns.
//
// Returns nil if ch is closed before any error arrives, or ctx is done.
// Returns ctx.Err() if the context is cancelled or times out.
//
// Example:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	results, errs := processItems(ctx, items)
//	if err := errors.First(ctx, errs); err != nil {
//	    cancel() // caller decides to stop siblings
//	    log.Println(err)
//	}
//	defer cancel()
func First(ctx context.Context, ch <-chan error) error {
	for {
		select {
		case err, ok := <-ch:
			if !ok {
				return nil
			}
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Fan merges multiple error channels into a single output channel that closes
// when all inputs have closed or ctx is done.
//
// Callers MUST either drain the returned channel to completion OR cancel ctx —
// failing to do so leaks the internal goroutines. The select in each forwarder
// respects ctx.Done() so cancellation is always safe.
//
// Example:
//
//	errs1, errs2 := stage1(ctx), stage2(ctx)
//	for err := range errors.Fan(ctx, errs1, errs2) {
//	    if err != nil { log.Println(err) }
//	}
func Fan(ctx context.Context, chans ...<-chan error) <-chan error {
	bufSize := len(chans)
	if bufSize < 1 {
		bufSize = 1
	}

	out := make(chan error, bufSize)
	var wg sync.WaitGroup
	wg.Add(len(chans))

	for _, ch := range chans {
		ch := ch
		go func() {
			defer wg.Done()
			for {
				select {
				case err, ok := <-ch:
					if !ok {
						return
					}
					select {
					case out <- err:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// Collect reads up to n non-nil errors from ch (or until ch closes or ctx is
// done) and returns them as a *MultiError.
//
// If the limit n is reached before the channel closes, the returned error
// wraps ErrLimitReached as its cause so callers can distinguish the two cases:
//
//	err := errors.Collect(ctx, errs, 10)
//	if errors.Is(err, errors.ErrLimitReached) {
//	    // stopped early — more errors may exist
//	}
func Collect(ctx context.Context, ch <-chan error, n int) error {
	m := NewMultiError(WithLimit(n))
	for {
		select {
		case err, ok := <-ch:
			if !ok {
				return m.Single()
			}
			if err != nil {
				m.Add(err)
				if m.Count() >= n {
					// Wrap in an *Error so errors.Is traversal finds ErrLimitReached.
					e := New(fmt.Sprintf("collected %d errors (limit reached)", n))
					e.cause = ErrLimitReached
					if inner := m.Single(); inner != nil {
						return New(e.Error()).Wrap(inner).Wrap(ErrLimitReached)
					}
					return e
				}
			}
		case <-ctx.Done():
			return m.Single()
		}
	}
}

// Stream — concurrent item processing with progressive error collection

// Stream processes a slice of items concurrently, collecting errors as they
// occur without stopping execution. Use Wait() to block until all items are
// done, or Each() to process errors as they arrive.
//
// Example — collect all:
//
//	s := errors.NewStream(ctx, items, process, 8)
//	if err := s.Wait(); err != nil {
//	    log.Println(err)
//	}
//
// Example — process as they arrive:
//
//	s := errors.NewStream(ctx, items, process, 8)
//	s.Each(func(err error) { log.Println(err) })
type Stream[T any] struct {
	ch        chan error
	done      chan struct{}
	stopCh    chan struct{}
	closeOnce sync.Once
	stopOnce  sync.Once

	// consumed guards Each/Wait — only one consumer is permitted.
	// 0 = available, 1 = consumed. Enforced with atomic CAS.
	consumed atomic.Int32
}

// NewStream creates a Stream that applies fn to every item in items using
// up to workers concurrent goroutines.
//
// workers <= 0 defaults to len(items), running all items at once.
// Respects ctx: in-flight work completes but no new items start once ctx
// is done.
//
// Example:
//
//	s := errors.NewStream(ctx, urls, func(url string) error {
//	    return fetch(url)
//	}, 8)
func NewStream[T any](ctx context.Context, items []T, fn func(T) error, workers ...int) *Stream[T] {
	w := len(items)
	if len(workers) > 0 && workers[0] > 0 {
		w = workers[0]
	}
	if w < 1 {
		w = 1
	}

	s := &Stream[T]{
		ch:     make(chan error, w),
		done:   make(chan struct{}),
		stopCh: make(chan struct{}),
	}

	go s.run(ctx, items, fn, w)
	return s
}

func (s *Stream[T]) run(ctx context.Context, items []T, fn func(T) error, workers int) {
	defer func() {
		s.closeOnce.Do(func() { close(s.ch) })
		close(s.done)
	}()

	work := make(chan T, workers)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range work {
				if err := fn(item); err != nil {
					select {
					case s.ch <- err:
					case <-s.stopCh:
						return
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

feed:
	for _, item := range items {
		select {
		case work <- item:
		case <-s.stopCh:
			break feed
		case <-ctx.Done():
			break feed
		}
	}
	close(work)
	wg.Wait()
}

// acquireConsumer atomically marks the stream as consumed.
// Panics if called more than once — Each and Wait are mutually exclusive.
func (s *Stream[T]) acquireConsumer(name string) {
	if !s.consumed.CompareAndSwap(0, 1) {
		panic(fmt.Sprintf("errors.Stream: %s called on an already-consumed Stream; Each and Wait are mutually exclusive", name))
	}
}

// Each calls fn for every error produced by the stream, in the order they
// arrive. Blocks until all items have been processed.
//
// Panics if called after Wait (or a second call to Each).
func (s *Stream[T]) Each(fn func(error)) {
	s.acquireConsumer("Each")
	for err := range s.ch {
		fn(err)
	}
	<-s.done
}

// Wait blocks until all items have been processed and returns a *MultiError
// containing every error collected, or nil if all items succeeded.
//
// Panics if called after Each (or a second call to Wait).
func (s *Stream[T]) Wait() error {
	s.acquireConsumer("Wait")
	m := NewMultiError()
	for err := range s.ch {
		m.Add(err)
	}
	<-s.done
	return m.Single()
}

// Stop signals the stream to stop processing new items and drains any
// buffered errors in the background. Safe to call multiple times.
// After Stop, Wait and Each will still return promptly but may not see
// all errors.
func (s *Stream[T]) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		// Drain the error channel in the background so run() goroutines
		// are not blocked trying to send, preventing a goroutine leak.
		go func() {
			for range s.ch {
			}
		}()
	})
}
