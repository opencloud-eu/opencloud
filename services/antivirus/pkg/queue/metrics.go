package queue

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	jobsPending = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "opencloud",
		Subsystem: "antivirus",
		Name:      "jobs_pending",
		Help:      "Number of unclaimed antivirus jobs by priority.",
	}, []string{"priority"})
	jobsInFlight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "opencloud",
		Subsystem: "antivirus",
		Name:      "jobs_in_flight",
		Help:      "Number of claimed, unacknowledged antivirus jobs by priority.",
	}, []string{"priority"})
	jobsEnqueued = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "opencloud",
		Subsystem: "antivirus",
		Name:      "jobs_enqueued_total",
		Help:      "Number of scan jobs durably enqueued by priority.",
	}, []string{"priority"})
	queueWait = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "opencloud",
		Subsystem: "antivirus",
		Name:      "queue_wait_seconds",
		Help:      "Time scan jobs waited in the durable queue before a worker claimed them.",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 300, 900},
	}, []string{"priority"})
)
