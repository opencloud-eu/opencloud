package command

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/opencloud/pkg/trash"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"

	"github.com/spf13/cobra"
)

func TrashCommand(cfg *config.Config) *cobra.Command {
	trashCmd := &cobra.Command{
		Use:   "trash",
		Short: "OpenCloud trash functionality",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnError(parser.ParseConfig(cfg, true))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Read the docs")
			return nil
		},
	}
	trashCmd.AddCommand(TrashPurgeEmptyDirsCommand(cfg))

	return trashCmd
}

func TrashPurgeEmptyDirsCommand(cfg *config.Config) *cobra.Command {
	trashPurgeCmd := &cobra.Command{
		Use:   "purge-empty-dirs",
		Short: "purge empty directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			basePath, _ := cmd.Flags().GetString("basepath")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			posix := cfg.StorageUsers.Driver == posixDriver
			if err := trash.PurgeTrashEmptyPaths(basePath, dryRun, posix); err != nil {
				fmt.Println(err)
				return err
			}

			return nil
		},
	}
	trashPurgeCmd.Flags().StringP("basepath", "p", "", "the basepath of the decomposedfs (e.g. /var/tmp/opencloud/storage/users)")
	_ = trashPurgeCmd.MarkFlagRequired("basepath")

	trashPurgeCmd.Flags().Bool("dry-run", true, "do not delete anything, just print what would be deleted")

	return trashPurgeCmd
}

func init() {
	register.AddCommand(TrashCommand)
}
