package defaults

import (
	"path/filepath"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/auth-authelia/pkg/config"
)

// FullDefaultConfig returns a fully initialized default configuration.
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()
	EnsureDefaults(cfg)
	Sanitize(cfg)
	return cfg
}

// DefaultConfig returns a basic default configuration.
func DefaultConfig() *config.Config {
	return &config.Config{
		Service: config.Service{
			Name: "auth-authelia",
		},
		// The config file is rendered by `opencloud init` into the OpenCloud config directory.
		ConfigPath:  filepath.Join(defaults.BaseConfigPath(), "authelia.yaml"),
		FilterNames: nil,
	}
}

// EnsureDefaults adds default values to the configuration if they are not set yet.
func EnsureDefaults(cfg *config.Config) {
	if cfg.LogLevel == "" {
		cfg.LogLevel = "error"
	}
}

// Sanitize sanitizes the configuration.
func Sanitize(_ *config.Config) {}
