package parser

import (
	"errors"
	"fmt"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/groups/pkg/config"
	"github.com/opencloud-eu/opencloud/services/groups/pkg/config/defaults"

	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
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

	if cfg.Commons.MultiTenantEnabled && cfg.Driver != "null" {
		return fmt.Errorf("Multi-tenant support is enabled. Only the 'null'-driver is supported by 'groups' service.")
	}
	if cfg.Drivers.LDAP.BindPassword == "" && cfg.Driver == "ldap" {
		return shared.MissingLDAPBindPassword(cfg.Service.Name)
	}

	return nil
}
