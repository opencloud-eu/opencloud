package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"go.opentelemetry.io/otel/trace"
)

// AccessLog is a middleware to log http requests at info level logging.
func AccessLog(logger log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := middleware.GetReqID(r.Context())
			// add Request Id to all responses
			w.Header().Set(middleware.RequestIDHeader, requestID)
			wrap := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(wrap, r)

			spanContext := trace.SpanContextFromContext(r.Context())
			logger.Info().
				Str("proto", r.Proto).
				Str(log.RequestIDString, requestID).
				Str("traceid", spanContext.TraceID().String()).
				Str("remote-addr", middleware.GetClientIP(r.Context())).
				Str("method", r.Method).
				Int("status", wrap.Status()).
				Str("path", r.URL.Path).
				Dur("duration", time.Since(start)).
				Int("bytes", wrap.BytesWritten()).
				Msg("access-log")
		})
	}
}
