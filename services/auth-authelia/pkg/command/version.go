package command

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/auth-authelia/pkg/config"

	"github.com/spf13/cobra"
)

// Version prints the version of this binary.
//
// The embedded Authelia service does not register with the go-micro service registry (it serves its
// own HTTP listener), so this command only prints the OpenCloud build version.
func Version(_ *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print the version of this binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Version: " + version.GetString())
			fmt.Printf("Compiled: %s\n", version.Compiled())
			return nil
		},
	}
}
