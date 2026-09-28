package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	ResultFailure = "failure"

	LabelEvent = "event"
)

type eventsMetrics struct {
	processed *prometheus.CounterVec
}

func WithEventsProcessed() Option {
	return func(m *Metrics) {
		if m.events != nil {
			return
		}
		m.events = &eventsMetrics{
			processed: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: m.namespace,
				Subsystem: m.subsystem,
				Name:      "events",
				Help:      "Number of consumed events",
			}, []string{LabelEvent, LabelResult}),
		}
		m.collectors = append(m.collectors, m.events.processed)
	}
}

func (m *Metrics) RecordSuccess(event string) {
	if m.events == nil {
		return
	}
	m.events.processed.WithLabelValues(event, ResultSuccess).Inc()
}

func (m *Metrics) RecordFailure(event string) {
	if m.events == nil {
		return
	}
	m.events.processed.WithLabelValues(event, ResultFailure).Inc()
}
