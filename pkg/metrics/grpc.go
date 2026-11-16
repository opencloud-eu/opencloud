package metrics

import (
	"context"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	merrors "go-micro.dev/v4/errors"
	"go-micro.dev/v4/server"
	"google.golang.org/grpc/status"
)

// grpcMetrics holds the metrics of the gRPC API of a service instance.
type grpcMetrics struct {
	enabled  prometheus.Gauge
	requests *grpcRequestMetrics // nil when request metrics are disabled
}

type grpcRequestMetrics struct {
	duration      *prometheus.HistogramVec
	inFlight      prometheus.GaugeFunc
	inFlightCount *atomic.Int64
}

func newGRPCMetrics(namespace, subsystem string) *grpcMetrics {
	return &grpcMetrics{
		enabled: newEnabledGauge(namespace, subsystem, APIGRPC, "gRPC API calls"),
	}
}

// WithGRPC declares that the service has a gRPC API, and whether it is enabled.
func WithGRPC(enabled bool) Option {
	return func(m *Metrics) {
		if m.GRPC == nil {
			m.GRPC = newGRPCMetrics(m.namespace, m.subsystem)
		}
		m.GRPC.setEnabled(enabled)
	}
}

// WithGRPCRequestMetrics enables the inbound gRPC request metrics when enabled
// is true, and declares the gRPC API if WithGRPC has not been given.
func WithGRPCRequestMetrics(enabled bool) Option {
	return func(m *Metrics) {
		if !enabled {
			return
		}
		if m.GRPC == nil {
			m.GRPC = newGRPCMetrics(m.namespace, m.subsystem)
		}
		m.GRPC.enableRequestMetrics(m.namespace, m.subsystem)
	}
}

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
}

// grpcStatusCode maps a gRPC status code to the equivalent HTTP status code,
// so that the code and result labels mean the same for both transports.
// go-micro errors already carry an HTTP status code and are checked first,
// otherwise the gRPC code is mapped using the grpc-gateway mapping.
func grpcStatusCode(err error) int {
	merr, ok := merrors.As(err)
	if ok && merr.GetCode() != 0 {
		return int(merr.GetCode())
	}
	return runtime.HTTPStatusFromCode(status.Code(err))
}

func (g *grpcMetrics) setEnabled(enabled bool) {
	if g == nil {
		return
	}
	g.enabled.Set(boolToFloat64(enabled))
}

func (g *grpcMetrics) enableRequestMetrics(namespace, subsystem string) {
	if g.requests != nil {
		return
	}
	g.requests = &grpcRequestMetrics{
		// TODO: use Prometheus native histograms, see https://github.com/opencloud-eu/opencloud/issues/3817
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "grpc_request_duration_seconds",
			Help:      "Duration of inbound gRPC requests in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{LabelMethod, LabelCode, LabelResult}),
		inFlightCount: new(atomic.Int64),
	}
	g.requests.inFlight = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "grpc_requests_in_flight",
		Help:      "Concurrent inbound gRPC requests.",
	}, func() float64 {
		return float64(g.requests.inFlightCount.Load())
	})
}

func (g *grpcMetrics) collectors() []prometheus.Collector {
	collectors := []prometheus.Collector{g.enabled}
	if g.requests != nil {
		collectors = append(collectors, g.requests.duration, g.requests.inFlight)
	}
	return collectors
}
