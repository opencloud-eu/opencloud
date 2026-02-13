package defaults

import (
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/config"
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
			Addr:   "127.0.0.1:9119",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		HTTP: config.HTTP{
			Addr:      "127.0.0.1:9115",
			Root:      "/",
			Namespace: "eu.opencloud.web",
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
				AllowCredentials: true,
			},
		},
		Service: config.Service{
			Name: "webdav",
		},
		OpenCloudPublicURL: "https://localhost:9200",
		WebdavNamespace:    "/users/{{.Id.OpaqueId}}",
		RevaGateway:        shared.DefaultRevaConfig().Address,
		OCDav: config.OCDav{
			Prefix:                "",
			SkipUserGroupsInToken: false,

			WebdavNamespace:            "/users/{{.Id.OpaqueId}}",
			FilesNamespace:             "/users/{{.Id.OpaqueId}}",
			SharesNamespace:            "/Shares",
			OCMNamespace:               "/public",
			PublicURL:                  "https://localhost:9200",
			Insecure:                   false,
			EnableHTTPTPC:              false,
			Timeout:                    84300,
			AllowPropfindDepthInfinity: false,
			NameValidation: config.NameValidation{
				InvalidChars: []string{"\f", "\r", "\n", "\\"},
				MaxLength:    255,
			},
		},
		FavoritesStore: config.FavoritesStore{
			Store: "memory",
		},
	}
}

// EnsureDefaults adds default values to the configuration if they are not set yet
func EnsureDefaults(cfg *config.Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = "error"
	}

	if cfg.GRPCClientTLS == nil && cfg.Commons != nil {
		cfg.GRPCClientTLS = structs.CopyOrZeroValue(cfg.Commons.GRPCClientTLS)
	}

	if cfg.Commons != nil {
		cfg.HTTP.TLS = cfg.Commons.HTTPServiceTLS
	}

	if cfg.OCDav.MachineAuthAPIKey == "" && cfg.Commons != nil && cfg.Commons.MachineAuthAPIKey != "" {
		cfg.OCDav.MachineAuthAPIKey = cfg.Commons.MachineAuthAPIKey
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
	// sanitize config
	if cfg.HTTP.Root != "/" {
		cfg.HTTP.Root = strings.TrimSuffix(cfg.HTTP.Root, "/")
	}

}
