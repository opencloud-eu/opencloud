package defaults

import (
	"path/filepath"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/users/pkg/config"
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
			Addr:   "127.0.0.1:9145",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		GRPC: config.GRPCConfig{
			Addr:      "127.0.0.1:9144",
			Namespace: "eu.opencloud.api",
			Protocol:  "tcp",
		},
		Service: config.Service{
			Name: "users",
		},
		Reva:   shared.DefaultRevaConfig(),
		Driver: "ldap",
		Drivers: config.Drivers{
			LDAP: config.LDAPDriver{
				URI:                      "ldaps://localhost:9235",
				CACert:                   filepath.Join(defaults.BaseDataPath(), "idm", "ldap.crt"),
				Insecure:                 false,
				UserBaseDN:               "ou=users,o=libregraph-idm",
				GroupBaseDN:              "ou=groups,o=libregraph-idm",
				UserScope:                "sub",
				GroupScope:               "sub",
				UserSubstringFilterType:  "any",
				UserFilter:               "",
				GroupFilter:              "",
				UserObjectClass:          "inetOrgPerson",
				GroupObjectClass:         "groupOfNames",
				BindDN:                   "uid=reva,ou=sysusers,o=libregraph-idm",
				DisableUserMechanism:     "attribute",
				LdapDisabledUsersGroupDN: "cn=DisabledUsersGroup,ou=groups,o=libregraph-idm",
				UserTypeAttribute:        "openCloudUserType",
				IDP:                      "https://localhost:9200",
				UserSchema: config.LDAPUserSchema{
					ID:          "openclouduuid",
					Mail:        "mail",
					DisplayName: "displayname",
					Username:    "uid",
					Enabled:     "openclouduserenabled",
				},
				GroupSchema: config.LDAPGroupSchema{
					ID:          "openclouduuid",
					Mail:        "mail",
					DisplayName: "cn",
					Groupname:   "cn",
					Member:      "member",
				},
			},
			JSON: config.JSONDriver{},
			OwnCloudSQL: config.OwnCloudSQLDriver{
				DBUsername:         "owncloud",
				DBPassword:         "secret",
				DBHost:             "mysql",
				DBPort:             3306,
				DBName:             "owncloud",
				IDP:                "https://localhost:9200",
				Nobody:             90,
				JoinUsername:       false,
				JoinOwnCloudUUID:   false,
				EnableMedialSearch: false,
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
}

// Sanitize sanitized the configuration
func Sanitize(cfg *config.Config) {
	// nothing to sanitize here atm
}
