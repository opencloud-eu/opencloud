package command

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/clihelper"
	"github.com/opencloud-eu/opencloud/pkg/config"
	oclog "github.com/opencloud-eu/opencloud/pkg/log"
)

// Execute is the entry point for the opencloud command.
func Execute() error {
	cfg := config.DefaultConfig()

	app := clihelper.DefaultApp(&cobra.Command{
		Use:   "opencloud",
		Short: "opencloud",
	})

	for _, commandFactory := range register.Commands {
		command := commandFactory(cfg)

		if command.GroupID != "" && !app.ContainsGroup(command.GroupID) {
			app.AddGroup(&cobra.Group{
				ID:    command.GroupID,
				Title: command.GroupID,
			})
		}

		app.AddCommand(command)
	}
	app.SetArgs(os.Args[1:])
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	return app.ExecuteContext(ctx)
}

func logger(name string) zerolog.Logger {
	return oclog.NewLogger(
		oclog.Name(name),
		oclog.Level("info"),
		oclog.Pretty(true),
		oclog.Color(true)).Logger
}
