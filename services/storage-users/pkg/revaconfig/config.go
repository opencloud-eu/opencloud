// Package revaconfig contains the config for the reva service
package revaconfig

import (
	"strconv"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
)

// StorageUsersConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func StorageUsersConfigFromStruct(cfg *config.Config) map[string]interface{} {
	rcfg := map[string]interface{}{
		"core": map[string]interface{}{
			"graceful_shutdown_timeout": cfg.GracefulShutdownTimeout,
		},
		"shared": map[string]interface{}{
			"jwt_secret":                cfg.TokenManager.JWTSecret,
			"gatewaysvc":                cfg.Reva.Address,
			"skip_user_groups_in_token": cfg.SkipUserGroupsInToken,
			"grpc_client_options":       cfg.Reva.GetGRPCClientConfig(),
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
				"storageprovider": map[string]interface{}{
					"driver":             cfg.Driver,
					"drivers":            StorageProviderDrivers(cfg),
					"mount_id":           cfg.MountID,
					"expose_data_server": cfg.ExposeDataServer,
					"data_server_url":    cfg.DataServerURL,
					"upload_expiration":  cfg.UploadExpiration,
					"events": map[string]interface{}{
						"nats_address":     cfg.Events.Addr,
						"nats_clusterid":   cfg.Events.ClusterID,
						"tls_insecure":     cfg.Events.TLSInsecure,
						"tls_root_ca_cert": cfg.Events.TLSRootCaCertPath,
						"nats_enable_tls":  cfg.Events.EnableTLS,
						"nats_username":    cfg.Events.AuthUsername,
						"nats_password":    cfg.Events.AuthPassword,
					},
				},
			},
			"interceptors": map[string]interface{}{
				"eventsmiddleware": map[string]interface{}{
					"group":            "sharing",
					"type":             "nats",
					"address":          cfg.Events.Addr,
					"clusterID":        cfg.Events.ClusterID,
					"tls-insecure":     cfg.Events.TLSInsecure,
					"tls-root-ca-cert": cfg.Events.TLSRootCaCertPath,
					"enable-tls":       cfg.Events.EnableTLS,
					"name":             generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus),
					"username":         cfg.Events.AuthUsername,
					"password":         cfg.Events.AuthPassword,
				},
				"prometheus": map[string]interface{}{
					"namespace": "opencloud",
					"subsystem": "storage_users",
				},
			},
		},
		"http": map[string]interface{}{
			"network": cfg.HTTP.Protocol,
			"address": cfg.HTTP.Addr,
			"middlewares": map[string]interface{}{
				"requestid": map[string]interface{}{},
			},
			// TODO build services dynamically
			"services": map[string]interface{}{
				"dataprovider": map[string]interface{}{
					"prefix":                 cfg.HTTP.Prefix,
					"driver":                 cfg.Driver,
					"drivers":                DataProviderDrivers(cfg),
					"nats_address":           cfg.Events.Addr,
					"nats_clusterID":         cfg.Events.ClusterID,
					"nats_tls_insecure":      cfg.Events.TLSInsecure,
					"nats_root_ca_cert_path": cfg.Events.TLSRootCaCertPath,
					"nats_enable_tls":        cfg.Events.EnableTLS,
					"nats_username":          cfg.Events.AuthUsername,
					"nats_password":          cfg.Events.AuthPassword,
					"data_txs": map[string]interface{}{
						"simple": map[string]interface{}{
							"cache_store":    "noop",
							"cache_database": "system",
							"cache_table":    "stat",
						},
						"spaces": map[string]interface{}{
							"cache_store":    "noop",
							"cache_database": "system",
							"cache_table":    "stat",
						},
						"tus": map[string]interface{}{
							"cache_store":    "noop",
							"cache_database": "system",
							"cache_table":    "stat",
							"cors_enabled":   true,
							// allow_origin is configured as a regex in tusd, so we concatenate the configured values into a regex
							"cors_allow_origin":      "(" + strings.ReplaceAll(strings.Join(cfg.HTTP.CORS.AllowedOrigins, "|"), "*", ".*") + ")",
							"cors_allow_credentials": cfg.HTTP.CORS.AllowCredentials,
							"cors_allow_methods":     strings.Join(cfg.HTTP.CORS.AllowedMethods, ","),
							"cors_allow_headers":     strings.Join(cfg.HTTP.CORS.AllowedHeaders, ","),
							"cors_max_age":           strconv.FormatUint(uint64(cfg.HTTP.CORS.MaxAge), 10),
							"cors_expose_headers":    strings.Join(cfg.HTTP.CORS.ExposedHeaders, ","),
						},
					},
				},
			},
		},
	}
	if cfg.ReadOnly {
		gcfg := rcfg["grpc"].(map[string]interface{})
		gcfg["interceptors"] = map[string]interface{}{
			"readonly": map[string]interface{}{},
		}
	}
	return rcfg
}
