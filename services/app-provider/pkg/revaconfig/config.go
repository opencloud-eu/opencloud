// Package revaconfig contains the config for the reva service
package revaconfig

import (
	"github.com/opencloud-eu/opencloud/services/app-provider/pkg/config"
)

// AppProviderConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func AppProviderConfigFromStruct(cfg *config.Config) map[string]interface{} {
	rcfg := map[string]interface{}{
		"shared": map[string]interface{}{
			"jwt_secret":           cfg.TokenManager.JWTSecret,
			"gatewaysvc":           cfg.Reva.Address,
			"grpc_client_options":  cfg.Reva.GetGRPCClientConfig(),
			"multi_tenant_enabled": cfg.Commons.MultiTenantEnabled,
		},
		"grpc": map[string]interface{}{
			"network": cfg.GRPC.Protocol,
			"address": cfg.GRPC.Addr,
			"tls_settings": map[string]interface{}{
				"enabled":     cfg.GRPC.TLS.Enabled,
				"certificate": cfg.GRPC.TLS.Cert,
				"key":         cfg.GRPC.TLS.Key,
			},
			"services": map[string]interface{}{
				"appprovider": map[string]interface{}{
					"app_provider_url": cfg.ExternalAddr,
					"driver":           cfg.Driver,
					"drivers": map[string]interface{}{
						"wopi": map[string]interface{}{
							"app_api_key":                   cfg.Drivers.WOPI.AppAPIKey,
							"app_desktop_only":              cfg.Drivers.WOPI.AppDesktopOnly,
							"app_icon_uri":                  cfg.Drivers.WOPI.AppIconURI,
							"app_int_url":                   cfg.Drivers.WOPI.AppInternalURL,
							"app_name":                      cfg.Drivers.WOPI.AppName,
							"app_url":                       cfg.Drivers.WOPI.AppURL,
							"app_disable_chat":              cfg.Drivers.WOPI.AppDisableChat,
							"insecure_connections":          cfg.Drivers.WOPI.Insecure,
							"iop_secret":                    cfg.Drivers.WOPI.IopSecret,
							"jwt_secret":                    cfg.TokenManager.JWTSecret,
							"wopi_url":                      cfg.Drivers.WOPI.WopiURL,
							"wopi_folder_url_base_url":      cfg.Drivers.WOPI.WopiFolderURLBaseURL,
							"wopi_folder_url_path_template": cfg.Drivers.WOPI.WopiFolderURLPathTemplate,
						},
					},
				},
			},
			"interceptors": map[string]interface{}{
				"prometheus": map[string]interface{}{
					"namespace": "opencloud",
					"subsystem": "app_provider",
				},
			},
		},
	}
	return rcfg
}
