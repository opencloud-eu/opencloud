package command

import (
	"context"
	"fmt"

	"github.com/oklog/run"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/server/http"

	"github.com/spf13/cobra"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := logging.Configure(cfg.Service.Name, cfg.Log)

			var (
				gr          = run.Group{}
				ctx, cancel = context.WithCancel(context.Background())
				m           = metrics.New()
			)

			defer cancel()

			m.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			server, err := debug.Server(
				debug.Logger(logger),
				debug.Config(cfg),
				debug.Context(ctx),
			)
			if err != nil {
				logger.Info().Err(err).Str("transport", "debug").Msg("Failed to initialize server")
				return err
			}

			gr.Add(server.ListenAndServe, func(_ error) {
				_ = server.Shutdown(ctx)
				cancel()
			})

			httpServer, err := http.Server(
				http.Logger(logger),
				http.Context(ctx),
				http.Config(cfg),
				http.Metrics(m),
				http.Namespace(cfg.HTTP.Namespace),
			)
			if err != nil {
				logger.Info().
					Err(err).
					Str("transport", "http").
					Msg("Failed to initialize server")

				return err
			}

			gr.Add(httpServer.Run, func(_ error) {
				if err == nil {
					logger.Info().
						Str("transport", "http").
						Str("server", cfg.Service.Name).
						Msg("Shutting down server")
				} else {
					logger.Error().Err(err).
						Str("transport", "http").
						Str("server", cfg.Service.Name).
						Msg("Shutting down server")
				}

				cancel()
			})

			return gr.Run()
		},
	}
}
