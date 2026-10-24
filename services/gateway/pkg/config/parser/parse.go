package parser

import (
	"errors"
	"fmt"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	defaults2 "github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/gateway/pkg/config"
	"github.com/opencloud-eu/opencloud/services/gateway/pkg/config/defaults"
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

func Validate(cfg *config.Config) error {
	if cfg.TokenManager.JWTSecret == "" {
		return shared.MissingJWTTokenError(cfg.Service.Name)
	}

	if cfg.StorageRegistry.StorageUsersMountID == "" {
		return fmt.Errorf("The storage users mount ID has not been configured for %s. "+
			"Make sure your %s config contains the proper values "+
			"(e.g. by running opencloud init or setting it manually in "+
			"the config/corresponding environment variable).",
			"gateway", defaults2.BaseConfigPath())
	}

	return nil
}
