package command

import (
	"fmt"
	"os"

	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
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

			reg := registry.GetRegistry()
			services, err := reg.GetService(cfg.HTTP.Namespace + "." + cfg.Service.Name)
			if err != nil {
				fmt.Println(fmt.Errorf("could not get %s services from the registry: %v", cfg.Service.Name, err))
				return err
			}

			if len(services) == 0 {
				fmt.Println("No running " + cfg.Service.Name + " service found.")
				return nil
			}

			table := tablewriter.NewTable(os.Stdout, tablewriter.WithHeaderAutoFormat(tw.Off))
			table.Header([]string{"Version", "Address", "Id"})
			for _, s := range services {
				for _, n := range s.Nodes {
					table.Append([]string{s.Version, n.Address, n.Id})
				}
			}
			table.Render()
			return nil
		},
	}
}
