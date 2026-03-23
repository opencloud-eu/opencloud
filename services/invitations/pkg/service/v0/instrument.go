package service

import (
	"context"

	"github.com/opencloud-eu/opencloud/services/invitations/pkg/invitations"
	"github.com/opencloud-eu/opencloud/services/invitations/pkg/metrics"
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

func (i instrument) DeleteById(ctx context.Context, id string) error {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000
		i.metrics.Latency.WithLabelValues().Observe(us)
		i.metrics.Duration.WithLabelValues().Observe(v)
	}))

	defer timer.ObserveDuration()
	i.metrics.Counter.WithLabelValues().Inc()

	return i.next.DeleteById(ctx, id)
}

func (i instrument) DeleteByInvitedEmail(ctx context.Context, email string) error {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000
		i.metrics.Latency.WithLabelValues().Observe(us)
		i.metrics.Duration.WithLabelValues().Observe(v)
	}))

	defer timer.ObserveDuration()
	i.metrics.Counter.WithLabelValues().Inc()

	return i.next.DeleteByInvitedEmail(ctx, email)
}

func (i instrument) List(ctx context.Context, userId string) ([]*invitations.Invitation, error) {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000

		i.metrics.Latency.WithLabelValues().Observe(us)
		i.metrics.Duration.WithLabelValues().Observe(v)
	}))

	defer timer.ObserveDuration()

	i.metrics.Counter.WithLabelValues().Inc()

	return i.next.List(ctx, userId)
}

func (i instrument) GetByInvitedEmail(ctx context.Context, email string) (*invitations.Invitation, error) {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000

		i.metrics.Latency.WithLabelValues().Observe(us)
		i.metrics.Duration.WithLabelValues().Observe(v)
	}))

	defer timer.ObserveDuration()

	i.metrics.Counter.WithLabelValues().Inc()

	return i.next.GetByInvitedEmail(ctx, email)
}

func (i instrument) GetByInviteId(ctx context.Context, id string) (*invitations.Invitation, error) {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000

		i.metrics.Latency.WithLabelValues().Observe(us)
		i.metrics.Duration.WithLabelValues().Observe(v)
	}))

	defer timer.ObserveDuration()

	i.metrics.Counter.WithLabelValues().Inc()

	return i.next.GetByInviteId(ctx, id)
}

// Invite implements the Service interface.
func (i instrument) Invite(ctx context.Context, invitation *invitations.Invitation) (*invitations.Invitation, error) {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000

		i.metrics.Latency.WithLabelValues().Observe(us)
		i.metrics.Duration.WithLabelValues().Observe(v)
	}))

	defer timer.ObserveDuration()

	i.metrics.Counter.WithLabelValues().Inc()

	return i.next.Invite(ctx, invitation)
}
