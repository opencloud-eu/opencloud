package command

import (
	"context"
	"errors"
	"testing"
	"time"

	nserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	bcl "github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes/backchannellogout"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/store"
)

func newOIDCNATSTestConfig(t *testing.T) (*config.Cache, nats.JetStreamContext) {
	t.Helper()
	username, password := "oidc-test", t.Name()
	server, err := nserver.NewServer(&nserver.Options{
		Host: "127.0.0.1", Port: -1, JetStream: true, StoreDir: t.TempDir(), Username: username, Password: password,
	})
	require.NoError(t, err)
	go server.Start()
	t.Cleanup(func() { server.Shutdown(); server.WaitForShutdown() })
	require.True(t, server.ReadyForConnections(5*time.Second))
	conn, err := nats.Connect(server.ClientURL(), nats.UserInfo(username, password))
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	js, err := conn.JetStream()
	require.NoError(t, err)
	return &config.Cache{
		Store: "nats-js-kv", Database: "cache-userinfo", Nodes: []string{server.ClientURL()}, TTL: time.Minute,
		AuthUsername: username, AuthPassword: password,
	}, js
}

func TestOIDCCacheMigratesEmptyNATS(t *testing.T) {
	for _, state := range []string{"new bucket", "expired entries", "delete markers only"} {
		t.Run(state, func(t *testing.T) {
			cfg, js := newOIDCNATSTestConfig(t)
			if state == "expired entries" {
				cfg.TTL = 100 * time.Millisecond
			}
			legacy := newUserInfoStore(cfg, cfg.Store, cfg.Database, cfg.Table, cfg.TTL)
			t.Cleanup(func() { _ = legacy.Close() })
			if state != "new bucket" {
				require.NoError(t, legacy.Write(&store.Record{Key: "old", Value: []byte("old claims")}))
				if state == "delete markers only" {
					require.NoError(t, legacy.Delete("old"))
				}
				bucket, err := js.KeyValue(cfg.Database)
				require.NoError(t, err)
				require.Eventually(t, func() bool {
					_, err := bucket.Keys()
					return errors.Is(err, nats.ErrNoKeysFound)
				}, 3*time.Second, 10*time.Millisecond)
			}
			cache := newUserInfoCache(cfg)
			t.Cleanup(func() { _ = cache.Close() })
			require.NoError(t, migrateUserInfoCache(context.Background(), cache, cfg), "empty legacy caches must not prevent proxy startup")
			key, err := bcl.NewKey("alice", "session")
			require.NoError(t, err)
			session, err := bcl.NewSuSe(key)
			require.NoError(t, err)
			_, err = bcl.GetLogoutRecords(session, cache)
			require.ErrorIs(t, err, store.ErrNotFound, "an empty current cache must behave as an already logged-out session")
		})
	}
}

func TestOIDCCacheWiresNATSMarkerCleanup(t *testing.T) {
	cfg, js := newOIDCNATSTestConfig(t)
	cache := newUserInfoCache(cfg)
	t.Cleanup(func() { _ = cache.Close() })
	require.NoError(t, cache.Write(&store.Record{Key: "deleted"}))
	require.NoError(t, cache.Delete("deleted"))
	legacy := newUserInfoStore(cfg, cfg.Store, cfg.Database, cfg.Table, cfg.TTL)
	t.Cleanup(func() { _ = legacy.Close() })
	require.NoError(t, legacy.Write(&store.Record{Key: "legacy"}))
	require.NoError(t, legacy.Delete("legacy"))
	cleaner, ok := cache.Store.(interface{ PurgeDeleted(context.Context) error })
	require.True(t, ok, "the configured NATS store must support marker maintenance")
	require.NoError(t, cleaner.PurgeDeleted(context.Background()), "cleanup must use the configured authentication")
	info, err := js.StreamInfo("KV_cache-userinfo-oidc-v2")
	require.NoError(t, err)
	require.Zero(t, info.State.Msgs)
	info, err = js.StreamInfo("KV_cache-userinfo")
	require.NoError(t, err)
	require.EqualValues(t, 1, info.State.Msgs, "cleanup must leave the legacy bucket untouched")
}
