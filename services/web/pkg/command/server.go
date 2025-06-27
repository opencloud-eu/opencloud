package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/oklog/run"
	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	ogrpc "github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/web/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/web/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/web/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/web/pkg/server/grpc"
	"github.com/opencloud-eu/opencloud/services/web/pkg/server/http"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		Category: "server",
		Before: func(_ *cli.Context) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		Action: func(c *cli.Context) error {
			logger := logging.Configure(cfg.Service.Name, cfg.Log)
			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}

			// actually read the contents of the config file and override defaults
			if cfg.File != "" {
				contents, err := os.ReadFile(cfg.File)
				if err != nil {
					logger.Err(err).Msg("error opening config file")
					return err
				}
				if err = json.Unmarshal(contents, &cfg.Web.Config); err != nil {
					logger.Fatal().Err(err).Msg("error unmarshalling config file")
					return err
				}
			}

			cfg.GrpcClient, err = ogrpc.NewClient(
				append(ogrpc.GetClientOptions(cfg.GRPCClientTLS), ogrpc.WithTraceProvider(traceProvider))...,
			)

			var (
				gr          = run.Group{}
				ctx, cancel = context.WithCancel(c.Context)
				m           = metrics.New()
			)

			defer cancel()

			{
				server, err := http.Server(
					http.Logger(logger),
					http.Context(ctx),
					http.Namespace(cfg.HTTP.Namespace),
					http.Config(cfg),
					http.Metrics(m),
					http.TraceProvider(traceProvider),
				)
				if err != nil {
					logger.Info().
						Err(err).
						Str("transport", "http").
						Msg("Failed to initialize server")

					return err
				}

				gr.Add(func() error {
					err := server.Run()
					if err != nil {
						logger.Error().
							Err(err).
							Str("transport", "http").
							Msg("Failed to start server")
					}
					return err
				}, func(err error) {
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
			}

			{
				grpcServer, err := grpc.Server(
					grpc.Config(cfg),
					grpc.Logger(logger),
					grpc.Name(cfg.Service.Name),
					grpc.Context(ctx),
					grpc.JWTSecret(cfg.TokenManager.JWTSecret),
					grpc.TraceProvider(traceProvider),
				)
				if err != nil {
					logger.Info().
						Err(err).
						Str("transport", "grpc").
						Msg("Failed to initialize server")
					return err
				}

				gr.Add(func() error {
					err := grpcServer.Run()
					if err != nil {
						logger.Error().
							Err(err).
							Str("transport", "grpc").
							Msg("Failed to start server")
					}
					return err
				}, func(err error) {
					if err == nil {
						logger.Info().
							Str("transport", "grpc").
							Str("server", cfg.Service.Name).
							Msg("Shutting down server")
					} else {
						logger.Error().Err(err).
							Str("transport", "grpc").
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
