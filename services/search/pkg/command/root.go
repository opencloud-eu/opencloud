package command

import (
	"os"

	"github.com/opencloud-eu/opencloud/pkg/clihelper"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"

	"github.com/spf13/cobra"
)

// GetCommands provides all commands for this service
func GetCommands(cfg *config.Config) []*cobra.Command {
	return []*cobra.Command{
		// start this service
		Server(cfg),

		// interaction with this service
		Index(cfg),
		Optimize(cfg),

		// infos about this service
		Health(cfg),
		Version(cfg),
	}
}

// Execute is the entry point for the opencloud-search command.
func Execute(cfg *config.Config) error {
	app := clihelper.DefaultApp(&cobra.Command{
		Use:   "search",
		Short: "Serve search API for OpenCloud",
	})
	app.AddCommand(GetCommands(cfg)...)
	app.SetArgs(os.Args[1:])

	return app.ExecuteContext(cfg.Context)
}
