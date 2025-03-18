package command

import (
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

// SendEmail triggers the sending of grouped email notifications for daily or weekly emails.
func SendEmail(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "send-email",
		Usage: "Send grouped email notifications with daily or weekly interval. Specify at least one of the flags '--daily' or '--weekly'.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "daily",
				Aliases: []string{"d"},
				Usage:   "Sends grouped daily email notifications.",
			},
			&cli.BoolFlag{
				Name:    "weekly",
				Aliases: []string{"w"},
				Usage:   "Sends grouped weekly email notifications.",
			},
		},
		Action: func(c *cli.Context) error {
			daily := c.Bool("daily")
			weekly := c.Bool("weekly")
			if !daily && !weekly {
				return errors.New("at least one of '--daily' or '--weekly' must be set")
			}
			connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTYPE_BUS)
			s, err := stream.NatsFromConfig(connName, false, stream.NatsConfig(cfg.Notifications.Events))
			if err != nil {
				return err
			}
			if daily {
				err = events.Publish(c.Context, s, events.SendEmailsEvent{
					Interval: "daily",
				})
				if err != nil {
					return err
				}
			}
			if weekly {
				err = events.Publish(c.Context, s, events.SendEmailsEvent{
					Interval: "weekly",
				})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
}
