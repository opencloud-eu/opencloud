package defaults

import (
	"time"

	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/config"
)

// FullDefaultConfig returns a fully initialized default configuration which is needed for doc generation.
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()
	EnsureDefaults(cfg)
	Sanitize(cfg)
	return cfg
}

// DefaultConfig returns the default config
func DefaultConfig() *config.Config {
	return &config.Config{
		Service: config.Service{
			Name: "policies",
		},
		Debug: config.Debug{
			Addr:   "127.0.0.1:9129",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		GRPC: config.GRPC{
			Addr:      "127.0.0.1:9125",
			Namespace: "eu.opencloud.api",
		},
		Events: config.Events{
			Endpoint:  "127.0.0.1:9233",
			Cluster:   "opencloud-cluster",
			EnableTLS: false,
		},
		Engine: config.Engine{
			Timeout: 10 * time.Second,
		},
	}
}

func EnsureDefaults(cfg *config.Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = "error"
	}

	if cfg.GRPCClientTLS == nil && cfg.Commons != nil {
		cfg.GRPCClientTLS = structs.CopyOrZeroValue(cfg.Commons.GRPCClientTLS)
	}
	if cfg.GRPC.TLS == nil && cfg.Commons != nil {
		cfg.GRPC.TLS = structs.CopyOrZeroValue(cfg.Commons.GRPCServiceTLS)
	}

}

func Sanitize(_ *config.Config) {}
