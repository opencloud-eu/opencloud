package command

import (
	"fmt"
	"log"
	"net"
	"net/rpc"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ListCommand is the entrypoint for the list command.
func ListCommand(cfg *config.Config) *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "list OpenCloud services running in the runtime (supervised mode)",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnError(parser.ParseConfig(cfg, true))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			host := viper.GetString("hostname")
			port := viper.GetString("port")

			client, err := rpc.DialHTTP("tcp", net.JoinHostPort(host, port))
			if err != nil {
				log.Fatalf("Failed to connect to the runtime. Has the runtime been started and did you configure the right runtime address (\"%s\")", cfg.Runtime.Host+":"+cfg.Runtime.Port)
			}

			var arg1 string

			if err := client.Call("Service.List", struct{}{}, &arg1); err != nil {
				log.Fatal(err)
			}

			fmt.Println(arg1)

			return nil
		},
	}

	listCmd.Flags().String("hostname", "localhost", "hostname of the runtime")
	_ = viper.BindEnv("hostname", "OC_RUNTIME_HOST")
	_ = viper.BindPFlag("hostname", listCmd.Flags().Lookup("hostname"))

	listCmd.Flags().String("port", "9250", "port of the runtime")
	_ = viper.BindEnv("port", "OC_RUNTIME_PORT")
	_ = viper.BindPFlag("port", listCmd.Flags().Lookup("port"))
	return listCmd
}

func init() {
	register.AddCommand(ListCommand)
}
