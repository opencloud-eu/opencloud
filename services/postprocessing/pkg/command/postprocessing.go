package command

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/config"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/config/parser"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/spf13/cobra"
)

// RestartPostprocessing cli command to restart postprocessing
func RestartPostprocessing(cfg *config.Config) *cobra.Command {
	restartPostprocessingCmd := &cobra.Command{
		Use:     "resume",
		Aliases: []string{"restart"},
		Short:   "resume postprocessing for an uploadID",

		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
			stream, err := stream.NatsFromConfig(connName, false, stream.NatsConfig{
				Endpoint:             cfg.Postprocessing.Events.Endpoint,
				Cluster:              cfg.Postprocessing.Events.Cluster,
				EnableTLS:            cfg.Postprocessing.Events.EnableTLS,
				TLSInsecure:          cfg.Postprocessing.Events.TLSInsecure,
				TLSRootCACertificate: cfg.Postprocessing.Events.TLSRootCACertificate,
				AuthUsername:         cfg.Postprocessing.Events.AuthUsername,
				AuthPassword:         cfg.Postprocessing.Events.AuthPassword,
			})
			if err != nil {
				return err
			}

			uid, _ := cmd.Flags().GetString("upload-id")
			step := ""
			if uid == "" {
				step, _ = cmd.Flags().GetString("step")
			}

			restart, _ := cmd.Flags().GetBool("restart")
			var ev events.Unmarshaller
			switch {
			case restart:
				ev = events.RestartPostprocessing{
					UploadID:  uid,
					Timestamp: utils.TSNow(),
				}
			default:
				ev = events.ResumePostprocessing{
					UploadID:  uid,
					Step:      events.Postprocessingstep(step),
					Timestamp: utils.TSNow(),
				}
			}

			if err := events.Publish(context.Background(), stream, ev); err != nil {
				return err
			}

			fmt.Println("Postprocessing event published successfully")
			return nil
		},
	}

	restartPostprocessingCmd.Flags().StringP(
		"upload-id",
		"u",
		"",
		"the uploadid to resume. Ignored if unset.",
	)
	restartPostprocessingCmd.Flags().StringP(
		"step",
		"s",
		"finished",
		"resume all uploads in the given postprocessing step. Ignored if upload-id is set.",
	)
	restartPostprocessingCmd.Flags().BoolP(
		"restart",
		"r",
		false,
		"restart postprocessing for the given uploadID. Ignores the step flag.",
	)

	return restartPostprocessingCmd
}
