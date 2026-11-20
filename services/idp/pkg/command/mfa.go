package command

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/config"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/mfa"
)

// MFA is the entrypoint for the commands to administrate two-factor authentication.
func MFA(cfg *config.Config) *cobra.Command {
	mfaCmd := &cobra.Command{
		Use:   "mfa",
		Short: "administrate the two-factor authentication of users",
	}
	mfaCmd.AddCommand(mfaList(cfg), mfaDisable(cfg))
	return mfaCmd
}

func mfaService(cfg *config.Config) (*mfa.Service, error) {
	if !cfg.MFAAvailable() {
		return nil, errors.New("two-factor authentication is not available, it needs IDP_MFA_ENABLED=true, the identity manager 'ldap' or 'cs3' and an encryption secret file")
	}
	return mfa.NewFromConfig(cfg, log.Configure(cfg.Service.Name, cfg.Commons, cfg.LogLevel))
}

func mfaList(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list the users with two-factor authentication enabled",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnError(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := mfaService(cfg)
			if err != nil {
				return err
			}
			enrollments, err := svc.List()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "USERNAME\tUSER ID\tENABLED SINCE")
			for _, e := range enrollments {
				fmt.Fprintf(w, "%s\t%s\t%s\n", e.User.Username, e.User.ID, e.EnabledAt.Format(time.RFC3339))
			}
			return w.Flush()
		},
	}
}

func mfaDisable(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "disable <username or user id>",
		Short: "disable two-factor authentication for a user, e.g. when the authenticator app and the recovery codes are lost",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnError(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := mfaService(cfg)
			if err != nil {
				return err
			}
			user, err := svc.AdminDisable(args[0])
			if errors.Is(err, mfa.ErrUserNotFound) {
				return fmt.Errorf("no user '%s' with two-factor authentication found", args[0])
			}
			if err != nil {
				return err
			}
			fmt.Printf("Two-factor authentication disabled for user '%s'. The user can log in with the password only and set it up again.\n", user.Username)
			return nil
		},
	}
}
