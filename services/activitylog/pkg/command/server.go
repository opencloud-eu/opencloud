package command

import (
	"context"
	"fmt"

	"github.com/oklog/run"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/urfave/cli/v2"
	microstore "go-micro.dev/v4/store"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	ogrpc "github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/config"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/server/http"
)

var _registeredEvents = []events.Unmarshaller{
	events.UploadReady{},
	events.FileTouched{},
	events.ContainerCreated{},
	events.FileDownloaded{},
	events.ItemTrashed{},
	events.ItemPurged{},
	events.ItemMoved{},
	events.ShareCreated{},
	events.ShareUpdated{},
	events.ShareRemoved{},
	events.LinkCreated{},
	events.LinkUpdated{},
	events.LinkRemoved{},
	events.SpaceShared{},
	events.SpaceUnshared{},
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
				logger.Error().Err(err).Msg("Failed to initialize tracer")
				return err
			}

			gr := run.Group{}
			ctx, cancel := context.WithCancel(c.Context)

			mtrcs := metrics.New()
			mtrcs.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			defer cancel()

			connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTYPE_BUS)
			evStream, err := stream.NatsFromConfig(connName, false, stream.NatsConfig(cfg.Events))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to initialize event stream")
				return err
			}

			evStore := store.Create(
				store.Store(cfg.Store.Store),
				store.TTL(cfg.Store.TTL),
				microstore.Nodes(cfg.Store.Nodes...),
				microstore.Database(cfg.Store.Database),
				microstore.Table(cfg.Store.Table),
				store.Authentication(cfg.Store.AuthUsername, cfg.Store.AuthPassword),
			)

			tm, err := pool.StringToTLSMode(cfg.GRPCClientTLS.Mode)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to parse tls mode")
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
				logger.Error().Err(err).Msg("Failed to initialize gateway selector")
				return fmt.Errorf("could not get reva client selector: %s", err)
			}

			grpcClient, err := ogrpc.NewClient(
				append(ogrpc.GetClientOptions(cfg.GRPCClientTLS), ogrpc.WithTraceProvider(tracerProvider))...,
			)
			if err != nil {
				return err
			}

			hClient := ehsvc.NewEventHistoryService("eu.opencloud.api.eventhistory", grpcClient)
			vClient := settingssvc.NewValueService("eu.opencloud.api.settings", grpcClient)

			{
				svc, err := http.Server(
					http.Logger(logger),
					http.Config(cfg),
					http.Context(ctx), // NOTE: not passing this "option" leads to a panic in go-micro
					http.TraceProvider(tracerProvider),
					http.Stream(evStream),
					http.Store(evStore),
					http.GatewaySelector(gatewaySelector),
					http.HistoryClient(hClient),
					http.ValueClient(vClient),
					http.RegisteredEvents(_registeredEvents),
				)

				if err != nil {
					logger.Error().Err(err).Str("transport", "http").Msg("Failed to initialize server")
					return err
				}

				gr.Add(func() error {
					return svc.Run()
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
