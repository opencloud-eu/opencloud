package backchannellogout

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
	"go-micro.dev/v4/store"
)

// Like a NATS bucket with no bucket-wide TTL, this store preserves metadata but
// does not implement expiry supplied with individual writes.
type cacheWithoutTTL struct{ store.Store }

func (s cacheWithoutTTL) Write(record *store.Record, _ ...store.WriteOption) error {
	r := *record
	r.Expiry = 0
	return s.Store.Write(&r)
}

func TestCacheEnforcesExpiryWithoutNativeTTLSupport(t *testing.T) {
	backing := cacheWithoutTTL{store.NewMemoryStore()}
	cache := NewCache(backing, true)
	now := time.Now()
	cache.now = func() time.Time { return now }
	expiresAt := now.Add(time.Hour)
	record := &store.Record{Key: "record", Value: []byte("value"), Metadata: map[string]any{"original": "unchanged"}}
	require.NoError(t, cache.Write(record, store.WriteExpiry(expiresAt)))
	require.Equal(t, map[string]any{"original": "unchanged"}, record.Metadata)

	for range 3 {
		now = now.Add(10 * time.Minute)
		read, err := cache.Read(record.Key)
		require.NoError(t, err)
		require.Len(t, read, 1)
		require.Equal(t, expiresAt.Sub(now), read[0].Expiry, "reads must not extend expiry")
	}
	now = expiresAt
	read, err := cache.Read(record.Key)
	require.NoError(t, err)
	require.Empty(t, read)
	_, err = backing.Read(record.Key)
	require.ErrorIs(t, err, store.ErrNotFound)
}

func TestCachePreservesTokenRevocationLifetime(t *testing.T) {
	backing := cacheWithoutTTL{store.NewMemoryStore()}
	cache := NewCache(backing, true)
	now := time.Now()
	cache.now = func() time.Time { return now }
	expiresAt := now.Add(time.Hour)
	key, err := NewTokenKey("alice", "session", "token")
	require.NoError(t, err)
	require.NoError(t, cache.Write(&store.Record{Key: key, Value: []byte("token")}, store.WriteExpiry(expiresAt)))
	now = now.Add(time.Minute)
	records, err := cache.Read(key)
	require.NoError(t, err)
	require.Len(t, records, 1)
	now = now.Add(time.Minute)
	require.NoError(t, RevokeToken(records[0], cache))
	revoked, err := IsTokenRevoked("token", cache)
	require.NoError(t, err)
	require.True(t, revoked)

	now = expiresAt.Add(-time.Second)
	revoked, err = IsTokenRevoked("token", cache)
	require.NoError(t, err)
	require.True(t, revoked)
	now = expiresAt
	require.NoError(t, cache.collectExpired(context.Background()))
	revoked, err = IsTokenRevoked("token", cache)
	require.NoError(t, err)
	require.False(t, revoked)
}

func TestCacheRetainsRevocationsWithoutKnownExpiry(t *testing.T) {
	cache := NewCache(cacheWithoutTTL{store.NewMemoryStore()}, true)
	now := time.Now()
	cache.now = func() time.Time { return now }
	require.NoError(t, RevokeToken(&store.Record{Value: []byte("token")}, cache))
	now = now.AddDate(1, 0, 0)
	require.NoError(t, cache.collectExpired(context.Background()))
	revoked, err := IsTokenRevoked("token", cache)
	require.NoError(t, err)
	require.True(t, revoked)
}

func TestCacheCleanupContinuesAfterCorruptEntry(t *testing.T) {
	backing := cacheWithoutTTL{store.NewMemoryStore()}
	cache := NewCache(backing, true)
	now := time.Now()
	cache.now = func() time.Time { return now }
	require.NoError(t, cache.Write(&store.Record{Key: "expired", Expiry: time.Minute}))
	require.NoError(t, backing.Write(&store.Record{Key: "corrupt", Metadata: map[string]any{expiryMetadataKey: "bad expiry"}}))
	now = now.Add(time.Hour)
	require.Error(t, cache.collectExpired(context.Background()))
	_, err := backing.Read("expired")
	require.ErrorIs(t, err, store.ErrNotFound)
	_, err = backing.Read("corrupt")
	require.NoError(t, err, "corrupt security data must not be silently dropped")
}

func TestCacheCleanupStopsOnCancellation(t *testing.T) {
	cache := NewCache(store.NewMemoryStore(), true)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { cache.CollectExpired(ctx, log.NopLogger()); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cache cleanup did not stop")
	}
}

func TestCacheCanDisableClaimsWithoutDisablingLogout(t *testing.T) {
	cache := NewCache(store.NewMemoryStore(), false)
	key := base64.URLEncoding.EncodeToString(make([]byte, 64))
	require.NoError(t, cache.Write(&store.Record{Key: key, Value: []byte("claims")}))
	_, err := cache.Read(key)
	require.ErrorIs(t, err, store.ErrNotFound)
	require.NoError(t, RevokeToken(&store.Record{Value: []byte(key), Expiry: time.Hour}, cache))
	revoked, err := IsTokenRevoked(key, cache)
	require.NoError(t, err)
	require.True(t, revoked)
}

func TestCacheMigratesEveryLegacyToken(t *testing.T) {
	legacy := cacheWithoutTTL{store.NewMemoryStore()}
	cache := NewCache(cacheWithoutTTL{store.NewMemoryStore()}, true)
	now := time.Now()
	cache.now = func() time.Time { return now }
	var tokenKeys []string
	for i := range 2 {
		bytes := make([]byte, 64)
		bytes[0] = byte(i)
		key := base64.URLEncoding.EncodeToString(bytes)
		tokenKeys = append(tokenKeys, key)
		data, err := msgpack.Marshal(map[string]any{"sub": "alice", "sid": "session", "exp": now.Add(time.Hour).Unix()})
		require.NoError(t, err)
		require.NoError(t, legacy.Write(&store.Record{Key: key, Value: data}))
		lookupKey, err := NewKey("alice", "session")
		require.NoError(t, err)
		require.NoError(t, legacy.Write(&store.Record{Key: lookupKey, Value: []byte(key)}))
	}
	require.NoError(t, cache.IndexLegacyTokens(context.Background(), legacy))
	records, err := GetLogoutRecords(mustNewSuSe(t, "alice", "session"), cache)
	require.NoError(t, err)
	require.Len(t, records, 2)
	for _, record := range records {
		require.NoError(t, RevokeToken(record, cache))
		require.NoError(t, cache.Delete(record.Key))
		require.NoError(t, cache.Delete(string(record.Value)))
	}
	// Another proxy starting with the same old cache must not restore a logout.
	require.NoError(t, cache.IndexLegacyTokens(context.Background(), legacy))
	for _, key := range tokenKeys {
		revoked, err := IsTokenRevoked(key, cache)
		require.NoError(t, err)
		require.True(t, revoked)
		_, err = cache.Read(key)
		require.True(t, errors.Is(err, store.ErrNotFound))
	}
	legacyKeys, err := legacy.List()
	require.NoError(t, err)
	require.Len(t, legacyKeys, 3, "migration must not modify a shared legacy bucket")
}

func TestRevocationRejectsMalformedExpiry(t *testing.T) {
	cache := NewCache(store.NewMemoryStore(), true)
	err := RevokeToken(&store.Record{Value: []byte("token"), Metadata: map[string]any{expiryMetadataKey: strconv.Itoa(1) + "x"}}, cache)
	require.Error(t, err)
}
