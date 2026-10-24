package command

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"

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
			concurrencyFlag, _ := cmd.Flags().GetInt("concurrency")

			if spaceFlag == "" && !allSpacesFlag {
				return errors.New("either --space or --all-spaces is required")
			}
			if int(concurrencyFlag) > cfg.ReindexMaxConcurrency {
				return fmt.Errorf("concurrency %d exceeds max allowed %d", concurrencyFlag, cfg.ReindexMaxConcurrency)
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

			// Cancel the operation when the user presses Ctrl+C (SIGINT) or the
			// process receives SIGTERM. The cancellation propagates over the
			// gRPC stream so the server stops indexing.
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			stream, err := c.IndexSpace(ctx, &searchsvc.IndexSpaceRequest{
				SpaceId:      spaceFlag,
				ForceReindex: forceRescanFlag,
				Concurrency:  int32(concurrencyFlag),
			})
			if err != nil {
				return err
			}

			for {
				progress, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					// The user aborted (Ctrl+C / SIGTERM). Exit quietly instead
					// of dumping a "context canceled" gRPC error.
					if errors.Is(ctx.Err(), context.Canceled) {
						fmt.Println("aborted, indexing has been stopped")
						return nil
					}
					return err
				}

				printProgress(progress)
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
	indexCmd.Flags().Int(
		"concurrency",
		3,
		"the number of concurrent indexing operations.",
	)

	return indexCmd
}

// printProgress prints a single progress line, e.g.
//
//	[ 1/12 SKIPPED] <space id> is disabled, it will be indexed once it is enabled again
//	[ 2/12 SUCCESS] <space id> indexed in 40.9ms
//	[ 3/12 ERROR  ] <space id> failed: <error>
//
// The counter is padded to the width of the total and the status to the
// longest status so that the messages line up.
func printProgress(progress *searchsvc.IndexSpaceResponse) {
	var status, msg string
	switch {
	case progress.GetStatus() == searchsvc.IndexSpaceResponse_STATUS_SKIPPED:
		status = "SKIPPED"
		msg = progress.GetSpaceId() + " is disabled, it will be indexed once it is enabled again"
	// servers not setting a status only report failures via the error field
	case progress.GetStatus() == searchsvc.IndexSpaceResponse_STATUS_ERROR || progress.GetError() != "":
		status = "ERROR"
		msg = progress.GetSpaceId() + " failed: " + progress.GetError()
	default:
		status = "SUCCESS"
		msg = fmt.Sprintf("%s indexed in %s", progress.GetSpaceId(), progress.GetSpaceDuration().AsDuration())
	}

	width := len(strconv.FormatInt(progress.GetTotalSpaces(), 10))
	fmt.Printf("[%*d/%d %-7s] %s\n", width, progress.GetIndexedSpaces(), progress.GetTotalSpaces(), status, msg)
}
