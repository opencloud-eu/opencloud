package metrics

import (
	ocmetrics "github.com/opencloud-eu/opencloud/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Namespace defines the namespace for the defines metrics.
	Namespace = "opencloud"

	// Subsystem defines the subsystem for the defines metrics.
	Subsystem = "search"

	buildInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "build_info",
		Help:      "Build information",
	}, []string{"version"})
	eventsOutstandingAcks = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "events_outstanding_acks",
		Help:      "Number of outstanding acks for events",
	})
	eventsUnprocessed = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "events_unprocessed",
		Help:      "Number of unprocessed events",
	})
	eventsRedelivered = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "events_redelivered",
		Help:      "Number of redelivered events",
	})
	searchDuration = promauto.NewHistogramVec(ocmetrics.DualHistogramOpts(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "search_duration_seconds",
		Help:      "Duration of search operations in seconds",
		Buckets:   []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60},
	}), []string{"status"})
	indexDuration = promauto.NewHistogramVec(ocmetrics.DualHistogramOpts(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Name:      "index_duration_seconds",
		Help:      "Duration of indexing operations in seconds",
		Buckets:   []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600, 1200},
	}), []string{"status"})
)

// Metrics defines the available metrics of this service.
type Metrics struct {
	// Counter  *prometheus.CounterVec
	BuildInfo             *prometheus.GaugeVec
	EventsOutstandingAcks prometheus.Gauge
	EventsUnprocessed     prometheus.Gauge
	EventsRedelivered     prometheus.Gauge
	SearchDuration        *prometheus.HistogramVec
	IndexDuration         *prometheus.HistogramVec
}

// New initializes the available metrics.
func New() *Metrics {
	m := &Metrics{
		BuildInfo:             buildInfo,
		EventsOutstandingAcks: eventsOutstandingAcks,
		EventsUnprocessed:     eventsUnprocessed,
		EventsRedelivered:     eventsRedelivered,
		SearchDuration:        searchDuration,
		IndexDuration:         indexDuration,
	}

	return m
}
