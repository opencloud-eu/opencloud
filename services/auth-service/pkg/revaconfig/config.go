package revaconfig

import (
	"github.com/opencloud-eu/opencloud/services/auth-service/pkg/config"
)

// AuthMachineConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func AuthMachineConfigFromStruct(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"core": map[string]interface{}{
			"tracing_enabled":      cfg.Tracing.Enabled,
			"tracing_exporter":     cfg.Tracing.Type,
			"tracing_endpoint":     cfg.Tracing.Endpoint,
			"tracing_collector":    cfg.Tracing.Collector,
			"tracing_service_name": cfg.Service.Name,
		},
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
				"authprovider": map[string]interface{}{
					"auth_manager": "serviceaccounts",
					"auth_managers": map[string]interface{}{
						"serviceaccounts": map[string]interface{}{
							"service_accounts": []map[string]interface{}{
								{
									"id":     cfg.ServiceAccount.ServiceAccountID,
									"secret": cfg.ServiceAccount.ServiceAccountSecret,
								},
							},
						},
					},
				},
			},
			"interceptors": map[string]interface{}{
				"prometheus": map[string]interface{}{
					"namespace": "opencloud",
					"subsystem": "auth_service",
				},
			},
		},
	}
}
