package defaults

import (
	"time"

	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/config"
)

// FullDefaultConfig returns a full sanitized config
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()
	EnsureDefaults(cfg)
	Sanitize(cfg)
	return cfg
}

// DefaultConfig is the default configuration
func DefaultConfig() *config.Config {
	return &config.Config{
		Debug: config.Debug{
			Addr:   "127.0.0.1:9255",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		Service: config.Service{
			Name: "postprocessing",
		},
		Postprocessing: config.Postprocessing{
			Events: config.Events{
				Endpoint:      "127.0.0.1:9233",
				Cluster:       "opencloud-cluster",
				MaxAckPending: 10_000,
				AckWait:       1 * time.Minute,
			},
			Workers:              3,
			RetryBackoffDuration: 5 * time.Second,
			MaxRetries:           14,
		},
		Store: config.Store{
			Store:    "nats-js-kv",
			Nodes:    []string{"127.0.0.1:9233"},
			Database: "postprocessing",
			Table:    "",
		},
	}
}

// EnsureDefaults ensures defaults on a config
func EnsureDefaults(cfg *config.Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = "error"
	}
}

// Sanitize does nothing atm
func Sanitize(cfg *config.Config) {
}
