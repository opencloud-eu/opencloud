package defaults

import (
	"path"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/idm/pkg/config"
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
			Addr:   "127.0.0.1:9239",
			Token:  "",
			Pprof:  false,
			Zpages: false,
		},
		Service: config.Service{
			Name: "idm",
		},
		CreateDemoUsers:    false,
		DemoUsersIssuerUrl: "https://localhost:9200",
		IDM: config.Settings{
			LDAPSAddr:    "127.0.0.1:9235",
			Cert:         path.Join(defaults.BaseDataPath(), "idm", "ldap.crt"),
			Key:          path.Join(defaults.BaseDataPath(), "idm", "ldap.key"),
			DatabasePath: path.Join(defaults.BaseDataPath(), "idm", "idm.boltdb"),
		},
	}
}

// EnsureDefaults adds default values to the configuration if they are not set yet
func EnsureDefaults(cfg *config.Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = "error"
	}

	if cfg.AdminUserID == "" && cfg.Commons != nil {
		cfg.AdminUserID = cfg.Commons.AdminUserID
	}
}

// Sanitize sanitizes the configuration
func Sanitize(cfg *config.Config) {
	// nothing to sanitize here
}
