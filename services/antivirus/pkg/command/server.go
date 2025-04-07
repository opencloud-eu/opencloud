package command

import (
	"context"
	"fmt"

	"github.com/oklog/run"
	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/service"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		Category: "server",
		Before: func(c *cli.Context) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		Action: func(c *cli.Context) error {
			var (
				gr          = run.Group{}
				ctx, cancel = context.WithCancel(c.Context)
				logger      = log.NewLogger(
					log.Name(cfg.Service.Name),
					log.Level(cfg.Log.Level),
					log.Pretty(cfg.Log.Pretty),
					log.Color(cfg.Log.Color),
					log.File(cfg.Log.File),
				)
			)
			defer cancel()
			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}
			{
				svc, err := service.NewAntivirus(cfg, logger, traceProvider)
				if err != nil {
					return cli.Exit(err.Error(), 1)
				}

				gr.Add(svc.Run, func(_ error) {
					cancel()
				})
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

				gr.Add(debugServer.ListenAndServe, func(_ error) {
					_ = debugServer.Shutdown(ctx)
					cancel()
				})
			}

			return gr.Run()
		},
	}
}
