package command

import (
	"context"
	"fmt"

	"github.com/oklog/run"
	"github.com/opencloud-eu/opencloud/pkg/broker"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/ocdav/pkg/config"
	"github.com/opencloud-eu/opencloud/services/ocdav/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/ocdav/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/ocdav/pkg/server/debug"
	"github.com/opencloud-eu/reva/v2/pkg/micro/ocdav"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
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
			gr := run.Group{}
			ctx, cancel := context.WithCancel(c.Context)

			defer cancel()

			gr.Add(func() error {
				// init reva shared config explicitly as the go-micro based ocdav does not use
				// the reva runtime. But we need e.g. the shared client settings to be initialized
				sc := map[string]interface{}{
					"jwt_secret":                cfg.TokenManager.JWTSecret,
					"gatewaysvc":                cfg.Reva.Address,
					"skip_user_groups_in_token": cfg.SkipUserGroupsInToken,
					"grpc_client_options":       cfg.Reva.GetGRPCClientConfig(),
				}
				if err := sharedconf.Decode(sc); err != nil {
					logger.Error().Err(err).Msg("error decoding shared config for ocdav")
				}
				opts := []ocdav.Option{
					ocdav.Name(cfg.HTTP.Namespace + "." + cfg.Service.Name),
					ocdav.Version(version.GetString()),
					ocdav.Context(ctx),
					ocdav.Logger(logger.Logger),
					ocdav.Address(cfg.HTTP.Addr),
					ocdav.AllowCredentials(cfg.HTTP.CORS.AllowCredentials),
					ocdav.AllowedMethods(cfg.HTTP.CORS.AllowedMethods),
					ocdav.AllowedHeaders(cfg.HTTP.CORS.AllowedHeaders),
					ocdav.AllowedOrigins(cfg.HTTP.CORS.AllowedOrigins),
					ocdav.FilesNamespace(cfg.FilesNamespace),
					ocdav.WebdavNamespace(cfg.WebdavNamespace),
					ocdav.OCMNamespace(cfg.OCMNamespace),
					ocdav.AllowDepthInfinity(cfg.AllowPropfindDepthInfinity),
					ocdav.SharesNamespace(cfg.SharesNamespace),
					ocdav.Timeout(cfg.Timeout),
					ocdav.Insecure(cfg.Insecure),
					ocdav.PublicURL(cfg.PublicURL),
					ocdav.Prefix(cfg.HTTP.Prefix),
					ocdav.GatewaySvc(cfg.Reva.Address),
					ocdav.JWTSecret(cfg.TokenManager.JWTSecret),
					ocdav.ProductName(cfg.Status.ProductName),
					ocdav.ProductVersion(cfg.Status.ProductVersion),
					ocdav.Product(cfg.Status.Product),
					ocdav.Version(cfg.Status.Version),
					ocdav.VersionString(cfg.Status.VersionString),
					ocdav.Edition(cfg.Status.Edition),
					ocdav.MachineAuthAPIKey(cfg.MachineAuthAPIKey),
					ocdav.Broker(broker.NoOp{}),
					// ocdav.FavoriteManager() // FIXME needs a proper persistence implementation https://github.com/owncloud/ocis/issues/1228
					// ocdav.LockSystem(), // will default to the CS3 lock system
					// ocdav.TLSConfig() // tls config for the http server
					ocdav.MetricsEnabled(true),
					ocdav.MetricsNamespace("opencloud"),
					ocdav.Tracing("Adding these strings is a workaround for ->", "https://github.com/cs3org/reva/issues/4131"),
					ocdav.WithTraceProvider(traceProvider),
					ocdav.RegisterTTL(registry.GetRegisterTTL()),
					ocdav.RegisterInterval(registry.GetRegisterInterval()),
				}

				s, err := ocdav.Service(opts...)
				if err != nil {
					return err
				}

				return s.Run()
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

			return gr.Run()
		},
	}
}
