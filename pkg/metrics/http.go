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
		m.HTTP.enableRequestMetrics(m.namespace, m.subsystem)
	}
}

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

		// To avoid having a high cardinality on the path label, which may include
		// different identifiers and query parameters and would result in a very
		// high number of individual values for the path label, we are using chi to
		// resolve the path against the pattern that it matches in its route
		// configuration, and we are using that as the value for the path label, as
		// the cardinality for those will remain quite small, since it uses the
		// placeholders for values instead of the actual values, so e.g. something
		// like "/resource/child/{id}" instead of
		// "/resource/child/d6b60c20-8711-4921-9cd6-b45a6a5b6ce1".
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

func (h *httpMetrics) setEnabled(enabled bool) {
	if h == nil {
		return
	}
	h.enabled.Set(boolToFloat64(enabled))
}

func (h *httpMetrics) enableRequestMetrics(namespace, subsystem string) {
	if h.requests != nil {
		return
	}
	h.requests = &httpRequestMetrics{
		// TODO(pbleser-oc): use Prometheus native histograms instead once the
		// minimum supported Prometheus (and Grafana) version allows it; track both
		// for now to remain backwards compatible.
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
