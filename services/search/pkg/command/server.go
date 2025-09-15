package command

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os/signal"

	"github.com/opencloud-eu/reva/v2/pkg/events/raw"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	ogrpc "github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/search/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	bleveQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
	"github.com/opencloud-eu/opencloud/services/search/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/search/pkg/server/grpc"
	svcEvent "github.com/opencloud-eu/opencloud/services/search/pkg/service/event"
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

			cfg.GrpcClient, err = ogrpc.NewClient(
				append(ogrpc.GetClientOptions(cfg.GRPCClientTLS), ogrpc.WithTraceProvider(traceProvider))...,
			)
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

			// initialize search engine
			var eng search.Engine
			switch cfg.Engine.Type {
			case "bleve":
				idx, err := bleve.NewIndex(cfg.Engine.Bleve.Datapath)
				if err != nil {
					return err
				}

				defer func() {
					if err = idx.Close(); err != nil {
						logger.Error().Err(err).Msg("could not close bleve index")
					}
				}()

				eng = bleve.NewBackend(idx, bleveQuery.DefaultCreator, logger)
			case "open-search":
				client, err := opensearchgoAPI.NewClient(opensearchgoAPI.Config{
					Client: opensearchgo.Config{
						Addresses:             cfg.Engine.OpenSearch.Client.Addresses,
						Username:              cfg.Engine.OpenSearch.Client.Username,
						Password:              cfg.Engine.OpenSearch.Client.Password,
						Header:                cfg.Engine.OpenSearch.Client.Header,
						CACert:                cfg.Engine.OpenSearch.Client.CACert,
						RetryOnStatus:         cfg.Engine.OpenSearch.Client.RetryOnStatus,
						DisableRetry:          cfg.Engine.OpenSearch.Client.DisableRetry,
						EnableRetryOnTimeout:  cfg.Engine.OpenSearch.Client.EnableRetryOnTimeout,
						MaxRetries:            cfg.Engine.OpenSearch.Client.MaxRetries,
						CompressRequestBody:   cfg.Engine.OpenSearch.Client.CompressRequestBody,
						DiscoverNodesOnStart:  cfg.Engine.OpenSearch.Client.DiscoverNodesOnStart,
						DiscoverNodesInterval: cfg.Engine.OpenSearch.Client.DiscoverNodesInterval,
						EnableMetrics:         cfg.Engine.OpenSearch.Client.EnableMetrics,
						EnableDebugLogger:     cfg.Engine.OpenSearch.Client.EnableDebugLogger,
						Transport: &http.Transport{
							TLSClientConfig: &tls.Config{
								MinVersion:         tls.VersionTLS12,
								InsecureSkipVerify: cfg.Engine.OpenSearch.Client.Insecure,
							},
						},
					},
				})
				if err != nil {
					return fmt.Errorf("failed to create OpenSearch client: %w", err)
				}

				openSearchBackend, err := opensearch.NewBackend(cfg.Engine.OpenSearch.ResourceIndex.Name, client)
				if err != nil {
					return fmt.Errorf("failed to create OpenSearch backend: %w", err)
				}

				eng = openSearchBackend
			default:
				return fmt.Errorf("unknown search engine: %s", cfg.Engine.Type)
			}

			// initialize gateway selector
			selector, err := pool.GatewaySelector(cfg.Reva.Address, pool.WithRegistry(registry.GetRegistry()), pool.WithTracerProvider(traceProvider))
			if err != nil {
				logger.Fatal().Err(err).Msg("could not get reva gateway selector")
				return err
			}

			// initialize search content extractor
			var extractor content.Extractor
			switch cfg.Extractor.Type {
			case "basic":
				if extractor, err = content.NewBasicExtractor(logger); err != nil {
					return err
				}
			case "tika":
				if extractor, err = content.NewTikaExtractor(selector, logger, cfg); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown search extractor: %s", cfg.Extractor.Type)
			}

			ss := search.NewService(selector, eng, extractor, mtrcs, logger, cfg)

			// setup the servers
			gr := runner.NewGroup()

			if !cfg.GRPC.Disabled {
				grpcServer, err := grpc.Server(
					grpc.Config(cfg),
					grpc.Logger(logger),
					grpc.Name(cfg.Service.Name),
					grpc.Context(ctx),
					grpc.Metrics(mtrcs),
					grpc.JWTSecret(cfg.TokenManager.JWTSecret),
					grpc.TraceProvider(traceProvider),
					grpc.GatewaySelector(selector),
					grpc.Searcher(ss),
				)
				if err != nil {
					logger.Error().Err(err).Str("transport", "grpc").Msg("Failed to initialize server")
					return err
				}

				gr.Add(runner.NewGoMicroGrpcServerRunner(cfg.Service.Name+".grpc", grpcServer))
			} else {
				logger.Info().Msg("gRPC server disabled, not starting gRPC service")
			}

			if !cfg.Events.Disabled {
				connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
				bus, err := raw.FromConfig(context.Background(), connName, raw.Config{
					Endpoint:             cfg.Events.Endpoint,
					Cluster:              cfg.Events.Cluster,
					EnableTLS:            cfg.Events.EnableTLS,
					TLSInsecure:          cfg.Events.TLSInsecure,
					TLSRootCACertificate: cfg.Events.TLSRootCACertificate,
					AuthUsername:         cfg.Events.AuthUsername,
					AuthPassword:         cfg.Events.AuthPassword,
					MaxAckPending:        cfg.Events.MaxAckPending,
					AckWait:              cfg.Events.AckWait,
				})

				eventSvc, err := svcEvent.New(ctx, bus, logger, traceProvider, mtrcs, ss, cfg.Events.DebounceDuration, cfg.Events.NumConsumers, cfg.Events.AsyncUploads)
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

			// always start a debug server
			{
				debugServer, err := debug.Server(
					debug.Logger(logger),
					debug.Context(ctx),
					debug.Config(cfg),
				)
				if err != nil {
					logger.Error().Err(err).Str("transport", "debug").Msg("Failed to initialize server")
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
