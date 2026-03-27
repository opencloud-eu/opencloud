package middleware

import (
	"net/http"

	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// CollaborationTracingMiddleware adds a new middleware in order to include
// more attributes in the traced span.
//
// In order not to mess with the expected responses, this middleware won't do
// anything if there is no available WOPI context set in the request (there is
// nothing to report). This means that the WopiContextAuthMiddleware should be
// set before this middleware.
func CollaborationTracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wopiContext, err := WopiContextFromCtx(r.Context())
		if err != nil {
			// if we can't get the context, skip this middleware
			next.ServeHTTP(w, r)
			return
		}

		span := trace.SpanFromContext(r.Context())

		wopiMethod := r.Header.Get("X-WOPI-Override")

		wopiFile := wopiContext.FileReference

		attrs := []attribute.KeyValue{
			attribute.String("wopi.session.id", r.Header.Get("X-WOPI-SessionId")),
			attribute.String("wopi.method", wopiMethod),
			attribute.String("cs3.resource.id.storage", wopiFile.GetResourceId().GetStorageId()),
			attribute.String("cs3.resource.id.opaque", wopiFile.GetResourceId().GetOpaqueId()),
			attribute.String("cs3.resource.id.space", wopiFile.GetResourceId().GetSpaceId()),
			attribute.String("cs3.resource.path", wopiFile.GetPath()),
		}

		if wopiUser, ok := ctxpkg.ContextGetUser(r.Context()); ok {
			attrs = append(attrs, []attribute.KeyValue{
				attribute.String("enduser.id", wopiUser.GetId().GetOpaqueId()),
				attribute.String("cs3.user.idp", wopiUser.GetId().GetIdp()),
				attribute.String("cs3.user.opaque", wopiUser.GetId().GetOpaqueId()),
				attribute.String("cs3.user.type", wopiUser.GetId().GetType().String()),
			}...)
		}
		span.SetAttributes(attrs...)

		next.ServeHTTP(w, r)
	})
}
