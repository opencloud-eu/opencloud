package parser

import (
	"errors"
	"fmt"
	"slices"

	"github.com/go-ldap/ldap/v3"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	defaults2 "github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
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

	if !slices.Contains([]string{"ldap", "cs3"}, cfg.Identity.Backend) {
		return fmt.Errorf("'%s' is not a valid identity backend	for the 'graph' service", cfg.Identity.Backend)
	}

	// ensure that the "cs3" identity backend is used in multi-tenant setups
	if cfg.Commons.MultiTenantEnabled && cfg.Identity.Backend != "cs3" {
		return fmt.Errorf("Multi-tenant support is enabled. The identity backend must be set to 'cs3' for the 'graph' service.")
	}
	if cfg.Identity.Backend == "ldap" {
		if err := validateLDAPSettings(cfg); err != nil {
			return err
		}
	}

	if cfg.Application.ID == "" {
		return fmt.Errorf("The application ID has not been configured for %s. "+
			"Make sure your %s config contains the proper values "+
			"(e.g. by running opencloud init or setting it manually in "+
			"the config/corresponding environment variable).",
			"graph", defaults2.BaseConfigPath())
	}

	switch cfg.API.UsernameMatch {
	case "default", "none":
	default:
		return fmt.Errorf("The username match validator is invalid for %s. "+
			"Make sure your %s config contains the proper values "+
			"(e.g. by running opencloud init or setting it manually in "+
			"the config/corresponding environment variable).",
			"graph", defaults2.BaseConfigPath())
	}

	if cfg.ServiceAccount.ServiceAccountID == "" {
		return shared.MissingServiceAccountID(cfg.Service.Name)
	}
	if cfg.ServiceAccount.ServiceAccountSecret == "" {
		return shared.MissingServiceAccountSecret(cfg.Service.Name)
	}

	// validate unified roles
	{
		var err error

		for _, uid := range cfg.UnifiedRoles.AvailableRoles {
			// check if the role is known
			if len(unifiedrole.GetRoles(unifiedrole.RoleFilterIDs(uid))) == 0 {
				// collect all possible errors to return them all at once
				err = errors.Join(err, fmt.Errorf("%w: %s", unifiedrole.ErrUnknownRole, uid))
			}
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func validateLDAPSettings(cfg *config.Config) error {
	if cfg.Identity.LDAP.BindPassword == "" {
		return shared.MissingLDAPBindPassword(cfg.Service.Name)
	}

	// ensure that "GroupBaseDN" is below "GroupBaseDN"
	if cfg.Identity.LDAP.WriteEnabled && cfg.Identity.LDAP.GroupCreateBaseDN != cfg.Identity.LDAP.GroupBaseDN {
		baseDN, err := ldap.ParseDN(cfg.Identity.LDAP.GroupBaseDN)
		if err != nil {
			return fmt.Errorf("Unable to parse the LDAP Group Base DN '%s': %w ", cfg.Identity.LDAP.GroupBaseDN, err)
		}
		createBaseDN, err := ldap.ParseDN(cfg.Identity.LDAP.GroupCreateBaseDN)
		if err != nil {
			return fmt.Errorf("Unable to parse the LDAP Group Create Base DN '%s': %w ", cfg.Identity.LDAP.GroupCreateBaseDN, err)
		}

		if !baseDN.AncestorOfFold(createBaseDN) {
			return fmt.Errorf("The LDAP Group Create Base DN (%s) must be subordinate to the LDAP Group Base DN (%s)", cfg.Identity.LDAP.GroupCreateBaseDN, cfg.Identity.LDAP.GroupBaseDN)
		}
	}
	return nil
}
