package backchannellogout

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/vmihailenco/msgpack/v5"
	"go-micro.dev/v4/store"
)

const expiryMetadataKey = "opencloud-oidc-expires-at"

// Cache preserves per-record expiry even on stores, such as NATS, that only
// support a bucket-wide TTL. The underlying store must have no bucket-wide TTL.
type Cache struct {
	store.Store
	cacheClaims bool
	now         func() time.Time
}

// NewCache wraps the dedicated OIDC cache. Disabling claims caching still keeps
// the session index and revocations needed to process logout requests.
func NewCache(underlying store.Store, cacheClaims bool) *Cache {
	return &Cache{Store: underlying, cacheClaims: cacheClaims, now: time.Now}
}

// List treats an empty NATS bucket like an empty store. The NATS adapter wraps
// ErrNoKeysFound when the bucket is new, expired, or contains only delete markers.
func (c *Cache) List(opts ...store.ListOption) ([]string, error) {
	return listCacheKeys(c.Store, opts...)
}

func listCacheKeys(cache store.Store, opts ...store.ListOption) ([]string, error) {
	keys, err := cache.List(opts...)
	if errors.Is(err, nats.ErrNoKeysFound) {
		return nil, nil
	}
	return keys, err
}

func isClaimsKey(key string) bool {
	decoded, err := keyEncoding.DecodeString(key)
	return err == nil && len(decoded) == 64
}

// Write records an absolute expiry alongside the data without modifying the
// caller's record. Native per-record expiry remains enabled where supported.
func (c *Cache) Write(record *store.Record, opts ...store.WriteOption) error {
	if !c.cacheClaims && isClaimsKey(record.Key) {
		return nil
	}
	r := *record
	r.Metadata = make(map[string]any, len(record.Metadata)+1)
	for key, value := range record.Metadata {
		r.Metadata[key] = value
	}
	options := store.WriteOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	expiresAt := "0"
	if options.TTL != 0 {
		expiresAt = strconv.FormatInt(c.now().Add(options.TTL).UnixNano(), 10)
		r.Expiry = options.TTL
	} else if !options.Expiry.IsZero() {
		expiresAt = strconv.FormatInt(options.Expiry.UnixNano(), 10)
		r.Expiry = options.Expiry.Sub(c.now())
	} else if record.Expiry != 0 {
		expiresAt = strconv.FormatInt(c.now().Add(record.Expiry).UnixNano(), 10)
	}
	r.Metadata[expiryMetadataKey] = expiresAt
	// Redis stores only Value and ignores Metadata and WriteExpiry. Preserve
	// the complete record in Value and also supply its native relative TTL.
	data, err := msgpack.Marshal(&r)
	if err != nil {
		return err
	}
	r.Value = data
	return c.Store.Write(&r, opts...)
}

// RevokeToken preserves the token's original absolute expiry. Security records
// for a given token must not acquire a new lifetime on each write.
func RevokeToken(record *store.Record, cache store.Store) error {
	var opts []store.WriteOption
	if value, ok := record.Metadata[expiryMetadataKey]; ok {
		nanos, err := strconv.ParseInt(fmt.Sprint(value), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid token expiry: %w", err)
		}
		if nanos != 0 {
			opts = append(opts, store.WriteExpiry(time.Unix(0, nanos)))
		}
	}
	return cache.Write(&store.Record{
		Key: RevokedTokenKey(string(record.Value)), Value: []byte{1}, Expiry: record.Expiry,
	}, opts...)
}

// Read excludes expired records and returns their remaining lifetime.
func (c *Cache) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	if !c.cacheClaims && isClaimsKey(key) {
		return nil, store.ErrNotFound
	}
	records, err := c.readRecords(key, opts...)
	if err != nil {
		return nil, err
	}
	active := make([]*store.Record, 0, len(records))
	for _, stored := range records {
		record := &store.Record{}
		if err := msgpack.Unmarshal(stored.Value, record); err != nil {
			return nil, fmt.Errorf("invalid OIDC cache record: %w", err)
		}
		expiry, err := c.expiry(record)
		if err != nil {
			return nil, err
		}
		if !expiry.IsZero() {
			remaining := expiry.Sub(c.now())
			if remaining <= 0 {
				// A failed cleanup must never make expired data usable again.
				_ = c.Store.Delete(record.Key)
				continue
			}
			record.Expiry = remaining
		}
		active = append(active, record)
	}
	return active, nil
}

func (c *Cache) readRecords(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	options := store.ReadOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	if !options.Prefix && !options.Suffix {
		return c.Store.Read(key, opts...)
	}
	// Read each listed key individually: the Redis plugin returns the search
	// prefix as every record's key, and the memory plugin can abort a prefix
	// read when any matching entry expires between listing and reading it.
	from := c.Store.Options()
	if options.Database != "" {
		from.Database = options.Database
	}
	if options.Table != "" {
		from.Table = options.Table
	}
	keys, err := c.List(store.ListFrom(from.Database, from.Table))
	if err != nil {
		return nil, err
	}
	var records []*store.Record
	for _, found := range keys {
		if (options.Prefix && !strings.HasPrefix(found, key)) || (options.Suffix && !strings.HasSuffix(found, key)) {
			continue
		}
		read, err := c.Store.Read(found, store.ReadFrom(from.Database, from.Table))
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		records = append(records, read...)
	}
	return records, nil
}

func (c *Cache) expiry(record *store.Record) (time.Time, error) {
	if value, ok := record.Metadata[expiryMetadataKey]; ok {
		nanos, err := strconv.ParseInt(fmt.Sprint(value), 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid OIDC cache expiry: %w", err)
		}
		if nanos == 0 {
			return time.Time{}, nil
		}
		return time.Unix(0, nanos), nil
	}
	// Entries written by an older proxy have no expiry metadata. Memory and
	// Redis supply their remaining TTL; legacy NATS claims contain exp instead.
	if record.Expiry > 0 {
		return c.now().Add(record.Expiry), nil
	}
	if isClaimsKey(record.Key) {
		var claims map[string]any
		if err := msgpack.Unmarshal(record.Value, &claims); err != nil {
			return time.Time{}, err
		}
		seconds, err := strconv.ParseInt(fmt.Sprint(claims["exp"]), 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(seconds, 0), nil
	}
	return time.Time{}, nil
}

// IndexLegacyTokens migrates claims cached by an older proxy before serving
// requests. A legacy session lookup contains only the last token's hash, so it
// is not sufficient to invalidate every token already present in the cache.
func (c *Cache) IndexLegacyTokens(ctx context.Context, legacy store.Store) error {
	keys, err := listCacheKeys(legacy)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !isClaimsKey(key) {
			continue
		}
		revoked, err := IsTokenRevoked(key, c)
		if err != nil {
			return err
		}
		if revoked {
			continue
		}
		records, err := legacy.Read(key)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		for _, record := range records {
			if _, current := record.Metadata[expiryMetadataKey]; current {
				continue
			}
			var claims map[string]any
			err := msgpack.Unmarshal(record.Value, &claims)
			expiresAt, expiryErr := c.expiry(record)
			if err != nil || expiryErr != nil || (!expiresAt.IsZero() && !c.now().Before(expiresAt)) {
				continue
			}
			subject, _ := claims["sub"].(string)
			session, _ := claims["sid"].(string)
			lookupKey, err := NewTokenKey(subject, session, key)
			if err == nil {
				existing, err := c.Read(lookupKey)
				if err != nil && !errors.Is(err, store.ErrNotFound) {
					return err
				}
				if len(existing) > 0 {
					continue
				}
				// Legacy exp may have been a fallback claims-cache TTL rather
				// than a token expiry. Do not guess when its revocation can end.
				if err := c.Write(&store.Record{Key: lookupKey, Value: []byte(key)}); err != nil {
					return err
				}
			} else {
				continue
			}
			var opts []store.WriteOption
			if !expiresAt.IsZero() {
				opts = append(opts, store.WriteExpiry(expiresAt))
			}
			if err := c.Write(record, opts...); err != nil {
				return err
			}
		}
	}
	return nil
}

// CollectExpired removes records that are no longer needed, including on stores
// without native per-record TTL support. It stops with the proxy's context.
func (c *Cache) CollectExpired(ctx context.Context, logger log.Logger) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.collectExpired(ctx); err != nil && ctx.Err() == nil {
				logger.Error().Err(err).Msg("failed to clean up OIDC cache")
			}
		}
	}
}

func (c *Cache) collectExpired(ctx context.Context) error {
	keys, err := c.List()
	if err != nil {
		return err
	}
	var cleanupErr error
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return err
		}
		records, err := c.Store.Read(key)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
			continue
		}
		for _, stored := range records {
			record := &store.Record{}
			if err := msgpack.Unmarshal(stored.Value, record); err != nil {
				cleanupErr = errors.Join(cleanupErr, err)
				continue
			}
			expiresAt, err := c.expiry(record)
			if err != nil {
				cleanupErr = errors.Join(cleanupErr, err)
				continue
			}
			if !expiresAt.IsZero() && !c.now().Before(expiresAt) {
				if err := c.Store.Delete(record.Key); err != nil && !errors.Is(err, store.ErrNotFound) {
					cleanupErr = errors.Join(cleanupErr, err)
				}
			}
		}
	}
	return cleanupErr
}
