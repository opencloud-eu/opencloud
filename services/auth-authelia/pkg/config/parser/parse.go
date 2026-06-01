package parser

import (
	"errors"
	"fmt"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/services/auth-authelia/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-authelia/pkg/config/defaults"
)

// ParseConfig loads configuration from known paths.
func ParseConfig(cfg *config.Config) error {
	err := occfg.BindSourcesToStructs(cfg.Service.Name, cfg)
	if err != nil {
		return err
	}

	defaults.EnsureDefaults(cfg)

	// load all env variables relevant to the config in the current context.
	if err := envdecode.Decode(cfg); err != nil {
		// no environment variable set for this config is an expected "error"
		if !errors.Is(err, envdecode.ErrNoTargetFieldsAreSet) {
			return err
		}
	}

	defaults.Sanitize(cfg)

	return Validate(cfg)
}

// Validate validates the configuration.
func Validate(cfg *config.Config) error {
	if cfg.ConfigPath == "" {
		return fmt.Errorf("the authelia config path is not configured for the %s service", cfg.Service.Name)
	}

	return nil
}
