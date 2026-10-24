package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/metrics"
)

// Instrumenter provides a middleware to create metrics
func Instrumenter(m metrics.Metrics) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			m.InFlightInc(r)
			defer m.InFlightDec(r)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			m.Duration(r, ww.Status(), time.Since(start))
		})
	}
}
