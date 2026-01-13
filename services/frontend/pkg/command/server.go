package command

import (
	"context"
	"fmt"
	"os/signal"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/frontend/pkg/config"
	"github.com/opencloud-eu/opencloud/services/frontend/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/frontend/pkg/revaconfig"
	"github.com/opencloud-eu/opencloud/services/frontend/pkg/server/debug"
	"github.com/opencloud-eu/reva/v2/cmd/revad/runtime"

	"github.com/spf13/cobra"
)

// Server is the entry point for the server command.
func Server(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := log.Configure(cfg.Service.Name, cfg.Commons, cfg.LogLevel)
			traceProvider, err := tracing.GetTraceProvider(cmd.Context(), cfg.Commons.TracesExporter, cfg.Service.Name)
			if err != nil {
				return err
			}

			var cancel context.CancelFunc
			if cfg.Context == nil {
				cfg.Context, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}
			ctx := cfg.Context

			gr := runner.NewGroup()

			{
				rCfg, err := revaconfig.FrontendConfigFromStruct(cfg, logger)
				if err != nil {
					return err
				}

				// run the appropriate reva servers based on the config
				if rServer := runtime.NewDrivenHTTPServerWithOptions(rCfg,
					runtime.WithLogger(&logger.Logger),
					runtime.WithRegistry(registry.GetRegistry()),
					runtime.WithTraceProvider(traceProvider),
				); rServer != nil {
					gr.Add(runner.NewRevaServiceRunner(cfg.Service.Name+".rhttp", rServer))
				}
				if rServer := runtime.NewDrivenGRPCServerWithOptions(rCfg,
					runtime.WithLogger(&logger.Logger),
					runtime.WithRegistry(registry.GetRegistry()),
					runtime.WithTraceProvider(traceProvider),
				); rServer != nil {
					gr.Add(runner.NewRevaServiceRunner(cfg.Service.Name+".rgrpc", rServer))
				}
			}

			{
				debugServer, err := debug.Server(
					debug.Logger(logger),
					debug.Context(ctx),
					debug.Config(cfg),
				)
				if err != nil {
					logger.Info().Err(err).Str("server", "debug").Msg("Failed to initialize server")
					return err
				}

				gr.Add(runner.NewGolangHttpServerRunner(cfg.Service.Name+".debug", debugServer))
			}

			httpSvc := registry.BuildHTTPService(cfg.HTTP.Namespace+"."+cfg.Service.Name, cfg.HTTP.Addr, version.GetString())
			if err := registry.RegisterService(ctx, logger, httpSvc, cfg.Debug.Addr); err != nil {
				logger.Fatal().Err(err).Msg("failed to register the http service")
			}

			// add event handler
			gr.Add(runner.New(cfg.Service.Name+".event",
				func() error {
					return ListenForEvents(ctx, cfg, logger)
				}, func() {
					logger.Info().Msg("stopping event handler")
				},
			))

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
