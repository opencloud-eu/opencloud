package command

import (
	"context"
	"fmt"
	"reflect"

	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	microstore "go-micro.dev/v4/store"

	"github.com/oklog/run"
	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/channels"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/config"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/service"
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

			grpcClient, err := grpc.NewClient(
				append(
					grpc.GetClientOptions(&cfg.GRPCClientTLS),
					grpc.WithTraceProvider(traceProvider),
				)...,
			)
			if err != nil {
				return err
			}

			gr := run.Group{}

			ctx, cancel := context.WithCancel(c.Context)
			defer cancel()

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

			// evs defines a list of events to subscribe to
			evs := []events.Unmarshaller{
				events.ShareCreated{},
				events.ShareExpired{},
				events.SpaceShared{},
				events.SpaceUnshared{},
				events.SpaceMembershipExpired{},
				events.ScienceMeshInviteTokenGenerated{},
				events.SendEmailsEvent{},
			}
			registeredEvents := make(map[string]events.Unmarshaller)
			for _, e := range evs {
				typ := reflect.TypeOf(e)
				registeredEvents[typ.String()] = e
			}

			connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTYPE_BUS)
			client, err := stream.NatsFromConfig(connName, false, stream.NatsConfig(cfg.Notifications.Events))
			if err != nil {
				return err
			}
			evts, err := events.Consume(client, "notifications", evs...)
			if err != nil {
				return err
			}
			channel, err := channels.NewMailChannel(*cfg, logger)
			if err != nil {
				return err
			}
			tm, err := pool.StringToTLSMode(cfg.Notifications.GRPCClientTLS.Mode)
			if err != nil {
				return err
			}
			gatewaySelector, err := pool.GatewaySelector(
				cfg.Notifications.RevaGateway,
				pool.WithTLSCACert(cfg.Notifications.GRPCClientTLS.CACert),
				pool.WithTLSMode(tm),
				pool.WithRegistry(registry.GetRegistry()),
				pool.WithTracerProvider(traceProvider),
			)
			if err != nil {
				logger.Fatal().Err(err).Str("addr", cfg.Notifications.RevaGateway).Msg("could not get reva gateway selector")
			}
			valueService := settingssvc.NewValueService("eu.opencloud.api.settings", grpcClient)
			historyClient := ehsvc.NewEventHistoryService("eu.opencloud.api.eventhistory", grpcClient)

			notificationStore := store.Create(
				store.Store(cfg.Store.Store),
				store.TTL(cfg.Store.TTL),
				microstore.Nodes(cfg.Store.Nodes...),
				microstore.Database(cfg.Store.Database),
				microstore.Table(cfg.Store.Table),
				store.Authentication(cfg.Store.AuthUsername, cfg.Store.AuthPassword),
			)

			svc := service.NewEventsNotifier(evts, channel, logger, gatewaySelector, valueService,
				cfg.ServiceAccount.ServiceAccountID, cfg.ServiceAccount.ServiceAccountSecret,
				cfg.Notifications.EmailTemplatePath, cfg.Notifications.DefaultLanguage, cfg.WebUIURL,
				cfg.Notifications.TranslationPath, cfg.Notifications.SMTP.Sender, notificationStore, historyClient, registeredEvents)

			gr.Add(svc.Run, func(error) {
				cancel()
			})

			return gr.Run()
		},
	}
}
