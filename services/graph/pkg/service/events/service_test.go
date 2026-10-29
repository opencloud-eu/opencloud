package events_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/test-go/testify/mock"

	"github.com/opencloud-eu/opencloud/internal/eventstest"
	"github.com/opencloud-eu/opencloud/internal/metricstest"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	g "github.com/opencloud-eu/opencloud/services/graph/pkg/service/events"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/stretchr/testify/require"
)

func TestSuccessfulCall(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithCancel(t.Context())

	bus := eventstest.NewTestBus()

	userId := fmt.Sprintf("user%d", 1000+rand.IntN(10000))

	backend := mocks.NewBackend(t)
	backend.EXPECT().UpdateLastSignInDate(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ time.Time) error {
		return nil
	})

	logger := log.NewLogger()

	reg := prometheus.NewRegistry()
	m, err := metrics.New(reg, &logger, func(_ []string) (string, string) { return "", "" })
	require.NoError(err)

	svc, err := g.NewService(ctx, bus, backend, m, &logger)
	require.NoError(err)
	startConsumer(t, svc, cancel)

	metricstest.RequireEqual(t, 0, m.UnsupportedEvents)
	metricstest.RequireIsNotSet(t, m.EventsProcessed)

	_ = bus.Publish(events.UserSignedIn{
		Timestamp: nil,
		Executant: &userv1beta1.UserId{
			OpaqueId: userId,
		},
	})

	waitForProcessedEvent(t, m.EventsProcessed, "success")
	require.Len(backend.Mock.Calls, 1)
	require.Len(backend.Mock.Calls[0].Arguments, 3)
	require.Equal(userId, backend.Mock.Calls[0].Arguments[1])

	metricstest.RequireEqual(t, 0, m.UnsupportedEvents)
	metricstest.RequireEqualWithLabels(t, 1, map[string]string{"event": "UserSignedIn", "result": "success"}, m.EventsProcessed)
}

func TestBackendReturningAnError(t *testing.T) {
	require := require.New(t)

	ctx, cancel := context.WithCancel(t.Context())

	bus := eventstest.NewTestBus()

	userId := fmt.Sprintf("user%d", 1000+rand.IntN(10000))

	backend := mocks.NewBackend(t)
	backend.EXPECT().UpdateLastSignInDate(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ time.Time) error {
		return errors.New("test")
	})

	logger := log.NewLogger()

	reg := prometheus.NewRegistry()
	m, err := metrics.New(reg, &logger, func(_ []string) (string, string) { return "", "" })
	require.NoError(err)

	svc, err := g.NewService(ctx, bus, backend, m, &logger)
	require.NoError(err)
	startConsumer(t, svc, cancel)

	metricstest.RequireEqual(t, 0, m.UnsupportedEvents)
	metricstest.RequireIsNotSet(t, m.EventsProcessed)

	_ = bus.Publish(events.UserSignedIn{
		Timestamp: nil,
		Executant: &userv1beta1.UserId{
			OpaqueId: userId,
		},
	})

	waitForProcessedEvent(t, m.EventsProcessed, "failure")
	require.Len(backend.Mock.Calls, 1)
	require.Len(backend.Mock.Calls[0].Arguments, 3)
	require.Equal(userId, backend.Mock.Calls[0].Arguments[1])

	metricstest.RequireEqual(t, 0, m.UnsupportedEvents)
	metricstest.RequireEqualWithLabels(t, 1, map[string]string{"event": "UserSignedIn", "result": "failure"}, m.EventsProcessed)
}

func startConsumer(t *testing.T, svc g.GraphEventConsumer, cancel context.CancelFunc) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- svc.Start() }()
	t.Cleanup(func() {
		cancel()
		closeErr := svc.Close()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Error("event consumer did not stop")
		}
		require.NoError(t, closeErr)
	})
}

func waitForProcessedEvent(t *testing.T, eventsProcessed *prometheus.CounterVec, result string) {
	t.Helper()
	// The event is counted after the backend returns, so waiting for the mock
	// alone does not ensure the metric has been updated.
	counter := eventsProcessed.WithLabelValues("UserSignedIn", result)
	require.Eventually(t,
		func() bool {
			m := &dto.Metric{}
			return counter.Write(m) == nil && m.GetCounter().GetValue() == 1
		},
		5*time.Second, 5*time.Millisecond, "event processing metric was not updated")
}
