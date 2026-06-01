package command

import (
	"context"
	"fmt"
	"os/signal"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/services/auth-authelia/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-authelia/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/auth-authelia/pkg/render"

	embed "github.com/authelia/authelia/v4/experimental/embed"
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

			// Render the Authelia configuration. The main config file is regenerated from the current
			// OpenCloud configuration on every start (so it never goes stale), while the secrets file is
			// generated once and reused. This mirrors how the idp service generates its config in
			// NewService, and is why the service does not depend on 'opencloud init' to produce the file.
			configPaths, err := render.Config(logger, cfg)
			if err != nil {
				logger.Error().Err(err).Str("config", cfg.ConfigPath).Msg("failed to render authelia configuration")
				return err
			}

			// The embedded Authelia services watch this context for cancellation. When the OpenCloud
			// supervisor (or a process signal) cancels it, Authelia performs a graceful shutdown.
			runCtx, runCancel := context.WithCancel(ctx)
			defer runCancel()

			// Build the embedded Authelia context from the rendered configuration files (Authelia
			// deep-merges them). This validates the configuration and instantiates all Authelia
			// providers (LDAP backend, storage, OIDC, ...).
			actx, validator, err := embed.NewWithContext(runCtx, configPaths, cfg.FilterNames)
			if err != nil {
				if validator != nil {
					for _, verr := range validator.Errors() {
						logger.Error().Err(verr).Strs("config", configPaths).Msg("invalid authelia configuration")
					}
				}
				logger.Error().Err(err).Strs("config", configPaths).Msg("failed to initialize embedded authelia")
				return err
			}

			// Run Authelia's provider startup checks before serving. This is what migrates the storage
			// schema to the latest version (creating the authentication_logs, banned_ip, ... tables);
			// without it the SQLite database stays empty and every auth attempt fails with "no such
			// table". The notifier (SMTP) and NTP startup checks are disabled in the rendered config so
			// an unreachable mail server or missing outbound NTP does not block startup.
			if err := embed.ProvidersStartupCheck(actx, true); err != nil {
				logger.Error().Err(err).Str("config", cfg.ConfigPath).Msg("authelia provider startup checks failed")
				return err
			}

			logger.Info().Str("config", cfg.ConfigPath).Msg("starting embedded authelia")

			gr := runner.NewGroup()
			gr.Add(runner.New(cfg.Service.Name, func() error {
				// ServiceRunAll blocks until runCtx is cancelled (or Authelia receives a process signal),
				// running Authelia's own HTTP server, metrics server and watchers.
				return embed.ServiceRunAll(actx)
			}, func() {
				runCancel()
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
