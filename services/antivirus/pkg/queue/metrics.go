package queue

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	metricNamespace = "opencloud"
	metricSubsystem = "antivirus"
	priorityLabel   = "priority"
)

var (
	defaultMetrics = &Metrics{
		JobsPending: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "jobs_pending",
			Help:      "Number of unclaimed antivirus jobs by priority.",
		}, []string{priorityLabel}),
		JobsInFlight: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "jobs_in_flight",
			Help:      "Number of claimed, unacknowledged antivirus jobs by priority.",
		}, []string{priorityLabel}),
		JobsEnqueued: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "jobs_enqueued_total",
			Help:      "Number of scan jobs durably enqueued by priority.",
		}, []string{priorityLabel}),
		JobsCompleted: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "jobs_completed_total",
			Help:      "Number of scan jobs finally acknowledged by priority.",
		}, []string{priorityLabel}),
		JobsDemoted: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "jobs_demoted_total",
			Help:      "Number of high-priority scan jobs moved to the low-priority lane.",
		}, []string{priorityLabel}),
		QueueWait: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: metricNamespace,
			Subsystem: metricSubsystem,
			Name:      "queue_wait_seconds",
			Help:      "Time scan jobs waited in the durable queue before a worker claimed them.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 300, 900},
		}, []string{priorityLabel}),
	}
)

// Metrics groups the antivirus queue metrics.
type Metrics struct {
	JobsPending   *prometheus.GaugeVec
	JobsInFlight  *prometheus.GaugeVec
	JobsEnqueued  *prometheus.CounterVec
	JobsCompleted *prometheus.CounterVec
	JobsDemoted   *prometheus.CounterVec
	QueueWait     *prometheus.HistogramVec
}

func newMetrics() *Metrics { return defaultMetrics }
