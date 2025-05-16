package store

import (
	"context"
	"path"

	olog "github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/vmihailenco/msgpack/v5"
	microstore "go-micro.dev/v4/store"
)

// CachedMDC is cache for the metadataclient
type CachedMDC struct {
	cfg    *config.Config
	logger olog.Logger
	next   MetadataClient

	filesCache microstore.Store
	dirsCache  microstore.Store
}

// SimpleDownload caches the answer from SimpleDownload or returns the cached one
func (c *CachedMDC) SimpleDownload(ctx context.Context, id string) ([]byte, error) {
	if b, err := c.filesCache.Read(id); err == nil && len(b) == 1 {
		return b[0].Value, nil
	}
	b, err := c.next.SimpleDownload(ctx, id)
	if err != nil {
		return nil, err
	}

	err = c.filesCache.Write(&microstore.Record{
		Key:    id,
		Value:  b,
		Expiry: c.cfg.Metadata.Cache.TTL,
	})
	if err != nil {
		c.logger.Error().Err(err).Msg("SimpleDownload: failed to update to files cache")
	}
	return b, nil
}

// SimpleUpload caches the answer from SimpleUpload and invalidates the cache
func (c *CachedMDC) SimpleUpload(ctx context.Context, id string, content []byte) error {
	b, err := c.filesCache.Read(id)
	if err == nil && len(b) == 1 && string(b[0].Value) == string(content) {
		// no need to bug mdc
		return nil
	}

	err = c.next.SimpleUpload(ctx, id, content)
	if err != nil {
		return err
	}

	// invalidate caches
	if err = c.dirsCache.Delete(path.Dir(id)); err != nil {
		c.logger.Error().Err(err).Msg("failed to clear dirs cache")
	}

	err = c.filesCache.Write(&microstore.Record{
		Key:    id,
		Value:  content,
		Expiry: c.cfg.Metadata.Cache.TTL,
	})
	if err != nil {
		c.logger.Error().Err(err).Msg("SimpleUpload: failed to update to files cache")
	}
	return nil
}

// Delete invalidates the cache when operation was successful
func (c *CachedMDC) Delete(ctx context.Context, id string) error {
	if err := c.next.Delete(ctx, id); err != nil {
		return err
	}

	// invalidate caches
	_ = c.removePrefix(c.filesCache, id)
	_ = c.removePrefix(c.dirsCache, id)
	return nil
}

// ReadDir caches the response from ReadDir or returnes the cached one
func (c *CachedMDC) ReadDir(ctx context.Context, id string) ([]string, error) {
	i, err := c.dirsCache.Read(id)
	if err == nil && len(i) == 1 {
		var ret []string
		if err = msgpack.Unmarshal(i[0].Value, &ret); err == nil {
			return ret, nil
		}
		c.logger.Error().Err(err).Msg("failed to unmarshal entry from dirs cache")
	}

	s, err := c.next.ReadDir(ctx, id)
	if err != nil {
		return nil, err
	}

	var value []byte
	if value, err = msgpack.Marshal(s); err != nil {
		c.logger.Error().Err(err).Msg("failed to marshal ReadDir result for dirs cache")
		return s, err
	}
	err = c.dirsCache.Write(&microstore.Record{
		Key:    id,
		Value:  value,
		Expiry: c.cfg.Metadata.Cache.TTL,
	})
	if err != nil {
		c.logger.Error().Err(err).Msg("ReadDir: failed to update dirs cache")
	}

	return s, err
}

// MakeDirIfNotExist invalidates the cache
func (c *CachedMDC) MakeDirIfNotExist(ctx context.Context, id string) error {
	err := c.next.MakeDirIfNotExist(ctx, id)
	if err != nil {
		return err
	}

	// invalidate caches
	if err = c.dirsCache.Delete(path.Dir(id)); err != nil {
		c.logger.Error().Err(err).Msg("failed to clear dirs cache")
	}
	return nil
}

// Init instantiates the caches
func (c *CachedMDC) Init(ctx context.Context, id string) error {
	c.dirsCache = store.Create(
		store.Store(c.cfg.Metadata.Cache.Store),
		store.TTL(c.cfg.Metadata.Cache.TTL),
		microstore.Nodes(c.cfg.Metadata.Cache.Nodes...),
		microstore.Database(c.cfg.Metadata.Cache.Database),
		microstore.Table(c.cfg.Metadata.Cache.DirectoryTable),
		store.DisablePersistence(c.cfg.Metadata.Cache.DisablePersistence),
		store.Authentication(c.cfg.Metadata.Cache.AuthUsername, c.cfg.Metadata.Cache.AuthPassword),
	)
	c.filesCache = store.Create(
		store.Store(c.cfg.Metadata.Cache.Store),
		store.TTL(c.cfg.Metadata.Cache.TTL),
		microstore.Nodes(c.cfg.Metadata.Cache.Nodes...),
		microstore.Database(c.cfg.Metadata.Cache.Database),
		microstore.Table(c.cfg.Metadata.Cache.FileTable),
		store.DisablePersistence(c.cfg.Metadata.Cache.DisablePersistence),
		store.Authentication(c.cfg.Metadata.Cache.AuthUsername, c.cfg.Metadata.Cache.AuthPassword),
	)
	return c.next.Init(ctx, id)
}

func (c *CachedMDC) removePrefix(cache microstore.Store, prefix string) error {
	c.logger.Debug().Str("prefix", prefix).Msg("removePrefix")
	keys, err := cache.List(microstore.ListPrefix(prefix))
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to list cache entries")
	}
	for _, k := range keys {
		c.logger.Debug().Str("key", k).Msg("removePrefix")
		if err := cache.Delete(k); err != nil {
			c.logger.Error().Err(err).Msg("failed to remove prefix from cache")
			return err
		}
	}
	return nil
}
