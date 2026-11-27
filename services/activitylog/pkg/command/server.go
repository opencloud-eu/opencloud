package command

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/opencloud-eu/reva/v3/pkg/events/stream"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
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

			kv, js, err := ConnectNatsKV(cfg.Store, logger)
			if err != nil {
				return err
			}
			activityLog, err := activitylog.New(kv,
				activitylog.Logger(logger),
				activitylog.MaxActivities(cfg.MaxActivities),
				activitylog.JetStream(js),
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

// minTTL is the smallest positive max age the NATS server accepts for a stream.
const minTTL = 100 * time.Millisecond

func ConnectNatsKV(cfg config.Store, logger log.Logger) (nats.KeyValue, nats.JetStreamContext, error) {
	// NATS rejects negative and too small positive max ages, fall back to no expiry
	if cfg.TTL < 0 || (cfg.TTL > 0 && cfg.TTL < minTTL) {
		logger.Warn().
			Str("bucket", cfg.Database).
			Dur("ttl", cfg.TTL).
			Msg("invalid TTL configured, falling back to the default (no expiry)")
		cfg.TTL = 0
	}

	// Connect to NATS servers
	secureOption := natspkg.Secure(cfg.EnableTLS, cfg.TLSInsecure, cfg.TLSRootCACertificate)
	conn, err := nats.Connect(strings.Join(cfg.Nodes, ","), secureOption, nats.UserInfo(cfg.AuthUsername, cfg.AuthPassword))
	if err != nil {
		return nil, nil, err
	}

	js, err := conn.JetStream()
	if err != nil {
		return nil, nil, err
	}

	kv, err := js.KeyValue(cfg.Database)
	if err != nil {
		if !errors.Is(err, nats.ErrBucketNotFound) {
			return nil, nil, errors.Wrapf(err, "Failed to get bucket (%s)", cfg.Database)
		}

		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: cfg.Database,
			TTL:    cfg.TTL,
		})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "Failed to create bucket (%s)", cfg.Database)
		}
		return kv, js, nil
	}

	// the bucket already exists, make sure its TTL matches the configured one
	streamName := "KV_" + cfg.Database
	info, err := js.StreamInfo(streamName)
	if err != nil {
		logger.Error().
			Err(err).
			Str("bucket", cfg.Database).
			Str("stream", streamName).
			Dur("ttl", cfg.TTL).
			Msg("could not read stream info of the existing bucket, continuing with its current TTL")
		return kv, js, nil
	}

	// keep a copy of the version key in the stream metadata, applying a TTL can purge the key
	version := ""
	if info.Config.Metadata[activitylog.ActivityLogVersionMetadataKey] == "" {
		entry, err := kv.Get(activitylog.ActivityLogVersionKey)
		switch {
		case err == nil:
			version = string(entry.Value())
		case errors.Is(err, nats.ErrKeyNotFound):
			// no legacy version key, nothing to copy
		default:
			logger.Error().
				Err(err).
				Str("bucket", cfg.Database).
				Str("stream", streamName).
				Dur("ttl", cfg.TTL).
				Msg("could not read the version key of the existing bucket, continuing with its current settings")
			return kv, js, nil
		}
	}

	if info.Config.MaxAge != cfg.TTL || version != "" {
		streamCfg := info.Config
		streamCfg.MaxAge = cfg.TTL
		// the duplicate window must not exceed the max age
		if cfg.TTL > 0 && streamCfg.Duplicates > cfg.TTL {
			streamCfg.Duplicates = cfg.TTL
		}
		if version != "" {
			if streamCfg.Metadata == nil {
				streamCfg.Metadata = map[string]string{}
			}
			streamCfg.Metadata[activitylog.ActivityLogVersionMetadataKey] = version
		}
		if _, err := js.UpdateStream(&streamCfg); err != nil {
			logger.Error().
				Err(err).
				Str("bucket", cfg.Database).
				Str("stream", streamName).
				Dur("ttl", cfg.TTL).
				Msg("could not update the stream config of the existing bucket, continuing with its current settings")
		}
	}

	return kv, js, nil
}
