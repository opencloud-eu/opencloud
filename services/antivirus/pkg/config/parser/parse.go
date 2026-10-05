package parser

import (
	"errors"
	"time"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config/defaults"

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

// Validate validates our little config
func Validate(cfg *config.Config) error {
	if cfg.Workers < 1 {
		return errors.New("ANTIVIRUS_WORKERS must be greater than zero")
	}
	if cfg.HighPriorityReservedWorkers < 0 || cfg.HighPriorityReservedWorkers >= cfg.Workers {
		return errors.New("ANTIVIRUS_HIGH_PRIORITY_RESERVED_WORKERS must be at least zero and less than ANTIVIRUS_WORKERS")
	}
	if cfg.PriorityThreshold < 1 {
		return errors.New("ANTIVIRUS_PRIORITY_THRESHOLD must be greater than zero")
	}
	if cfg.PriorityWindow <= 0 {
		return errors.New("ANTIVIRUS_PRIORITY_WINDOW must be greater than zero")
	}
	if cfg.PriorityCooldown <= 0 {
		return errors.New("ANTIVIRUS_PRIORITY_COOLDOWN must be greater than zero")
	}
	if cfg.QueueAckWait <= 0 {
		return errors.New("ANTIVIRUS_QUEUE_ACK_WAIT must be greater than zero")
	}
	if cfg.QueueAckWait >= 2*time.Minute {
		return errors.New("ANTIVIRUS_QUEUE_ACK_WAIT must be less than 2m")
	}
	if cfg.QueueReplicas < 1 {
		return errors.New("ANTIVIRUS_QUEUE_REPLICAS must be greater than zero")
	}
	return nil
}
