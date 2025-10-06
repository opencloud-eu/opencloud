package revaconfig

import (
	"github.com/opencloud-eu/opencloud/services/users/pkg/config"
)

// UsersConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func UsersConfigFromStruct(cfg *config.Config) map[string]interface{} {
	rcfg := map[string]interface{}{
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
			// TODO build services dynamically
			"services": map[string]interface{}{
				"userprovider": map[string]interface{}{
					"driver": cfg.Driver,
					"drivers": map[string]interface{}{
						"json": map[string]interface{}{
							"users": cfg.Drivers.JSON.File,
						},
						"ldap": ldapConfigFromString(cfg.Drivers.LDAP),
						"owncloudsql": map[string]interface{}{
							"dbusername":           cfg.Drivers.OwnCloudSQL.DBUsername,
							"dbpassword":           cfg.Drivers.OwnCloudSQL.DBPassword,
							"dbhost":               cfg.Drivers.OwnCloudSQL.DBHost,
							"dbport":               cfg.Drivers.OwnCloudSQL.DBPort,
							"dbname":               cfg.Drivers.OwnCloudSQL.DBName,
							"idp":                  cfg.Drivers.OwnCloudSQL.IDP,
							"nobody":               cfg.Drivers.OwnCloudSQL.Nobody,
							"join_username":        cfg.Drivers.OwnCloudSQL.JoinUsername,
							"join_ownclouduuid":    cfg.Drivers.OwnCloudSQL.JoinOwnCloudUUID,
							"enable_medial_search": cfg.Drivers.OwnCloudSQL.EnableMedialSearch,
						},
					},
				},
			},
			"interceptors": map[string]interface{}{
				"prometheus": map[string]interface{}{
					"namespace": "opencloud",
					"subsystem": "users",
				},
			},
		},
	}
	return rcfg
}

func ldapConfigFromString(cfg config.LDAPDriver) map[string]interface{} {
	return map[string]interface{}{
		"uri":                        cfg.URI,
		"cacert":                     cfg.CACert,
		"insecure":                   cfg.Insecure,
		"bind_username":              cfg.BindDN,
		"bind_password":              cfg.BindPassword,
		"user_base_dn":               cfg.UserBaseDN,
		"group_base_dn":              cfg.GroupBaseDN,
		"user_scope":                 cfg.UserScope,
		"group_scope":                cfg.GroupScope,
		"user_substring_filter_type": cfg.UserSubstringFilterType,
		"user_filter":                cfg.UserFilter,
		"group_filter":               cfg.GroupFilter,
		"user_objectclass":           cfg.UserObjectClass,
		"group_objectclass":          cfg.GroupObjectClass,
		"user_disable_mechanism":     cfg.DisableUserMechanism,
		"user_enabled_property":      cfg.UserSchema.Enabled,
		"user_type_property":         cfg.UserTypeAttribute,
		"group_local_disabled_dn":    cfg.LdapDisabledUsersGroupDN,
		"idp":                        cfg.IDP,
		"user_schema": map[string]interface{}{
			"id":              cfg.UserSchema.ID,
			"tenantId":        cfg.UserSchema.TenantID,
			"idIsOctetString": cfg.UserSchema.IDIsOctetString,
			"mail":            cfg.UserSchema.Mail,
			"displayName":     cfg.UserSchema.DisplayName,
			"userName":        cfg.UserSchema.Username,
		},
		"group_schema": map[string]interface{}{
			"id":              cfg.GroupSchema.ID,
			"idIsOctetString": cfg.GroupSchema.IDIsOctetString,
			"mail":            cfg.GroupSchema.Mail,
			"displayName":     cfg.GroupSchema.DisplayName,
			"groupName":       cfg.GroupSchema.Groupname,
			"member":          cfg.GroupSchema.Member,
		},
	}
}
