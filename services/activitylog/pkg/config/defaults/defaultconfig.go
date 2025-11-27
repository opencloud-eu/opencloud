package defaults

import (
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/config"
)

// FullDefaultConfig returns the full default config
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()
	EnsureDefaults(cfg)
	Sanitize(cfg)
	return cfg
}

// DefaultConfig return the default configuration
func DefaultConfig() *config.Config {
	return &config.Config{
		Debug: config.Debug{
			Addr:   "127.0.0.1:9197",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		Service: config.Service{
			Name: "activitylog",
		},
		Events: config.Events{
			Endpoint:  "127.0.0.1:9233",
			Cluster:   "opencloud-cluster",
			EnableTLS: false,
		},
		Store: config.Store{
			Store:    "nats-js-kv",
			Nodes:    []string{"127.0.0.1:9233"},
			Database: "activitylog",
			Table:    "",
		},
		RevaGateway:     shared.DefaultRevaConfig().Address,
		DefaultLanguage: "en",
		HTTP: config.HTTP{
			Addr:      "127.0.0.1:9195",
			Root:      "/",
			Namespace: "eu.opencloud.web",
			CORS: config.CORS{
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET"},
				AllowedHeaders:   []string{"Authorization", "Origin", "Content-Type", "Accept", "X-Requested-With", "X-Request-Id", "Ocs-Apirequest"},
				AllowCredentials: true,
			},
		},
		WriteBufferDuration: 10 * time.Second,
		MaxActivities:       6000,
	}
}

// EnsureDefaults ensures the config contains default values
func EnsureDefaults(cfg *config.Config) {
	// provide with defaults for shared logging, since we need a valid destination address for "envdecode".
	if cfg.Log == nil && cfg.Commons != nil && cfg.Commons.Log != nil {
		cfg.Log = &config.Log{
			Level:  cfg.Commons.Log.Level,
			Pretty: cfg.Commons.Log.Pretty,
			Color:  cfg.Commons.Log.Color,
			File:   cfg.Commons.Log.File,
		}
	} else if cfg.Log == nil {
		cfg.Log = &config.Log{}
	}

	if cfg.GRPCClientTLS == nil && cfg.Commons != nil {
		cfg.GRPCClientTLS = structs.CopyOrZeroValue(cfg.Commons.GRPCClientTLS)
	}

	if cfg.TokenManager == nil && cfg.Commons != nil && cfg.Commons.TokenManager != nil {
		cfg.TokenManager = &config.TokenManager{
			JWTSecret: cfg.Commons.TokenManager.JWTSecret,
		}
	} else if cfg.TokenManager == nil {
		cfg.TokenManager = &config.TokenManager{}
	}

	if cfg.Commons != nil {
		cfg.HTTP.TLS = cfg.Commons.HTTPServiceTLS
	}
}

// Sanitize sanitizes the config
func Sanitize(cfg *config.Config) {
	// sanitize config
}
