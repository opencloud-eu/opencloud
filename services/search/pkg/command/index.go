package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v4/client"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config/parser"
)

// Index is the entrypoint for the server command.
func Index(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "index",
		Usage:    "index the files for one one more users",
		Category: "index management",
		Aliases:  []string{"i"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "space",
				Aliases: []string{"s"},
				Usage:   "the id of the space to travers and index the files of. This or --all-spaces is required.",
			},
			&cli.BoolFlag{
				Name:  "all-spaces",
				Usage: "index all spaces instead. This or --space is required.",
			},
		},
		Before: func(_ *cli.Context) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		Action: func(ctx *cli.Context) error {
			if ctx.String("space") == "" && !ctx.Bool("all-spaces") {
				return errors.New("either --space or --all-spaces is required")
			}

			traceProvider, err := tracing.GetTraceProvider(ctx.Context, cfg.Commons.TracesExporter, cfg.Service.Name)
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
				SpaceId: ctx.String("space"),
			}, func(opts *client.CallOptions) { opts.RequestTimeout = 10 * time.Minute })
			if err != nil {
				fmt.Println("failed to index space: " + err.Error())
				return err
			}
			return nil
		},
	}
}
