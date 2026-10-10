package command

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	nserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	bcl "github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes/backchannellogout"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
	"go-micro.dev/v4/store"
)

func TestOIDCCacheBackends(t *testing.T) {
	for _, backend := range []string{"memory", "nats-js-kv", "redis"} {
		t.Run(backend, func(t *testing.T) {
			cfg := &config.Cache{Store: backend, Database: "cache-userinfo", TTL: time.Second}
			var natsURL string
			switch backend {
			case "nats-js-kv":
				server, err := nserver.NewServer(&nserver.Options{Host: "127.0.0.1", Port: -1, JetStream: true, StoreDir: t.TempDir()})
				require.NoError(t, err)
				go server.Start()
				t.Cleanup(func() { server.Shutdown(); server.WaitForShutdown() })
				require.True(t, server.ReadyForConnections(5*time.Second))
				natsURL = server.ClientURL()
				cfg.Nodes = []string{natsURL}
			case "redis":
				binary, err := exec.LookPath("redis-server")
				if err != nil {
					t.Skip("redis-server is required for the Redis backend integration test")
				}
				socket := filepath.Join(t.TempDir(), "redis.sock")
				cmd := exec.Command(binary, "--port", "0", "--unixsocket", socket, "--save", "", "--appendonly", "no")
				require.NoError(t, cmd.Start())
				t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
				require.Eventually(t, func() bool { _, err := os.Stat(socket); return err == nil }, 5*time.Second, 10*time.Millisecond)
				cfg.Nodes = []string{"unix://" + socket}
			}

			legacy := newUserInfoStore(cfg, cfg.Store, cfg.Database, cfg.Table, cfg.TTL)
			t.Cleanup(func() { _ = legacy.Close() })
			cache := newUserInfoCache(cfg)
			defer cache.Close()
			var tokenKeys []string
			data, err := msgpack.Marshal(map[string]any{"sub": "alice", "sid": "session", "exp": time.Now().Add(time.Hour).Unix()})
			require.NoError(t, err)
			for i := range 2 {
				bytes := make([]byte, 64)
				bytes[0] = byte(i)
				key := base64.URLEncoding.EncodeToString(bytes)
				tokenKeys = append(tokenKeys, key)
				require.NoError(t, legacy.Write(&store.Record{Key: key, Value: data, Expiry: time.Hour}))
				lookup, err := bcl.NewKey("alice", "session")
				require.NoError(t, err)
				require.NoError(t, legacy.Write(&store.Record{Key: lookup, Value: []byte(key), Expiry: time.Hour}))
			}
			// Persistent backends import all old claims, not just the last lookup.
			require.NoError(t, cache.IndexLegacyTokens(context.Background(), legacy))
			lookup, err := bcl.NewKey("alice", "session")
			require.NoError(t, err)
			suse, err := bcl.NewSuSe(lookup)
			require.NoError(t, err)
			records, err := bcl.GetLogoutRecords(suse, cache)
			require.NoError(t, err)
			require.Len(t, records, 2)
			for _, record := range records {
				require.NoError(t, bcl.RevokeToken(record, cache))
				require.NoError(t, cache.Delete(record.Key))
				require.NoError(t, cache.Delete(string(record.Value)))
			}
			// Exercise known token lifetimes independently of migration's unknown TTL.
			require.NoError(t, bcl.RevokeToken(&store.Record{Value: []byte("known-expiry"), Expiry: time.Hour}, cache))
			if backend != "memory" {
				peer := newUserInfoCache(cfg)
				t.Cleanup(func() { _ = peer.Close() })
				require.NoError(t, migrateUserInfoCache(context.Background(), peer, cfg))
				cache = peer
			}
			if natsURL != "" {
				conn, err := nats.Connect(natsURL)
				require.NoError(t, err)
				defer conn.Close()
				js, err := conn.JetStream()
				require.NoError(t, err)
				old, err := js.StreamInfo("KV_cache-userinfo")
				require.NoError(t, err)
				require.Equal(t, cfg.TTL, old.Config.MaxAge)
				current, err := js.StreamInfo("KV_cache-userinfo-oidc-v2")
				require.NoError(t, err)
				require.Zero(t, current.Config.MaxAge, "bucket TTL must not discard revocations")
				require.Eventually(t, func() bool {
					records, err := legacy.Read(tokenKeys[0])
					return (err == nil || err == store.ErrNotFound) && len(records) == 0
				}, 5*time.Second, 20*time.Millisecond)
			}
			for _, key := range append(tokenKeys, "known-expiry") {
				revoked, err := bcl.IsTokenRevoked(key, cache)
				require.NoError(t, err)
				require.True(t, revoked, "revocations must survive migration, another proxy, and the legacy bucket TTL")
			}
		})
	}
}
