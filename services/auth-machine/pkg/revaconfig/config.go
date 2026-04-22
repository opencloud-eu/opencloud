package revaconfig

import (
	"github.com/opencloud-eu/opencloud/services/auth-machine/pkg/config"
)

// AuthMachineConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func AuthMachineConfigFromStruct(cfg *config.Config) map[string]any {
	return map[string]any{
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
				"authprovider": map[string]any{
					"auth_manager": "machine",
					"auth_managers": map[string]any{
						"machine": map[string]any{
							"api_key":      cfg.MachineAuthAPIKey,
							"gateway_addr": cfg.Reva.Address,
						},
					},
				},
			},
			"interceptors": map[string]any{
				"prometheus": map[string]any{
					"namespace": "opencloud",
					"subsystem": "auth_machine",
				},
			},
		},
	}
}
