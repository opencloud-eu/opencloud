package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"

	"github.com/gofrs/uuid"
	"github.com/opencloud-eu/reva/v2/cmd/revad/runtime"
	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/frontend/pkg/config"
	"github.com/opencloud-eu/opencloud/services/frontend/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/frontend/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/frontend/pkg/revaconfig"
	"github.com/opencloud-eu/opencloud/services/frontend/pkg/server/debug"
)

// Server is the entry point for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		Category: "server",
		Before: func(c *cli.Context) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		Action: func(c *cli.Context) error {
			logger := logging.Configure(cfg.Service.Name, cfg.Log)
			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}

			var cancel context.CancelFunc
			ctx := cfg.Context
			if ctx == nil {
				ctx, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}

			gr := runner.NewGroup()

			{
				rCfg, err := revaconfig.FrontendConfigFromStruct(cfg, logger)
				if err != nil {
					return err
				}
				pidFile := path.Join(os.TempDir(), "revad-"+cfg.Service.Name+"-"+uuid.Must(uuid.NewV4()).String()+".pid")
				reg := registry.GetRegistry()

				revaSrv := runtime.RunDrivenServerWithOptions(rCfg, pidFile,
					runtime.WithLogger(&logger.Logger),
					runtime.WithRegistry(reg),
					runtime.WithTraceProvider(traceProvider),
				)
				gr.Add(runner.NewRevaServiceRunner("frontend_revad", revaSrv))
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

				gr.Add(runner.NewGolangHttpServerRunner("frontend_debug", debugServer))
			}

			httpSvc := registry.BuildHTTPService(cfg.HTTP.Namespace+"."+cfg.Service.Name, cfg.HTTP.Addr, version.GetString())
			if err := registry.RegisterService(ctx, logger, httpSvc, cfg.Debug.Addr); err != nil {
				logger.Fatal().Err(err).Msg("failed to register the http service")
			}

			// add event handler
			gr.Add(runner.New("frontend_event",
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
