package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/olekukonko/errors"
	"github.com/spf13/cobra"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	natspkg "github.com/opencloud-eu/opencloud/pkg/nats"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/runner"
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
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/activitylog"
	svcEvents "github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/events"
	svcHttp "github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/http"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
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
			defer cancel()

			mtrcs := metrics.New()
			mtrcs.BuildInfo.WithLabelValues(version.GetString()).Set(1)

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

			kv, err := ConnectNatsKV(cfg.Store)
			if err != nil {
				return err
			}
			activityLog, err := activitylog.New(kv,
				activitylog.Logger(logger),
				activitylog.MaxActivities(cfg.MaxActivities),
				activitylog.WriteBufferDuration(cfg.WriteBufferDuration),
			)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to initialize activity log")
				return err
			}

			if !cfg.HTTP.Disabled {

				hClient := ehsvc.NewEventHistoryService("eu.opencloud.api.eventhistory", grpcClient)

				svc, err := svcHttp.New(
					activityLog,
					svcHttp.Logger(logger),
					svcHttp.GatewaySelector(gatewaySelector),
					svcHttp.RegisteredEvents(_registeredEvents),
					//svcHttp.TraceProvider(tracerProvider),
					svcHttp.HistoryClient(hClient),
				)
				if err != nil {
					logger.Error().Err(err).Msg("handler init")
					return err
				}
				// TODO svc = service.NewInstrument(svc, metrics)
				// TODO svc = service.NewLogging(svc, logger) // this logs service specific data
				// TODO svc = service.NewTracing(svc, traceProvider)
				vClient := settingssvc.NewValueService("eu.opencloud.api.settings", grpcClient)

				server, err := http.Server(
					http.ValueClient(vClient),
					http.Logger(logger),
					http.Context(ctx),
					http.Config(cfg),
					http.Service(svc),
				)
				if err != nil {
					logger.Info().
						Err(err).
						Str("transport", "http").
						Msg("Failed to initialize server")

					return err
				}

				gr.Add(runner.NewGoMicroHttpServerRunner(cfg.Service.Name+".http", server))
			} else {
				logger.Info().Msg("HTTP server disabled, not starting HTTP service")
			}

			if !cfg.Events.Disabled {

				connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
				evStream, err := stream.NatsFromConfig(connName, false, stream.NatsConfig{
					Endpoint:             cfg.Events.Endpoint,
					Cluster:              cfg.Events.Cluster,
					EnableTLS:            cfg.Events.EnableTLS,
					TLSInsecure:          cfg.Events.TLSInsecure,
					TLSRootCACertificate: cfg.Events.TLSRootCACertificate,
					AuthUsername:         cfg.Events.AuthUsername,
					AuthPassword:         cfg.Events.AuthPassword,
				})
				if err != nil {
					logger.Error().Err(err).Msg("Failed to initialize event stream")
					return err
				}

				eventSvc, err := svcEvents.New(
					activityLog,
					evStream,
					svcEvents.Context(ctx),
					svcEvents.Logger(logger),
					svcEvents.ServiceAccount(cfg.ServiceAccount),
					svcEvents.GatewaySelector(gatewaySelector),
					svcEvents.RegisteredEvents(_registeredEvents),
					svcEvents.NumConsumers(cfg.NumConsumers),
				)
				if err != nil {
					logger.Error().Err(err).Str("transport", "event").Msg("Failed to initialize server")
					return err
				}

				gr.Add(runner.New(cfg.Service.Name+".svc", func() error {
					return eventSvc.Run()
				}, func() {
					eventSvc.Close()
				}))
			} else {
				logger.Info().Msg("event listening disabled, not starting event service")
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

func ConnectNatsKV(cfg config.Store) (nats.KeyValue, error) {
	// Connect to NATS servers
	secureOption := natspkg.Secure(cfg.EnableTLS, cfg.TLSInsecure, cfg.TLSRootCACertificate)
	conn, err := nats.Connect(strings.Join(cfg.Nodes, ","), secureOption, nats.UserInfo(cfg.AuthUsername, cfg.AuthPassword))
	if err != nil {
		return nil, err
	}

	js, err := conn.JetStream()
	if err != nil {
		return nil, err
	}

	kv, err := js.KeyValue(cfg.Database)
	if err != nil {
		if !errors.Is(err, nats.ErrBucketNotFound) {
			return nil, errors.Wrapf(err, "Failed to get bucket (%s)", cfg.Database)
		}

		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: cfg.Database,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create bucket (%s)", cfg.Database)
		}
	}

	return kv, nil
}
