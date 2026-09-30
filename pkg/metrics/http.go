package metrics

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	LabelMethod       = "method"
	LabelPath         = "path"
	LabelCode         = "code"
	ResultClientError = "clientError"
	ResultServerError = "serverError"

	unmatchedRoutePattern = "unknown"
)

type httpMetrics struct {
	duration         *prometheus.HistogramVec
	requests         *prometheus.CounterVec
	requestsInFlight *atomic.Int64
}

// WithHTTPMetrics enables the inbound HTTP request metrics.
func WithHTTPMetrics() Option {
	return func(m *Metrics) {
		if m.HTTP != nil {
			return
		}
		h := &httpMetrics{
			duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: m.namespace,
				Subsystem: m.subsystem,
				Name:      "http_request_duration_seconds",
				Help:      "Duration of inbound HTTP requests in seconds.",
				Buckets:   prometheus.DefBuckets,
			}, []string{LabelMethod, LabelPath, LabelCode, LabelResult}),
			requests: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: m.namespace,
				Subsystem: m.subsystem,
				Name:      "http_requests_total",
				Help:      "Number of inbound HTTP requests.",
			}, []string{LabelMethod, LabelPath, LabelCode, LabelResult}),
			requestsInFlight: new(atomic.Int64),
		}
		m.HTTP = h
		m.collectors = append(m.collectors, h.duration, h.requests)
		m.collectors = append(m.collectors, prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: m.namespace,
			Subsystem: m.subsystem,
			Unit:      "requests",
			Name:      "http_requests",
			Help:      "Concurrent inbound HTTP requests.",
		}, func() float64 {
			return float64(h.requestsInFlight.Load())
		}))
	}
}

func (h *httpMetrics) RecordDuration(method, pattern string, statusCode int, duration time.Duration) {
	if h == nil {
		return
	}
	result := ResultSuccess
	switch {
	case statusCode >= 500:
		result = ResultServerError
	case statusCode >= 400:
		result = ResultClientError
	}
	code := strconv.Itoa(statusCode)
	h.duration.WithLabelValues(method, pattern, code, result).Observe(duration.Seconds())
	h.requests.WithLabelValues(method, pattern, code, result).Inc()
}

func (h *httpMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h == nil {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		h.requestsInFlight.Add(1)
		defer h.requestsInFlight.Add(-1)

		wrapper := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(wrapper, r)

		pattern := unmatchedRoutePattern
		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			if p := rctx.RoutePattern(); p != "" {
				pattern = p
			}
		}
		h.RecordDuration(r.Method, pattern, wrapper.Status(), time.Since(start))
	})
}
