package parser

import (
	"errors"
	"fmt"
	"strings"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/config"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/config/defaults"
	"github.com/opencloud-eu/reva/v3/pkg/events"

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
	if cfg.Postprocessing.Delayprocessing != 0 {
		if !contains(cfg.Postprocessing.Steps, events.PPStepDelay) {
			if len(cfg.Postprocessing.Steps) > 0 {
				s := strings.Join(append(cfg.Postprocessing.Steps, string(events.PPStepDelay)), ",")
				fmt.Printf("Added delay step to the list of postprocessing steps. NOTE: Use envvar `POSTPROCESSING_STEPS=%s` to suppress this message and choose the order of postprocessing steps.\n", s)
			}

			cfg.Postprocessing.Steps = append(cfg.Postprocessing.Steps, string(events.PPStepDelay))
		}
	}
	return nil
}

func contains(all []string, candidate events.Postprocessingstep) bool {
	for _, s := range all {
		if s == string(candidate) {
			return true
		}
	}
	return false
}
