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
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
				AllowedHeaders:   []string{"Authorization", "Origin", "Content-Type", "Accept", "X-Requested-With", "X-Request-Id", "Cache-Control"},
				AllowCredentials: true,
			},
		},
		Service: config.Service{
			Name: "webdav",
		},
		OpenCloudPublicURL: "https://localhost:9200",
		WebdavNamespace:    "/users/{{.Id.OpaqueId}}",
		RevaGateway:        shared.DefaultRevaConfig().Address,

		ThumbnailGeneratorURL:     "http://127.0.0.1:9186",
		ThumbnailGeneratorTimeout: "30s",
		ThumbnailMaxInputFileSize: "50MB",
		ThumbnailCacheBackend:     "file",
		ThumbnailResolutions: []string{
			"16", "32", "64", "128", "192", "256", "320", "384", "448",
			"512", "640", "768", "1024", "1280", "1920", "2560", "3840", "7680",
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
}

// Sanitize sanitized the configuration
func Sanitize(cfg *config.Config) {
	// sanitize config
	if cfg.HTTP.Root != "/" {
		cfg.HTTP.Root = strings.TrimSuffix(cfg.HTTP.Root, "/")
	}

	// a yaml configuration that joined the resolutions into one entry kept
	// working in the thumbnails service, so it keeps working here
	if len(cfg.ThumbnailResolutions) == 1 && strings.Contains(cfg.ThumbnailResolutions[0], ",") {
		cfg.ThumbnailResolutions = strings.Split(cfg.ThumbnailResolutions[0], ",")
	}
}
