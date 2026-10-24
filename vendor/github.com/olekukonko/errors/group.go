// Group runs multiple functions concurrently and collects all errors into a
// *MultiError. It is the error-aware counterpart to sync/errgroup: errgroup
// stops at the first failure; Group collects every failure.

package errors

import (
	"context"
	"sync"
)

// Group runs goroutines concurrently and collects every error they return.
// The zero value is ready to use; options may be applied via NewGroup.
//
// Example — fan-out with full error collection:
//
//	g := errors.NewGroup()
//	g.Go(func() error { return validateUser(id) })
//	g.Go(func() error { return validatePerms(id) })
//	if err := g.Wait(); err != nil {
//	    // err is *MultiError containing all failures
//	    log.Println(err)
//	}
type Group struct {
	wg            sync.WaitGroup
	errs          *MultiError
	ctx           context.Context
	cancel        context.CancelFunc
	cancelOnFirst bool
}

// GroupOption configures a Group.
type GroupOption func(*Group)

// GroupWithContext attaches ctx to the Group. The ctx is passed to
// context-aware Go calls (GoCtx). If cancelOnFirst is true, the context
// is cancelled as soon as the first error is returned by any goroutine —
// useful for "cancel siblings on first failure" patterns.
func GroupWithContext(ctx context.Context, cancelOnFirst bool) GroupOption {
	return func(g *Group) {
		g.ctx, g.cancel = context.WithCancel(ctx)
		g.cancelOnFirst = cancelOnFirst
	}
}

// GroupWithLimit sets a maximum error limit on the underlying MultiError.
// Errors beyond the limit are dropped.
func GroupWithLimit(n int) GroupOption {
	return func(g *Group) {
		g.errs = NewMultiError(WithLimit(n))
	}
}

// NewGroup creates a Group with the given options applied.
func NewGroup(opts ...GroupOption) *Group {
	g := &Group{
		errs: NewMultiError(),
	}
	for _, o := range opts {
		o(g)
	}
	if g.ctx == nil {
		g.ctx = context.Background()
	}
	return g
}

// Go starts fn in a new goroutine. Errors returned by fn are collected;
// nil returns are ignored. Thread-safe: MultiError.Add handles its own locking.
// cancelOnFirst is read-only after construction so no lock is needed.
func (g *Group) Go(fn func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := fn(); err != nil {
			g.errs.Add(err) // MultiError.Add is internally mutex-protected
			if g.cancelOnFirst && g.cancel != nil {
				g.cancel()
			}
		}
	}()
}

// GoCtx starts fn in a new goroutine, passing the group's context.
// If the group was created with GroupWithContext, fn receives a context
// that is cancelled when cancelOnFirst triggers or the parent is done.
func (g *Group) GoCtx(fn func(ctx context.Context) error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := fn(g.ctx); err != nil {
			g.errs.Add(err) // MultiError.Add is internally mutex-protected
			if g.cancelOnFirst && g.cancel != nil {
				g.cancel()
			}
		}
	}()
}

// Wait blocks until all goroutines have finished and returns a *MultiError
// containing every error collected, or nil if all succeeded.
// Always returns *MultiError (never collapses to a raw error) so callers
// can reliably type-assert the result.
func (g *Group) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel() // release context resources
	}
	if !g.errs.Has() {
		return nil
	}
	return g.errs
}

// Errors returns a snapshot of errors collected so far.
// Safe to call concurrently with Go/GoCtx; may be incomplete before Wait returns.
func (g *Group) Errors() []error {
	return g.errs.Errors()
}
