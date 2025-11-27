package command

import (
	"context"
	"fmt"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"

	"time"

	applicationsv1beta1 "github.com/cs3org/go-cs3apis/cs3/auth/applications/v1beta1"
	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/config/parser"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc/metadata"
)

// Create is the entrypoint for the app auth create command
func Create(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "create",
		Usage:    "create an app auth token for a user",
		Category: "maintenance",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "user-name",
				Value: "",
				Usage: "user to create the app-token for",
			},
			&cli.StringFlag{
				Name:  "expiration",
				Value: "72h",
				Usage: "expiration of the app password, e.g. 72h, 1h, 1m, 1s. Default is 72h.",
			},
		},
		Before: func(_ *cli.Context) error {
			return configlog.ReturnError(parser.ParseConfig(cfg))
		},
		Action: func(c *cli.Context) error {
			traceProvider, err := tracing.GetTraceProvider(c.Context, cfg.Commons.TracesExporter, cfg.Service.Name)
			if err != nil {
				return err
			}

			gatewaySelector, err := pool.GatewaySelector(
				cfg.Reva.Address,
				append(
					cfg.Reva.GetRevaOptions(),
					pool.WithRegistry(registry.GetRegistry()),
					pool.WithTracerProvider(traceProvider),
				)...)
			if err != nil {
				return err
			}

			next, err := gatewaySelector.Next()
			if err != nil {
				return err
			}

			userName := c.String("user-name")
			if userName == "" {
				fmt.Printf("Username to create app token for: ")
				if _, err := fmt.Scanln(&userName); err != nil {
					return err
				}
			}

			ctx := context.Background()
			authRes, err := next.Authenticate(ctx, &gatewayv1beta1.AuthenticateRequest{
				Type:         "machine",
				ClientId:     "username:" + userName,
				ClientSecret: cfg.MachineAuthAPIKey,
			})
			if err != nil {
				return err
			}
			if authRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
				return fmt.Errorf("error authenticating user: %s", authRes.GetStatus().GetMessage())
			}

			granteeCtx := ctxpkg.ContextSetUser(context.Background(), &userpb.User{Id: authRes.GetUser().GetId()})
			granteeCtx = metadata.AppendToOutgoingContext(granteeCtx, ctxpkg.TokenHeader, authRes.GetToken())

			scopes, err := scope.AddOwnerScope(map[string]*authpb.Scope{})
			if err != nil {
				return err
			}

			expiry, err := time.ParseDuration(c.String("expiration"))
			if err != nil {
				return err
			}

			appPassword, err := next.GenerateAppPassword(granteeCtx, &applicationsv1beta1.GenerateAppPasswordRequest{
				TokenScope: scopes,
				Label:      "Generated via CLI",
				Expiration: &typesv1beta1.Timestamp{
					Seconds: uint64(time.Now().Add(expiry).Unix()),
				},
			})
			if err != nil {
				return err
			}

			fmt.Printf("App token created for %s", authRes.GetUser().GetUsername())
			fmt.Println()
			fmt.Printf(" token: %s", appPassword.GetAppPassword().GetPassword())
			fmt.Println()

			return nil
		},
	}
}
