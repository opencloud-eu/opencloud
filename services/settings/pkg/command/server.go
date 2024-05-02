package command

import (
	"context"
	"fmt"
	"os/signal"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	ogrpc "github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/config"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/server/grpc"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/server/http"
	svc "github.com/opencloud-eu/opencloud/services/settings/pkg/service/v0"
	"github.com/urfave/cli/v2"
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

			var cancel context.CancelFunc
			ctx := cfg.Context
			if ctx == nil {
				ctx, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}

			mtrcs := metrics.New()
			mtrcs.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			handle := svc.NewDefaultLanguageService(cfg, svc.NewService(cfg, logger))

			servers := runner.NewGroup()

			// prepare an HTTP server and add it to the group run.
			httpServer, err := http.Server(
				http.Name(cfg.Service.Name),
				http.Logger(logger),
				http.Context(ctx),
				http.Config(cfg),
				http.Metrics(mtrcs),
				http.ServiceHandler(handle),
				http.TraceProvider(traceProvider),
			)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Error initializing http service")
				return fmt.Errorf("could not initialize http service: %w", err)
			}
			servers.Add(runner.NewGoMicroHttpServerRunner("settings_http", httpServer))

			// prepare a gRPC server and add it to the group run.
			grpcServer := grpc.Server(
				grpc.Name(cfg.Service.Name),
				grpc.Logger(logger),
				grpc.Context(ctx),
				grpc.Config(cfg),
				grpc.Metrics(mtrcs),
				grpc.ServiceHandler(handle),
				grpc.TraceProvider(traceProvider),
			)
			servers.Add(runner.NewGoMicroGrpcServerRunner("settings_grpc", grpcServer))

			// prepare a debug server and add it to the group run.
			debugServer, err := debug.Server(
				debug.Logger(logger),
				debug.Context(ctx),
				debug.Config(cfg),
			)
			if err != nil {
				logger.Error().Err(err).Str("server", "debug").Msg("Failed to initialize server")
				return err
			}

			servers.Add(runner.NewGolangHttpServerRunner("settings_debug", debugServer))

			grResults := servers.Run(ctx)

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
