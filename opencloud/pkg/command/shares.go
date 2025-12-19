package command

import (
	"errors"

	"github.com/spf13/viper"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"
	oclog "github.com/opencloud-eu/opencloud/pkg/log"
	mregistry "github.com/opencloud-eu/opencloud/pkg/registry"
	sharing "github.com/opencloud-eu/opencloud/services/sharing/pkg/config"
	sharingparser "github.com/opencloud-eu/opencloud/services/sharing/pkg/config/parser"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/share/manager/jsoncs3"
	"github.com/opencloud-eu/reva/v2/pkg/share/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// SharesCommand is the entrypoint for the groups command.
func SharesCommand(cfg *config.Config) *cobra.Command {
	sharesCmd := &cobra.Command{
		Use:   "shares",
		Short: `cli tools to manage entries in the share manager.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Parse base config
			if err := parser.ParseConfig(cfg, true); err != nil {
				return configlog.ReturnError(err)
			}

			// Parse sharing config
			cfg.Sharing.Commons = cfg.Commons
			return configlog.ReturnError(sharingparser.ParseConfig(cfg.Sharing))
		},
	}
	sharesCmd.AddCommand(cleanupCmd(cfg))

	return sharesCmd
}

func init() {
	register.AddCommand(SharesCommand)
}

func cleanupCmd(cfg *config.Config) *cobra.Command {
	cleanCmd := &cobra.Command{
		Use:   "cleanup",
		Short: `clean up stale entries in the share manager.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Parse base config
			if err := parser.ParseConfig(cfg, true); err != nil {
				return configlog.ReturnError(err)
			}

			// Parse sharing config
			cfg.Sharing.Commons = cfg.Commons
			return configlog.ReturnError(sharingparser.ParseConfig(cfg.Sharing))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanup(cmd, cfg)
		},
	}
	cleanCmd.Flags().String("service-account-id", "", "Name of the service account to use for the cleanup")
	_ = cleanCmd.MarkFlagRequired("service-account-id")
	_ = viper.BindEnv("service-account-id", "OC_SERVICE_ACCOUNT_ID")
	_ = viper.BindPFlag("service-account-id", cleanCmd.Flags().Lookup("service-account-id"))

	cleanCmd.Flags().String("service-account-secret", "", "Secret for the service account")
	_ = cleanCmd.MarkFlagRequired("service-account-secret")
	_ = viper.BindEnv("service-account-secret", "OC_SERVICE_ACCOUNT_SECRET")
	_ = viper.BindPFlag("service-account-secret", cleanCmd.Flags().Lookup("service-account-secret"))

	return cleanCmd
}

func cleanup(_ *cobra.Command, cfg *config.Config) error {
	driver := cfg.Sharing.UserSharingDriver
	// cleanup is only implemented for the jsoncs3 share manager
	if driver != "jsoncs3" {
		return configlog.ReturnError(errors.New("cleanup is only implemented for the jsoncs3 share manager"))
	}

	rcfg := revaShareConfig(cfg.Sharing)
	f, ok := registry.NewFuncs[driver]
	if !ok {
		return configlog.ReturnError(errors.New("Unknown share manager type '" + driver + "'"))
	}
	mgr, err := f(rcfg[driver].(map[string]interface{}))
	if err != nil {
		return configlog.ReturnError(err)
	}

	// Initialize registry to make service lookup work
	_ = mregistry.GetRegistry()

	// get an authenticated context
	gatewaySelector, err := pool.GatewaySelector(cfg.Sharing.Reva.Address)
	if err != nil {
		return configlog.ReturnError(err)
	}

	client, err := gatewaySelector.Next()
	if err != nil {
		return configlog.ReturnError(err)
	}

	serviceAccountIDFlag := viper.GetString("service-account-id")
	serviceAccountSecretFlag := viper.GetString("service-account-secret")
	serviceUserCtx, err := utils.GetServiceUserContext(serviceAccountIDFlag, client, serviceAccountSecretFlag)
	if err != nil {
		return configlog.ReturnError(err)
	}

	l := logger()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	serviceUserCtx = l.WithContext(serviceUserCtx)

	mgr.(*jsoncs3.Manager).CleanupStaleShares(serviceUserCtx)

	return nil
}

func revaShareConfig(cfg *sharing.Config) map[string]interface{} {
	return map[string]interface{}{
		"json": map[string]interface{}{
			"file":         cfg.UserSharingDrivers.JSON.File,
			"gateway_addr": cfg.Reva.Address,
		},
		"sql": map[string]interface{}{ // cernbox sql
			"db_username":                   cfg.UserSharingDrivers.SQL.DBUsername,
			"db_password":                   cfg.UserSharingDrivers.SQL.DBPassword,
			"db_host":                       cfg.UserSharingDrivers.SQL.DBHost,
			"db_port":                       cfg.UserSharingDrivers.SQL.DBPort,
			"db_name":                       cfg.UserSharingDrivers.SQL.DBName,
			"password_hash_cost":            cfg.UserSharingDrivers.SQL.PasswordHashCost,
			"enable_expired_shares_cleanup": cfg.UserSharingDrivers.SQL.EnableExpiredSharesCleanup,
			"janitor_run_interval":          cfg.UserSharingDrivers.SQL.JanitorRunInterval,
		},
		"owncloudsql": map[string]interface{}{
			"gateway_addr":     cfg.Reva.Address,
			"storage_mount_id": cfg.UserSharingDrivers.OwnCloudSQL.UserStorageMountID,
			"db_username":      cfg.UserSharingDrivers.OwnCloudSQL.DBUsername,
			"db_password":      cfg.UserSharingDrivers.OwnCloudSQL.DBPassword,
			"db_host":          cfg.UserSharingDrivers.OwnCloudSQL.DBHost,
			"db_port":          cfg.UserSharingDrivers.OwnCloudSQL.DBPort,
			"db_name":          cfg.UserSharingDrivers.OwnCloudSQL.DBName,
		},
		"cs3": map[string]interface{}{
			"gateway_addr":        cfg.UserSharingDrivers.CS3.ProviderAddr,
			"provider_addr":       cfg.UserSharingDrivers.CS3.ProviderAddr,
			"service_user_id":     cfg.UserSharingDrivers.CS3.SystemUserID,
			"service_user_idp":    cfg.UserSharingDrivers.CS3.SystemUserIDP,
			"machine_auth_apikey": cfg.UserSharingDrivers.CS3.SystemUserAPIKey,
		},
		"jsoncs3": map[string]interface{}{
			"gateway_addr":        cfg.Reva.Address,
			"provider_addr":       cfg.UserSharingDrivers.JSONCS3.ProviderAddr,
			"service_user_id":     cfg.UserSharingDrivers.JSONCS3.SystemUserID,
			"service_user_idp":    cfg.UserSharingDrivers.JSONCS3.SystemUserIDP,
			"machine_auth_apikey": cfg.UserSharingDrivers.JSONCS3.SystemUserAPIKey,
		},
	}
}

func logger() *zerolog.Logger {
	log := oclog.NewLogger(
		oclog.Name("migrate"),
		oclog.Level("info"),
		oclog.Pretty(true),
		oclog.Color(true)).Logger
	return &log
}
