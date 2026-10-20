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
	ResultClientError = "client-error"
	ResultServerError = "server-error"

	unmatchedRoutePattern = "unknown"
)

type httpMetrics struct {
	enabled  prometheus.Gauge
	requests *httpRequestMetrics // nil when request metrics are disabled
}

type httpRequestMetrics struct {
	duration      *prometheus.HistogramVec
	total         *prometheus.CounterVec
	inFlight      prometheus.GaugeFunc
	inFlightCount *atomic.Int64
}

func newHTTPMetrics(namespace, subsystem string) *httpMetrics {
	return &httpMetrics{
		enabled: newEnabledGauge(namespace, subsystem, APIHTTP),
	}
}

// WithHTTPMetrics enables the inbound HTTP request metrics.
func WithHTTPMetrics() Option {
	return func(m *Metrics) {
		if m.HTTP == nil {
			m.HTTP = newHTTPMetrics(m.namespace, m.subsystem)
		}
		m.HTTP.enableRequestMetrics(m.namespace, m.subsystem)
	}
}

func (h *httpMetrics) Enable()  { h.enabled.Set(1) }
func (h *httpMetrics) Disable() { h.enabled.Set(0) }

func (h *httpMetrics) RecordDuration(method, pattern string, statusCode int, duration time.Duration) {
	if h == nil || h.requests == nil {
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
	h.requests.duration.WithLabelValues(method, pattern, code, result).Observe(duration.Seconds())
	h.requests.total.WithLabelValues(method, pattern, code, result).Inc()
}

func (h *httpMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h == nil || h.requests == nil {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		h.requests.inFlightCount.Add(1)
		defer h.requests.inFlightCount.Add(-1)

		wrapper := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(wrapper, r)

		pattern := unmatchedRoutePattern
		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			if p := rctx.RoutePattern(); p != "" {
				pattern = p
			}
		}
		status := wrapper.Status()
		if status == 0 {
			// chi's WrapResponseWriter reports 0 when the handler never called
			// WriteHeader and wrote no body; net/http would have sent 200 OK.
			status = http.StatusOK
		}
		h.RecordDuration(r.Method, pattern, status, time.Since(start))
	})
}

func (h *httpMetrics) enableRequestMetrics(namespace, subsystem string) {
	if h.requests != nil {
		return
	}
	h.requests = &httpRequestMetrics{
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_request_duration_seconds",
			Help:      "Duration of inbound HTTP requests in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{LabelMethod, LabelPath, LabelCode, LabelResult}),
		total: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_requests_total",
			Help:      "Number of inbound HTTP requests.",
		}, []string{LabelMethod, LabelPath, LabelCode, LabelResult}),
		inFlightCount: new(atomic.Int64),
	}
	h.requests.inFlight = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Unit:      "requests",
		Name:      "http_requests",
		Help:      "Concurrent inbound HTTP requests.",
	}, func() float64 {
		return float64(h.requests.inFlightCount.Load())
	})
}

func (h *httpMetrics) collectors() []prometheus.Collector {
	collectors := []prometheus.Collector{h.enabled}
	if h.requests != nil {
		collectors = append(collectors, h.requests.duration, h.requests.total, h.requests.inFlight)
	}
	return collectors
}
