package command

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/urfave/cli/v2"
	mreg "go-micro.dev/v4/registry"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/version"
)

const (
	_skipServiceListingFlagName = "skip-services"
)

// VersionCommand is the entrypoint for the version command.
func VersionCommand(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "print the version of this binary and all running service instances",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  _skipServiceListingFlagName,
				Usage: "skip service listing",
			},
		},
		Category: "info",
		Action: func(c *cli.Context) error {
			fmt.Println("Version: " + version.GetString())
			fmt.Printf("Edition: %s\n", version.Edition)
			fmt.Printf("Compiled: %s\n", version.Compiled())

			if c.Bool(_skipServiceListingFlagName) {
				return nil
			}

			fmt.Print("\n")

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
}

func init() {
	register.AddCommand(VersionCommand)
}
