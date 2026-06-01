package command

import (
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/auth-authelia/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-authelia/pkg/config/parser"

	"github.com/spf13/cobra"
)

// Health is the entrypoint for the health command.
//
// Authelia exposes its own health endpoint on the listener it manages itself (see the rendered
// authelia.yaml). The OpenCloud service therefore only validates that its configuration can be
// parsed and that the Authelia configuration file is referenced.
func Health(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "check health status",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnError(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := log.Configure(cfg.Service.Name, cfg.Commons, cfg.LogLevel)
			logger.Debug().Str("config", cfg.ConfigPath).Msg("Health configuration is valid")
			return nil
		},
	}
}
