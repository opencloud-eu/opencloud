package defaults

import (
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/ocdav/pkg/config"
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
			Addr:   "127.0.0.1:9163",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		HTTP: config.HTTPConfig{
			Addr:      "127.0.0.1:9350",
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
		Service: config.Service{
			Name: "ocdav",
		},
		Reva:              shared.DefaultRevaConfig(),
		WebdavNamespace:   "/users/{{.Id.OpaqueId}}",
		FilesNamespace:    "/users/{{.Id.OpaqueId}}",
		SharesNamespace:   "/Shares",
		OCMNamespace:      "/public",
		PublicURL:         "https://localhost:9200",
		Insecure:          false,
		Timeout:           84300,
		MachineAuthAPIKey: "",
		Status: config.Status{
			Version:        version.Legacy,
			VersionString:  version.LegacyString,
			ProductVersion: version.GetString(),
			Product:        "OpenCloud",
			ProductName:    "OpenCloud",
			Edition:        "",
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

// Sanitize sanitizes the configuration
func Sanitize(cfg *config.Config) {
	// nothing to sanitize here atm
}
