package command

import (
	"context"
	"fmt"
	"os/signal"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	ocmetrics "github.com/opencloud-eu/opencloud/pkg/metrics"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	svcProtogen "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/policies/v0"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/config"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine/opa"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/server/debug"
	svcEvent "github.com/opencloud-eu/opencloud/services/policies/pkg/service/event"
	svcGRPC "github.com/opencloud-eu/opencloud/services/policies/pkg/service/grpc"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/spf13/cobra"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: fmt.Sprintf("start the %s service without runtime (unsupervised mode)", "policies"),
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

			logger := log.Configure(cfg.Service.Name, cfg.Commons, cfg.LogLevel).SubloggerWithRequestID(ctx)

			traceProvider, err := tracing.GetTraceProvider(cmd.Context(), cfg.Commons.TracesExporter, cfg.Service.Name)
			if err != nil {
				return err
			}

			pathPrefixMap := map[string]func() string{
				"config:": defaults.BaseConfigPath,
				"data:":   defaults.BaseDataPath,
			}

			e, err := opa.NewOPA(cfg.Engine.Timeout, logger, cfg.Engine, pathPrefixMap)
			if err != nil {
				return err
			}

			m, err := metrics.New(ocmetrics.NewLoggingPrometheusRegisterer(prometheus.DefaultRegisterer, &logger), &logger)
			if err != nil {
				return err
			}
			if cfg.GRPC.Disabled {
				m.GrpcEnabled.Set(0)
			} else {
				m.GrpcEnabled.Set(1)
			}
			if cfg.Events.Disabled {
				m.EventsEnabled.Set(0)
			} else {
				m.EventsEnabled.Set(1)
			}

			gr := runner.NewGroup()
			if !cfg.GRPC.Disabled { // only run the GRPC consumer when enabled, https://github.com/opencloud-eu/opencloud/issues/1312
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

				grpcSvc, err := svcGRPC.New(e, m)
				if err != nil {
					return err
				}

				if err := svcProtogen.RegisterPoliciesProviderHandler(
					svc.Server(),
					grpcSvc,
				); err != nil {
					return err
				}

				gr.Add(runner.NewGoMicroGrpcServerRunner(cfg.Service.Name+".grpc", svc))
			}

			if !cfg.Events.Disabled { // only run the event consumer when enabled, https://github.com/opencloud-eu/opencloud/issues/1312
				connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
				bus, err := stream.NatsFromConfig(connName, false, cfg.Events.ToNatsConfig())
				if err != nil {
					return err
				}

				eventSvc, err := svcEvent.New(ctx, bus, logger, traceProvider, e, cfg.Postprocessing.Query, m)
				if err != nil {
					return err
				}

				gr.Add(runner.New(cfg.Service.Name+".svc", func() error {
					return eventSvc.Run()
				}, func() {
					eventSvc.Close()
				}))
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
