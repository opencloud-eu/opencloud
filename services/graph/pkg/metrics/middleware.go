package metrics

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// A middleware that tracks the duration of every inbound Graph API HTTP call
// and calls a function to delegate the storage of that duration into a
// histogram metric, analyzing the incoming query and deconstructing it into
// method, path pattern, as well as the resulting status code.
//
// It also tracks the number of concurrent HTTP requests that are in flight,
// using a Gauge that it increments and decrements when it wraps the next
// handler.
//
// Note that to avoid a high cardinality on the path label, the URL is matched
// against the chi routing rules, passing the path pattern to the function
// instead of the actual URI.
func HTTPMetrics(inFlight *atomic.Int64, observe func(method, pattern string, statusCode int, duration time.Duration)) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			inFlight.Add(1)
			defer inFlight.Add(-1)

			wrapper := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(wrapper, r)

			duration := time.Since(start)

			method := r.Method
			routePattern := UnmatchedRoutePattern
			if rctx := chi.RouteContext(r.Context()); rctx != nil {
				if pattern := rctx.RoutePattern(); pattern != "" {
					routePattern = pattern
					if method == "" {
						method = rctx.RouteMethod
					}
				}
			}

			observe(r.Method, routePattern, wrapper.Status(), duration)
		})
	}
}
