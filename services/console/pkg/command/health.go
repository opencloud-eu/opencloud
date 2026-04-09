package command

import (
	"github.com/spf13/cobra"

	"github.com/opencloud-eu/opencloud/services/console/pkg/config"
)

// Health is the entrypoint for the health command.
func Health(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:     "health",
		Short:   "check health status",
		PreRunE: preRunE(cfg),
		RunE: func(cmd *cobra.Command, args []string) error {
			// fixMe: what to check?
			// - remote service health // conn
			// - web service rpc health, theme state?
			return nil
		},
	}
}
