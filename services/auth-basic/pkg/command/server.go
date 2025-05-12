package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"

	"github.com/gofrs/uuid"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/ldap"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/auth-basic/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-basic/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/auth-basic/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/auth-basic/pkg/revaconfig"
	"github.com/opencloud-eu/opencloud/services/auth-basic/pkg/server/debug"
	"github.com/opencloud-eu/reva/v2/cmd/revad/runtime"
	"github.com/urfave/cli/v2"
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

			// the reva runtime calls `os.Exit` in the case of a failure and there is no way for the OpenCloud
			// runtime to catch it and restart a reva service. Therefore, we need to ensure the service has
			// everything it needs, before starting the service.
			// In this case: CA certificates
			if cfg.AuthProvider == "ldap" {
				ldapCfg := cfg.AuthProviders.LDAP
				if err := ldap.WaitForCA(logger, ldapCfg.Insecure, ldapCfg.CACert); err != nil {
					logger.Error().Err(err).Msg("The configured LDAP CA cert does not exist")
					return err
				}
			}

			{
				pidFile := path.Join(os.TempDir(), "revad-"+cfg.Service.Name+"-"+uuid.Must(uuid.NewV4()).String()+".pid")
				rCfg := revaconfig.AuthBasicConfigFromStruct(cfg)
				reg := registry.GetRegistry()

				revaSrv := runtime.RunDrivenServerWithOptions(rCfg, pidFile,
					runtime.WithLogger(&logger.Logger),
					runtime.WithRegistry(reg),
					runtime.WithTraceProvider(traceProvider),
				)
				gr.Add(runner.NewRevaServiceRunner("auth-basic_revad", revaSrv))
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

				gr.Add(runner.NewGolangHttpServerRunner("auth-basic_debug", debugServer))
			}

			grpcSvc := registry.BuildGRPCService(cfg.GRPC.Namespace+"."+cfg.Service.Name, cfg.GRPC.Protocol, cfg.GRPC.Addr, version.GetString())
			if err := registry.RegisterService(ctx, logger, grpcSvc, cfg.Debug.Addr); err != nil {
				logger.Fatal().Err(err).Msg("failed to register the grpc service")
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
