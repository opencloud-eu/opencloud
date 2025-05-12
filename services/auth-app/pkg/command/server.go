package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"

	"github.com/gofrs/uuid"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	ogrpc "github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/revaconfig"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/server/http"
	"github.com/opencloud-eu/reva/v2/cmd/revad/runtime"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/urfave/cli/v2"
)

// Server is the entry point for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		Category: "server",
		Before: func(_ *cli.Context) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		Action: func(c *cli.Context) error {
			if cfg.AllowImpersonation {
				fmt.Println("WARNING: Impersonation is enabled. Admins can impersonate all users.")
			}

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

				pidFile := path.Join(os.TempDir(), "revad-"+cfg.Service.Name+"-"+uuid.Must(uuid.NewV4()).String()+".pid")
				rCfg := revaconfig.AuthAppConfigFromStruct(cfg)
				reg := registry.GetRegistry()

				revaSrv := runtime.RunDrivenServerWithOptions(rCfg, pidFile,
					runtime.WithLogger(&logger.Logger),
					runtime.WithRegistry(reg),
					runtime.WithTraceProvider(traceProvider),
				)
				gr.Add(runner.NewRevaServiceRunner("auth-app_revad", revaSrv))

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
