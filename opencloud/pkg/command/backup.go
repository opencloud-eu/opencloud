package command

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/backup"
	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"
	decomposedbs "github.com/opencloud-eu/reva/v2/pkg/storage/fs/decomposed/blobstore"
	decomposeds3bs "github.com/opencloud-eu/reva/v2/pkg/storage/fs/decomposeds3/blobstore"
)

// BackupCommand is the entrypoint for the backup command
func BackupCommand(cfg *config.Config) *cobra.Command {
	bckCmd := &cobra.Command{
		Use:   "backup",
		Short: "OpenCloud backup functionality",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnError(parser.ParseConfig(cfg, true))
		},
	}
	bckCmd.AddCommand(ConsistencyCommand(cfg))
	return bckCmd
}

// ConsistencyCommand is the entrypoint for the consistency Command
func ConsistencyCommand(cfg *config.Config) *cobra.Command {
	consCmd := &cobra.Command{
		Use:   "consistency",
		Short: "check backup consistency",
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				bs  backup.ListBlobstore
				err error
			)
			basePath, _ := cmd.Flags().GetString("basepath")
			blobstoreFlag, _ := cmd.Flags().GetString("blobstore")
			switch blobstoreFlag {
			case "decomposeds3":
				bs, err = decomposeds3bs.New(
					cfg.StorageUsers.Drivers.DecomposedS3.Endpoint,
					cfg.StorageUsers.Drivers.DecomposedS3.Region,
					cfg.StorageUsers.Drivers.DecomposedS3.Bucket,
					cfg.StorageUsers.Drivers.DecomposedS3.AccessKey,
					cfg.StorageUsers.Drivers.DecomposedS3.SecretKey,
					cfg.StorageUsers.Drivers.DecomposedS3.SSECKey,
					decomposeds3bs.Options{},
				)
			case "decomposed":
				bs, err = decomposedbs.New(basePath)
			case "none":
				bs = nil
			default:
				err = errors.New("blobstore type not supported")
			}
			if err != nil {
				fmt.Println(err)
				return err
			}
			fail, _ := cmd.Flags().GetBool("fail")
			if err := backup.CheckProviderConsistency(basePath, bs, fail); err != nil {
				fmt.Println(err)
				return err
			}

			return nil
		},
	}
	consCmd.Flags().StringP("basepath", "p", "", "the basepath of the decomposedfs (e.g. /var/tmp/opencloud/storage/users)")
	_ = consCmd.MarkFlagRequired("basepath")
	consCmd.Flags().StringP("blobstore", "b", "decomposed", "the blobstore type. Can be (none, decomposed, decomposeds3). Default decomposed")
	consCmd.Flags().Bool("fail", false, "exit with non-zero status if consistency check fails")
	return consCmd
}

func init() {
	register.AddCommand(BackupCommand)
}
