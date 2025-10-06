package revaconfig

import (
	"github.com/opencloud-eu/opencloud/services/auth-machine/pkg/config"
)

// AuthMachineConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func AuthMachineConfigFromStruct(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"shared": map[string]interface{}{
			"jwt_secret":                cfg.TokenManager.JWTSecret,
			"gatewaysvc":                cfg.Reva.Address,
			"skip_user_groups_in_token": cfg.SkipUserGroupsInToken,
			"grpc_client_options":       cfg.Reva.GetGRPCClientConfig(),
			"multi_tenant_enabled":      cfg.Commons.MultiTenantEnabled,
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
				"authprovider": map[string]interface{}{
					"auth_manager": "machine",
					"auth_managers": map[string]interface{}{
						"machine": map[string]interface{}{
							"api_key":      cfg.MachineAuthAPIKey,
							"gateway_addr": cfg.Reva.Address,
						},
					},
				},
			},
			"interceptors": map[string]interface{}{
				"prometheus": map[string]interface{}{
					"namespace": "opencloud",
					"subsystem": "auth_machine",
				},
			},
		},
	}
}
