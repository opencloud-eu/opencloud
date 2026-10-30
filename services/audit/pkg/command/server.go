package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/audit/pkg/config"
	"github.com/opencloud-eu/opencloud/services/audit/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/audit/pkg/server/debug"
	svc "github.com/opencloud-eu/opencloud/services/audit/pkg/service"
	"github.com/opencloud-eu/opencloud/services/audit/pkg/types"
	"github.com/opencloud-eu/opencloud/services/audit/pkg/writer"
	"github.com/opencloud-eu/reva/v2/pkg/audit"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/raw"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"

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
			var cancel context.CancelFunc
			if cfg.Context == nil {
				cfg.Context, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}
			ctx := cfg.Context
			logger := log.Configure(cfg.Service.Name, cfg.Commons, cfg.LogLevel)
			gr := runner.NewGroup()

			// we need an additional context for the audit server in order to
			// cancel it anytime
			svcCtx, svcCancel := context.WithCancel(ctx)
			defer svcCancel()

			connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
			if cfg.Legacy.Enabled {
				logger.Warn().Msg("the legacy audit log is deprecated, use the audit stream")
				if err := addLegacy(svcCtx, svcCancel, gr, cfg, connName, logger); err != nil {
					return err
				}
			} else {
				js, err := raw.JetStream(ctx, connName, raw.Config{
					Endpoint:             cfg.Events.Endpoint,
					Cluster:              cfg.Events.Cluster,
					TLSInsecure:          cfg.Events.TLSInsecure,
					TLSRootCACertificate: cfg.Events.TLSRootCACertificate,
					EnableTLS:            cfg.Events.EnableTLS,
					AuthUsername:         cfg.Events.AuthUsername,
					AuthPassword:         cfg.Events.AuthPassword,
				})
				if err != nil {
					return err
				}
				var sinks []writer.Sink
				if cfg.Output.Stdout {
					sinks = append(sinks, writer.NewLineSink("stdout", os.Stdout))
				}
				if cfg.Output.OTLP.Endpoint != "" {
					otlp, err := writer.NewOTLPSink(ctx, writer.OTLPConfig{
						Endpoint: cfg.Output.OTLP.Endpoint,
						Insecure: cfg.Output.OTLP.Insecure,
						Service:  cfg.Service.Name,
						Version:  version.GetString(),
					})
					if err != nil {
						return fmt.Errorf("could not initialize the OTLP output: %w", err)
					}
					defer func() { _ = otlp.Shutdown(context.Background()) }()
					sinks = append(sinks, otlp)
				}
				if len(sinks) == 0 {
					return errors.New("no audit output configured, enable AUDIT_OUTPUT_STDOUT or set AUDIT_OUTPUT_OTLP_ENDPOINT")
				}
				w := writer.New(logger, sinks...)
				streamCfg := audit.StreamConfig{Replicas: cfg.Stream.Replicas, MaxAge: cfg.Stream.MaxAge}

				gr.Add(runner.New(cfg.Service.Name+".writer", func() error {
					return w.Run(svcCtx, js, streamCfg)
				}, func() {
					svcCancel()
				}))
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

// addLegacy runs the legacy audit log on the domain events of the main queue.
func addLegacy(ctx context.Context, cancel context.CancelFunc, gr *runner.GroupRunner, cfg *config.Config, connName string, logger log.Logger) error {
	client, err := stream.NatsFromConfig(connName, false, stream.NatsConfig(cfg.Events))
	if err != nil {
		return err
	}
	evts, err := events.Consume(client, "audit", types.RegisteredEvents()...)
	if err != nil {
		return err
	}
	gr.Add(runner.New(cfg.Service.Name+".svc", func() error {
		svc.AuditLoggerFromConfig(ctx, cfg.Auditlog, evts, logger)
		return nil
	}, func() {
		cancel()
	}))
	return nil
}
