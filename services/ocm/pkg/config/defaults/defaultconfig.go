package defaults

import (
	"path/filepath"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/ocm/pkg/config"
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
			Addr:   "127.0.0.1:9281",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		HTTP: config.HTTPConfig{
			Addr:      "127.0.0.1:9280",
			Namespace: "eu.opencloud.web",
			Protocol:  "tcp",
			Prefix:    "",
			CORS: config.CORS{
				AllowedOrigins: []string{"https://localhost:9200"},
				AllowedMethods: []string{
					"OPTIONS",
					"HEAD",
					"GET",
					"PUT",
					"POST",
					"DELETE",
					"MKCOL",
					"PROPFIND",
					"PROPPATCH",
					"MOVE",
					"COPY",
					"REPORT",
					"SEARCH",
				},
				AllowedHeaders: []string{
					"Origin",
					"Accept",
					"Content-Type",
					"Depth",
					"Authorization",
					"Ocs-Apirequest",
					"If-None-Match",
					"If-Match",
					"Destination",
					"Overwrite",
					"X-Request-Id",
					"X-Requested-With",
					"Tus-Resumable",
					"Tus-Checksum-Algorithm",
					"Upload-Concat",
					"Upload-Length",
					"Upload-Metadata",
					"Upload-Defer-Length",
					"Upload-Expires",
					"Upload-Checksum",
					"Upload-Offset",
					"X-HTTP-Method-Override",
					"Cache-Control",
				},
				AllowCredentials: false,
			},
		},
		GRPC: config.GRPCConfig{
			Addr:      "127.0.0.1:9282",
			Namespace: "eu.opencloud.api",
		},
		Reva: shared.DefaultRevaConfig(),
		Service: config.Service{
			Name: "ocm",
		},
		Events: config.Events{
			Endpoint: "127.0.0.1:9233",
			Cluster:  "opencloud-cluster",
		},
		ScienceMesh: config.ScienceMesh{
			Prefix:             "sciencemesh",
			InviteAcceptDialog: "/open-cloud-mesh/accept-invite",
		},
		OCMD: config.OCMD{
			Prefix: "ocm",
		},
		OCMInviteManager: config.OCMInviteManager{
			Driver: "json",
			Drivers: config.OCMInviteManagerDrivers{
				JSON: config.OCMInviteManagerJSONDriver{
					File: filepath.Join(defaults.BaseDataPath(), "storage", "ocm", "ocminvites.json"),
				},
			},
			TokenExpiration: 24 * time.Hour,
			Timeout:         30 * time.Second,
			Insecure:        false,
		},
		OCMProviderAuthorizerDriver: "json",
		OCMProviderAuthorizerDrivers: config.OCMProviderAuthorizerDrivers{
			JSON: config.OCMProviderAuthorizerJSONDriver{
				Providers: filepath.Join(defaults.BaseConfigPath(), "ocmproviders.json"),
			},
		},
		OCMShareProvider: config.OCMShareProvider{
			Driver: "json",
			Drivers: config.OCMShareProviderDrivers{
				JSON: config.OCMShareProviderJSONDriver{
					File: filepath.Join(defaults.BaseDataPath(), "storage", "ocm", "ocmshares.json"),
				},
			},
			Insecure: false,
		},
		OCMCore: config.OCMCore{
			Driver: "json",
			Drivers: config.OCMCoreDrivers{
				JSON: config.OCMCoreJSONDriver{
					File: filepath.Join(defaults.BaseDataPath(), "storage", "ocm", "ocmshares.json"),
				},
			},
		},
		OCMStorageProvider: config.OCMStorageProvider{
			Insecure:      false,
			StorageRoot:   filepath.Join(defaults.BaseDataPath(), "storage", "ocm"),
			DataServerURL: "http://localhost:9280/data",
		},
	}
}

// EnsureDefaults ensures the config contains default values
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

	if cfg.GRPCClientTLS == nil && cfg.Commons != nil {
		cfg.GRPCClientTLS = structs.CopyOrZeroValue(cfg.Commons.GRPCClientTLS)
	}

	if cfg.GRPC.TLS == nil && cfg.Commons != nil {
		cfg.GRPC.TLS = structs.CopyOrZeroValue(cfg.Commons.GRPCServiceTLS)
	}

	if (cfg.Commons != nil && cfg.Commons.OpenCloudURL != "") &&
		(cfg.HTTP.CORS.AllowedOrigins == nil ||
			len(cfg.HTTP.CORS.AllowedOrigins) == 1 &&
				cfg.HTTP.CORS.AllowedOrigins[0] == "https://localhost:9200") {
		cfg.HTTP.CORS.AllowedOrigins = []string{cfg.Commons.OpenCloudURL}
	}
}

// Sanitize sanitizes the config
func Sanitize(cfg *config.Config) {
	// nothing to sanitize here atm
}
