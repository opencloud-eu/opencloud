package defaults

import (
	"path/filepath"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
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
			Addr:   "127.0.0.1:9104",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		HTTP: config.HTTP{
			Addr:      "127.0.0.1:9100",
			Root:      "/",
			Namespace: "eu.opencloud.web",
			CacheTTL:  604800, // 7 days

			CORS: config.CORS{
				AllowedOrigins: []string{"https://localhost:9200"},
				AllowedMethods: []string{
					"OPTIONS",
					"HEAD",
					"GET",
					"PUT",
					"PATCH",
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
				},
				AllowCredentials: false,
			},
		},
		Service: config.Service{
			Name: "web",
		},
		Asset: config.Asset{
			CorePath:   filepath.Join(defaults.BaseDataPath(), "web/assets/core"),
			AppsPath:   filepath.Join(defaults.BaseDataPath(), "web/assets/apps"),
			ThemesPath: filepath.Join(defaults.BaseDataPath(), "web/assets/themes"),
		},
		GatewayAddress: "eu.opencloud.api.gateway",
		Web: config.Web{
			ThemeServer: "https://localhost:9200",
			ThemePath:   "/themes/opencloud/theme.json",
			Config: config.WebConfig{
				Server: "https://localhost:9200",
				Theme:  "",
				OpenIDConnect: config.OIDC{
					MetadataURL:  "",
					Authority:    "https://localhost:9200",
					ClientID:     "web",
					ResponseType: "code",
					Scope:        "openid profile email",
				},
				Apps: []string{"files", "search", "text-editor", "pdf-viewer", "external", "admin-settings", "epub-reader", "preview", "app-store"},
				Options: config.Options{
					ContextHelpersReadMore: true,
					AccountEditLink:        &config.AccountEditLink{},
					Editor:                 &config.Editor{},
					FeedbackLink:           &config.FeedbackLink{},
					Embed:                  &config.Embed{},
					ConcurrentRequests: &config.ConcurrentRequests{
						Shares: &config.ConcurrentRequestsShares{},
					},
					Upload:                 &config.Upload{},
					TokenStorageLocal:      true,
					UserListRequiresFilter: false,
				},
			},
		},
	}
}

// EnsureDefaults adds default values to the configuration if they are not set yet
func EnsureDefaults(cfg *config.Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = "error"
	}

	if cfg.TokenManager == nil && cfg.Commons != nil && cfg.Commons.TokenManager != nil {
		cfg.TokenManager = &config.TokenManager{
			JWTSecret: cfg.Commons.TokenManager.JWTSecret,
		}
	} else if cfg.TokenManager == nil {
		cfg.TokenManager = &config.TokenManager{}
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
		cfg.HTTP.Root = strings.TrimRight(cfg.HTTP.Root, "/")
	}
	// build well known openid-configuration endpoint if it is not set
	if cfg.Web.Config.OpenIDConnect.MetadataURL == "" {
		cfg.Web.Config.OpenIDConnect.MetadataURL = strings.TrimRight(cfg.Web.Config.OpenIDConnect.Authority, "/") + "/.well-known/openid-configuration"
	}
	// remove AccountEdit parent if no value is set
	if cfg.Web.Config.Options.AccountEditLink.Href == "" {
		cfg.Web.Config.Options.AccountEditLink = nil
	}
	// remove Editor parent if no value is set
	if !cfg.Web.Config.Options.Editor.AutosaveEnabled {
		cfg.Web.Config.Options.Editor = nil
	}
	// remove FeedbackLink parent if no value is set
	if cfg.Web.Config.Options.FeedbackLink.Href == "" &&
		cfg.Web.Config.Options.FeedbackLink.AriaLabel == "" &&
		cfg.Web.Config.Options.FeedbackLink.Description == "" {
		cfg.Web.Config.Options.FeedbackLink = nil
	}
	// remove Upload parent if no value is set
	if cfg.Web.Config.Options.Upload.CompanionURL == "" {
		cfg.Web.Config.Options.Upload = nil
	}
	// remove Embed parent if no value is set
	if cfg.Web.Config.Options.Embed.Enabled == "" &&
		cfg.Web.Config.Options.Embed.Target == "" &&
		cfg.Web.Config.Options.Embed.MessagesOrigin == "" &&
		cfg.Web.Config.Options.Embed.DelegateAuthentication &&
		cfg.Web.Config.Options.Embed.DelegateAuthenticationOrigin == "" {
		cfg.Web.Config.Options.Embed = nil
	}
}
