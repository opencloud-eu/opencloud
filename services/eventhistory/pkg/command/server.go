package command

import (
	"context"
	"fmt"

	"github.com/oklog/run"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/urfave/cli/v2"
	microstore "go-micro.dev/v4/store"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	ogrpc "github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/config"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/server/grpc"
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
			logger := logging.Configure(cfg.Service.Name, cfg.Log)
			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}

			cfg.GrpcClient, err = ogrpc.NewClient(
				append(ogrpc.GetClientOptions(cfg.GRPCClientTLS), ogrpc.WithTraceProvider(traceProvider))...,
			)
			if err != nil {
				return err
			}

			var (
				gr          = run.Group{}
				ctx, cancel = context.WithCancel(c.Context)
				m           = metrics.New()
			)

			defer cancel()

			m.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTYPE_BUS)
			consumer, err := stream.NatsFromConfig(connName, false, stream.NatsConfig(cfg.Events))
			if err != nil {
				return err
			}

			st := store.Create(
				store.Store(cfg.Store.Store),
				store.TTL(cfg.Store.TTL),
				microstore.Nodes(cfg.Store.Nodes...),
				microstore.Database(cfg.Store.Database),
				microstore.Table(cfg.Store.Table),
				store.Authentication(cfg.Store.AuthUsername, cfg.Store.AuthPassword),
			)

			service := grpc.NewService(
				grpc.Logger(logger),
				grpc.Context(ctx),
				grpc.Config(cfg),
				grpc.Name(cfg.Service.Name),
				grpc.Namespace(cfg.GRPC.Namespace),
				grpc.Address(cfg.GRPC.Addr),
				grpc.Metrics(m),
				grpc.Consumer(consumer),
				grpc.Persistence(st),
				grpc.TraceProvider(traceProvider),
			)

			gr.Add(service.Run, func(err error) {
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
