package command

import (
	"context"
	"crypto/tls"
	"fmt"
	"os/signal"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	pkgcrypto "github.com/opencloud-eu/opencloud/pkg/crypto"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/services/nats/pkg/config"
	"github.com/opencloud-eu/opencloud/services/nats/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/nats/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/nats/pkg/server/debug"
	"github.com/opencloud-eu/opencloud/services/nats/pkg/server/nats"

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

			var cancel context.CancelFunc
			if cfg.Context == nil {
				cfg.Context, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}
			ctx := cfg.Context

			gr := runner.NewGroup()
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

				gr.Add(runner.NewGolangHttpServerRunner(cfg.Service.Name+".debug", debugServer))
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
				logging.NewLogWrapper(logger),
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

			gr.Add(runner.New(cfg.Service.Name+".svc", func() error {
				return natsServer.ListenAndServe()
			}, func() {
				// INFO: this is a hotfix, we need to replace suture and wait for nats to
				//       gracefully shutdown using rungroups
				// 		 see https://github.com/opencloud-eu/opencloud/issues/2282
				logger.Info().Msg("Gracefully shutting down the NATS server...")
				natsServer.Shutdown()
				time.Sleep(5 * time.Second) // wait for the server to shutdown gracefully
				logger.Info().Msg("NATS server shutdown")
			}))

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
