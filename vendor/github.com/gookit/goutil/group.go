package goutil

import (
	"context"

	"github.com/gookit/goutil/structs"
	"github.com/gookit/goutil/syncs"
)

// ErrGroup is a collection of goroutines working on subtasks that
// are part of the same overall task.
type ErrGroup = syncs.ErrGroup

// NewCtxErrGroup instance. use for batch run tasks, can with context.
func NewCtxErrGroup(ctx context.Context, limit ...int) (*ErrGroup, context.Context) {
	return syncs.NewCtxErrGroup(ctx, limit...)
}

// NewErrGroup instance. use for batch run tasks
func NewErrGroup(limit ...int) *ErrGroup {
	return syncs.NewErrGroup(limit...)
}

// RunFn func
type RunFn func(ctx *structs.Data) error

// QuickRun struct
type QuickRun struct {
	ctx *structs.Data
	// err error
	fns []RunFn
}

// NewQuickRun instance
func NewQuickRun() *QuickRun {
	return &QuickRun{
		ctx: structs.NewData(),
	}
}

// Add func for run
func (p *QuickRun) Add(fns ...RunFn) *QuickRun {
	p.fns = append(p.fns, fns...)
	return p
}

// Run all func
func (p *QuickRun) Run() error {
	for i, fn := range p.fns {
		p.ctx.Set("index", i)
		if err := fn(p.ctx); err != nil {
			return err
		}
	}
	return nil
}
