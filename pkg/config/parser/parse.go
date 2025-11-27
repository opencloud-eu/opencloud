package parser

import (
	"errors"

	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

// ParseConfig loads the opencloud configuration and
// copies applicable parts into the commons part, from
// where the services can copy it into their own config
func ParseConfig(cfg *config.Config, skipValidate bool) error {
	err := config.BindSourcesToStructs("opencloud", cfg)
	if err != nil {
		return err
	}

	EnsureDefaults(cfg)

	// load all env variables relevant to the config in the current context.
	if err := envdecode.Decode(cfg); err != nil {
		// no environment variable set for this config is an expected "error"
		if !errors.Is(err, envdecode.ErrNoTargetFieldsAreSet) {
			return err
		}
	}

	EnsureCommons(cfg)

	if skipValidate {
		return nil
	}

	return Validate(cfg)
}

// EnsureDefaults ensures that all pointers in the
// OpenCloud config (not the services configs) are initialized
func EnsureDefaults(cfg *config.Config) {
	if cfg.Log == nil {
		cfg.Log = &shared.Log{}
	}
	if cfg.TokenManager == nil {
		cfg.TokenManager = &shared.TokenManager{}
	}
	if cfg.Cache == nil {
		cfg.Cache = &shared.Cache{}
	}
	if cfg.GRPCClientTLS == nil {
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}
	}
	if cfg.GRPCServiceTLS == nil {
		cfg.GRPCServiceTLS = &shared.GRPCServiceTLS{}
	}
	if cfg.Reva == nil {
		cfg.Reva = &shared.Reva{}
	}
}

// EnsureCommons copies applicable parts of the OpenCloud config into the commons part
func EnsureCommons(cfg *config.Config) {
	// ensure the commons part is initialized
	if cfg.Commons == nil {
		cfg.Commons = &shared.Commons{}
	}

	cfg.Commons.Log = structs.CopyOrZeroValue(cfg.Log)
	cfg.Commons.Cache = structs.CopyOrZeroValue(cfg.Cache)

	if cfg.GRPCClientTLS != nil {
		cfg.Commons.GRPCClientTLS = cfg.GRPCClientTLS
	}

	if cfg.GRPCServiceTLS != nil {
		cfg.Commons.GRPCServiceTLS = cfg.GRPCServiceTLS
	}

	cfg.Commons.HTTPServiceTLS = cfg.HTTPServiceTLS

	cfg.Commons.TokenManager = structs.CopyOrZeroValue(cfg.TokenManager)

	// copy machine auth api key to the commons part if set
	if cfg.MachineAuthAPIKey != "" {
		cfg.Commons.MachineAuthAPIKey = cfg.MachineAuthAPIKey
	}

	if cfg.SystemUserAPIKey != "" {
		cfg.Commons.SystemUserAPIKey = cfg.SystemUserAPIKey
	}

	// copy transfer secret to the commons part if set
	if cfg.TransferSecret != "" {
		cfg.Commons.TransferSecret = cfg.TransferSecret
	}

	// make sure url signing secret is set and copy it to the commons part
	// fall back to transfer secret for url signing secret to avoid
	// issues when upgrading from an older release
	if cfg.URLSigningSecret == "" {
		cfg.URLSigningSecret = cfg.TransferSecret
	}
	cfg.Commons.URLSigningSecret = cfg.URLSigningSecret

	// copy metadata user id to the commons part if set
	if cfg.SystemUserID != "" {
		cfg.Commons.SystemUserID = cfg.SystemUserID
	}

	// copy admin user id to the commons part if set
	if cfg.AdminUserID != "" {
		cfg.Commons.AdminUserID = cfg.AdminUserID
	}

	if cfg.OpenCloudURL != "" {
		cfg.Commons.OpenCloudURL = cfg.OpenCloudURL
	}

	cfg.Commons.Reva = structs.CopyOrZeroValue(cfg.Reva)
}

// Validate checks that all required configs are set. If a required config value
// is missing an error will be returned.
func Validate(cfg *config.Config) error {
	if cfg.TokenManager.JWTSecret == "" {
		return shared.MissingJWTTokenError("opencloud")
	}

	if cfg.TransferSecret == "" {
		return shared.MissingRevaTransferSecretError("opencloud")
	}

	if cfg.URLSigningSecret == "" {
		return shared.MissingURLSigningSecret("opencloud")
	}

	if cfg.MachineAuthAPIKey == "" {
		return shared.MissingMachineAuthApiKeyError("opencloud")
	}

	if cfg.SystemUserID == "" {
		return shared.MissingSystemUserID("opencloud")
	}

	return nil
}
