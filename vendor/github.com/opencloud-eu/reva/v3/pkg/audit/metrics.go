package audit

import (
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// All services run in one binary, so the metrics are registered once and
// labelled by service.
var (
	emittedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "opencloud",
		Subsystem: "audit",
		Name:      "events_emitted_total",
		Help:      "Audit events handed to the emitter, including the ones dropped later.",
	}, []string{"service", "event", "outcome"})

	droppedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "opencloud",
		Subsystem: "audit",
		Name:      "events_dropped_total",
		Help:      "Audit events that were not stored, by event and reason.",
	}, []string{"service", "event", "reason"})

	publishDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "opencloud",
		Subsystem: "audit",
		Name:      "publish_duration_seconds",
		Help:      "Time to store an audit event in JetStream.",
		Buckets:   []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 5},
	}, []string{"service"})

	metricsOnce sync.Once
)

func registerMetrics() {
	metricsOnce.Do(func() {
		for _, c := range []prometheus.Collector{emittedTotal, droppedTotal, publishDuration} {
			if err := prometheus.Register(c); err != nil {
				var are prometheus.AlreadyRegisteredError
				if !errors.As(err, &are) {
					panic(err)
				}
			}
		}
	})
}
