package command

import (
	"os"

	"github.com/opencloud-eu/opencloud/pkg/clihelper"

	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/services/console/pkg/command/web"
	"github.com/opencloud-eu/opencloud/services/console/pkg/config"
)

// GetCommands provides all commands for this service
func GetCommands(cfg *config.Config) cli.Commands {
	return []*cli.Command{
		Server(cfg),
		Health(cfg),
		Version(cfg),
		web.Commands(cfg),
	}
}

// Execute is the entry point for the opencloud-console command.
func Execute(cfg *config.Config) error {
	app := clihelper.DefaultApp(&cli.App{
		Name:     "console",
		Usage:    "Serve console API for OpenCloud",
		Commands: GetCommands(cfg),
	})
	return app.RunContext(cfg.Context, os.Args)
}
