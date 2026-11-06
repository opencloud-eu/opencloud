package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/version"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	mreg "go-micro.dev/v4/registry"
)

const (
	_skipServiceListingFlagName = "skip-services"
)

// VersionCommand is the entrypoint for the version command.
func VersionCommand(cfg *config.Config) *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "print the version of this binary and all running service instances",
		GroupID: CommandGroupServer,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Version: " + version.GetString())
			fmt.Printf("Edition: %s\n", version.Edition)
			fmt.Printf("Compiled: %s\n", version.Compiled())

			skipServiceListing, _ := cmd.Flags().GetBool(_skipServiceListingFlagName)
			if skipServiceListing {
				return nil
			}

			fmt.Print("\n")

			configlog.Error(parser.ParseConfig(cfg, true))
			reg := registry.GetRegistry()
			serviceList, err := reg.ListServices()
			if err != nil {
				fmt.Printf("could not list services: %v\n", err)
				return err
			}

			var services []*mreg.Service
			for _, s := range serviceList {
				s, err := reg.GetService(s.Name)
				if err != nil {
					fmt.Printf("could not get service: %v\n", err)
					return err
				}
				services = append(services, s...)
			}

			if len(services) == 0 {
				fmt.Println("No running services found.")
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
	versionCmd.Flags().Bool(_skipServiceListingFlagName, false, "skip service listing")
	return versionCmd
}

func init() {
	register.AddCommand(VersionCommand)
}
