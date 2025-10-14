package command

import (
	"context"
	"fmt"
	"os/signal"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/server/http"
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

			var cancel context.CancelFunc
			if cfg.Context == nil {
				cfg.Context, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}
			ctx := cfg.Context

			mtrcs := metrics.New()
			mtrcs.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			var kv jetstream.KeyValue
			// Allow to run without a NATS store (e.g. for the standalone Education provisioning service)
			if len(cfg.Store.Nodes) > 0 {
				//Connect to NATS servers
				natsOptions := nats.Options{
					Servers:  cfg.Store.Nodes,
					User:     cfg.Store.AuthUsername,
					Password: cfg.Store.AuthPassword,
				}
				conn, err := natsOptions.Connect()
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

			gr := runner.NewGroup()
			{
				server, err := http.Server(
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
