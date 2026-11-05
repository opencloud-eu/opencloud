package metrics

import (
	ocmetrics "github.com/opencloud-eu/opencloud/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Namespace defines the namespace for the defines metrics.
	Namespace = "opencloud"

	// Subsystem defines the subsystem for the defines metrics.
	Subsystem = "webfinger"
)

// Metrics defines the available metrics of this service.
type Metrics struct {
	BuildInfo *prometheus.GaugeVec
	Counter   *prometheus.CounterVec
	Latency   *prometheus.SummaryVec
	Duration  *prometheus.HistogramVec
}

// New initializes the available metrics.
func New(opts ...Option) *Metrics {
	options := newOptions(opts...)

	m := &Metrics{
		BuildInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "build_info",
			Help:      "Build information",
		}, []string{"version"}),
		Counter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "webfinger_total",
			Help:      "How many webfinger requests processed",
		}, []string{}),
		Latency: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "webfinger_latency_microseconds",
			Help:      "Webfinger request latencies in microseconds",
		}, []string{}),
		Duration: prometheus.NewHistogramVec(ocmetrics.DualHistogramOpts(prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "webfinger_duration_seconds",
			Help:      "Webfinger request time in seconds",
		}), []string{}),
	}

	if err := prometheus.Register(m.BuildInfo); err != nil {
		options.Logger.Error().
			Err(err).
			Str("metric", "BuildInfo").
			Msg("Failed to register prometheus metric")
	}

	if err := prometheus.Register(m.Counter); err != nil {
		options.Logger.Error().
			Err(err).
			Str("metric", "counter").
			Msg("Failed to register prometheus metric")
	}

	if err := prometheus.Register(m.Latency); err != nil {
		options.Logger.Error().
			Err(err).
			Str("metric", "latency").
			Msg("Failed to register prometheus metric")
	}

	if err := prometheus.Register(m.Duration); err != nil {
		options.Logger.Error().
			Err(err).
			Str("metric", "duration").
			Msg("Failed to register prometheus metric")
	}

	return m
}
