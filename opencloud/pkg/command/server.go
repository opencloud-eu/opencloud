package command

import (
	"os"
	"strings"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/opencloud/pkg/runtime"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"

	"github.com/spf13/cobra"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "start a fullstack server (runtime and all services in supervised mode)",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			defaultOIDCIssuerToAuthelia()
			return configlog.ReturnError(parser.ParseConfig(cfg, false))
		},
		GroupID: CommandGroupServer,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prefer the in-memory registry as the default when running in single-binary mode
			r := runtime.New(cfg)
			return r.Start(cmd.Context())
		},
	}
}

// defaultOIDCIssuerToAuthelia points the shared OIDC issuer at the embedded Authelia provider, which
// is the default IdP in supervised mode. Authelia serves OIDC under the '/authelia' base path, so the
// issuer is '<OC_URL>/authelia'. Every issuer-consuming service honors OC_OIDC_ISSUER, so setting it
// here (before config parsing) propagates the value to all of them without per-service wiring.
//
// It only sets a default: an explicit OC_OIDC_ISSUER is left untouched (e.g. for an external IdP, or
// when falling back to the lico 'idp' service, which serves OIDC at the root and needs OC_URL).
func defaultOIDCIssuerToAuthelia() {
	if _, ok := os.LookupEnv("OC_OIDC_ISSUER"); ok {
		return
	}

	ocURL := os.Getenv("OC_URL")
	if ocURL == "" {
		ocURL = "https://localhost:9200"
	}

	_ = os.Setenv("OC_OIDC_ISSUER", strings.TrimRight(ocURL, "/")+"/authelia")
}

func init() {
	register.AddCommand(Server)
}
