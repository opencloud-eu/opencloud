package defaults

import (
	"strings"

	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/config"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/relations"
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
			Addr:   "127.0.0.1:9279",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		HTTP: config.HTTP{
			Addr:      "127.0.0.1:9275",
			Root:      "/",
			Namespace: "eu.opencloud.web",
			CORS: config.CORS{
				AllowedOrigins:   []string{"https://localhost:9200"},
				AllowCredentials: false,
			},
		},
		Service: config.Service{
			Name: "webfinger",
		},

		OpenCloudURL: "https://localhost:9200",
		Relations:    []string{relations.OpenIDConnectRel, relations.OpenCloudInstanceRel},
		Instances: []config.Instance{
			{
				Claim: "sub",
				Regex: ".+",
				Href:  "{{.OC_URL}}",
				Titles: map[string]string{
					"en": "OpenCloud Instance",
				},
			},
		},
		IDP:      "https://localhost:9200",
		Insecure: false,
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

	if cfg.Commons != nil {
		cfg.HTTP.TLS = cfg.Commons.HTTPServiceTLS
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
