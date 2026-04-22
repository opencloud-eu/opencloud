// Package revaconfig contains the reva config for storage-shares.
package revaconfig

import (
	"github.com/opencloud-eu/opencloud/services/storage-shares/pkg/config"
)

// StorageSharesConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func StorageSharesConfigFromStruct(cfg *config.Config) map[string]any {
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
			"services": map[string]any{
				"sharesstorageprovider": map[string]any{
					"usershareprovidersvc": cfg.SharesProviderEndpoint,
					"mount_id":             cfg.MountID,
				},
			},
			"interceptors": map[string]any{
				"prometheus": map[string]any{
					"namespace": "opencloud",
					"subsystem": "storage_shares",
				},
			},
		},
	}
	if cfg.ReadOnly {
		gcfg := rcfg["grpc"].(map[string]any)
		gcfg["interceptors"] = map[string]any{
			"readonly": map[string]any{},
		}
	}
	return rcfg
}
