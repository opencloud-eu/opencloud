// Package auditctx passes the audit request context on to the next service,
// like the useragent interceptor does with the user agent.
package auditctx

import (
	"context"

	"github.com/opencloud-eu/reva/v3/pkg/audit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// NewUnary returns a unary interceptor that keeps the audit request context
// for this service and its outgoing calls.
func NewUnary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(withRequestContext(ctx), req)
	}
}

// NewStream returns a stream interceptor that keeps the audit request context
// for this service and its outgoing calls.
func NewStream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &wrappedServerStream{ServerStream: ss, ctx: withRequestContext(ss.Context())})
	}
}

func withRequestContext(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	lst := md.Get(audit.Header)
	if len(lst) == 0 {
		return ctx
	}
	return audit.ContextSetRequest(ctx, audit.ParseRequestContext(lst[0]))
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (ss *wrappedServerStream) Context() context.Context {
	return ss.ctx
}
