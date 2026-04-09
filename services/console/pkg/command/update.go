package command

import (
	"fmt"
	"net/http"

	"github.com/centrifugal/centrifuge-go"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"

	"github.com/opencloud-eu/reva/v2/pkg/events/stream"

	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	websvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/web/v0"
	"github.com/opencloud-eu/opencloud/services/console/pkg/config"
	"github.com/opencloud-eu/opencloud/services/console/pkg/console"
	"github.com/opencloud-eu/opencloud/services/console/pkg/console/remote"
	"github.com/opencloud-eu/opencloud/services/console/pkg/features"
	"github.com/opencloud-eu/opencloud/services/console/pkg/web"
)

func Update(cfg *config.Config) *cobra.Command {
	connectCmd := &cobra.Command{
		Use:   "update",
		Short: "update instance related console settings",
	}

	subscribeCmd := &cobra.Command{
		Use:     "subscribe",
		Short:   "get updates automatically",
		PreRunE: preRunE(cfg),
		RunE: func(cmd *cobra.Command, args []string) error {
			common, err := newCommons(cfg)
			if err != nil {
				return fmt.Errorf("failed to build environment: %w", err)
			}

			client, err := remote.NewCentrifugoSubscription(remote.CentrifugoSubscriptionOptions{
				Token:      common.token,
				URLBuilder: common.urlBuilder,
				Claims:     common.claims,
				CentrifugeConfig: centrifuge.Config{
					Token:     common.token.Raw,
					TLSConfig: features.DefaultTLSConfig,
				},
				CentrifugeSubscriptionConfig: centrifuge.SubscriptionConfig{
					Recoverable: true,
					JoinLeave:   true,
				},
				Logger: common.logger,
			})
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}
			defer func() {
				_ = client.Close()
			}()

			handler := remote.Handler{
				ThemeAssigned: func(m remote.Message[remote.MessagePayloadThemeAssigned]) error {
					return common.webService.ThemeApply(cmd.Context())
				},
				ThemeUnassigned: func(m remote.Message[struct{}]) error {
					return common.webService.ThemeRemove(cmd.Context())
				},
			}

			if err := client.Handle(handler); err != nil {
				return fmt.Errorf("failed to subscribe: %w", err)
			}

			<-cmd.Context().Done()

			return nil
		},
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "run update related commands manually",
	}

	{
		webThemeGetCmd := &cobra.Command{
			Use:     "web-theme-get",
			Short:   "download and apply configured console web theme",
			PreRunE: preRunE(cfg),
			RunE: func(cmd *cobra.Command, args []string) error {
				common, err := newCommons(cfg)
				if err != nil {
					return fmt.Errorf("failed to build environment: %w", err)
				}

				if err := common.webService.ThemeApply(cmd.Context()); err != nil {
					return fmt.Errorf("failed to apply theme: %w", err)
				}

				return nil
			},
		}

		runCmd.AddCommand(
			webThemeGetCmd,
		)
	}

	connectCmd.AddCommand(
		runCmd,
		subscribeCmd,
	)

	return connectCmd
}

type commons struct {
	token      *jwt.Token
	claims     *console.Claims
	webService web.Service
	logger     log.Logger
	urlBuilder console.URLBuilder
}

func newCommons(cfg *config.Config) (commons, error) {
	grpcClient, err := console.NewGRPCClient(cfg.Context, cfg.Commons.TracesExporter, cfg.Service.Name, cfg.GRPCClientTLS)
	if err != nil {
		return commons{}, err
	}

	token, claims, err := console.ParseUnverifiedJWTToken(cfg.RemoteConsole.JWTToken)
	if err != nil {
		return commons{}, err
	}

	urlBuilder, err := console.NewURLBuilder(claims)
	if err != nil {
		return commons{}, err
	}

	logger := log.Configure(cfg.Service.Name, cfg.Commons, cfg.LogLevel)
	consoleRepository, err := remote.NewHTTPRepository(remote.HTTPRepositoryOptions{
		Client:     &http.Client{Transport: features.DefaultHTTPTransport},
		Token:      token,
		Claims:     claims,
		URLBuilder: urlBuilder,
		Logger:     logger,
	})
	if err != nil {
		return commons{}, fmt.Errorf("failed to create console repository: %w", err)
	}

	webRepository, err := web.NewGRPCRepository(web.GRPCRepositoryOptions{
		WebService: websvc.NewWebService("eu.opencloud.api.web", grpcClient),
		Logger:     logger,
	})
	if err != nil {
		return commons{}, fmt.Errorf("failed to create web repository: %w", err)
	}

	eventStream, err := stream.NatsFromConfig(
		generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus),
		false,
		stream.NatsConfig(cfg.Events),
	)
	if err != nil {
		return commons{}, err
	}

	webSSEService, err := web.NewSSEService(web.SSEServiceOptions{
		EventStream: eventStream,
		Logger:      logger,
	})
	if err != nil {
		return commons{}, fmt.Errorf("failed to create web service: %w", err)
	}

	webCoreService, err := web.NewCoreService(web.CoreServiceOptions{
		Repository:        webRepository,
		ConsoleRepository: consoleRepository,
		Next:              webSSEService,
		Logger:            logger,
	})
	if err != nil {
		return commons{}, fmt.Errorf("failed to create web service: %w", err)
	}

	return commons{
		webService: webCoreService,
		token:      token,
		claims:     claims,
		logger:     logger,
		urlBuilder: urlBuilder,
	}, nil
}
