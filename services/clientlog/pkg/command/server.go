package command

import (
	"context"
	"fmt"

	"github.com/oklog/run"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/clientlog/pkg/config"
	"github.com/opencloud-eu/opencloud/services/clientlog/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/clientlog/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/clientlog/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/clientlog/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/clientlog/pkg/service"
)

// all events we care about
var _registeredEvents = []events.Unmarshaller{
	events.UploadReady{},
	events.ItemTrashed{},
	events.ItemRestored{},
	events.ItemMoved{},
	events.ContainerCreated{},
	events.FileLocked{},
	events.FileUnlocked{},
	events.FileTouched{},
	events.SpaceShared{},
	events.SpaceShareUpdated{},
	events.SpaceUnshared{},
	events.ShareCreated{},
	events.ShareRemoved{},
	events.ShareUpdated{},
	events.LinkCreated{},
	events.LinkUpdated{},
	events.LinkRemoved{},
	events.BackchannelLogout{},
}

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
			tracerProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}

			gr := run.Group{}
			ctx, cancel := context.WithCancel(c.Context)

			mtrcs := metrics.New()
			mtrcs.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			defer cancel()

			connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
			s, err := stream.NatsFromConfig(connName, false, stream.NatsConfig(cfg.Events))
			if err != nil {
				return err
			}

			tm, err := pool.StringToTLSMode(cfg.GRPCClientTLS.Mode)
			if err != nil {
				return err
			}
			gatewaySelector, err := pool.GatewaySelector(
				cfg.RevaGateway,
				pool.WithTLSCACert(cfg.GRPCClientTLS.CACert),
				pool.WithTLSMode(tm),
				pool.WithRegistry(registry.GetRegistry()),
				pool.WithTracerProvider(tracerProvider),
			)
			if err != nil {
				return fmt.Errorf("could not get reva client selector: %s", err)
			}

			{
				svc, err := service.NewClientlogService(
					service.Logger(logger),
					service.Config(cfg),
					service.Stream(s),
					service.GatewaySelector(gatewaySelector),
					service.RegisteredEvents(_registeredEvents),
					service.TraceProvider(tracerProvider),
				)

				if err != nil {
					logger.Info().Err(err).Str("transport", "http").Msg("Failed to initialize server")
					return err
				}

				gr.Add(func() error {
					return svc.Run()
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
