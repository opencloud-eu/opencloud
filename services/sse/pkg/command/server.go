package command

import (
	"context"
	"fmt"
	"os/signal"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/sse/pkg/config"
	"github.com/opencloud-eu/opencloud/services/sse/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/sse/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/sse/pkg/server/http"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"

	"github.com/spf13/cobra"
)

// all events we care about
var _registeredEvents = []events.Unmarshaller{
	events.SendSSE{},
}

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var cancel context.CancelFunc
			if cfg.Context == nil {
				cfg.Context, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}
			ctx := cfg.Context

			logger := log.Configure(cfg.Service.Name, cfg.Commons, cfg.LogLevel)

			tracerProvider, err := tracing.GetTraceProvider(cmd.Context(), cfg.Commons.TracesExporter, cfg.Service.Name)
			if err != nil {
				return err
			}

			gr := runner.NewGroup()
			{
				connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
				natsStream, err := stream.NatsFromConfig(connName, true, stream.NatsConfig(cfg.Events))
				if err != nil {
					return err
				}

				server, err := http.Server(
					http.Logger(logger),
					http.Context(ctx),
					http.Config(cfg),
					http.Consumer(natsStream),
					http.RegisteredEvents(_registeredEvents),
					http.TracerProvider(tracerProvider),
				)
				if err != nil {
					return err
				}

				gr.Add(runner.NewGoMicroHttpServerRunner(cfg.Service.Name+".http", server))
			}

			{
				debugServer, err := debug.Server(
					debug.Logger(logger),
					debug.Context(ctx),
					debug.Config(cfg),
				)
				if err != nil {
					logger.Info().Err(err).Str("transport", "debug").Msg("Failed to initialize server")
					return err
				}

				gr.Add(runner.NewGolangHttpServerRunner(cfg.Service.Name+".debug", debugServer))
			}

			grResults := gr.Run(ctx)

			// return the first non-nil error found in the results
			for _, grResult := range grResults {
				if grResult.RunnerError != nil {
					return grResult.RunnerError
				}
			}
			return nil
		},
	}
}
