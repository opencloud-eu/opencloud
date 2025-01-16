package command

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	ocisinit "github.com/opencloud-eu/opencloud/opencloud/pkg/init"
	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	cli "github.com/urfave/cli/v2"
)

// InitCommand is the entrypoint for the init command
func InitCommand(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "initialise an OpenCloud config",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "insecure",
				EnvVars: []string{"OC_INSECURE"},
				Value:   "ask",
				Usage:   "Allow insecure OpenCloud config",
			},
			&cli.BoolFlag{
				Name:    "diff",
				Aliases: []string{"d"},
				Usage:   "Show the difference between the current config and the new one",
				Value:   false,
			},
			&cli.BoolFlag{
				Name:    "force-overwrite",
				Aliases: []string{"f"},
				EnvVars: []string{"OC_FORCE_CONFIG_OVERWRITE"},
				Value:   false,
				Usage:   "Force overwrite existing config file",
			},
			&cli.StringFlag{
				Name:    "config-path",
				Value:   defaults.BaseConfigPath(),
				Usage:   "Config path for the OpenCloud runtime",
				EnvVars: []string{"OC_CONFIG_DIR", "OC_BASE_DATA_PATH"},
			},
			&cli.StringFlag{
				Name:    "admin-password",
				Aliases: []string{"ap"},
				EnvVars: []string{"ADMIN_PASSWORD", "IDM_ADMIN_PASSWORD"},
				Usage:   "Set admin password instead of using a random generated one",
			},
		},
		Action: func(c *cli.Context) error {
			insecureFlag := c.String("insecure")
			insecure := false
			if insecureFlag == "ask" {
				answer := strings.ToLower(stringPrompt("Do you want to configure OpenCloud with certificate checking disabled?\n This is not recommended for public instances! [yes | no = default]"))
				if answer == "yes" || answer == "y" {
					insecure = true
				}
			} else if insecureFlag == strings.ToLower("true") || insecureFlag == strings.ToLower("yes") || insecureFlag == strings.ToLower("y") {
				insecure = true
			}
			err := ocisinit.CreateConfig(insecure, c.Bool("force-overwrite"), c.Bool("diff"), c.String("config-path"), c.String("admin-password"))
			if err != nil {
				log.Fatalf("Could not create config: %s", err)
			}
			return nil
		},
	}
}

func init() {
	register.AddCommand(InitCommand)
}

func stringPrompt(label string) string {
	input := ""
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, label+" ")
		input, _ = reader.ReadString('\n')
		if input != "" {
			break
		}
	}
	return strings.TrimSpace(input)
}
