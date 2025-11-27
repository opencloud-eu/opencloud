package defaults

import (
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/app-registry/pkg/config"
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
			Addr:   "127.0.0.1:9243",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		GRPC: config.GRPCConfig{
			Addr:      "127.0.0.1:9242",
			Namespace: "eu.opencloud.api",
			Protocol:  "tcp",
		},
		Service: config.Service{
			Name: "app-registry",
		},
		Reva: shared.DefaultRevaConfig(),
	}
}

func defaultMimeTypeConfig() []config.MimeTypeConfig {
	return []config.MimeTypeConfig{
		{
			MimeType:    "application/pdf",
			Extension:   "pdf",
			Name:        "PDF",
			Description: "PDF document",
		},
		{
			MimeType:      "application/vnd.oasis.opendocument.text",
			Extension:     "odt",
			Name:          "OpenDocument",
			Description:   "OpenDocument text document",
			AllowCreation: true,
		},
		{
			MimeType:      "application/vnd.oasis.opendocument.spreadsheet",
			Extension:     "ods",
			Name:          "OpenSpreadsheet",
			Description:   "OpenDocument spreadsheet document",
			AllowCreation: true,
		},
		{
			MimeType:      "application/vnd.oasis.opendocument.presentation",
			Extension:     "odp",
			Name:          "OpenPresentation",
			Description:   "OpenDocument presentation document",
			AllowCreation: true,
		},
		{
			MimeType:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			Extension:     "docx",
			Name:          "Microsoft Word",
			Description:   "Microsoft Word document",
			AllowCreation: true,
		},
		{
			MimeType:      "application/vnd.openxmlformats-officedocument.wordprocessingml.form",
			Extension:     "docxf",
			Name:          "Form Document",
			Description:   "Form Document",
			AllowCreation: true,
		},
		{
			MimeType:      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			Extension:     "xlsx",
			Name:          "Microsoft Excel",
			Description:   "Microsoft Excel document",
			AllowCreation: true,
		},
		{
			MimeType:      "application/vnd.openxmlformats-officedocument.presentationml.presentation",
			Extension:     "pptx",
			Name:          "Microsoft PowerPoint",
			Description:   "Microsoft PowerPoint document",
			AllowCreation: true,
		},
		{
			MimeType:    "application/vnd.jupyter",
			Extension:   "ipynb",
			Name:        "Jupyter Notebook",
			Description: "Jupyter Notebook",
		},
		{
			MimeType:      "text/markdown",
			Extension:     "md",
			Name:          "Markdown file",
			Description:   "Markdown file",
			AllowCreation: true,
		},
		{
			MimeType:    "application/compressed-markdown",
			Extension:   "zmd",
			Name:        "Compressed markdown file",
			Description: "Compressed markdown file",
		},
		{
			MimeType:    "application/vnd.geogebra.slides",
			Extension:   "ggs",
			Name:        "GeoGebra Slides",
			Description: "GeoGebra Slides",
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

	if cfg.GRPC.TLS == nil && cfg.Commons != nil {
		cfg.GRPC.TLS = structs.CopyOrZeroValue(cfg.Commons.GRPCServiceTLS)
	}
}

// Sanitize the config
func Sanitize(cfg *config.Config) {
	if cfg.AppRegistry.MimeTypeConfig == nil {
		cfg.AppRegistry.MimeTypeConfig = defaultMimeTypeConfig()
	}
}
