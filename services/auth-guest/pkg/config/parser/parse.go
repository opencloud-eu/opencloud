// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"errors"
	"fmt"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	ocdefaults "github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config/defaults"

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

// Validate validates the config
func Validate(cfg *config.Config) error {
	if cfg.TokenManager == nil || cfg.TokenManager.JWTSecret == "" {
		return shared.MissingJWTTokenError(cfg.Service.Name)
	}
	if cfg.JWT.Secret == "" {
		return fmt.Errorf("the guest session secret has not been set properly in your config for %s. "+
			"Make sure your %s config contains the proper values "+
			"(e.g. by using 'opencloud init --diff' and applying the patch or setting a value manually in "+
			"the config/corresponding environment variable AUTH_GUEST_SESSION_JWT_SECRET)",
			cfg.Service.Name, ocdefaults.BaseConfigPath())
	}
	// The guest session token and the reva access token are both HS256 JWTs. Signing them
	// with the same key would make them interchangeable.
	if cfg.JWT.Secret == cfg.TokenManager.JWTSecret {
		return fmt.Errorf("the guest session secret (AUTH_GUEST_SESSION_JWT_SECRET) of %s must differ from the jwt secret (OC_JWT_SECRET)", cfg.Service.Name)
	}
	return nil
}
