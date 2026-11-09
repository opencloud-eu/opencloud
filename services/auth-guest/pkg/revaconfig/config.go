package revaconfig

import (
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
)

// GuestLinksConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func GuestLinksConfigFromStruct(cfg *config.Config) map[string]any {
	rcfg := map[string]any{
		"shared": map[string]any{
			"jwt_secret":           cfg.TokenManager.JWTSecret,
			"gatewaysvc":           cfg.RevaGateway,
			"grpc_client_options":  cfg.GRPCClientTLS,
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
				"authprovider": map[string]any{
					"auth_manager": "guestlinks",
					"auth_managers": map[string]any{
						"guestlinks": map[string]any{
							"gateway_addr":           cfg.RevaGateway,
							"jwt_secret":             cfg.SessionSecret(),
							"service_account_id":     cfg.ServiceAccount.ServiceAccountID,
							"service_account_secret": cfg.ServiceAccount.ServiceAccountSecret,
						},
					},
				},
			},
			"interceptors": map[string]any{
				"prometheus": map[string]any{
					"namespace": "opencloud",
					"subsystem": "auth_guest",
				},
			},
		},
	}
	return rcfg
}
