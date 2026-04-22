package revaconfig

import (
	"github.com/opencloud-eu/opencloud/services/storage-publiclink/pkg/config"
)

// StoragePublicLinkConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func StoragePublicLinkConfigFromStruct(cfg *config.Config) map[string]any {
	rcfg := map[string]any{
		"shared": map[string]any{
			"jwt_secret":                cfg.TokenManager.JWTSecret,
			"gatewaysvc":                cfg.Reva.Address,
			"skip_user_groups_in_token": cfg.SkipUserGroupsInToken,
			"grpc_client_options":       cfg.Reva.GetGRPCClientConfig(),
			"multi_tenant_enabled":      cfg.Commons.MultiTenantEnabled,
		},
		"grpc": map[string]any{
			"network": cfg.GRPC.Protocol,
			"address": cfg.GRPC.Addr,
			"tls_settings": map[string]any{
				"enabled":     cfg.GRPC.TLS.Enabled,
				"certificate": cfg.GRPC.TLS.Cert,
				"key":         cfg.GRPC.TLS.Key,
			},
			"interceptors": map[string]any{
				"log": map[string]any{},
				"prometheus": map[string]any{
					"namespace": "opencloud",
					"subsystem": "storage_publiclink",
				},
			},
			"services": map[string]any{
				"publicstorageprovider": map[string]any{
					"mount_id":     cfg.StorageProvider.MountID,
					"gateway_addr": cfg.Reva.Address,
				},
				"authprovider": map[string]any{
					"auth_manager": "publicshares",
					"auth_managers": map[string]any{
						"publicshares": map[string]any{
							"gateway_addr": cfg.Reva.Address,
						},
					},
				},
			},
		},
	}
	return rcfg
}
