package backchannellogout

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	natsjskv "github.com/go-micro/plugins/v4/store/nats-js-kv"
	nserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/store"
)

func newNATSCleanupTestCache(t *testing.T) (*Cache, nats.JetStreamContext, nats.KeyValue) {
	t.Helper()
	server, err := nserver.NewServer(&nserver.Options{Host: "127.0.0.1", Port: -1, JetStream: true, StoreDir: t.TempDir()})
	require.NoError(t, err)
	go server.Start()
	t.Cleanup(func() { server.Shutdown(); server.WaitForShutdown() })
	require.True(t, server.ReadyForConnections(5*time.Second))
	options := nats.GetDefaultOptions()
	options.Servers = []string{server.ClientURL()}
	conn, err := options.Connect()
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	js, err := conn.JetStream()
	require.NoError(t, err)
	backing := natsjskv.NewStore(store.Nodes(server.ClientURL()), store.Database("oidc-cleanup"), store.Table("oidc"), natsjskv.EncodeKeys(), natsjskv.DefaultTTL(0))
	cache := NewCache(WithNATSCleanup(backing, options.Connect), true)
	t.Cleanup(func() { _ = cache.Close() })
	_, err = cache.List()
	require.NoError(t, err)
	bucket, err := js.KeyValue("oidc-cleanup")
	require.NoError(t, err)
	return cache, js, bucket
}

func TestNATSCleanupRemovesDeleteMarkers(t *testing.T) {
	cache, js, _ := newNATSCleanupTestCache(t)
	now := time.Now()
	cache.now = func() time.Time { return now }
	require.NoError(t, cache.collectExpired(context.Background()), "an empty bucket needs no cleanup")
	for i := range 20 {
		require.NoError(t, cache.Write(&store.Record{Key: fmt.Sprintf("expired-%d", i), Value: []byte("claims"), Expiry: time.Minute}))
	}
	require.NoError(t, RevokeToken(&store.Record{Value: []byte("active"), Expiry: time.Hour}, cache))
	now = now.Add(2 * time.Minute)
	require.NoError(t, cache.collectExpired(context.Background()))
	info, err := js.StreamInfo("KV_oidc-cleanup")
	require.NoError(t, err)
	require.EqualValues(t, 1, info.State.Msgs, "expired values and their markers must be physically removed")
	revoked, err := IsTokenRevoked("active", cache)
	require.NoError(t, err)
	require.True(t, revoked, "valid revocations must remain stored")

	require.NoError(t, cache.Delete(RevokedTokenKey("active")))
	keys, err := cache.List()
	require.NoError(t, err)
	require.Empty(t, keys)
	require.NoError(t, cache.collectExpired(context.Background()), "markers must be collected even when List returns no live keys")
	info, err = js.StreamInfo("KV_oidc-cleanup")
	require.NoError(t, err)
	require.Zero(t, info.State.Msgs)

	// Lazy expiry during Read also creates a marker, collected on the next pass.
	require.NoError(t, cache.Write(&store.Record{Key: "lazy", Expiry: time.Minute}))
	now = now.Add(2 * time.Minute)
	read, err := cache.Read("lazy")
	require.NoError(t, err)
	require.Empty(t, read)
	require.NoError(t, cache.collectExpired(context.Background()))
	info, err = js.StreamInfo("KV_oidc-cleanup")
	require.NoError(t, err)
	require.Zero(t, info.State.Msgs)
}

func TestNATSCleanupPreservesConcurrentWritesAndOtherTables(t *testing.T) {
	cache, js, bucket := newNATSCleanupTestCache(t)
	backend := cache.Store.(*natsCacheStore)
	encoder := backend.Store.(interface{ NatsKey(string, string) string })
	reusedKey := "reused"
	require.NoError(t, cache.Write(&store.Record{Key: reusedKey, Value: []byte("old")}))
	require.NoError(t, cache.Delete(reusedKey))
	otherKey := encoder.NatsKey("other-table", "deleted")
	_, err := bucket.Put(otherKey, []byte("other table"))
	require.NoError(t, err)
	require.NoError(t, bucket.Delete(otherKey))
	proxy := &beforeNATSPurge{JetStreamContext: js, before: func() {
		// Reuse the key after the snapshot, immediately before the purge.
		require.NoError(t, cache.Write(&store.Record{Key: reusedKey, Value: []byte("new"), Expiry: time.Hour}))
	}}
	require.NoError(t, backend.purgeDeleted(context.Background(), proxy))
	read, err := cache.Read(reusedKey)
	require.NoError(t, err)
	require.Len(t, read, 1)
	require.Equal(t, []byte("new"), read[0].Value)
	info, err := js.StreamInfo("KV_oidc-cleanup")
	require.NoError(t, err)
	require.EqualValues(t, 2, info.State.Msgs, "keep the concurrent value and the other table's marker")
}

func TestNATSCleanupReportsFailureAndRetries(t *testing.T) {
	cache, js, _ := newNATSCleanupTestCache(t)
	backend := cache.Store.(*natsCacheStore)
	require.NoError(t, cache.Write(&store.Record{Key: "deleted"}))
	require.NoError(t, cache.Delete("deleted"))
	failure := errors.New("purge unavailable")
	proxy := &beforeNATSPurge{JetStreamContext: js, err: failure}
	require.ErrorIs(t, backend.purgeDeleted(context.Background(), proxy), failure)
	info, err := js.StreamInfo("KV_oidc-cleanup")
	require.NoError(t, err)
	require.EqualValues(t, 1, info.State.Msgs, "keep the marker so cleanup can retry")
	connect := backend.connect
	backend.connect = func() (*nats.Conn, error) { return nil, failure }
	require.ErrorIs(t, cache.collectExpired(context.Background()), failure, "maintenance failures must reach the cleanup caller")
	backend.connect = connect
	require.NoError(t, backend.PurgeDeleted(context.Background()))
	info, err = js.StreamInfo("KV_oidc-cleanup")
	require.NoError(t, err)
	require.Zero(t, info.State.Msgs)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, backend.PurgeDeleted(ctx), context.Canceled)
}

type beforeNATSPurge struct {
	nats.JetStreamContext
	before func()
	err    error
}

func (p *beforeNATSPurge) PurgeStream(name string, opts ...nats.JSOpt) error {
	if p.before != nil {
		p.before()
		p.before = nil
	}
	if p.err != nil {
		return p.err
	}
	return p.JetStreamContext.PurgeStream(name, opts...)
}
