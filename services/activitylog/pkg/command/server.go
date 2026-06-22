package command

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/spf13/cobra"

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
func Server(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := log.Configure(cfg.Service.Name, cfg.Commons, cfg.LogLevel)
			tracerProvider, err := tracing.GetTraceProvider(cmd.Context(), cfg.Commons.TracesExporter, cfg.Service.Name)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to initialize tracer")
				return err
			}

			gr := runner.NewGroup()
			ctx, cancel := context.WithCancel(cmd.Context())

			mtrcs := metrics.New()
			mtrcs.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			defer cancel()

			connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
			evStream, err := stream.NatsFromConfig(connName, false, stream.NatsConfig(cfg.Events))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to initialize event stream")
				return err
			}

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
					http.GatewaySelector(gatewaySelector),
					http.HistoryClient(hClient),
					http.ValueClient(vClient),
					http.RegisteredEvents(_registeredEvents),
				)

				if err != nil {
					logger.Error().Err(err).Str("transport", "http").Msg("Failed to initialize server")
					return err
				}

				gr.Add(runner.NewGoMicroHttpServerRunner(cfg.Service.Name+".http", svc))
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
