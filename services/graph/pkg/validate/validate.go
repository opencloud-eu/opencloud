package validate

import (
	"context"
	"sync/atomic"

	"github.com/go-playground/validator/v10"
)

var defaultValidator atomic.Value

func init() {
	v := validator.New()

	initLibregraph(v)

	defaultValidator.Store(v)
}

// Default returns the default validator.
func Default() *validator.Validate { return defaultValidator.Load().(*validator.Validate) }

// StructCtx validates a struct and returns the error.
func StructCtx(ctx context.Context, s any) error {
	return Default().StructCtx(ctx, s)
}
