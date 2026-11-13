package httprate

import "context"

type ctxKey int

const (
	incrementKey ctxKey = iota
	requestLimitKey
)

func WithIncrement(ctx context.Context, value int) context.Context {
	return context.WithValue(ctx, incrementKey, value)
}

func getIncrement(ctx context.Context) int {
	if value, ok := ctx.Value(incrementKey).(int); ok {
		return value
	}
	return 1
}

func WithRequestLimit(ctx context.Context, value int) context.Context {
	return context.WithValue(ctx, requestLimitKey, value)
}

func getRequestLimit(ctx context.Context) int {
	if value, ok := ctx.Value(requestLimitKey).(int); ok {
		return value
	}
	return 0
}
