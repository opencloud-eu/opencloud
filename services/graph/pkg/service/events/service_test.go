package events_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/prometheus/client_golang/prometheus"
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

	var wg sync.WaitGroup
	wg.Add(1)

	userId := fmt.Sprintf("user%d", 1000+rand.IntN(10000))

	backend := mocks.NewBackend(t)
	backend.EXPECT().UpdateLastSignInDate(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ time.Time) error {
		defer wg.Done()
		return nil
	})

	logger := log.NewLogger()

	reg := prometheus.NewRegistry()
	m, err := metrics.New(reg, &logger, func(_ []string) (string, string) { return "", "" })
	require.NoError(err)

	svc, err := g.NewService(ctx, bus, backend, m, &logger)
	require.NoError(err)
	t.Cleanup(func() { svc.Close() })
	t.Cleanup(cancel)
	go func() {
		require.NoError(svc.Start())
	}()

	metricstest.RequireEqual(t, 0, m.UnsupportedEvents)
	metricstest.RequireIsNotSet(t, m.EventsProcessed)

	_ = bus.Publish(events.UserSignedIn{
		Timestamp: nil,
		Executant: &userv1beta1.UserId{
			OpaqueId: userId,
		},
	})

	wg.Wait()
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

	var wg sync.WaitGroup
	wg.Add(1)

	userId := fmt.Sprintf("user%d", 1000+rand.IntN(10000))

	backend := mocks.NewBackend(t)
	backend.EXPECT().UpdateLastSignInDate(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, _ string, _ time.Time) error {
		defer wg.Done()
		return errors.New("test")
	})

	logger := log.NewLogger()

	reg := prometheus.NewRegistry()
	m, err := metrics.New(reg, &logger, func(_ []string) (string, string) { return "", "" })
	require.NoError(err)

	svc, err := g.NewService(ctx, bus, backend, m, &logger)
	require.NoError(err)
	t.Cleanup(func() { svc.Close() })
	t.Cleanup(cancel)
	go func() {
		require.NoError(svc.Start())
	}()

	metricstest.RequireEqual(t, 0, m.UnsupportedEvents)
	metricstest.RequireIsNotSet(t, m.EventsProcessed)

	_ = bus.Publish(events.UserSignedIn{
		Timestamp: nil,
		Executant: &userv1beta1.UserId{
			OpaqueId: userId,
		},
	})

	wg.Wait()
	require.Len(backend.Mock.Calls, 1)
	require.Len(backend.Mock.Calls[0].Arguments, 3)
	require.Equal(userId, backend.Mock.Calls[0].Arguments[1])

	metricstest.RequireEqual(t, 0, m.UnsupportedEvents)
	metricstest.RequireEqualWithLabels(t, 1, map[string]string{"event": "UserSignedIn", "result": "failure"}, m.EventsProcessed)
}
