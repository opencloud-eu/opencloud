package command

import (
	"os"

	"github.com/opencloud-eu/opencloud/pkg/clihelper"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"

	"github.com/spf13/cobra"
)

// GetCommands provides all commands for this service
func GetCommands(cfg *config.Config) []*cobra.Command {
	return []*cobra.Command{
		// start this service
		Server(cfg),

		// interaction with this service
		Uploads(cfg),
		TrashBin(cfg),
		Blobstore(cfg),
		Revisions(cfg),

		// infos about this service
		Health(cfg),
		Version(cfg),
	}
}

// Execute is the entry point for the opencloud-storage-users command.
func Execute(cfg *config.Config) error {
	app := clihelper.DefaultApp(&cobra.Command{
		Use:   "storage-users",
		Short: "Provide storage for users and projects in OpenCloud",
	})
	app.AddCommand(GetCommands(cfg)...)
	app.SetArgs(os.Args[1:])

	return app.ExecuteContext(cfg.Context)
}
