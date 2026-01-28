package defaults

import (
	"path/filepath"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
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
			Addr:   "127.0.0.1:9159",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		GRPC: config.GRPCConfig{
			Addr:      "127.0.0.1:9157",
			Namespace: "eu.opencloud.api",
			Protocol:  "tcp",
		},
		HTTP: config.HTTPConfig{
			Addr:      "127.0.0.1:9158",
			Namespace: "eu.opencloud.web",
			Protocol:  "tcp",
			Prefix:    "data",
			CORS: config.CORS{
				AllowedOrigins: []string{"https://localhost:9200"},
				AllowedMethods: []string{
					"POST",
					"HEAD",
					"PATCH",
					"OPTIONS",
					"GET",
					"DELETE",
				},
				AllowedHeaders: []string{
					"Authorization",
					"Origin",
					"X-Requested-With",
					"X-Request-Id",
					"X-HTTP-Method-Override",
					"Content-Type",
					"Upload-Length",
					"Upload-Offset",
					"Tus-Resumable",
					"Upload-Metadata",
					"Upload-Defer-Length",
					"Upload-Concat",
					"Upload-Incomplete",
					"Upload-Draft-Interop-Version",
				},
				AllowCredentials: false,
				ExposedHeaders: []string{
					"Upload-Offset",
					"Location",
					"Upload-Length",
					"Tus-Version",
					"Tus-Resumable",
					"Tus-Max-Size",
					"Tus-Extension",
					"Upload-Metadata",
					"Upload-Defer-Length",
					"Upload-Concat",
					"Upload-Incomplete",
					"Upload-Draft-Interop-Version",
				},
				MaxAge: 86400,
			},
		},
		Service: config.Service{
			Name: "storage-users",
		},
		Reva:                    shared.DefaultRevaConfig(),
		DataServerURL:           "http://localhost:9158/data",
		DataGatewayURL:          "http://localhost:9140/data",
		RevaGatewayGRPCAddr:     "127.0.0.1:9142",
		TransferExpires:         86400,
		UploadExpiration:        24 * 60 * 60,
		GracefulShutdownTimeout: 30,
		Driver:                  "posix",
		Drivers: config.Drivers{
			OwnCloudSQL: config.OwnCloudSQLDriver{
				Root:                  filepath.Join(defaults.BaseDataPath(), "storage", "owncloud"),
				ShareFolder:           "/Shares",
				UserLayout:            "{{.Username}}",
				UploadInfoDir:         filepath.Join(defaults.BaseDataPath(), "storage", "uploadinfo"),
				DBUsername:            "owncloud",
				DBPassword:            "owncloud",
				DBHost:                "",
				DBPort:                3306,
				DBName:                "owncloud",
				UsersProviderEndpoint: "eu.opencloud.api.users",
			},
			DecomposedS3: config.DecomposedS3Driver{
				Propagator:                 "sync",
				Root:                       filepath.Join(defaults.BaseDataPath(), "storage", "users"),
				ShareFolder:                "/Shares",
				UserLayout:                 "{{.Id.OpaqueId}}",
				Region:                     "default",
				SendContentMd5:             true,
				ConcurrentStreamParts:      false,
				NumThreads:                 4,
				PersonalSpaceAliasTemplate: "{{.SpaceType}}/{{.User.Username | lower}}",
				PersonalSpacePathTemplate:  "",
				GeneralSpaceAliasTemplate:  "{{.SpaceType}}/{{.SpaceName | replace \" \" \"-\" | lower}}",
				GeneralSpacePathTemplate:   "",
				PermissionsEndpoint:        "eu.opencloud.api.settings",
				MaxAcquireLockCycles:       20,
				MaxConcurrency:             5,
				LockCycleDurationFactor:    30,
				DisableMultipart:           false,
				PartSize:                   0,
				AsyncUploads:               true,
			},
			Decomposed: config.DecomposedDriver{
				Propagator:                 "sync",
				Root:                       filepath.Join(defaults.BaseDataPath(), "storage", "users"),
				ShareFolder:                "/Shares",
				UserLayout:                 "{{.Id.OpaqueId}}",
				PersonalSpaceAliasTemplate: "{{.SpaceType}}/{{.User.Username | lower}}",
				PersonalSpacePathTemplate:  "",
				GeneralSpaceAliasTemplate:  "{{.SpaceType}}/{{.SpaceName | replace \" \" \"-\" | lower}}",
				GeneralSpacePathTemplate:   "",
				PermissionsEndpoint:        "eu.opencloud.api.settings",
				MaxAcquireLockCycles:       20,
				MaxConcurrency:             5,
				LockCycleDurationFactor:    30,
				AsyncUploads:               true,
			},
			Posix: config.PosixDriver{
				UseSpaceGroups:             false,
				Root:                       filepath.Join(defaults.BaseDataPath(), "storage", "users"),
				PersonalSpaceAliasTemplate: "{{.SpaceType}}/{{.User.Username | lower}}",
				PersonalSpacePathTemplate:  "users/{{.User.Id.OpaqueId}}",
				GeneralSpaceAliasTemplate:  "{{.SpaceType}}/{{.SpaceName | replace \" \" \"-\" | lower}}",
				GeneralSpacePathTemplate:   "projects/{{.SpaceId}}",
				PermissionsEndpoint:        "eu.opencloud.api.settings",
				AsyncUploads:               true,
				ScanDebounceDelay:          1 * time.Second,
				WatchFS:                    false,
				EnableFSRevisions:          false,
				InotifyStatsFrequency:      5 * time.Minute,
			},
		},
		Events: config.Events{
			Addr:      "127.0.0.1:9233",
			ClusterID: "opencloud-cluster",
			EnableTLS: false,
		},
		FilemetadataCache: config.FilemetadataCache{
			Store:    "memory",
			Nodes:    []string{"127.0.0.1:9233"},
			Database: "storage-users",
			TTL:      24 * 60 * time.Second,
		},
		IDCache: config.IDCache{
			Store:    "nats-js-kv",
			Nodes:    []string{"127.0.0.1:9233"},
			Database: "ids-storage-users",
		},
		Tasks: config.Tasks{
			PurgeTrashBin: config.PurgeTrashBin{
				ProjectDeleteBefore:  30 * 24 * time.Hour,
				PersonalDeleteBefore: 30 * 24 * time.Hour,
			},
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

	if cfg.Tasks.PurgeTrashBin.UserID == "" && cfg.Commons != nil {
		cfg.Tasks.PurgeTrashBin.UserID = cfg.Commons.AdminUserID
	}

	if (cfg.Commons != nil && cfg.Commons.OpenCloudURL != "") &&
		(cfg.HTTP.CORS.AllowedOrigins == nil ||
			len(cfg.HTTP.CORS.AllowedOrigins) == 1 &&
				cfg.HTTP.CORS.AllowedOrigins[0] == "https://localhost:9200") {
		cfg.HTTP.CORS.AllowedOrigins = []string{cfg.Commons.OpenCloudURL}
	}
}

// Sanitize sanitized the configuration
func Sanitize(cfg *config.Config) {
	// nothing to sanitize here atm
}
