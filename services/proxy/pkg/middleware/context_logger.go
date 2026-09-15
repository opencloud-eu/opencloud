package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// ContextLogger is a middleware to use a logger associated with the request's
// context which includes general information of the request.
func ContextLogger(logger log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := logger.With().
				Str("remoteAddr", middleware.GetClientIP(r.Context())).
				Str(log.RequestIDString, r.Header.Get("X-Request-ID")).
				Str("proto", r.Proto).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("query", r.URL.RawQuery).
				Str("fragment", r.URL.Fragment).
				Logger().WithContext(r.Context())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
