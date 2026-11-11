package metrics

import "github.com/prometheus/client_golang/prometheus"

type eventsMetrics struct {
	enabled   prometheus.Gauge
	processed *prometheus.CounterVec
}

func newEventsMetrics(namespace, subsystem string) *eventsMetrics {
	return &eventsMetrics{
		enabled: newEnabledGauge(namespace, subsystem, APIEvents, "events"),
		processed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "events",
			Help:      "Number of consumed events",
		}, []string{LabelEvent, LabelResult}),
	}
}

// WithEvents declares that the service consumes events, and whether it is enabled.
func WithEvents(enabled bool) Option {
	return func(m *Metrics) {
		if m.Events == nil {
			m.Events = newEventsMetrics(m.namespace, m.subsystem)
		}
		m.Events.setEnabled(enabled)
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
