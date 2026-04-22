package revaconfig

import (
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	pkgconfig "github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/services/storage-system/pkg/config"
)

// StorageSystemFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func StorageSystemFromStruct(cfg *config.Config) map[string]any {
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
			"services": map[string]any{
				"gateway": map[string]any{
					// registries are located on the gateway
					"authregistrysvc":    localEndpoint,
					"storageregistrysvc": localEndpoint,
					// user metadata is located on the users services
					"userprovidersvc":  localEndpoint,
					"groupprovidersvc": localEndpoint,
					"permissionssvc":   localEndpoint,
					// other
					"disable_home_creation_on_login": true, // metadata manually creates a space
					// metadata always uses the simple upload, so no transfer secret or datagateway needed
					"cache_store":    "noop",
					"cache_database": "system",
				},
				"userprovider": map[string]any{
					"driver": "memory",
					"drivers": map[string]any{
						"memory": map[string]any{
							"users": map[string]any{
								"serviceuser": map[string]any{
									"id": map[string]any{
										"opaqueId": cfg.SystemUserID,
										"idp":      "internal",
										"type":     userpb.UserType_USER_TYPE_SERVICE,
									},
									"username":     "serviceuser",
									"display_name": "System User",
								},
							},
						},
					},
				},
				"authregistry": map[string]any{
					"driver": "static",
					"drivers": map[string]any{
						"static": map[string]any{
							"rules": map[string]any{
								"machine": localEndpoint,
							},
						},
					},
				},
				"authprovider": map[string]any{
					"auth_manager": "machine",
					"auth_managers": map[string]any{
						"machine": map[string]any{
							"api_key":      cfg.SystemUserAPIKey,
							"gateway_addr": localEndpoint,
						},
					},
				},
				"permissions": map[string]any{
					"driver": "demo",
					"drivers": map[string]any{
						"demo": map[string]any{},
					},
				},
				"storageregistry": map[string]any{
					"driver": "static",
					"drivers": map[string]any{
						"static": map[string]any{
							"rules": map[string]any{
								"/": map[string]any{
									"address": localEndpoint,
								},
							},
						},
					},
				},
				"storageprovider": map[string]any{
					"driver":          cfg.Driver,
					"drivers":         metadataDrivers(localEndpoint, cfg),
					"data_server_url": cfg.DataServerURL,
				},
			},
			"interceptors": map[string]any{
				"prometheus": map[string]any{
					"namespace": "opencloud",
					"subsystem": "storage_system",
				},
			},
		},
		"http": map[string]any{
			"network": cfg.HTTP.Protocol,
			"address": cfg.HTTP.Addr,
			// no datagateway needed as the metadata clients directly talk to the dataprovider with the simple protocol
			"services": map[string]any{
				"dataprovider": map[string]any{
					"prefix":  "data",
					"driver":  cfg.Driver,
					"drivers": metadataDrivers(localEndpoint, cfg),
					"data_txs": map[string]any{
						"simple": map[string]any{
							"cache_store":    "noop",
							"cache_database": "system",
							"cache_table":    "stat",
						},
						"spaces": map[string]any{
							"cache_store":    "noop",
							"cache_database": "system",
							"cache_table":    "stat",
						},
						"tus": map[string]any{
							"cache_store":    "noop",
							"cache_database": "system",
							"cache_table":    "stat",
						},
					},
				},
			},
			"middlewares": map[string]any{
				"prometheus": map[string]any{
					"namespace": "opencloud",
					"subsystem": "storage_system",
				},
			},
		},
	}
	return rcfg
}

func metadataDrivers(localEndpoint string, cfg *config.Config) map[string]any {
	m := map[string]any{
		"metadata_backend":           "messagepack",
		"root":                       cfg.Drivers.Decomposed.Root,
		"user_layout":                "{{.Id.OpaqueId}}",
		"treetime_accounting":        false,
		"treesize_accounting":        false,
		"permissionssvc":             localEndpoint,
		"max_acquire_lock_cycles":    cfg.Drivers.Decomposed.MaxAcquireLockCycles,
		"lock_cycle_duration_factor": cfg.Drivers.Decomposed.LockCycleDurationFactor,
		"multi_tenant_enabled":       false, // storage-system doesn't use tenants, even if it's enabled for storage-users
		"disable_versioning":         true,
		"statcache": map[string]any{
			"cache_store":    "noop",
			"cache_database": "system",
		},
		"filemetadatacache": map[string]any{
			"cache_store":               cfg.FileMetadataCache.Store,
			"cache_nodes":               cfg.FileMetadataCache.Nodes,
			"cache_database":            cfg.FileMetadataCache.Database,
			"cache_ttl":                 cfg.FileMetadataCache.TTL,
			"cache_disable_persistence": cfg.FileMetadataCache.DisablePersistence,
			"cache_auth_username":       cfg.FileMetadataCache.AuthUsername,
			"cache_auth_password":       cfg.FileMetadataCache.AuthPassword,
		},
	}

	return map[string]any{
		"ocis":       m, // deprecated: use decomposed
		"decomposed": m,
	}
}
