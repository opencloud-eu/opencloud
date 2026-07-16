package command

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config/parser"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Index is the entrypoint for the server command.
func Index(cfg *config.Config) *cobra.Command {
	indexCmd := &cobra.Command{
		Use:     "index",
		Short:   "index the files for one one more users",
		Aliases: []string{"i"},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			allSpacesFlag, _ := cmd.Flags().GetBool("all-spaces")
			spaceFlag, _ := cmd.Flags().GetString("space")
			forceRescanFlag, _ := cmd.Flags().GetBool("force-rescan")
			endpointFlag, _ := cmd.Flags().GetString("endpoint")
			insecureFlag, _ := cmd.Flags().GetBool("insecure")
			if spaceFlag == "" && !allSpacesFlag {
				return errors.New("either --space or --all-spaces is required")
			}

			var dialOpts []grpc.DialOption
			if cfg.GRPCClientTLS.Mode == "insecure" || insecureFlag {
				dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
			} else {
				dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
					MinVersion: tls.VersionTLS12,
				})))
			}

			conn, err := grpc.NewClient(endpointFlag, dialOpts...)
			if err != nil {
				return fmt.Errorf("failed to dial %s: %w", endpointFlag, err)
			}
			defer conn.Close()

			c := searchsvc.NewSearchProviderClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			_, err = c.IndexSpace(ctx, &searchsvc.IndexSpaceRequest{
				SpaceId:      spaceFlag,
				ForceReindex: forceRescanFlag,
			})
			if err != nil {
				fmt.Println("failed to index space: " + err.Error())
				return err
			}
			return nil
		},
	}
	indexCmd.Flags().StringP(
		"space",
		"s",
		"",
		"the id of the space to travers and index the files of. This or --all-spaces is required.")

	indexCmd.Flags().Bool(
		"all-spaces",
		false,
		"index all spaces instead. This or --space is required.",
	)
	indexCmd.Flags().Bool(
		"force-rescan",
		false,
		"force a rescan of all files, even if they are already indexed. This will make the indexing process much slower, but ensures that the index is up-to-date using the current search service configuration.",
	)
	indexCmd.Flags().String(
		"endpoint",
		"127.0.0.1:9220",
		"the address of the search service gRPC endpoint.",
	)
	indexCmd.Flags().Bool(
		"insecure",
		false,
		"disable TLS for the gRPC connection.",
	)

	indexCmd.AddCommand(Migrate(cfg), Prune(cfg))

	return indexCmd
}
