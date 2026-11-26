package command

import (
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/config"
	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/opencloud-eu/reva/v3/pkg/events/stream"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SendEmail triggers the sending of grouped email notifications for daily or weekly emails.
func SendEmail(cfg *config.Config) *cobra.Command {
	sendEmailCmd := &cobra.Command{
		Use:   "send-email",
		Short: "Send grouped email notifications with daily or weekly interval. Specify at least one of the flags '--daily' or '--weekly'.",
		RunE: func(cmd *cobra.Command, args []string) error {
			daily, _ := cmd.Flags().GetBool("daily")
			weekly, _ := cmd.Flags().GetBool("weekly")
			if !daily && !weekly {
				return errors.New("at least one of '--daily' or '--weekly' must be set")
			}
			connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
			s, err := stream.NatsFromConfig(connName, false, stream.NatsConfig(cfg.Notifications.Events))
			if err != nil {
				return err
			}
			if daily {
				err = events.Publish(cmd.Context(), s, events.SendEmailsEvent{
					Interval: "daily",
				})
				if err != nil {
					return err
				}
			}
			if weekly {
				err = events.Publish(cmd.Context(), s, events.SendEmailsEvent{
					Interval: "weekly",
				})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	sendEmailCmd.Flags().BoolP(
		"daily",
		"d",
		false,
		"Sends grouped daily email notifications.",
	)

	sendEmailCmd.Flags().BoolP(
		"weekly",
		"w",
		false,
		"Sends grouped weekly email notifications.",
	)
	return sendEmailCmd
}
