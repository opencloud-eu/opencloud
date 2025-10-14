package defaults

import (
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/frontend/pkg/config"
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
			Addr:   "127.0.0.1:9141",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		HTTP: config.HTTPConfig{
			Addr:      "127.0.0.1:9140",
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
					"PATCH",
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
		Service: config.Service{
			Name: "frontend",
		},
		Reva:                     shared.DefaultRevaConfig(),
		PublicURL:                "https://localhost:9200",
		EnableFavorites:          false,
		UploadMaxChunkSize:       1e+7,
		UploadHTTPMethodOverride: "",
		DefaultUploadProtocol:    "tus",
		DefaultLinkPermissions:   1,
		SearchMinLength:          3,
		Edition:                  "",
		CheckForUpdates:          true,
		Checksums: config.Checksums{
			SupportedTypes:      []string{"sha1", "md5", "adler32"},
			PreferredUploadType: "sha1",
		},
		AppHandler: config.AppHandler{
			Prefix:            "app",
			SecureViewAppAddr: "eu.opencloud.api.collaboration",
		},
		Archiver: config.Archiver{
			Insecure:    false,
			Prefix:      "archiver",
			MaxNumFiles: 10000,
			MaxSize:     1073741824,
		},
		DataGateway: config.DataGateway{
			Prefix: "data",
		},
		OCS: config.OCS{
			Prefix:                      "ocs",
			SharePrefix:                 "/Shares",
			HomeNamespace:               "/users/{{.Id.OpaqueId}}",
			AdditionalInfoAttribute:     "{{.Mail}}",
			StatCacheType:               "memory",
			StatCacheNodes:              []string{"127.0.0.1:9233"},
			StatCacheDatabase:           "cache-stat",
			StatCacheTTL:                300 * time.Second,
			ListOCMShares:               true,
			PublicShareMustHavePassword: true,
			IncludeOCMSharees:           false,
		},
		Middleware: config.Middleware{
			Auth: config.Auth{
				CredentialsByUserAgent: map[string]string{},
			},
		},
		LDAPServerWriteEnabled: true,
		AutoAcceptShares:       true,
		Events: config.Events{
			Endpoint:  "127.0.0.1:9233",
			Cluster:   "opencloud-cluster",
			EnableTLS: false,
		},
		MaxConcurrency: 1,
		PasswordPolicy: config.PasswordPolicy{
			MinCharacters:          8,
			MinLowerCaseCharacters: 1,
			MinUpperCaseCharacters: 1,
			MinDigits:              1,
			MinSpecialCharacters:   1,
		},
		Groupware: config.Groupware{
			Enabled: false,
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

	if cfg.MachineAuthAPIKey == "" && cfg.Commons != nil && cfg.Commons.MachineAuthAPIKey != "" {
		cfg.MachineAuthAPIKey = cfg.Commons.MachineAuthAPIKey
	}

	if (cfg.Commons != nil && cfg.Commons.OpenCloudURL != "") &&
		(cfg.HTTP.CORS.AllowedOrigins == nil ||
			len(cfg.HTTP.CORS.AllowedOrigins) == 1 &&
				cfg.HTTP.CORS.AllowedOrigins[0] == "https://localhost:9200") {
		cfg.HTTP.CORS.AllowedOrigins = []string{cfg.Commons.OpenCloudURL}
	}
}

// Sanitize sanitized the configuration
func Sanitize(cfg *config.Config) {
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 5
	}
}
