package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config/parser"

	"github.com/spf13/cobra"
	"go-micro.dev/v4/client"
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
			forceReindexFlag, _ := cmd.Flags().GetBool("force-reindex")
			if spaceFlag == "" && !allSpacesFlag {
				return errors.New("either --space or --all-spaces is required")
			}

			traceProvider, err := tracing.GetTraceProvider(cmd.Context(), cfg.Commons.TracesExporter, cfg.Service.Name)
			if err != nil {
				return err
			}
			grpcClient, err := grpc.NewClient(
				append(grpc.GetClientOptions(cfg.GRPCClientTLS),
					grpc.WithTraceProvider(traceProvider),
				)...,
			)
			if err != nil {
				return err
			}

			c := searchsvc.NewSearchProviderService("eu.opencloud.api.search", grpcClient)
			_, err = c.IndexSpace(context.Background(), &searchsvc.IndexSpaceRequest{
				SpaceId:      spaceFlag,
				ForceReindex: forceReindexFlag,
			}, func(opts *client.CallOptions) { opts.RequestTimeout = 10 * time.Minute })
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

	return indexCmd
}
