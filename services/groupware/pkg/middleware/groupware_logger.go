package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

func GroupwareLogger(logger log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrap := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(wrap, r)

			level := logger.Debug()
			err := recover()
			if err != nil {
				level = logger.Error()
			}

			if !level.Enabled() {
				return
			}

			if err != nil {
				switch e := err.(type) {
				case error:
					level = level.Err(e)
				default:
					level = level.Any("panic", e)
				}
			}

			ctx := r.Context()

			requestID := middleware.GetReqID(ctx)
			traceID := GetTraceID(ctx)

			level.Str(log.RequestIDString, requestID)

			if traceID != "" {
				level.Str("traceId", traceID)
			}

			level.
				Str("proto", r.Proto).
				Str("method", r.Method).
				Int("status", wrap.Status()).
				Str("path", r.URL.Path).
				Dur("duration", time.Since(start)).
				Int("bytes", wrap.BytesWritten()).
				Msg("")
		})
	}
}
