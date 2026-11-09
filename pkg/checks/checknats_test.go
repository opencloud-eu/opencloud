package checks_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/test-go/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/checks"
	"github.com/opencloud-eu/opencloud/pkg/nats"
)

func TestNewNatsCheckWithAuth(t *testing.T) {
	ns, err := server.NewServer(&server.Options{
		Host:     "127.0.0.1",
		Port:     server.RANDOM_PORT,
		NoLog:    true,
		NoSigs:   true,
		Username: "opencloud",
		Password: "secret",
	})
	require.NoError(t, err)
	go ns.Start()
	t.Cleanup(ns.Shutdown)
	require.True(t, ns.ReadyForConnections(5*time.Second))

	addr := ns.ClientURL()
	ctx := context.Background()

	require.Error(t, checks.NewNatsCheck(addr)(ctx), "anonymous connection must be rejected")
	require.Error(t, checks.NewNatsCheck(addr, nats.Auth("opencloud", "wrong"))(ctx))
	require.NoError(t, checks.NewNatsCheck(addr, nats.Auth("opencloud", "secret"))(ctx))
}

func TestNewNatsCheckWithoutAuth(t *testing.T) {
	ns, err := server.NewServer(&server.Options{
		Host:   "127.0.0.1",
		Port:   server.RANDOM_PORT,
		NoLog:  true,
		NoSigs: true,
	})
	require.NoError(t, err)
	go ns.Start()
	t.Cleanup(ns.Shutdown)
	require.True(t, ns.ReadyForConnections(5*time.Second))

	// no username configured: Auth adds no option and the secure option may be nil as well
	require.NoError(t, checks.NewNatsCheck(ns.ClientURL(), nats.Secure(false, false, ""), nats.Auth("", ""))(context.Background()))
}
