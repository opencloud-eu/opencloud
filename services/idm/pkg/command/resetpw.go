package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/idm/pkg/config"
	"github.com/opencloud-eu/opencloud/services/idm/pkg/config/parser"

	"github.com/go-ldap/ldap/v3"
	"github.com/libregraph/idm/pkg/ldbbolt"
	"github.com/libregraph/idm/server"
	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/term"
)

// ResetPassword is the entrypoint for the resetpassword command
func ResetPassword(cfg *config.Config) *cobra.Command {
	resetPasswordCmd := &cobra.Command{
		Use:   "resetpassword",
		Short: "Reset user password",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := log.Configure(cfg.Service.Name, cfg.Commons, cfg.LogLevel)
			ctx, cancel := context.WithCancel(cmd.Context())

			defer cancel()
			userNameFlag, _ := cmd.Flags().GetString("user-name")
			return resetPassword(ctx, logger, cfg, userNameFlag)
		},
	}
	resetPasswordCmd.Flags().StringP(
		"user-name",
		"u",
		"admin",
		"User name",
	)

	return resetPasswordCmd
}

func resetPassword(_ context.Context, logger log.Logger, cfg *config.Config, userName string) error {
	servercfg := server.Config{
		Logger:      log.LogrusWrap(logger.Logger),
		LDAPHandler: "boltdb",
		LDAPBaseDN:  "o=libregraph-idm",

		BoltDBFile: cfg.IDM.DatabasePath,
	}

	userDN := fmt.Sprintf("uid=%s,ou=users,%s", userName, servercfg.LDAPBaseDN)
	fmt.Printf("Resetting password for user '%s'.\n", userDN)
	if _, err := os.Stat(servercfg.BoltDBFile); errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "IDM database does not exist.\n")
		return err
	}

	newPw, err := getPassword()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
		return err
	}

	bdb := &ldbbolt.LdbBolt{}

	opts := bolt.Options{
		Timeout: 1 * time.Millisecond,
	}
	if err := bdb.Configure(servercfg.Logger, servercfg.LDAPBaseDN, servercfg.BoltDBFile, &opts); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: '%s'. Please stop any running OpenCloud idm instance, as this tool requires exclusive access to the database.\n", err)
		return err
	}
	defer bdb.Close()

	if err := bdb.Initialize(); err != nil {
		return err
	}

	pwRequest := ldap.NewPasswordModifyRequest(userDN, "", newPw)
	if err := bdb.UpdatePassword(pwRequest); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update user password: %v\n", err)
	}
	fmt.Printf("Password for user '%s' updated.\n", userDN)
	return nil
}

func getPassword() (string, error) {
	fmt.Print("Enter new password: ")
	bytePassword, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}
	fmt.Println("")
	fmt.Print("Re-enter new password: ")
	bytePasswordVerify, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}
	fmt.Println("")

	password := string(bytePassword)
	passwordVerify := string(bytePasswordVerify)

	if password != passwordVerify {
		return "", errors.New("Passwords do not match")
	}
	return password, nil
}
