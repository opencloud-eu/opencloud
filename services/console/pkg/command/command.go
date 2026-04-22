package command

import (
	"github.com/spf13/cobra"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/services/console/pkg/config"
	"github.com/opencloud-eu/opencloud/services/console/pkg/config/parser"
)

func preRunE(cfg *config.Config) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return configlog.ReturnError(parser.ParseConfig(cfg))
	}
}
