package defaults

import (
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/gateway/pkg/config"
)

// FullDefaultConfig returns a fully initialized default configuration
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()
	EnsureDefaults(cfg)
	Sanitize(cfg)
	return cfg
}

// DefaultConfig returns a basic default configuration
func DefaultConfig() *config.Config {
	return &config.Config{
		Debug: config.Debug{
			Addr:   "127.0.0.1:9143",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		GRPC: config.GRPCConfig{
			Addr:      "127.0.0.1:9142",
			Namespace: "eu.opencloud.api",
			Protocol:  "tcp",
		},
		Service: config.Service{
			Name: "gateway",
		},
		Reva:                       shared.DefaultRevaConfig(),
		CommitShareToStorageGrant:  true,
		ShareFolder:                "Shares",
		DisableHomeCreationOnLogin: true,
		TransferExpires:            24 * 60 * 60,
		Cache: config.Cache{
			ProviderCacheStore:      "noop",
			ProviderCacheNodes:      []string{"127.0.0.1:9233"},
			ProviderCacheDatabase:   "cache-providers",
			ProviderCacheTTL:        300 * time.Second,
			CreateHomeCacheStore:    "memory",
			CreateHomeCacheNodes:    []string{"127.0.0.1:9233"},
			CreateHomeCacheDatabase: "cache-createhome",
			CreateHomeCacheTTL:      300 * time.Second,
		},

		FrontendPublicURL: "https://localhost:9200",

		AppRegistryEndpoint:       "eu.opencloud.api.app-registry",
		AuthAppEndpoint:           "eu.opencloud.api.auth-app",
		AuthBasicEndpoint:         "eu.opencloud.api.auth-basic",
		AuthMachineEndpoint:       "eu.opencloud.api.auth-machine",
		AuthServiceEndpoint:       "eu.opencloud.api.auth-service",
		GroupsEndpoint:            "eu.opencloud.api.groups",
		PermissionsEndpoint:       "eu.opencloud.api.settings",
		SharingEndpoint:           "eu.opencloud.api.sharing",
		StoragePublicLinkEndpoint: "eu.opencloud.api.storage-publiclink",
		StorageSharesEndpoint:     "eu.opencloud.api.storage-shares",
		StorageUsersEndpoint:      "eu.opencloud.api.storage-users",
		UsersEndpoint:             "eu.opencloud.api.users",
		OCMEndpoint:               "eu.opencloud.api.ocm",

		StorageRegistry: config.StorageRegistry{
			Driver: "spaces",
			JSON:   "",
		},
	}
}

// EnsureDefaults adds default values to the configuration if they are not set yet
func EnsureDefaults(cfg *config.Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = "error"
	}

	if cfg.Reva == nil && cfg.Commons != nil {
		cfg.Reva = structs.CopyOrZeroValue(cfg.Commons.Reva)
	}

	if cfg.TokenManager == nil && cfg.Commons != nil && cfg.Commons.TokenManager != nil {
		cfg.TokenManager = &config.TokenManager{
			JWTSecret: cfg.Commons.TokenManager.JWTSecret,
		}
	} else if cfg.TokenManager == nil {
		cfg.TokenManager = &config.TokenManager{}
	}

	if cfg.TransferSecret == "" && cfg.Commons != nil && cfg.Commons.TransferSecret != "" {
		cfg.TransferSecret = cfg.Commons.TransferSecret
	}

	if cfg.GRPC.TLS == nil && cfg.Commons != nil {
		cfg.GRPC.TLS = structs.CopyOrZeroValue(cfg.Commons.GRPCServiceTLS)
	}
}

// Sanitize sanitized the configuration
func Sanitize(cfg *config.Config) {
	// nothing to sanitize here atm
}
