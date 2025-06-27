package theme

import (
	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/services/console/pkg/config"
)

func Commands(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "theme",
		Usage: "web-theme related commands",
		Subcommands: []*cli.Command{
			Pull(cfg),
		},
	}
}
