// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/revaconfig"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/server/http"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
	svcEvents "github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/events"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/jwt"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/token"
	"github.com/opencloud-eu/reva/v2/cmd/revad/runtime"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

var _registeredEvents = []events.Unmarshaller{
	events.ShareCreated{},
	events.ShareRemoved{},
	events.ShareExpired{},
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
				return err
			}

			tm, err := pool.StringToTLSMode(cfg.GRPCClientTLS.Mode)
			if err != nil {
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
				return fmt.Errorf("could not get reva client selector: %s", err)
			}

			gr := runner.NewGroup()
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			mtrcs := metrics.New()
			mtrcs.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			tokenSvc := token.NewTokenService()
			store := storage.NewFileManager(cfg.Storage.RootDirectory)
			jwtService := jwt.NewJwtService(cfg.SessionSecret(), cfg.JWT.TTL)

			var evStream events.Stream
			if !cfg.Events.Disabled {
				connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
				evStream, err = stream.NatsFromConfig(connName, false, stream.NatsConfig{
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
			}

			authGuest := authguest.NewAuthGuestService(tokenSvc, store,
				authguest.GatewaySelector(gatewaySelector),
				authguest.ServiceAccount(cfg.ServiceAccount),
				authguest.JWT(jwtService),
				authguest.EventsPublisher(evStream),
			)

			if !cfg.HTTP.Disabled {
				server, err := http.Server(
					http.Logger(logger),
					http.Context(ctx),
					http.Config(cfg),
					http.Service(authGuest),
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
				consumer, err := svcEvents.NewEventConsumer(
					evStream,
					svcEvents.Logger(logger),
					svcEvents.Context(ctx),
					svcEvents.RegisteredEvents(_registeredEvents),
					svcEvents.NumConsumers(cfg.NumConsumers),
					svcEvents.AuthGuestService(authGuest),
				)
				if err != nil {
					logger.Error().Err(err).Str("transport", "event").Msg("Failed to initialize server")
					return err
				}

				gr.Add(runner.New(cfg.Service.Name+".svc", func() error {
					return consumer.Run()
				}, func() {
					consumer.Close()
				}))
			} else {
				logger.Info().Msg("event listening disabled, not starting event service")
			}
			{
				//FIXME: Does this need to be optional? Similar to cfg.HTTP.Disabled?
				// run the appropriate reva servers based on the config
				rCfg := revaconfig.GuestLinksConfigFromStruct(cfg)
				if rServer := runtime.NewDrivenGRPCServerWithOptions(rCfg,
					runtime.WithLogger(&logger.Logger),
					runtime.WithRegistry(registry.GetRegistry()),
					runtime.WithTraceProvider(tracerProvider),
				); rServer != nil {
					gr.Add(runner.NewRevaServiceRunner(cfg.Service.Name+".rgrpc", rServer))
				}
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
			grpcSvc := registry.BuildGRPCService(cfg.GRPC.Namespace+"."+cfg.Service.Name, cfg.GRPC.Protocol, cfg.GRPC.Addr, version.GetString())
			if err := registry.RegisterService(ctx, logger, grpcSvc, cfg.Debug.Addr); err != nil {
				logger.Fatal().Err(err).Msg("failed to register the grpc service")
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
