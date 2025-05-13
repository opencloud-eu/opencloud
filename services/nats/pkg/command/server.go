package command

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/oklog/run"

	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	pkgcrypto "github.com/opencloud-eu/opencloud/pkg/crypto"
	"github.com/opencloud-eu/opencloud/services/nats/pkg/config"
	"github.com/opencloud-eu/opencloud/services/nats/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/nats/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/nats/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/nats/pkg/server/nats"
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

			var tlsConf *tls.Config
			if cfg.Nats.EnableTLS {
				// Generate a self-signing cert if no certificate is present
				if err := pkgcrypto.GenCert(cfg.Nats.TLSCert, cfg.Nats.TLSKey, logger); err != nil {
					logger.Fatal().Err(err).Msgf("Could not generate test-certificate")
				}

				crt, err := tls.LoadX509KeyPair(cfg.Nats.TLSCert, cfg.Nats.TLSKey)
				if err != nil {
					return err
				}

				clientAuth := tls.RequireAndVerifyClientCert
				if cfg.Nats.TLSSkipVerifyClientCert {
					clientAuth = tls.NoClientCert
				}

				tlsConf = &tls.Config{
					MinVersion:   tls.VersionTLS12,
					ClientAuth:   clientAuth,
					Certificates: []tls.Certificate{crt},
				}
			}
			natsServer, err := nats.NewNATSServer(
				ctx,
				logger,
				nats.Host(cfg.Nats.Host),
				nats.Port(cfg.Nats.Port),
				nats.ClusterID(cfg.Nats.ClusterID),
				nats.StoreDir(cfg.Nats.StoreDir),
				nats.TLSConfig(tlsConf),
				nats.AllowNonTLS(!cfg.Nats.EnableTLS),
			)
			if err != nil {
				return err
			}

			gr.Add(func() error {
				err := make(chan error, 1)
				select {
				case <-ctx.Done():
					return nil
				case err <- natsServer.ListenAndServe():
					return <-err
				}

			}, func(err error) {
				if err == nil {
					logger.Info().
						Str("transport", "nats").
						Str("server", cfg.Service.Name).
						Msg("letting other services deregister")

					time.Sleep(3 * time.Second)

					logger.Info().
						Str("transport", "nats").
						Str("server", cfg.Service.Name).
						Msg("Shutting down server")
				} else {
					logger.Error().Err(err).
						Str("transport", "nats").
						Str("server", cfg.Service.Name).
						Msg("Shutting down server")
				}

				natsServer.Shutdown()
				cancel()
			})

			return gr.Run()
		},
	}
}
