package command

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	ocinit "github.com/opencloud-eu/opencloud/opencloud/pkg/init"
	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/defaults"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// InitCommand is the entrypoint for the init command
func InitCommand(_ *config.Config) *cobra.Command {
	initCmd := &cobra.Command{
		Use:     "init",
		Short:   "initialise an OpenCloud config",
		GroupID: CommandGroupServer,
		RunE: func(cmd *cobra.Command, args []string) error {
			insecureFlag := viper.GetString("insecure")
			insecure := false
			if insecureFlag == "ask" {
				answer := strings.ToLower(stringPrompt("Do you want to configure OpenCloud with certificate checking disabled?\n This is not recommended for public instances! [yes | no = default]"))
				if answer == "yes" || answer == "y" {
					insecure = true
				}
			} else if insecureFlag == strings.ToLower("true") || insecureFlag == strings.ToLower("yes") || insecureFlag == strings.ToLower("y") {
				insecure = true
			}
			forceOverwriteFlag := viper.GetBool("force-overwrite")
			diffFlag, _ := cmd.Flags().GetBool("diff")
			configPathFlag := viper.GetString("config-path")
			adminPasswordFlag := viper.GetString("admin-password")
			err := ocinit.CreateConfig(insecure, forceOverwriteFlag, diffFlag, configPathFlag, adminPasswordFlag)
			if err != nil {
				log.Fatalf("Could not create config: %s", err)
			}
			return nil
		},
	}
	initCmd.Flags().String("insecure", "ask", "Allow insecure OpenCloud config")
	_ = viper.BindEnv("insecure", "OC_INSECURE")
	_ = viper.BindPFlag("insecure", initCmd.Flags().Lookup("insecure"))

	initCmd.Flags().BoolP("diff", "d", false, "Show the difference between the current config and the new one")

	initCmd.Flags().BoolP("force-overwrite", "f", false, "Force overwrite existing config file")
	_ = viper.BindEnv("force-overwrite", "OC_FORCE_CONFIG_OVERWRITE")
	_ = viper.BindPFlag("force-overwrite", initCmd.Flags().Lookup("force-overwrite"))

	initCmd.Flags().String("config-path", defaults.BaseConfigPath(), "Config path for the OpenCloud runtime")
	_ = viper.BindEnv("config-path", "OC_CONFIG_DIR")
	_ = viper.BindEnv("config-path", "OC_BASE_DATA_PATH")
	_ = viper.BindPFlag("config-path", initCmd.Flags().Lookup("config-path"))

	initCmd.Flags().String("admin-password", "", "Set admin password instead of using a random generated one")
	_ = viper.BindEnv("admin-password", "ADMIN_PASSWORD")
	_ = viper.BindEnv("admin-password", "IDM_ADMIN_PASSWORD")
	_ = viper.BindPFlag("admin-password", initCmd.Flags().Lookup("admin-password"))
	return initCmd
}

func init() {
	register.AddCommand(InitCommand)
}

func stringPrompt(label string) string {
	input := ""
	reader := bufio.NewReader(os.Stdin)
	for {
		_, _ = fmt.Fprint(os.Stderr, label+" ")
		input, _ = reader.ReadString('\n')
		if input != "" {
			break
		}
	}
	return strings.TrimSpace(input)
}
