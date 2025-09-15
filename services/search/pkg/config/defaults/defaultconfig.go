package defaults

import (
	"path/filepath"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

// FullDefaultConfig returns a fully initialized default configuration
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()

	EnsureDefaults(cfg)

	return cfg
}

// DefaultConfig returns a basic default configuration
func DefaultConfig() *config.Config {
	return &config.Config{
		Debug: config.Debug{
			Addr:  "127.0.0.1:9224",
			Token: "",
		},
		GRPC: config.GRPCConfig{
			Addr:      "127.0.0.1:9220",
			Namespace: "eu.opencloud.api",
		},
		Service: config.Service{
			Name: "search",
		},
		Reva: shared.DefaultRevaConfig(),
		Engine: config.Engine{
			Type: "bleve",
			Bleve: config.EngineBleve{
				Datapath: filepath.Join(defaults.BaseDataPath(), "search"),
			},
			OpenSearch: config.EngineOpenSearch{
				ResourceIndex: config.EngineOpenSearchResourceIndex{
					Name: "opencloud-resource",
				},
			},
		},
		Extractor: config.Extractor{
			Type:             "basic",
			CS3AllowInsecure: false,
			Tika: config.ExtractorTika{
				TikaURL:        "http://127.0.0.1:9998",
				CleanStopWords: true,
			},
		},
		Events: config.Events{
			Endpoint:         "127.0.0.1:9233",
			Cluster:          "opencloud-cluster",
			DebounceDuration: 1000,
			AsyncUploads:     true,
			NumConsumers:     1,
			EnableTLS:        false,
			MaxAckPending:    1000,
			AckWait:          1 * time.Minute,
		},
		ContentExtractionSizeLimit: 20 * 1024 * 1024, // Limit content extraction to <20MB files by default
		BatchSize:                  500,
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
	// provide with defaults for shared tracing, since we need a valid destination address for "envdecode".
	if cfg.Tracing == nil && cfg.Commons != nil && cfg.Commons.Tracing != nil {
		cfg.Tracing = &config.Tracing{
			Enabled:   cfg.Commons.Tracing.Enabled,
			Type:      cfg.Commons.Tracing.Type,
			Endpoint:  cfg.Commons.Tracing.Endpoint,
			Collector: cfg.Commons.Tracing.Collector,
		}
	} else if cfg.Tracing == nil {
		cfg.Tracing = &config.Tracing{}
	}

	if cfg.TokenManager == nil && cfg.Commons != nil && cfg.Commons.TokenManager != nil {
		cfg.TokenManager = &config.TokenManager{
			JWTSecret: cfg.Commons.TokenManager.JWTSecret,
		}
	} else if cfg.TokenManager == nil {
		cfg.TokenManager = &config.TokenManager{}
	}

	if cfg.Reva == nil && cfg.Commons != nil {
		cfg.Reva = structs.CopyOrZeroValue(cfg.Commons.Reva)
	}
	if cfg.GRPCClientTLS == nil && cfg.Commons != nil {
		cfg.GRPCClientTLS = structs.CopyOrZeroValue(cfg.Commons.GRPCClientTLS)
	}
	if cfg.GRPC.TLS == nil && cfg.Commons != nil {
		cfg.GRPC.TLS = structs.CopyOrZeroValue(cfg.Commons.GRPCServiceTLS)
	}
}

// Sanitize sanitizes the configuration
func Sanitize(cfg *config.Config) {
	// no http endpoint to be sanitized
}
