package theme

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	websvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/web/v0"
	"github.com/opencloud-eu/opencloud/services/console/pkg/config"
	"github.com/opencloud-eu/opencloud/services/console/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/console/pkg/console"
	"github.com/opencloud-eu/opencloud/services/console/pkg/web"
)

func Pull(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "pull",
		Usage: "pull the latest theme from the configured source",
		Before: func(c *cli.Context) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		Action: func(cCtx *cli.Context) error {
			grpcClient, err := console.NewGRPCClient(cfg)
			if err != nil {
				return err
			}

			token, err := console.ParseJWTToken(cfg.RemoteConsole.JWTToken, cfg.RemoteConsole.JWTTokenKey)
			if err != nil {
				return err
			}

			webRepository, err := web.NewGRPCRepository(web.GRPCRepositoryOptions{
				WebService: websvc.NewWebService("eu.opencloud.api.web", grpcClient),
			})
			if err != nil {
				return fmt.Errorf("failed to create web repository: %w", err)
			}

			consoleRepository, err := web.NewConsoleHTTPRepository(web.ConsoleHTTPRepositoryOptions{
				ConsoleAPIRoot: "https://host.docker.internal:3000/api/",
				HTTPClient:     console.DefaultHTTPClient,
			})
			if err != nil {
				return fmt.Errorf("failed to create console repository: %w", err)
			}

			webService, err := web.NewService(web.ServiceOptions{
				Repository:        webRepository,
				ConsoleRepository: consoleRepository,
			})
			if err != nil {
				return fmt.Errorf("failed to create web service: %w", err)
			}

			if err := webService.ThemeApply(context.Background(), token); err != nil {
				return fmt.Errorf("failed to apply theme: %w", err)
			}

			return nil
		},
	}
}
