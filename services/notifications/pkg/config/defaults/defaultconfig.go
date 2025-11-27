package defaults

import (
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/config"
)

// FullDefaultConfig returns a fully initialized default configuration
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()
	EnsureDefaults(cfg)
	Sanitize(cfg)
	return cfg
}

// NOTE: Most of this configuration is not needed to keep it as simple as possible
// TODO: Clean up unneeded configuration

// DefaultConfig returns a basic default configuration
func DefaultConfig() *config.Config {
	return &config.Config{
		Debug: config.Debug{
			Addr:   "127.0.0.1:9174",
			Zpages: false,
			Pprof:  false,
		},
		Service: config.Service{
			Name: "notifications",
		},
		WebUIURL: "https://localhost:9200",
		Notifications: config.Notifications{
			SMTP: config.SMTP{
				Encryption: "none",
			},
			Events: config.Events{
				Endpoint:  "127.0.0.1:9233",
				Cluster:   "opencloud-cluster",
				EnableTLS: false,
			},
			RevaGateway: shared.DefaultRevaConfig().Address,
		},
		Store: config.Store{
			Store:    "nats-js-kv",
			Nodes:    []string{"127.0.0.1:9233"},
			Database: "notifications",
			Table:    "",
			TTL:      336 * time.Hour,
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

	if cfg.Notifications.GRPCClientTLS == nil && cfg.Commons != nil {
		cfg.Notifications.GRPCClientTLS = structs.CopyOrZeroValue(cfg.Commons.GRPCClientTLS)
	}
}

// Sanitize sanitizes the configuration
func Sanitize(cfg *config.Config) {
	// nothing to sanitize here atm
}
