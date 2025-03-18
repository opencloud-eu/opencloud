package command

import (
	"context"
	"fmt"

	"github.com/oklog/run"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	svcProtogen "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/policies/v0"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/config"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine/opa"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/server/debug"
	svcEvent "github.com/opencloud-eu/opencloud/services/policies/pkg/service/event"
	svcGRPC "github.com/opencloud-eu/opencloud/services/policies/pkg/service/grpc"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    fmt.Sprintf("start the %s service without runtime (unsupervised mode)", "authz"),
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
				).SubloggerWithRequestID(ctx)
			)
			defer cancel()

			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}

			e, err := opa.NewOPA(cfg.Engine.Timeout, logger, cfg.Engine)
			if err != nil {
				return err
			}

			{
				grpcClient, err := grpc.NewClient(
					append(
						grpc.GetClientOptions(cfg.GRPCClientTLS),
						grpc.WithTraceProvider(traceProvider),
					)...,
				)
				if err != nil {
					return err
				}

				svc, err := grpc.NewServiceWithClient(
					grpcClient,
					grpc.Logger(logger),
					grpc.TLSEnabled(cfg.GRPC.TLS.Enabled),
					grpc.TLSCert(
						cfg.GRPC.TLS.Cert,
						cfg.GRPC.TLS.Key,
					),
					grpc.Name(cfg.Service.Name),
					grpc.Context(ctx),
					grpc.Address(cfg.GRPC.Addr),
					grpc.Namespace(cfg.GRPC.Namespace),
					grpc.Version(version.GetString()),
					grpc.TraceProvider(traceProvider),
				)
				if err != nil {
					return err
				}

				grpcSvc, err := svcGRPC.New(e)
				if err != nil {
					return err
				}

				if err := svcProtogen.RegisterPoliciesProviderHandler(
					svc.Server(),
					grpcSvc,
				); err != nil {
					return err
				}

				gr.Add(svc.Run, func(_ error) {
					cancel()
				})
			}

			{

				connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTYPE_BUS)
				bus, err := stream.NatsFromConfig(connName, false, stream.NatsConfig(cfg.Events))
				if err != nil {
					return err
				}

				eventSvc, err := svcEvent.New(ctx, bus, logger, traceProvider, e, cfg.Postprocessing.Query)
				if err != nil {
					return err
				}

				gr.Add(eventSvc.Run, func(_ error) {
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
