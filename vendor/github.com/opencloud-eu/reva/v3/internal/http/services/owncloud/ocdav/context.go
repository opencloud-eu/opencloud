package ocdav

import (
	"context"

	cs3storage "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
)

type tokenStatInfoKey struct{}

// ContextWithTokenStatInfo adds the token stat info to the context
func ContextWithTokenStatInfo(ctx context.Context, info *cs3storage.ResourceInfo) context.Context {
	return context.WithValue(ctx, tokenStatInfoKey{}, info)
}

// TokenStatInfoFromContext returns the token stat info from the context
func TokenStatInfoFromContext(ctx context.Context) (*cs3storage.ResourceInfo, bool) {
	v, ok := ctx.Value(tokenStatInfoKey{}).(*cs3storage.ResourceInfo)
	return v, ok
}
