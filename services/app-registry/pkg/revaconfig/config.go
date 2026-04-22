package revaconfig

import (
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/app-registry/pkg/config"
)

// AppRegistryConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func AppRegistryConfigFromStruct(cfg *config.Config, logger log.Logger) map[string]any {
	rcfg := map[string]any{
		"shared": map[string]any{
			"jwt_secret":           cfg.TokenManager.JWTSecret,
			"gatewaysvc":           cfg.Reva.Address,
			"grpc_client_options":  cfg.Reva.GetGRPCClientConfig(),
			"multi_tenant_enabled": cfg.Commons.MultiTenantEnabled,
		},
		"grpc": map[string]any{
			"network": cfg.GRPC.Protocol,
			"address": cfg.GRPC.Addr,
			"tls_settings": map[string]any{
				"enabled":     cfg.GRPC.TLS.Enabled,
				"certificate": cfg.GRPC.TLS.Cert,
				"key":         cfg.GRPC.TLS.Key,
			},
			"services": map[string]any{
				"appregistry": map[string]any{
					"driver": "static",
					"drivers": map[string]any{
						"static": map[string]any{
							"mime_types": mimetypes(cfg, logger),
						},
					},
				},
			},
			"interceptors": map[string]any{
				"prometheus": map[string]any{
					"namespace": "opencloud",
					"subsystem": "app_registry",
				},
			},
		},
	}
	return rcfg
}

func mimetypes(cfg *config.Config, logger log.Logger) []map[string]any {
	var m []map[string]any
	if err := mapstructure.Decode(cfg.AppRegistry.MimeTypeConfig, &m); err != nil {
		logger.Error().Err(err).Msg("Failed to decode appregistry mimetypes to mapstructure")
		return nil
	}
	return m
}
