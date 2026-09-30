package metrics

import "github.com/prometheus/client_golang/prometheus"

type eventsMetrics struct {
	processed *prometheus.CounterVec
}

// WithEventsProcessed enables the events processed metric.
func WithEventsProcessed() Option {
	return func(m *Metrics) {
		if m.Events != nil {
			return
		}
		m.Events = &eventsMetrics{
			processed: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: m.namespace,
				Subsystem: m.subsystem,
				Name:      "events",
				Help:      "Number of consumed events",
			}, []string{LabelEvent, LabelResult}),
		}
		m.collectors = append(m.collectors, m.Events.processed)
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
