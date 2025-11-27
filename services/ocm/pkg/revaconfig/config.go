package revaconfig

import (
	"math"
	"net/url"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/ocm/pkg/config"
)

// OCMConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func OCMConfigFromStruct(cfg *config.Config, logger log.Logger) map[string]interface{} {

	// Construct the ocm provider domain from the OpenCloud URL
	providerDomain := ""
	u, err := url.Parse(cfg.Commons.OpenCloudURL)
	switch {
	case err != nil:
		logger.Error().Err(err).Msg("could not parse OpenCloud URL")
	case u.Host == "OpenCloud":
		logger.Error().Msg("OpenCloud URL has no host")
	default:
		providerDomain = u.Host
	}

	return map[string]interface{}{
		"shared": map[string]interface{}{
			"jwt_secret":           cfg.TokenManager.JWTSecret,
			"gatewaysvc":           cfg.Reva.Address, // Todo or address?
			"grpc_client_options":  cfg.Reva.GetGRPCClientConfig(),
			"multi_tenant_enabled": cfg.Commons.MultiTenantEnabled,
		},
		"http": map[string]interface{}{
			"network": cfg.HTTP.Protocol,
			"address": cfg.HTTP.Addr,
			"middlewares": map[string]interface{}{
				"cors": map[string]interface{}{
					"allowed_origins":   cfg.HTTP.CORS.AllowedOrigins,
					"allowed_methods":   cfg.HTTP.CORS.AllowedMethods,
					"allowed_headers":   cfg.HTTP.CORS.AllowedHeaders,
					"allow_credentials": cfg.HTTP.CORS.AllowCredentials,
					// currently unused
					//"options_passthrough": ,
					//"debug": ,
					//"max_age": ,
					//"priority": ,
					//"exposed_headers": ,
				},
				"auth": map[string]interface{}{
					"credentials_by_user_agent": cfg.Middleware.Auth.CredentialsByUserAgent,
				},
				"prometheus": map[string]interface{}{
					"namespace": "opencloud",
					"subsystem": "ocm",
				},
				"requestid": map[string]interface{}{},
			},
			// TODO build services dynamically
			"services": map[string]interface{}{
				"wellknown": map[string]interface{}{
					"prefix": ".well-known",
					"ocmprovider": map[string]interface{}{
						"ocm_prefix":           cfg.OCMD.Prefix,
						"endpoint":             cfg.Commons.OpenCloudURL,
						"provider":             "OpenCloud",
						"webdav_root":          "/dav/ocm",
						"webapp_root":          cfg.ScienceMesh.Prefix,
						"invite_accept_dialog": cfg.ScienceMesh.InviteAcceptDialog,
						"enable_webapp":        false,
						"enable_datatx":        false,
					},
				},
				"sciencemesh": map[string]interface{}{
					"prefix":                 cfg.ScienceMesh.Prefix,
					"smtp_credentials":       map[string]string{},
					"gatewaysvc":             cfg.Reva.Address,
					"mesh_directory_url":     cfg.ScienceMesh.MeshDirectoryURL,
					"directory_service_urls": cfg.ScienceMesh.DirectoryServiceURLs,
					"provider_domain":        providerDomain,
					"events": map[string]interface{}{
						"natsaddress":          cfg.Events.Endpoint,
						"natsclusterid":        cfg.Events.Cluster,
						"tlsinsecure":          cfg.Events.TLSInsecure,
						"tlsrootcacertificate": cfg.Events.TLSRootCACertificate,
						"authusername":         cfg.Events.AuthUsername,
						"authpassword":         cfg.Events.AuthPassword,
					},
				},
				"ocmd": map[string]interface{}{
					"prefix":                        cfg.OCMD.Prefix,
					"gatewaysvc":                    cfg.Reva.Address,
					"expose_recipient_display_name": cfg.OCMD.ExposeRecipientDisplayName,
				},
				"dataprovider": map[string]interface{}{
					"prefix": "data",
					"driver": "ocmreceived",
					"drivers": map[string]interface{}{
						"ocmreceived": map[string]interface{}{
							"insecure":               cfg.OCMStorageProvider.Insecure,
							"storage_root":           cfg.OCMStorageProvider.StorageRoot,
							"service_account_id":     cfg.ServiceAccount.ID,
							"service_account_secret": cfg.ServiceAccount.Secret,
						},
					},
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
						},
					},
				},
			},
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
				"ocminvitemanager": map[string]interface{}{
					"driver": cfg.OCMInviteManager.Driver,
					"drivers": map[string]interface{}{
						"json": map[string]interface{}{
							"file": cfg.OCMInviteManager.Drivers.JSON.File,
						},
					},
					"provider_domain":  providerDomain,
					"token_expiration": cfg.OCMInviteManager.TokenExpiration.String(),
					"ocm_timeout":      int(math.Round(cfg.OCMInviteManager.Timeout.Seconds())),
					"ocm_insecure":     cfg.OCMInviteManager.Insecure,
				},
				"ocmproviderauthorizer": map[string]interface{}{
					"driver": cfg.OCMProviderAuthorizerDriver,
					"drivers": map[string]interface{}{
						"json": map[string]interface{}{
							"providers": cfg.OCMProviderAuthorizerDrivers.JSON.Providers,
						},
					},
				},
				"ocmshareprovider": map[string]interface{}{
					"driver": cfg.OCMShareProvider.Driver,
					"drivers": map[string]interface{}{
						"json": map[string]interface{}{
							"file": cfg.OCMShareProvider.Drivers.JSON.File,
						},
					},
					"gatewaysvc":      cfg.Reva.Address,
					"provider_domain": providerDomain,
					"webdav_endpoint": cfg.Commons.OpenCloudURL,
					"webapp_template": cfg.OCMShareProvider.WebappTemplate,
					"client_insecure": cfg.OCMShareProvider.Insecure,
				},
				"ocmcore": map[string]interface{}{
					"driver": cfg.OCMCore.Driver,
					"drivers": map[string]interface{}{
						"json": map[string]interface{}{
							"file": cfg.OCMCore.Drivers.JSON.File,
						},
					},
				},
				"storageprovider": map[string]interface{}{
					"driver": "ocmreceived",
					"drivers": map[string]interface{}{
						"ocmreceived": map[string]interface{}{
							"insecure":     cfg.OCMStorageProvider.Insecure,
							"storage_root": cfg.OCMStorageProvider.StorageRoot,
						},
					},
					"data_server_url": cfg.OCMStorageProvider.DataServerURL,
				},
				"authprovider": map[string]interface{}{
					"auth_manager": "ocmshares",
					"auth_managers": map[string]interface{}{
						"ocmshares": map[string]interface{}{
							"gatewaysvc": cfg.Reva.Address,
						},
					},
				},
			},
		},
	}
}
