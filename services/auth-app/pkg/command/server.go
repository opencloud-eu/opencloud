package command

import (
	"context"
	"fmt"
	"os/signal"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	ogrpc "github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/revaconfig"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/server/http"
	"github.com/opencloud-eu/reva/v2/cmd/revad/runtime"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"

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
			if cfg.AllowImpersonation {
				fmt.Println("WARNING: Impersonation is enabled. Admins can impersonate all users.")
			}

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
				// run the appropriate reva servers based on the config
				rCfg := revaconfig.AuthAppConfigFromStruct(cfg)
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

				gr.Add(runner.NewGolangHttpServerRunner("auth-app_debug", debugServer))
			}

			grpcSvc := registry.BuildGRPCService(cfg.GRPC.Namespace+"."+cfg.Service.Name, cfg.GRPC.Protocol, cfg.GRPC.Addr, version.GetString())
			if err := registry.RegisterService(ctx, logger, grpcSvc, cfg.Debug.Addr); err != nil {
				logger.Fatal().Err(err).Msg("failed to register the grpc service")
			}

			tm, err := pool.StringToTLSMode(cfg.GRPCClientTLS.Mode)
			if err != nil {
				return err
			}
			gatewaySelector, err := pool.GatewaySelector(
				cfg.Reva.Address,
				append(
					cfg.Reva.GetRevaOptions(),
					pool.WithTLSCACert(cfg.GRPCClientTLS.CACert),
					pool.WithTLSMode(tm),
					pool.WithRegistry(registry.GetRegistry()),
					pool.WithTracerProvider(traceProvider),
				)...)
			if err != nil {
				return err
			}

			grpcClient, err := ogrpc.NewClient(
				append(ogrpc.GetClientOptions(cfg.GRPCClientTLS), ogrpc.WithTraceProvider(traceProvider))...,
			)
			if err != nil {
				return err
			}

			{
				rClient := settingssvc.NewRoleService("eu.opencloud.api.settings", grpcClient)
				server, err := http.Server(
					http.Logger(logger),
					http.Context(ctx),
					http.Config(cfg),
					http.GatewaySelector(gatewaySelector),
					http.RoleClient(rClient),
					http.TracerProvider(traceProvider),
				)
				if err != nil {
					logger.Fatal().Err(err).Msg("failed to initialize http server")
				}

				gr.Add(runner.NewGoMicroHttpServerRunner("auth-app_http", server))
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
