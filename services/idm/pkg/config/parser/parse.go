package parser

import (
	"errors"
	"net"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/idm/pkg/config"
	"github.com/opencloud-eu/opencloud/services/idm/pkg/config/defaults"

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
	if cfg.CreateDemoUsers && cfg.AdminUserID == "" {
		return shared.MissingAdminUserID(cfg.Service.Name)
	}

	if cfg.ServiceUserPasswords.Idm == "" {
		return shared.MissingServiceUserPassword(cfg.Service.Name, "IDM")
	}

	if cfg.AdminUserID != "" && cfg.ServiceUserPasswords.OCAdmin == "" {
		return shared.MissingServiceUserPassword(cfg.Service.Name, "admin")
	}

	if cfg.ServiceUserPasswords.Idp == "" {
		return shared.MissingServiceUserPassword(cfg.Service.Name, "IDP")
	}

	if cfg.ServiceUserPasswords.Reva == "" {
		return shared.MissingServiceUserPassword(cfg.Service.Name, "REVA")
	}

	ip, err := net.ResolveTCPAddr("tcp", cfg.IDM.LDAPAddr) // validate the LDAP address if set

	if err != nil {
		return errors.New("invalid configuration: 'ldap_addr' is not a valid address")
	}

	if !ip.IP.IsLoopback() {
		// loopback addresses are allowed to be used with ldap_addr, but not with ldaps_addr, for security reasons
		return errors.New("invalid configuration: 'ldap_addr' is set but 'ldaps_addr' is not set. For security reasons, the 'ldap_addr' setting is only allowed to be used with loopback addresses. Please set 'ldaps_addr' to a valid address and port to listen for LDAPS connections")
	}

	if cfg.IDM.LDAPSAddr != "" {
		if cfg.IDM.Cert == "" {
			return errors.New("invalid configuration: 'ldaps_addr' is set but 'cert' is not set. Please set 'cert' to a valid path to a TLS certificate")
		}
		if cfg.IDM.Key == "" {
			return errors.New("invalid configuration: 'ldaps_addr' is set but 'key' is not set. Please set 'key' to a valid path to a TLS certificate key")
		}
	}

	return nil
}
