package command

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	bcl "github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes/backchannellogout"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	microstore "go-micro.dev/v4/store"
)

func newUserInfoCache(cfg *config.Cache) *bcl.Cache {
	storeType := cfg.Store
	cacheClaims := storeType != store.TypeNoop
	if !cacheClaims || storeType == store.TypeOCMem {
		// Security records must not use the shared, capacity-evicted ocmem
		// cache, which also ignores the configured database namespace.
		storeType = store.TypeMemory
	}
	database := cfg.Database
	if database == "" {
		database = "cache-userinfo"
	}
	// Keep the no-TTL bucket separate from other services and older proxies
	// which may use the configured database with a bucket-wide TTL.
	// Redis ignores Database, so give its table a dedicated prefix as well.
	return bcl.NewCache(newUserInfoStore(cfg, storeType, database+"-oidc-v2", cfg.Table+"/oidc-v2/", 0), cacheClaims)
}

func migrateUserInfoCache(ctx context.Context, cache *bcl.Cache, cfg *config.Cache) error {
	// Memory and noop stores cannot contain entries from a previous process.
	if cfg.Store == "" || cfg.Store == "mem" || cfg.Store == store.TypeMemory || cfg.Store == store.TypeNoop || cfg.Store == store.TypeOCMem {
		return nil
	}
	legacy := newUserInfoStore(cfg, cfg.Store, cfg.Database, cfg.Table, cfg.TTL)
	defer legacy.Close()
	return cache.IndexLegacyTokens(ctx, legacy)
}

func newUserInfoStore(cfg *config.Cache, storeType, database, table string, ttl time.Duration) microstore.Store {
	return store.Create(
		store.Store(storeType),
		store.TTL(ttl),
		microstore.Nodes(cfg.Nodes...),
		microstore.Database(database),
		microstore.Table(table),
		store.DisablePersistence(cfg.DisablePersistence),
		store.Authentication(cfg.AuthUsername, cfg.AuthPassword),
		store.TLSEnabled(cfg.EnableTLS),
		store.TLSInsecure(cfg.TLSInsecure),
		store.TLSRootCA(cfg.TLSRootCACertificate),
	)
}
