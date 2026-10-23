package metrics

import (
	"context"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go-micro.dev/v4/server"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// grpcMetrics holds the metrics of the gRPC API of a service instance.
type grpcMetrics struct {
	enabled  prometheus.Gauge
	requests *grpcRequestMetrics // nil when request metrics are disabled
}

type grpcRequestMetrics struct {
	duration      *prometheus.HistogramVec
	total         *prometheus.CounterVec
	inFlight      prometheus.GaugeFunc
	inFlightCount *atomic.Int64
}

func newGRPCMetrics(namespace, subsystem string) *grpcMetrics {
	return &grpcMetrics{
		enabled: newEnabledGauge(namespace, subsystem, APIGRPC),
	}
}

// WithGRPCMetrics enables the inbound gRPC request metrics.
func WithGRPCMetrics() Option {
	return func(m *Metrics) {
		if m.GRPC == nil {
			m.GRPC = newGRPCMetrics(m.namespace, m.subsystem)
		}
		m.GRPC.enableRequestMetrics(m.namespace, m.subsystem)
	}
}

func (g *grpcMetrics) Enable()  { g.enabled.Set(1) }
func (g *grpcMetrics) Disable() { g.enabled.Set(0) }

// HandlerWrapper returns a go-micro handler wrapper that records the gRPC
// requests served by this instance.
func (g *grpcMetrics) HandlerWrapper() server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp any) error {
			if g == nil || g.requests == nil {
				return fn(ctx, req, rsp)
			}

			start := time.Now()
			g.requests.inFlightCount.Add(1)
			defer g.requests.inFlightCount.Add(-1)

			err := fn(ctx, req, rsp)
			g.RecordDuration(req.Method(), err, time.Since(start))
			return err
		}
	}
}

// RecordDuration records a single gRPC call.
func (g *grpcMetrics) RecordDuration(method string, err error, duration time.Duration) {
	if g == nil || g.requests == nil {
		return
	}
	code := grpcStatusCode(err)
	result := ResultSuccess
	switch {
	case code >= 500:
		result = ResultServerError
	case code >= 400:
		result = ResultClientError
	}
	g.requests.duration.WithLabelValues(method, strconv.Itoa(code), result).Observe(duration.Seconds())
	g.requests.total.WithLabelValues(method, strconv.Itoa(code), result).Inc()
}

// grpcStatusCode maps a gRPC status code to the equivalent HTTP status code,
// so that the code and result labels mean the same for both transports.
func grpcStatusCode(err error) int {
	switch status.Code(err) {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func (g *grpcMetrics) enableRequestMetrics(namespace, subsystem string) {
	if g.requests != nil {
		return
	}
	g.requests = &grpcRequestMetrics{
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "grpc_request_duration_seconds",
			Help:      "Duration of inbound gRPC requests in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{LabelMethod, LabelCode, LabelResult}),
		total: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "grpc_requests_total",
			Help:      "Number of inbound gRPC requests.",
		}, []string{LabelMethod, LabelCode, LabelResult}),
		inFlightCount: new(atomic.Int64),
	}
	g.requests.inFlight = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Unit:      "requests",
		Name:      "grpc_requests",
		Help:      "Concurrent inbound gRPC requests.",
	}, func() float64 {
		return float64(g.requests.inFlightCount.Load())
	})
}

func (g *grpcMetrics) collectors() []prometheus.Collector {
	collectors := []prometheus.Collector{g.enabled}
	if g.requests != nil {
		collectors = append(collectors, g.requests.duration, g.requests.total, g.requests.inFlight)
	}
	return collectors
}
