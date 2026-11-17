package parser

import (
	"errors"
	"fmt"
	"net/url"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	ocdefaults "github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config/defaults"
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

// Validate validates the configuration
func Validate(cfg *config.Config) error {
	if cfg.TokenManager.JWTSecret == "" {
		return shared.MissingJWTTokenError(cfg.Service.Name)
	}
	if cfg.Wopi.Secret == "" {
		return shared.MissingWOPISecretError(cfg.Service.Name)
	}
	if _, err := middleware.ParseTrustedProxies(cfg.HTTP.TrustedProxies); err != nil {
		return fmt.Errorf("invalid COLLABORATION_HTTP_TRUSTED_PROXIES in service %s: %w", cfg.Service.Name, err)
	}
	url, err := url.Parse(cfg.Wopi.WopiSrc)
	if err != nil {
		return fmt.Errorf("The WOPI Src has not been set properly in your config for %s. "+
			"Make sure your %s config contains the proper values "+
			"(e.g. by running opencloud init or setting it manually in "+
			"the config/corresponding environment variable): %s",
			cfg.Service.Name, ocdefaults.BaseConfigPath(), err.Error())
	}
	if url.Path != "" {
		return fmt.Errorf("The WOPI Src must not contain a path in your config for %s. "+
			"Make sure your %s config contains the proper values "+
			"(e.g. by running opencloud init or setting it manually in "+
			"the config/corresponding environment variable)",
			cfg.Service.Name, ocdefaults.BaseConfigPath())
	}

	return nil
}
