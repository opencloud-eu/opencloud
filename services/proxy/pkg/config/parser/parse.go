package parser

import (
	"errors"
	"fmt"
	"strings"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config/defaults"

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
	if cfg.MachineAuthAPIKey == "" {
		return shared.MissingMachineAuthApiKeyError(cfg.Service.Name)
	}

	if cfg.TransferSecret == "" {
		return shared.MissingRevaTransferSecretError(cfg.Service.Name)
	}

	if cfg.OIDC.AccessTokenVerifyMethod != config.AccessTokenVerificationNone &&
		cfg.OIDC.AccessTokenVerifyMethod != config.AccessTokenVerificationJWT {
		return fmt.Errorf(
			"Invalid value '%s' for 'access_token_verify_method' in service %s. Possible values are: '%s' or '%s'.",
			cfg.OIDC.AccessTokenVerifyMethod, cfg.Service.Name,
			config.AccessTokenVerificationJWT, config.AccessTokenVerificationNone,
		)
	}
	if cfg.OIDC.AccessTokenVerifyMethod == "none" && cfg.OIDC.SkipUserInfo {
		return fmt.Errorf(
			"Incompatible value '%t' for 'skip_user_info' in service %s. Must be false when 'access_token_verify_method' is 'none'.",
			cfg.OIDC.SkipUserInfo, cfg.Service.Name,
		)
	}
	if len(cfg.OIDC.Audiences) > 0 && cfg.OIDC.AccessTokenVerifyMethod != config.AccessTokenVerificationJWT {
		return fmt.Errorf("OIDC audiences (PROXY_OIDC_AUDIENCES) in service %s require access_token_verify_method to be 'jwt'", cfg.Service.Name)
	}
	for _, audience := range cfg.OIDC.Audiences {
		if strings.TrimSpace(audience) == "" {
			return fmt.Errorf("OIDC audiences (PROXY_OIDC_AUDIENCES) in service %s must not contain empty or whitespace-only entries", cfg.Service.Name)
		}
	}

	if cfg.ServiceAccount.ServiceAccountID == "" {
		return shared.MissingServiceAccountID(cfg.Service.Name)
	}

	if cfg.ServiceAccount.ServiceAccountSecret == "" {
		return shared.MissingServiceAccountSecret(cfg.Service.Name)
	}

	if cfg.Commons.URLSigningSecret == "" {
		return shared.MissingURLSigningSecret(cfg.Service.Name)
	}

	return nil
}
