package command

import (
	"context"
	"fmt"
	"os/signal"

	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/services/audit/pkg/config"
	"github.com/opencloud-eu/opencloud/services/audit/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/audit/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/audit/pkg/server/debug"
	svc "github.com/opencloud-eu/opencloud/services/audit/pkg/service"
	"github.com/opencloud-eu/opencloud/services/audit/pkg/types"
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
			var cancel context.CancelFunc
			if cfg.Context == nil {
				cfg.Context, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}
			ctx := cfg.Context

			logger := logging.Configure(cfg.Service.Name, cfg.Log)
			gr := runner.NewGroup()

			connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
			client, err := stream.NatsFromConfig(connName, false, stream.NatsConfig(cfg.Events))
			if err != nil {
				return err
			}
			evts, err := events.Consume(client, "audit", types.RegisteredEvents()...)
			if err != nil {
				return err
			}

			// we need an additional context for the audit server in order to
			// cancel it anytime
			svcCtx, svcCancel := context.WithCancel(ctx)
			defer svcCancel()

			gr.Add(runner.New("audit_svc", func() error {
				svc.AuditLoggerFromConfig(svcCtx, cfg.Auditlog, evts, logger)
				return nil
			}, func() {
				svcCancel()
			}))

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

				gr.Add(runner.NewGolangHttpServerRunner("audit_debug", debugServer))
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
