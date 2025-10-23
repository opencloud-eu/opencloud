package revaconfig

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/opencloud-eu/opencloud/pkg/capabilities"
	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/frontend/pkg/config"
)

// FrontendConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func FrontendConfigFromStruct(cfg *config.Config, logger log.Logger) (map[string]interface{}, error) {
	webURL, err := url.Parse(cfg.PublicURL)
	if err != nil {
		return nil, err
	}
	webURL.Path = path.Join(webURL.Path, "external")
	webOpenInAppURL := webURL.String()

	passwordPolicyCfg, err := passwordPolicyConfig(cfg)
	if err != nil {
		logger.Err(err).Send()
		return nil, err
	}

	archivers := []map[string]interface{}{
		{
			"enabled":       true,
			"version":       "2.0.0",
			"formats":       []string{"tar", "zip"},
			"archiver_url":  path.Join("/", cfg.Archiver.Prefix),
			"max_num_files": strconv.FormatInt(cfg.Archiver.MaxNumFiles, 10),
			"max_size":      strconv.FormatInt(cfg.Archiver.MaxSize, 10),
		},
	}

	appProviders := []map[string]interface{}{
		{
			"enabled":      true,
			"version":      "1.1.0",
			"apps_url":     "/app/list",
			"open_url":     "/app/open",
			"open_web_url": "/app/open-with-web",
			"new_url":      "/app/new",
		},
	}

	filesCfg := map[string]interface{}{
		"private_links":     true,
		"bigfilechunking":   false,
		"blacklisted_files": []string{},
		"undelete":          true,
		"versioning":        true,
		"tags":              true,
		"archivers":         archivers,
		"app_providers":     appProviders,
		"favorites":         cfg.EnableFavorites,
		"full_text_search":  cfg.FullTextSearch,
	}

	if cfg.DefaultUploadProtocol == "tus" {
		filesCfg["tus_support"] = map[string]interface{}{
			"version":              "1.0.0",
			"resumable":            "1.0.0",
			"extension":            "creation,creation-with-upload",
			"http_method_override": cfg.UploadHTTPMethodOverride,
			"max_chunk_size":       cfg.UploadMaxChunkSize,
		}
	}

	readOnlyUserAttributes := []string{}
	if cfg.ReadOnlyUserAttributes != nil {
		readOnlyUserAttributes = cfg.ReadOnlyUserAttributes
	}

	changePasswordDisabled := !cfg.LDAPServerWriteEnabled
	if slices.Contains(readOnlyUserAttributes, "user.passwordProfile") {
		changePasswordDisabled = true
	}

	return map[string]interface{}{
		"shared": map[string]interface{}{
			"jwt_secret":                cfg.TokenManager.JWTSecret,
			"gatewaysvc":                cfg.Reva.Address, // Todo or address?
			"skip_user_groups_in_token": cfg.SkipUserGroupsInToken,
			"grpc_client_options":       cfg.Reva.GetGRPCClientConfig(),
			"multi_tenant_enabled":      cfg.Commons.MultiTenantEnabled,
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
					"subsystem": "frontend",
				},
				"requestid": map[string]interface{}{},
			},
			// TODO build services dynamically
			"services": map[string]interface{}{
				// this reva service called "appprovider" comes from
				// `internal/http/services/appprovider` and is a translation
				// layer from the grpc app registry to http, used by e.g. OpenCloud Web
				// It should not be confused with `internal/grpc/services/appprovider`
				// which is currently only the driver for the CS3org WOPI server
				"appprovider": map[string]interface{}{
					"prefix":                 cfg.AppHandler.Prefix,
					"transfer_shared_secret": cfg.TransferSecret,
					"timeout":                86400,
					"insecure":               cfg.AppHandler.Insecure,
					"webbaseuri":             webOpenInAppURL,
					"web": map[string]interface{}{
						"urlparamsmapping": map[string]string{
							// param -> value mapper
							// these mappers are static and are only subject to change when changed in oC Web
							"fileId": "fileid",
							"app":    "appname",
						},
						"staticurlparams": map[string]string{
							"contextRouteName": "files-spaces-personal", // FIXME: remove when https://github.com/owncloud/web/pull/7437 arrived in OpenCloud
						},
					},
					"secure_view_app_addr": cfg.AppHandler.SecureViewAppAddr,
				},
				"archiver": map[string]interface{}{
					"prefix":        cfg.Archiver.Prefix,
					"timeout":       86400,
					"insecure":      cfg.Archiver.Insecure,
					"max_num_files": cfg.Archiver.MaxNumFiles,
					"max_size":      cfg.Archiver.MaxSize,
				},
				"datagateway": map[string]interface{}{
					"prefix":                 cfg.DataGateway.Prefix,
					"transfer_shared_secret": cfg.TransferSecret,
					"timeout":                86400,
					"insecure":               true,
				},
				"ocs": map[string]interface{}{
					"storage_registry_svc": cfg.Reva.Address,
					"share_prefix":         cfg.OCS.SharePrefix,
					"home_namespace":       cfg.OCS.HomeNamespace,
					"stat_cache_config": map[string]interface{}{
						"cache_store":               cfg.OCS.StatCacheType,
						"cache_nodes":               cfg.OCS.StatCacheNodes,
						"cache_database":            cfg.OCS.StatCacheDatabase,
						"cache_table":               cfg.OCS.StatCacheTable,
						"cache_ttl":                 cfg.OCS.StatCacheTTL,
						"cache_disable_persistence": cfg.OCS.StatCacheDisablePersistence,
						"cache_auth_username":       cfg.OCS.StatCacheAuthUsername,
						"cache_auth_password":       cfg.OCS.StatCacheAuthPassword,
					},
					"prefix":                    cfg.OCS.Prefix,
					"additional_info_attribute": cfg.OCS.AdditionalInfoAttribute,
					"machine_auth_apikey":       cfg.MachineAuthAPIKey,
					"enable_denials":            cfg.OCS.EnableDenials,
					"list_ocm_shares":           cfg.OCS.ListOCMShares,
					"cache_warmup_driver":       cfg.OCS.CacheWarmupDriver,
					"cache_warmup_drivers": map[string]interface{}{
						"cbox": map[string]interface{}{
							"db_username": cfg.OCS.CacheWarmupDrivers.CBOX.DBUsername,
							"db_password": cfg.OCS.CacheWarmupDrivers.CBOX.DBPassword,
							"db_host":     cfg.OCS.CacheWarmupDrivers.CBOX.DBHost,
							"db_port":     cfg.OCS.CacheWarmupDrivers.CBOX.DBPort,
							"db_name":     cfg.OCS.CacheWarmupDrivers.CBOX.DBName,
							"namespace":   cfg.OCS.CacheWarmupDrivers.CBOX.Namespace,
							"gatewaysvc":  cfg.Reva.Address,
						},
					},
					"config": map[string]interface{}{
						"version": "1.7",
						"website": "OpenCloud",
						"host":    cfg.PublicURL,
						"contact": "",
						"ssl":     "false",
					},
					"default_upload_protocol": cfg.DefaultUploadProtocol,
					"capabilities": map[string]interface{}{
						"capabilities": map[string]interface{}{
							"core": map[string]interface{}{
								"poll_interval": 60,
								"webdav_root":   "remote.php/webdav",
								"status": map[string]interface{}{
									"installed":      true,
									"maintenance":    false,
									"needsDbUpgrade": false,
									"version":        version.Legacy,
									"versionstring":  version.LegacyString,
									"edition":        cfg.Edition,
									"productname":    "OpenCloud",
									"product":        "OpenCloud",
									"productversion": version.GetString(),
									"hostname":       "",
								},
								"check_for_updates":   cfg.CheckForUpdates,
								"support_url_signing": true,
								"support_sse":         !cfg.DisableSSE,
							},
							"graph": map[string]interface{}{
								"personal_data_export": true,
								"users": map[string]interface{}{
									"read_only_attributes":          readOnlyUserAttributes,
									"create_disabled":               !cfg.LDAPServerWriteEnabled,
									"delete_disabled":               !cfg.LDAPServerWriteEnabled,
									"change_password_self_disabled": changePasswordDisabled,
									"edit_login_allowed_disabled":   cfg.EditLoginAllowedDisabled,
								},
							},
							"checksums": map[string]interface{}{
								"supported_types":       cfg.Checksums.SupportedTypes,
								"preferred_upload_type": cfg.Checksums.PreferredUploadType,
							},
							"files": filesCfg,
							"dav": map[string]interface{}{
								"reports": []string{"search-files"},
							},
							"files_sharing": map[string]interface{}{
								"api_enabled":                       true,
								"group_sharing":                     true,
								"sharing_roles":                     true,
								"deny_access":                       cfg.OCS.EnableDenials,
								"auto_accept_share":                 true,
								"share_with_group_members_only":     true,
								"share_with_membership_groups_only": true,
								"default_permissions":               22,
								"search_min_length":                 cfg.SearchMinLength,
								"public": map[string]interface{}{
									"alias":                      true,
									"enabled":                    true,
									"send_mail":                  true,
									"defaultPublicLinkShareName": "Public link",
									"social_share":               true,
									"upload":                     true,
									"multiple":                   true,
									"supports_upload_only":       true,
									"default_permissions":        cfg.DefaultLinkPermissions,
									"password": map[string]interface{}{
										"enforced": false,
										"enforced_for": map[string]interface{}{
											"read_only":         cfg.OCS.PublicShareMustHavePassword,
											"read_write":        cfg.OCS.WriteablePublicShareMustHavePassword,
											"read_write_delete": cfg.OCS.WriteablePublicShareMustHavePassword,
											"upload_only":       cfg.OCS.WriteablePublicShareMustHavePassword,
										},
									},
									"expire_date": map[string]interface{}{
										"enabled": false,
									},
									"can_edit": true,
								},
								"user": map[string]interface{}{
									"send_mail":       true,
									"profile_picture": false,
									"settings": []map[string]interface{}{
										{
											"enabled": true,
											"version": "1.0.0",
										},
									},
									"expire_date": map[string]interface{}{
										"enabled": true,
									},
								},
								"user_enumeration": map[string]interface{}{
									"enabled":            true,
									"group_members_only": true,
								},
								"federation": map[string]interface{}{
									"outgoing": cfg.EnableFederatedSharingOutgoing,
									"incoming": cfg.EnableFederatedSharingIncoming,
								},
							},
							"spaces": map[string]interface{}{
								"version":    "1.0.0",
								"enabled":    true,
								"projects":   true,
								"share_jail": true,
								"max_quota":  cfg.MaxQuota,
							},
							"theme": capabilities.Default().Theme,
							"search": map[string]interface{}{
								"property": map[string]interface{}{
									"name": map[string]interface{}{
										"enabled": true,
									},
									"mtime": map[string]interface{}{
										"keywords": []string{"today", "last 7 days", "last 30 days", "this year", "last year"},
										"enabled":  true,
									},
									"size": map[string]interface{}{
										"enabled": false,
									},
									"mediatype": map[string]interface{}{
										"keywords": []string{"document", "spreadsheet", "presentation", "pdf", "image", "video", "audio", "folder", "archive"},
										"enabled":  true,
									},
									"type": map[string]interface{}{
										"enabled": true,
									},
									"tag": map[string]interface{}{
										"enabled": true,
									},
									"tags": map[string]interface{}{
										"enabled": true,
									},
									"content": map[string]interface{}{
										"enabled": true,
									},
									"scope": map[string]interface{}{
										"enabled": true,
									},
								},
							},
							"password_policy": passwordPolicyCfg,
							"notifications": map[string]interface{}{
								"endpoints":    []string{"list", "get", "delete"},
								"configurable": cfg.ConfigurableNotifications,
							},
							"groupware": map[string]interface{}{
								"enabled": cfg.Groupware.Enabled,
							},
						},
						"version": map[string]interface{}{
							"product":        "OpenCloud",
							"edition":        "",
							"major":          version.ParsedLegacy().Major(),
							"minor":          version.ParsedLegacy().Minor(),
							"micro":          version.ParsedLegacy().Patch(),
							"string":         version.LegacyString,
							"productversion": version.GetString(),
						},
					},
					"include_ocm_sharees":   cfg.OCS.IncludeOCMSharees,
					"show_email_in_results": cfg.OCS.ShowUserEmailInResults,
				},
			},
		},
	}, nil
}

func readMultilineFile(path string) (map[string]struct{}, error) {
	if !fileExists(path) {
		path = filepath.Join(defaults.BaseConfigPath(), path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	data := make(map[string]struct{})
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			data[line] = struct{}{}
		}
	}
	return data, err
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func passwordPolicyConfig(cfg *config.Config) (map[string]interface{}, error) {
	_maxCharacters := 72
	if cfg.PasswordPolicy.Disabled {
		return map[string]interface{}{
			"max_characters":        _maxCharacters,
			"banned_passwords_list": nil,
		}, nil
	}
	var bannedPasswordsList map[string]struct{}
	var err error
	if cfg.PasswordPolicy.BannedPasswordsList != "" {
		bannedPasswordsList, err = readMultilineFile(cfg.PasswordPolicy.BannedPasswordsList)
		if err != nil {
			return nil, fmt.Errorf("failed to load the banned passwords from a file %s: %w", cfg.PasswordPolicy.BannedPasswordsList, err)
		}
	}
	return map[string]interface{}{
		"max_characters":           _maxCharacters,
		"min_digits":               cfg.PasswordPolicy.MinDigits,
		"min_characters":           cfg.PasswordPolicy.MinCharacters,
		"min_lowercase_characters": cfg.PasswordPolicy.MinLowerCaseCharacters,
		"min_uppercase_characters": cfg.PasswordPolicy.MinUpperCaseCharacters,
		"min_special_characters":   cfg.PasswordPolicy.MinSpecialCharacters,
		"banned_passwords_list":    bannedPasswordsList,
	}, nil
}
