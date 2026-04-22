package web

import (
	"context"
	"io"
)

type Service interface {
	ThemeApply(context.Context) error
	ThemeRemove(context.Context) error
}

type Repository interface {
	ThemeExists(ctx context.Context, id string) (bool, error)
	ThemeRemove(ctx context.Context, id string) error
	ThemeAdd(ctx context.Context, id string, r io.Reader) error
}

type ConsoleRepository interface {
	ThemeGet(ctx context.Context) (io.ReadCloser, error)
}
