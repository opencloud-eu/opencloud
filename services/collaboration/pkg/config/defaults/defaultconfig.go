package defaults

import (
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
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
		Service: config.Service{
			Name: "collaboration",
		},
		App: config.App{
			Name:        "Collabora",
			Description: "Open office documents with Collabora",
			Icon:        "image-edit",
			Addr:        "https://127.0.0.1:9980",
			Insecure:    false,
			ProofKeys: config.ProofKeys{
				// they'll be enabled by default
				Duration: "12h",
			},
		},
		Store: config.Store{
			Store:    "nats-js-kv",
			Nodes:    []string{"127.0.0.1:9233"},
			Database: "collaboration",
			Table:    "",
			TTL:      30 * time.Minute,
		},
		GRPC: config.GRPC{
			Addr:      "127.0.0.1:9301",
			Protocol:  "tcp",
			Namespace: "eu.opencloud.api",
		},
		HTTP: config.HTTP{
			Addr:      "127.0.0.1:9300",
			Namespace: "eu.opencloud.web",
		},
		Debug: config.Debug{
			Addr:   "127.0.0.1:9304",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		Wopi: config.Wopi{
			WopiSrc: "https://localhost:9300",
		},
		CS3Api: config.CS3Api{
			Gateway: config.Gateway{
				Name: shared.DefaultRevaConfig().Address,
			},
			DataGateway: config.DataGateway{
				Insecure: false,
			},
			APPRegistrationInterval: 30 * time.Second,
		},
	}
}

// EnsureDefaults adds default values to the configuration if they are not set yet
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

	if cfg.TokenManager == nil && cfg.Commons != nil && cfg.Commons.TokenManager != nil {
		cfg.TokenManager = &config.TokenManager{
			JWTSecret: cfg.Commons.TokenManager.JWTSecret,
		}
	} else if cfg.TokenManager == nil {
		cfg.TokenManager = &config.TokenManager{}
	}
	if cfg.CS3Api.GRPCClientTLS == nil && cfg.Commons != nil {
		cfg.CS3Api.GRPCClientTLS = structs.CopyOrZeroValue(cfg.Commons.GRPCClientTLS)
	}
}

// Sanitize sanitized the configuration
func Sanitize(cfg *config.Config) {
	// sanitize config
}
