package command

import (
	"context"
	"fmt"
	"os/signal"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	natspkg "github.com/opencloud-eu/opencloud/pkg/nats"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/server/http"
	evc "github.com/opencloud-eu/opencloud/services/graph/pkg/service/events"
	svc "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

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

			prom := prometheus.DefaultRegisterer

			// note that the function we pass here is tasked with decomposing Graph HTTP API
			// request URL patterns into information that is then used for labels in metrics
			// to track HTTP request processing durations, and it is located there to be close
			// to the HTTP API route definitions, to improve chances of adapting it accordingly
			// whenever those routes should change in the future
			mtrcs, err := metrics.New(prom, &logger, svc.DecomposeGraphApiRequestPattern)
			if err != nil {
				return err
			}

			var kv jetstream.KeyValue
			// Allow to run without a NATS store (e.g. for the standalone Education provisioning service)
			if len(cfg.Store.Nodes) > 0 {
				// Connect to NATS servers
				secureOption := natspkg.Secure(cfg.Store.EnableTLS, cfg.Store.TLSInsecure, cfg.Store.TLSRootCACertificate)
				conn, err := nats.Connect(strings.Join(cfg.Store.Nodes, ","), secureOption, nats.UserInfo(cfg.Store.AuthUsername, cfg.Store.AuthPassword))
				if err != nil {
					return err
				}

				js, err := jetstream.New(conn)
				if err != nil {
					return err
				}
				kv, err = js.KeyValue(ctx, cfg.Store.Database)
				if err != nil {
					if !errors.Is(err, jetstream.ErrBucketNotFound) {
						return fmt.Errorf("failed to get bucket (%s): %w", cfg.Store.Database, err)
					}

					kv, err = js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
						Bucket: cfg.Store.Database,
					})
					if err != nil {
						return fmt.Errorf("failed to create bucket (%s): %w", cfg.Store.Database, err)
					}
				}
			}

			identityBackendName := cfg.Identity.Backend // contains the name of the backend implementation to use

			// since the identity backend in use is of prime importance to understand issues through logs, every
			// log entry should contain a 'backend' entry with the name of the backend in use from here on:
			logger = log.Logger{Logger: logger.With().Str("backend", identityBackendName).Logger()}

			identityBackend, eduBackend, err := identity.CreateIdentityBackends(
				identityBackendName,
				cfg,
				&logger,
				prom,
				traceProvider,
			)

			if err != nil {
				logger.Error().Err(err).Msg("Error initializing the identity backend")
				return fmt.Errorf("could not initialize identity backend: %w", err)
			}

			var eventsStream events.Stream
			if cfg.Events.Endpoint != "" {
				var err error
				connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
				eventsStream, err = stream.NatsFromConfig(connName, false, cfg.Events.ToNatsConfig())
				if err != nil {
					logger.Error().Err(err).Msg("Error initializing events publisher")
					return fmt.Errorf("could not initialize events publisher: %w", err)
				}
			}

			gr := runner.NewGroup()

			if !cfg.HTTP.Disabled {
				mtrcs.HttpEnabled.Set(1)

				server, err := http.Server(
					identityBackend,
					eduBackend,
					eventsStream,
					http.Logger(logger),
					http.Context(ctx),
					http.Config(cfg),
					http.Metrics(mtrcs),
					http.TraceProvider(traceProvider),
					http.NatsKeyValue(kv),
				)
				if err != nil {
					logger.Error().Err(err).Str("transport", "http").Msg("Failed to initialize server")
					return err
				}
				gr.Add(runner.NewGoMicroHttpServerRunner(cfg.Service.Name+".http", server))
			} else {
				mtrcs.HttpEnabled.Set(0)
				logger.Info().Str("transport", "http").Msg("HTTP server is disabled")
			}

			if !cfg.Events.DisabledConsumer {
				mtrcs.EventsEnabled.Set(1)

				// even if events are enabled, we still need to differentiate between whether this process
				// show be consuming events or not (and even when that is disabled, we still need to be
				// able to produce events), which is why this is a separate setting;
				// for context, see https://github.com/opencloud-eu/opencloud/issues/1312

				logger := &log.Logger{Logger: logger.With().Str("transport", "events").Logger()}
				eventConsumer, err := evc.NewService(cfg.Context, eventsStream, identityBackend, mtrcs, logger)
				if err != nil {
					return fmt.Errorf("could not initialize events consumer: %w", err)
				}

				gr.Add(runner.New(cfg.Service.Name+".svc", func() error {
					return eventConsumer.Start()
				}, func() {
					err := eventConsumer.Close()
					if err != nil {
						logger.Error().Err(err).Msg("failed to stop event consumer")
					}
				}))
			} else {
				mtrcs.EventsEnabled.Set(0)
				logger.Info().Str("transport", "events").Msg("event consumer is disabled")
			}

			{
				server, err := debug.Server(
					debug.Logger(logger),
					debug.Context(ctx),
					debug.Config(cfg),
				)
				if err != nil {
					logger.Info().Err(err).Str("transport", "debug").Msg("Failed to initialize server")
					return err
				}

				gr.Add(runner.NewGolangHttpServerRunner(cfg.Service.Name+".debug", server))
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
