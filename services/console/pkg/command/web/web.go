package web

import (
	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/services/console/pkg/command/web/theme"
	"github.com/opencloud-eu/opencloud/services/console/pkg/config"
)

func Commands(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "web",
		Usage: "web related commands",
		Subcommands: []*cli.Command{
			theme.Commands(cfg),
		},
	}
}
