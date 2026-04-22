package ratelimiter

import (
	"context"

	"go-micro.dev/v4/errors"
	"go-micro.dev/v4/server"
)

// NewHandlerWrapper creates a blocking server side rate limiter.
func NewHandlerWrapper(limit int) server.HandlerWrapper {
	if limit <= 0 {
		return func(h server.HandlerFunc) server.HandlerFunc {
			return h
		}
	}

	token := make(chan struct{}, limit)
	for i := 0; i < limit; i++ {
		token <- struct{}{}
	}

	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp any) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case t := <-token:
				defer func() {
					token <- t
				}()
				return h(ctx, req, rsp)
			default:
				return errors.New("go.micro.server", "Rate limit exceeded", 429)
			}

		}
	}
}
