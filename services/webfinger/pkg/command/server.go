package command

import (
	"context"
	"fmt"
	"os/signal"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/config"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/relations"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/server/http"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/service/v0"
	"github.com/urfave/cli/v2"
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
			ctx := cfg.Context
			if ctx == nil {
				ctx, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}

			m := metrics.New(metrics.Logger(logger))
			m.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			gr := runner.NewGroup()
			{
				relationProviders, err := getRelationProviders(cfg)
				if err != nil {
					logger.Error().Err(err).Msg("relation provider init")
					return err
				}

				svc, err := service.New(
					service.Logger(logger),
					service.Config(cfg),
					service.WithRelationProviders(relationProviders),
				)
				if err != nil {
					logger.Error().Err(err).Msg("handler init")
					return err
				}
				svc = service.NewInstrument(svc, m)
				svc = service.NewLogging(svc, logger) // this logs service specific data
				svc = service.NewTracing(svc, traceProvider)

				server, err := http.Server(
					http.Logger(logger),
					http.Context(ctx),
					http.Config(cfg),
					http.Service(svc),
					http.TraceProvider(traceProvider),
				)

				if err != nil {
					logger.Info().
						Err(err).
						Str("server", "http").
						Msg("Failed to initialize server")

					return err
				}

				gr.Add(runner.NewGoMicroHttpServerRunner("webfinger_http", server))
			}

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

				gr.Add(runner.NewGolangHttpServerRunner("webfinger_debug", debugServer))
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

func getRelationProviders(cfg *config.Config) (map[string]service.RelationProvider, error) {
	rels := map[string]service.RelationProvider{}
	for _, relationURI := range cfg.Relations {
		switch relationURI {
		case relations.OpenIDConnectRel:
			rels[relationURI] = relations.OpenIDDiscovery(cfg.IDP)
		case relations.OpenCloudInstanceRel:
			var err error
			rels[relationURI], err = relations.OpenCloudInstance(cfg.Instances, cfg.OpenCloudURL)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown relation '%s'", relationURI)
		}
	}
	return rels, nil
}
