package service

import (
	"context"
	"net/url"

	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/webfinger"
	"github.com/prometheus/client_golang/prometheus"
)

// NewInstrument returns a service that instruments metrics.
func NewInstrument(next Service, metrics *metrics.Metrics) Service {
	return instrument{
		next:    next,
		metrics: metrics,
	}
}

type instrument struct {
	next    Service
	metrics *metrics.Metrics
}

// Webfinger implements the Service interface.
func (i instrument) Webfinger(ctx context.Context, queryTarget *url.URL, rels []string, platform string) (webfinger.JSONResourceDescriptor, error) {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000

		i.metrics.Latency.WithLabelValues().Observe(us)
		i.metrics.Duration.WithLabelValues().Observe(v)
	}))

	defer timer.ObserveDuration()

	i.metrics.Counter.WithLabelValues().Inc()

	return i.next.Webfinger(ctx, queryTarget, rels, platform)
}
