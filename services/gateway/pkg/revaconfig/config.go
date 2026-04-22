package revaconfig

import (
	"encoding/json"
	"os"
	"strings"

	pkgconfig "github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/gateway/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// GatewayConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func GatewayConfigFromStruct(cfg *config.Config, logger log.Logger) map[string]any {
	localEndpoint := pkgconfig.LocalEndpoint(cfg.GRPC.Protocol, cfg.GRPC.Addr)

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
			// TODO build services dynamically
			"services": map[string]any{
				"gateway": map[string]any{
					"applicationauthsvc": cfg.AuthAppEndpoint,
					// registries are located on the gateway
					"authregistrysvc":    localEndpoint,
					"storageregistrysvc": localEndpoint,
					"appregistrysvc":     cfg.AppRegistryEndpoint,
					// user metadata is located on the users services
					"preferencessvc":   cfg.UsersEndpoint,
					"userprovidersvc":  cfg.UsersEndpoint,
					"groupprovidersvc": cfg.GroupsEndpoint,
					"permissionssvc":   cfg.PermissionsEndpoint,
					// sharing is located on the sharing service
					"usershareprovidersvc":          cfg.SharingEndpoint,
					"publicshareprovidersvc":        cfg.SharingEndpoint,
					"ocmshareprovidersvc":           cfg.OCMEndpoint,
					"ocminvitemanagersvc":           cfg.OCMEndpoint,
					"ocmproviderauthorizersvc":      cfg.OCMEndpoint,
					"ocmcoresvc":                    cfg.OCMEndpoint,
					"commit_share_to_storage_grant": cfg.CommitShareToStorageGrant,
					"share_folder":                  cfg.ShareFolder, // ShareFolder is the location where to create shares in the recipient's storage provider.
					// other
					"disable_home_creation_on_login": cfg.DisableHomeCreationOnLogin,
					"datagateway":                    strings.TrimRight(cfg.FrontendPublicURL, "/") + "/data",
					"transfer_shared_secret":         cfg.TransferSecret,
					"transfer_expires":               cfg.TransferExpires,
					// cache and TTLs
					"provider_cache_config": map[string]any{
						"cache_store":         cfg.Cache.ProviderCacheStore,
						"cache_nodes":         cfg.Cache.ProviderCacheNodes,
						"cache_database":      cfg.Cache.ProviderCacheDatabase,
						"cache_table":         "provider",
						"cache_ttl":           cfg.Cache.ProviderCacheTTL,
						"disable_persistence": cfg.Cache.ProviderCacheDisablePersistence,
						"cache_auth_username": cfg.Cache.ProviderCacheAuthUsername,
						"cache_auth_password": cfg.Cache.ProviderCacheAuthPassword,
					},
					"create_personal_space_cache_config": map[string]any{
						"cache_store":               cfg.Cache.CreateHomeCacheStore,
						"cache_nodes":               cfg.Cache.CreateHomeCacheNodes,
						"cache_database":            cfg.Cache.CreateHomeCacheDatabase,
						"cache_table":               "create_personal_space",
						"cache_ttl":                 cfg.Cache.CreateHomeCacheTTL,
						"cache_disable_persistence": cfg.Cache.CreateHomeCacheDisablePersistence,
						"cache_auth_username":       cfg.Cache.CreateHomeCacheAuthUsername,
						"cache_auth_password":       cfg.Cache.CreateHomeCacheAuthPassword,
					},
				},
				"authregistry": map[string]any{
					"driver": "static",
					"drivers": map[string]any{
						"static": map[string]any{
							"rules": map[string]any{
								"appauth":         cfg.AuthAppEndpoint,
								"basic":           cfg.AuthBasicEndpoint,
								"machine":         cfg.AuthMachineEndpoint,
								"publicshares":    cfg.StoragePublicLinkEndpoint,
								"serviceaccounts": cfg.AuthServiceEndpoint,
								"ocmshares":       cfg.OCMEndpoint,
							},
						},
					},
				},
				"storageregistry": map[string]any{
					"driver": cfg.StorageRegistry.Driver,
					"drivers": map[string]any{
						"spaces": map[string]any{
							"providers": spacesProviders(cfg, logger),
						},
					},
				},
			},
			"interceptors": map[string]any{
				"prometheus": map[string]any{
					"namespace": "opencloud",
					"subsystem": "gateway",
				},
			},
		},
	}
	return rcfg
}

func spacesProviders(cfg *config.Config, logger log.Logger) map[string]map[string]any {

	// if a list of rules is given it overrides the generated rules from below
	if len(cfg.StorageRegistry.Rules) > 0 {
		rules := map[string]map[string]any{}
		for i := range cfg.StorageRegistry.Rules {
			parts := strings.SplitN(cfg.StorageRegistry.Rules[i], "=", 2)
			rules[parts[0]] = map[string]any{"address": parts[1]}
		}
		return rules
	}

	// check if the rules have to be read from a json file
	if cfg.StorageRegistry.JSON != "" {
		data, err := os.ReadFile(cfg.StorageRegistry.JSON)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to read storage registry rules from JSON file: " + cfg.StorageRegistry.JSON)
			return nil
		}
		var rules map[string]map[string]any
		if err = json.Unmarshal(data, &rules); err != nil {
			logger.Error().Err(err).Msg("Failed to unmarshal storage registry rules")
			return nil
		}
		return rules
	}
	// generate rules based on default config
	return map[string]map[string]any{
		cfg.StorageUsersEndpoint: {
			"providerid": cfg.StorageRegistry.StorageUsersMountID,
			"spaces": map[string]any{
				"personal": map[string]any{
					"mount_point":   "/users",
					"path_template": "/users/{{.Space.Owner.Id.OpaqueId}}",
				},
				"project": map[string]any{
					"mount_point":   "/projects",
					"path_template": "/projects/{{.Space.Name}}",
				},
			},
		},
		cfg.StorageSharesEndpoint: {
			"providerid": utils.ShareStorageProviderID,
			"spaces": map[string]any{
				"virtual": map[string]any{
					// The root of the share jail is mounted here
					"mount_point": "/users/{{.CurrentUser.Id.OpaqueId}}/Shares",
				},
				"grant": map[string]any{
					// Grants are relative to a space root that the gateway will determine with a stat
					"mount_point": ".",
				},
				"mountpoint": map[string]any{
					// The jail needs to be filled with mount points
					// .Space.Name is a path relative to the mount point
					"mount_point":   "/users/{{.CurrentUser.Id.OpaqueId}}/Shares",
					"path_template": "/users/{{.CurrentUser.Id.OpaqueId}}/Shares/{{.Space.Name}}",
				},
			},
		},
		// public link storage returns the mount id of the actual storage
		cfg.StoragePublicLinkEndpoint: {
			"providerid": utils.PublicStorageProviderID,
			"spaces": map[string]any{
				"grant": map[string]any{
					"mount_point": ".",
				},
				"mountpoint": map[string]any{
					"mount_point":   "/public",
					"path_template": "/public/{{.Space.Root.OpaqueId}}",
				},
			},
		},
		cfg.OCMEndpoint: {
			"providerid": utils.OCMStorageProviderID,
			"spaces": map[string]any{
				"grant": map[string]any{
					"mount_point": ".",
				},
				"mountpoint": map[string]any{
					"mount_point":   "/ocm",
					"path_template": "/ocm/{{.Space.Root.OpaqueId}}",
				},
			},
		},
		// medatada storage not part of the global namespace
	}
}
