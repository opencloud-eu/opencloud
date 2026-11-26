package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	merrors "go-micro.dev/v4/errors"
	"google.golang.org/grpc/status"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config/parser"
)

// Optimize merges the segments of the search index of the running service.
func Optimize(cfg *config.Config) *cobra.Command {
	optimizeCmd := &cobra.Command{
		Use:   "optimize",
		Short: "merge the segments of the search index into one, without indexing anything again",
		PreRunE: func(*cobra.Command, []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, conn, err := dialSearchProvider(cmd, cfg)
			if err != nil {
				return err
			}
			defer conn.Close()

			// the cancellation reaches the server, which stops the merge
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			fmt.Println("optimizing the search index...")
			res, err := c.OptimizeIndex(ctx, &searchsvc.OptimizeIndexRequest{})
			switch {
			// a merge that finished before the signal counts as done
			case err != nil && errors.Is(ctx.Err(), context.Canceled):
				fmt.Println("aborted, the optimization has been stopped")
				return nil
			case err != nil:
				// the service answers with a go-micro error, its json sits in the status message
				return errors.New(merrors.Parse(status.Convert(err).Message()).GetDetail())
			}
			fmt.Printf("optimized the search index in %s\n", res.GetDuration().AsDuration())
			return nil
		},
	}
	addConnFlags(optimizeCmd)

	return optimizeCmd
}
