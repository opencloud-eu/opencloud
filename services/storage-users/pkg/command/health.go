package command

import (
	"fmt"
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config/parser"

	"github.com/spf13/cobra"
)

// Health is the entrypoint for the health command.
func Health(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "check health status",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnError(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := log.Configure(cfg.Service.Name, cfg.Commons, cfg.LogLevel)

			resp, err := http.Get(
				fmt.Sprintf(
					"http://%s/healthz",
					cfg.Debug.Addr,
				),
			)

			if err != nil {
				logger.Fatal().
					Err(err).
					Msg("Failed to request health check")
			}

			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				logger.Fatal().
					Int("code", resp.StatusCode).
					Msg("Health seems to be in bad state")
			}

			logger.Debug().
				Int("code", resp.StatusCode).
				Msg("Health got a good state")

			return nil
		},
	}
}
