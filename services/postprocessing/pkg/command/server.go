package command

import (
	"context"
	"fmt"
	"os"

	"github.com/oklog/run"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/urfave/cli/v2"
	microstore "go-micro.dev/v4/store"

	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/config"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/service"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    fmt.Sprintf("start %s service without runtime (unsupervised mode)", cfg.Service.Name),
		Category: "server",
		Before: func(c *cli.Context) error {
			err := parser.ParseConfig(cfg)
			if err != nil {
				fmt.Printf("%v", err)
				os.Exit(1)
			}
			return err
		},
		Action: func(c *cli.Context) error {
			var (
				gr          = run.Group{}
				logger      = logging.Configure(cfg.Service.Name, cfg.Log)
				ctx, cancel = context.WithCancel(c.Context)
			)
			defer cancel()

			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}

			{
				st := store.Create(
					store.Store(cfg.Store.Store),
					store.TTL(cfg.Store.TTL),
					microstore.Nodes(cfg.Store.Nodes...),
					microstore.Database(cfg.Store.Database),
					microstore.Table(cfg.Store.Table),
					store.Authentication(cfg.Store.AuthUsername, cfg.Store.AuthPassword),
				)

				svc, err := service.NewPostprocessingService(ctx, logger, st, traceProvider, cfg)
				if err != nil {
					return err
				}
				gr.Add(func() error {
					err := make(chan error, 1)
					select {
					case <-ctx.Done():
						return nil

					case err <- svc.Run():
						return <-err
					}
				}, func(err error) {
					if err != nil {
						logger.Info().
							Str("transport", "stream").
							Str("server", cfg.Service.Name).
							Msg("Shutting down server")
					} else {
						logger.Error().Err(err).
							Str("transport", "stream").
							Str("server", cfg.Service.Name).
							Msg("Shutting down server")
					}

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
					logger.Info().Err(err).Str("transport", "debug").Msg("Failed to initialize server")
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
