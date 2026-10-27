package metrics

import "github.com/prometheus/client_golang/prometheus"

type eventsMetrics struct {
	enabled   prometheus.Gauge
	processed *prometheus.CounterVec
}

func newEventsMetrics(namespace, subsystem string) *eventsMetrics {
	return &eventsMetrics{
		enabled: newEnabledGauge(namespace, subsystem, APIEvents),
		processed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "events",
			Help:      "Number of consumed events",
		}, []string{LabelEvent, LabelResult}),
	}
}

func (e *eventsMetrics) RecordSuccess(event string) {
	if e == nil {
		return
	}
	e.processed.WithLabelValues(event, ResultSuccess).Inc()
}

func (e *eventsMetrics) RecordFailure(event string) {
	if e == nil {
		return
	}
	e.processed.WithLabelValues(event, ResultFailure).Inc()
}

func (e *eventsMetrics) setEnabled(enabled bool) {
	if e == nil {
		return
	}
	e.enabled.Set(boolToFloat64(enabled))
}

func (e *eventsMetrics) collectors() []prometheus.Collector {
	return []prometheus.Collector{e.enabled, e.processed}
}
