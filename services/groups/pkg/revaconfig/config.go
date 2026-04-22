package revaconfig

import (
	"github.com/opencloud-eu/opencloud/services/groups/pkg/config"
)

// GroupsConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func GroupsConfigFromStruct(cfg *config.Config) map[string]any {
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
			// TODO build services dynamically
			"services": map[string]any{
				"groupprovider": map[string]any{
					"driver": cfg.Driver,
					"drivers": map[string]any{
						"json": map[string]any{
							"groups": cfg.Drivers.JSON.File,
						},
						"ldap": ldapConfigFromString(cfg.Drivers.LDAP),
						"rest": map[string]any{
							"client_id":           cfg.Drivers.REST.ClientID,
							"client_secret":       cfg.Drivers.REST.ClientSecret,
							"redis_address":       cfg.Drivers.REST.RedisAddr,
							"redis_username":      cfg.Drivers.REST.RedisUsername,
							"redis_password":      cfg.Drivers.REST.RedisPassword,
							"id_provider":         cfg.Drivers.REST.IDProvider,
							"api_base_url":        cfg.Drivers.REST.APIBaseURL,
							"oidc_token_endpoint": cfg.Drivers.REST.OIDCTokenEndpoint,
							"target_api":          cfg.Drivers.REST.TargetAPI,
						},
					},
				},
			},
			"interceptors": map[string]any{
				"prometheus": map[string]any{
					"namespace": "opencloud",
					"subsystem": "groups",
				},
			},
		},
	}
}

func ldapConfigFromString(cfg config.LDAPDriver) map[string]any {
	return map[string]any{
		"uri":                         cfg.URI,
		"cacert":                      cfg.CACert,
		"insecure":                    cfg.Insecure,
		"bind_username":               cfg.BindDN,
		"bind_password":               cfg.BindPassword,
		"user_base_dn":                cfg.UserBaseDN,
		"group_base_dn":               cfg.GroupBaseDN,
		"user_scope":                  cfg.UserScope,
		"group_scope":                 cfg.GroupScope,
		"group_substring_filter_type": cfg.GroupSubstringFilterType,
		"user_filter":                 cfg.UserFilter,
		"group_filter":                cfg.GroupFilter,
		"user_objectclass":            cfg.UserObjectClass,
		"group_objectclass":           cfg.GroupObjectClass,
		"idp":                         cfg.IDP,
		"user_schema": map[string]any{
			"id":              cfg.UserSchema.ID,
			"idIsOctetString": cfg.UserSchema.IDIsOctetString,
			"mail":            cfg.UserSchema.Mail,
			"displayName":     cfg.UserSchema.DisplayName,
			"userName":        cfg.UserSchema.Username,
		},
		"group_schema": map[string]any{
			"id":              cfg.GroupSchema.ID,
			"idIsOctetString": cfg.GroupSchema.IDIsOctetString,
			"mail":            cfg.GroupSchema.Mail,
			"displayName":     cfg.GroupSchema.DisplayName,
			"groupName":       cfg.GroupSchema.Groupname,
			"member":          cfg.GroupSchema.Member,
		},
	}
}
