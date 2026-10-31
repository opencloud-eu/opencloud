package command

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/version"

	"github.com/opencloud-eu/opencloud/services/auth-api/pkg/config"
	"github.com/spf13/cobra"
)

// Version prints the service versions of all running instances.
func Version(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print the version of this binary and the running service instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Version: " + version.GetString())
			fmt.Printf("Compiled: %s\n", version.Compiled())
			fmt.Println("")

			return nil
		},
	}
}
