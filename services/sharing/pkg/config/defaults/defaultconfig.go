package defaults

import (
	"path/filepath"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/sharing/pkg/config"
)

// FullDefaultConfig returns a fully initialized default configuration
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()
	EnsureDefaults(cfg)
	Sanitize(cfg)
	return cfg
}

// DefaultConfig returns a basic default configuration
func DefaultConfig() *config.Config {
	return &config.Config{
		Debug: config.Debug{
			Addr:   "127.0.0.1:9151",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		GRPC: config.GRPCConfig{
			Addr:      "127.0.0.1:9150",
			Namespace: "eu.opencloud.api",
			Protocol:  "tcp",
		},
		Service: config.Service{
			Name: "sharing",
		},
		Reva:              shared.DefaultRevaConfig(),
		UserSharingDriver: "jsoncs3",
		UserSharingDrivers: config.UserSharingDrivers{
			JSON: config.UserSharingJSONDriver{
				File: filepath.Join(defaults.BaseDataPath(), "storage", "shares.json"),
			},
			CS3: config.UserSharingCS3Driver{
				ProviderAddr:  "eu.opencloud.api.storage-system",
				SystemUserIDP: "internal",
			},
			JSONCS3: config.UserSharingJSONCS3Driver{
				ProviderAddr:   "eu.opencloud.api.storage-system",
				SystemUserIDP:  "internal",
				MaxConcurrency: 1,
			},
			OwnCloudSQL: config.UserSharingOwnCloudSQLDriver{
				DBUsername: "owncloud",
				DBHost:     "mysql",
				DBPort:     3306,
				DBName:     "owncloud",
			},
		},
		PublicSharingDriver: "jsoncs3",
		PublicSharingDrivers: config.PublicSharingDrivers{
			JSON: config.PublicSharingJSONDriver{
				File: filepath.Join(defaults.BaseDataPath(), "storage", "publicshares.json"),
			},
			CS3: config.PublicSharingCS3Driver{
				ProviderAddr:  "eu.opencloud.api.storage-system", // system storage
				SystemUserIDP: "internal",
			},
			JSONCS3: config.PublicSharingJSONCS3Driver{
				ProviderAddr:  "eu.opencloud.api.storage-system", // system storage
				SystemUserIDP: "internal",
			},
			// TODO implement and add owncloudsql publicshare driver
		},
		Events: config.Events{
			Addr:      "127.0.0.1:9233",
			ClusterID: "opencloud-cluster",
			EnableTLS: false,
		},
		EnableExpiredSharesCleanup:  true,
		PublicShareMustHavePassword: true,
		PasswordPolicy: config.PasswordPolicy{
			MinCharacters:          8,
			MinLowerCaseCharacters: 1,
			MinUpperCaseCharacters: 1,
			MinDigits:              1,
			MinSpecialCharacters:   1,
		},
	}
}

// EnsureDefaults adds default values to the configuration if they are not set yet
func EnsureDefaults(cfg *config.Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = "error"
	}

	if cfg.Reva == nil && cfg.Commons != nil {
		cfg.Reva = structs.CopyOrZeroValue(cfg.Commons.Reva)
	}

	if cfg.TokenManager == nil && cfg.Commons != nil && cfg.Commons.TokenManager != nil {
		cfg.TokenManager = &config.TokenManager{
			JWTSecret: cfg.Commons.TokenManager.JWTSecret,
		}
	} else if cfg.TokenManager == nil {
		cfg.TokenManager = &config.TokenManager{}
	}

	if cfg.GRPC.TLS == nil && cfg.Commons != nil {
		cfg.GRPC.TLS = structs.CopyOrZeroValue(cfg.Commons.GRPCServiceTLS)
	}

	if cfg.UserSharingDrivers.CS3.SystemUserAPIKey == "" && cfg.Commons != nil && cfg.Commons.SystemUserAPIKey != "" {
		cfg.UserSharingDrivers.CS3.SystemUserAPIKey = cfg.Commons.SystemUserAPIKey
	}

	if cfg.UserSharingDrivers.CS3.SystemUserID == "" && cfg.Commons != nil && cfg.Commons.SystemUserID != "" {
		cfg.UserSharingDrivers.CS3.SystemUserID = cfg.Commons.SystemUserID
	}

	if cfg.UserSharingDrivers.JSONCS3.SystemUserAPIKey == "" && cfg.Commons != nil && cfg.Commons.SystemUserAPIKey != "" {
		cfg.UserSharingDrivers.JSONCS3.SystemUserAPIKey = cfg.Commons.SystemUserAPIKey
	}

	if cfg.UserSharingDrivers.JSONCS3.SystemUserID == "" && cfg.Commons != nil && cfg.Commons.SystemUserID != "" {
		cfg.UserSharingDrivers.JSONCS3.SystemUserID = cfg.Commons.SystemUserID
	}

	if cfg.PublicSharingDrivers.CS3.SystemUserAPIKey == "" && cfg.Commons != nil && cfg.Commons.SystemUserAPIKey != "" {
		cfg.PublicSharingDrivers.CS3.SystemUserAPIKey = cfg.Commons.SystemUserAPIKey
	}

	if cfg.PublicSharingDrivers.CS3.SystemUserID == "" && cfg.Commons != nil && cfg.Commons.SystemUserID != "" {
		cfg.PublicSharingDrivers.CS3.SystemUserID = cfg.Commons.SystemUserID
	}

	if cfg.PublicSharingDrivers.JSONCS3.SystemUserAPIKey == "" && cfg.Commons != nil && cfg.Commons.SystemUserAPIKey != "" {
		cfg.PublicSharingDrivers.JSONCS3.SystemUserAPIKey = cfg.Commons.SystemUserAPIKey
	}

	if cfg.PublicSharingDrivers.JSONCS3.SystemUserID == "" && cfg.Commons != nil && cfg.Commons.SystemUserID != "" {
		cfg.PublicSharingDrivers.JSONCS3.SystemUserID = cfg.Commons.SystemUserID
	}
}

// Sanitize sanitized the configuration
func Sanitize(cfg *config.Config) {
	// nothing to sanitize here atm
}
