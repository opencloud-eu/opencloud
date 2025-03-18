package revaconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/sharing/pkg/config"
)

// SharingConfigFromStruct will adapt an OpenCloud config struct into a reva mapstructure to start a reva service.
func SharingConfigFromStruct(cfg *config.Config, logger log.Logger) (map[string]interface{}, error) {
	passwordPolicyCfg, err := passwordPolicyConfig(cfg)
	if err != nil {
		logger.Err(err).Send()
		return nil, err
	}
	rcfg := map[string]interface{}{
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
				"usershareprovider": map[string]interface{}{
					"driver": cfg.UserSharingDriver,
					"drivers": map[string]interface{}{
						"json": map[string]interface{}{
							"file":         cfg.UserSharingDrivers.JSON.File,
							"gateway_addr": cfg.Reva.Address,
						},
						"sql": map[string]interface{}{ // cernbox sql
							"db_username":                   cfg.UserSharingDrivers.SQL.DBUsername,
							"db_password":                   cfg.UserSharingDrivers.SQL.DBPassword,
							"db_host":                       cfg.UserSharingDrivers.SQL.DBHost,
							"db_port":                       cfg.UserSharingDrivers.SQL.DBPort,
							"db_name":                       cfg.UserSharingDrivers.SQL.DBName,
							"password_hash_cost":            cfg.UserSharingDrivers.SQL.PasswordHashCost,
							"enable_expired_shares_cleanup": cfg.UserSharingDrivers.SQL.EnableExpiredSharesCleanup,
							"janitor_run_interval":          cfg.UserSharingDrivers.SQL.JanitorRunInterval,
						},
						"owncloudsql": map[string]interface{}{
							"gateway_addr":     cfg.Reva.Address,
							"storage_mount_id": cfg.UserSharingDrivers.OwnCloudSQL.UserStorageMountID,
							"db_username":      cfg.UserSharingDrivers.OwnCloudSQL.DBUsername,
							"db_password":      cfg.UserSharingDrivers.OwnCloudSQL.DBPassword,
							"db_host":          cfg.UserSharingDrivers.OwnCloudSQL.DBHost,
							"db_port":          cfg.UserSharingDrivers.OwnCloudSQL.DBPort,
							"db_name":          cfg.UserSharingDrivers.OwnCloudSQL.DBName,
						},
						"cs3": map[string]interface{}{
							"gateway_addr":        cfg.UserSharingDrivers.CS3.ProviderAddr,
							"provider_addr":       cfg.UserSharingDrivers.CS3.ProviderAddr,
							"service_user_id":     cfg.UserSharingDrivers.CS3.SystemUserID,
							"service_user_idp":    cfg.UserSharingDrivers.CS3.SystemUserIDP,
							"machine_auth_apikey": cfg.UserSharingDrivers.CS3.SystemUserAPIKey,
						},
						"jsoncs3": map[string]interface{}{
							"gateway_addr":        cfg.Reva.Address,
							"provider_addr":       cfg.UserSharingDrivers.JSONCS3.ProviderAddr,
							"service_user_id":     cfg.UserSharingDrivers.JSONCS3.SystemUserID,
							"service_user_idp":    cfg.UserSharingDrivers.JSONCS3.SystemUserIDP,
							"machine_auth_apikey": cfg.UserSharingDrivers.JSONCS3.SystemUserAPIKey,
							"ttl":                 cfg.UserSharingDrivers.JSONCS3.CacheTTL,
							"max_concurrency":     cfg.UserSharingDrivers.JSONCS3.MaxConcurrency,
							"events": map[string]interface{}{
								"natsaddress":          cfg.Events.Addr,
								"natsclusterid":        cfg.Events.ClusterID,
								"tlsinsecure":          cfg.Events.TLSInsecure,
								"tlsrootcacertificate": cfg.Events.TLSRootCaCertPath,
								"authusername":         cfg.Events.AuthUsername,
								"authpassword":         cfg.Events.AuthPassword,
							},
						},
					},
				},
				"publicshareprovider": map[string]interface{}{
					"gateway_addr":                       cfg.Reva.Address,
					"writeable_share_must_have_password": cfg.WriteableShareMustHavePassword,
					"public_share_must_have_password":    cfg.PublicShareMustHavePassword,
					"password_policy":                    passwordPolicyCfg,
					"driver":                             cfg.PublicSharingDriver,
					"drivers": map[string]interface{}{
						"json": map[string]interface{}{
							"file":         cfg.PublicSharingDrivers.JSON.File,
							"gateway_addr": cfg.Reva.Address,
						},
						"sql": map[string]interface{}{
							"db_username":                   cfg.PublicSharingDrivers.SQL.DBUsername,
							"db_password":                   cfg.PublicSharingDrivers.SQL.DBPassword,
							"db_host":                       cfg.PublicSharingDrivers.SQL.DBHost,
							"db_port":                       cfg.PublicSharingDrivers.SQL.DBPort,
							"db_name":                       cfg.PublicSharingDrivers.SQL.DBName,
							"password_hash_cost":            cfg.PublicSharingDrivers.SQL.PasswordHashCost,
							"enable_expired_shares_cleanup": cfg.PublicSharingDrivers.SQL.EnableExpiredSharesCleanup,
							"janitor_run_interval":          cfg.PublicSharingDrivers.SQL.JanitorRunInterval,
						},
						"cs3": map[string]interface{}{
							"gateway_addr":        cfg.PublicSharingDrivers.CS3.ProviderAddr,
							"provider_addr":       cfg.PublicSharingDrivers.CS3.ProviderAddr,
							"service_user_id":     cfg.PublicSharingDrivers.CS3.SystemUserID,
							"service_user_idp":    cfg.PublicSharingDrivers.CS3.SystemUserIDP,
							"machine_auth_apikey": cfg.PublicSharingDrivers.CS3.SystemUserAPIKey,
						},
						"jsoncs3": map[string]interface{}{
							"gateway_addr":                  cfg.Reva.Address,
							"provider_addr":                 cfg.PublicSharingDrivers.JSONCS3.ProviderAddr,
							"service_user_id":               cfg.PublicSharingDrivers.JSONCS3.SystemUserID,
							"service_user_idp":              cfg.PublicSharingDrivers.JSONCS3.SystemUserIDP,
							"machine_auth_apikey":           cfg.PublicSharingDrivers.JSONCS3.SystemUserAPIKey,
							"enable_expired_shares_cleanup": cfg.EnableExpiredSharesCleanup,
						},
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
					"name":             generators.GenerateConnectionName(cfg.Service.Name, generators.NTYPE_BUS),
					"username":         cfg.Events.AuthUsername,
					"password":         cfg.Events.AuthPassword,
				},
				"prometheus": map[string]interface{}{
					"namespace": "opencloud",
					"subsystem": "sharing",
				},
			},
		},
	}
	return rcfg, nil
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
