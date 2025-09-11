package command

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/justinas/alice"
	"github.com/oklog/run"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	pkgmiddleware "github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	policiessvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/policies/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/proxy"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/server/debug"
	proxyHTTP "github.com/opencloud-eu/opencloud/services/proxy/pkg/server/http"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/userroles"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/signedurl"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/urfave/cli/v2"
	"go-micro.dev/v4/selector"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		Category: "server",
		Before: func(c *cli.Context) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		Action: func(c *cli.Context) error {
			userInfoCache := store.Create(
				store.Store(cfg.OIDC.UserinfoCache.Store),
				store.TTL(cfg.OIDC.UserinfoCache.TTL),
				microstore.Nodes(cfg.OIDC.UserinfoCache.Nodes...),
				microstore.Database(cfg.OIDC.UserinfoCache.Database),
				microstore.Table(cfg.OIDC.UserinfoCache.Table),
				store.DisablePersistence(cfg.OIDC.UserinfoCache.DisablePersistence),
				store.Authentication(cfg.OIDC.UserinfoCache.AuthUsername, cfg.OIDC.UserinfoCache.AuthPassword),
			)

			signingKeyStore := store.Create(
				store.Store(cfg.PreSignedURL.SigningKeys.Store),
				store.TTL(cfg.PreSignedURL.SigningKeys.TTL),
				microstore.Nodes(cfg.PreSignedURL.SigningKeys.Nodes...),
				microstore.Database("proxy"),
				microstore.Table("signing-keys"),
				store.Authentication(cfg.PreSignedURL.SigningKeys.AuthUsername, cfg.PreSignedURL.SigningKeys.AuthPassword),
			)

			logger := logging.Configure(cfg.Service.Name, cfg.Log)
			traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
			if err != nil {
				return err
			}
			cfg.GrpcClient, err = grpc.NewClient(
				append(
					grpc.GetClientOptions(cfg.GRPCClientTLS),
					grpc.WithTraceProvider(traceProvider))...)
			if err != nil {
				return err
			}

			oidcHTTPClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						MinVersion:         tls.VersionTLS12,
						InsecureSkipVerify: cfg.OIDC.Insecure, //nolint:gosec
					},
					DisableKeepAlives: true,
				},
				Timeout: time.Second * 10,
			}

			oidcClient := oidc.NewOIDCClient(
				oidc.WithAccessTokenVerifyMethod(cfg.OIDC.AccessTokenVerifyMethod),
				oidc.WithLogger(logger),
				oidc.WithHTTPClient(oidcHTTPClient),
				oidc.WithOidcIssuer(cfg.OIDC.Issuer),
				oidc.WithJWKSOptions(cfg.OIDC.JWKS),
			)

			m := metrics.New()

			gr := run.Group{}
			ctx, cancel := context.WithCancel(c.Context)

			defer cancel()

			m.BuildInfo.WithLabelValues(version.GetString()).Set(1)

			rp, err := proxy.NewMultiHostReverseProxy(
				proxy.Logger(logger),
				proxy.Config(cfg),
			)
			if err != nil {
				return fmt.Errorf("failed to initialize reverse proxy: %w", err)
			}

			reg := registry.GetRegistry()

			gatewaySelector, err := pool.GatewaySelector(
				cfg.Reva.Address,
				append(
					cfg.Reva.GetRevaOptions(),
					pool.WithRegistry(reg),
					pool.WithTracerProvider(traceProvider),
				)...)
			if err != nil {
				logger.Fatal().Err(err).Msg("Failed to get gateway selector")
			}

			serviceSelector := selector.NewSelector(selector.Registry(reg))

			var userProvider backend.UserBackend
			switch cfg.AccountBackend {
			case "cs3":
				userProvider = backend.NewCS3UserBackend(
					backend.WithLogger(logger),
					backend.WithRevaGatewaySelector(gatewaySelector),
					backend.WithSelector(serviceSelector),
					backend.WithMachineAuthAPIKey(cfg.MachineAuthAPIKey),
					backend.WithOIDCissuer(cfg.OIDC.Issuer),
					backend.WithServiceAccount(cfg.ServiceAccount),
					backend.WithAutoProvisionClaims(cfg.AutoProvisionClaims),
				)
			default:
				logger.Fatal().Msgf("Invalid accounts backend type '%s'", cfg.AccountBackend)
			}

			var publisher events.Stream
			if cfg.Events.Endpoint != "" {
				var err error
				connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
				publisher, err = stream.NatsFromConfig(connName, false, stream.NatsConfig(cfg.Events))
				if err != nil {
					logger.Error().
						Err(err).
						Msg("Error initializing events publisher")
					return fmt.Errorf("could not initialize events publisher %w", err)
				}
			}

			lh := staticroutes.StaticRouteHandler{
				Prefix:          cfg.HTTP.Root,
				UserInfoCache:   userInfoCache,
				Logger:          logger,
				Config:          *cfg,
				OidcClient:      oidcClient,
				OidcHttpClient:  oidcHTTPClient,
				Proxy:           rp,
				EventsPublisher: publisher,
				UserProvider:    userProvider,
			}
			if err != nil {
				return fmt.Errorf("failed to initialize reverse proxy: %w", err)
			}

			{
				middlewares := loadMiddlewares(logger, cfg, userInfoCache, signingKeyStore, traceProvider, *m, userProvider, publisher, gatewaySelector, serviceSelector)

				server, err := proxyHTTP.Server(
					proxyHTTP.Handler(lh.Handler()),
					proxyHTTP.Logger(logger),
					proxyHTTP.Context(ctx),
					proxyHTTP.Config(cfg),
					proxyHTTP.Metrics(metrics.New()),
					proxyHTTP.Middlewares(middlewares),
				)
				if err != nil {
					logger.Error().
						Err(err).
						Str("server", "http").
						Msg("Failed to initialize server")

					return err
				}

				gr.Add(func() error {
					return server.Run()
				}, func(err error) {
					if err == nil {
						logger.Info().
							Str("transport", "http").
							Str("server", cfg.Service.Name).
							Msg("Shutting down server")
					} else {
						logger.Error().Err(err).
							Str("transport", "http").
							Str("server", cfg.Service.Name).
							Msg("Shutting down server")
					}

					cancel()
				})
			}

			{
				debugServer, err := debug.Server(
					debug.Logger(logger),
					debug.Context(ctx),
					debug.Config(cfg),
				)
				if err != nil {
					logger.Error().Err(err).Str("server", "debug").Msg("Failed to initialize server")
					return err
				}

				gr.Add(debugServer.ListenAndServe, func(_ error) {
					_ = debugServer.Shutdown(ctx)
					cancel()
				})
			}

			return gr.Run()
		},
	}
}

func loadMiddlewares(logger log.Logger, cfg *config.Config,
	userInfoCache, signingKeyStore microstore.Store,
	traceProvider trace.TracerProvider, metrics metrics.Metrics,
	userProvider backend.UserBackend, publisher events.Publisher,
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient], serviceSelector selector.Selector) alice.Chain {

	rolesClient := settingssvc.NewRoleService("eu.opencloud.api.settings", cfg.GrpcClient)
	policiesProviderClient := policiessvc.NewPoliciesProviderService("eu.opencloud.api.policies", cfg.GrpcClient)

	var roleAssigner userroles.UserRoleAssigner
	switch cfg.RoleAssignment.Driver {
	case "default":
		roleAssigner = userroles.NewDefaultRoleAssigner(
			userroles.WithRoleService(rolesClient),
			userroles.WithLogger(logger),
		)
	case "oidc":
		roleAssigner = userroles.NewOIDCRoleAssigner(
			userroles.WithRoleService(rolesClient),
			userroles.WithLogger(logger),
			userroles.WithRolesClaim(cfg.RoleAssignment.OIDCRoleMapper.RoleClaim),
			userroles.WithRoleMapping(cfg.RoleAssignment.OIDCRoleMapper.RolesMap),
			userroles.WithRevaGatewaySelector(gatewaySelector),
			userroles.WithServiceAccount(cfg.ServiceAccount),
		)
	default:
		logger.Fatal().Msgf("Invalid role assignment driver '%s'", cfg.RoleAssignment.Driver)
	}

	oidcHTTPClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: cfg.OIDC.Insecure, //nolint:gosec
			},
			DisableKeepAlives: true,
		},
		Timeout: time.Second * 10,
	}

	var authenticators []middleware.Authenticator
	if cfg.EnableBasicAuth {
		logger.Warn().Msg("basic auth enabled, use only for testing or development")
		authenticators = append(authenticators, middleware.BasicAuthenticator{
			Logger:       logger,
			UserProvider: userProvider,
		})
	}

	if cfg.AuthMiddleware.AllowAppAuth {
		authenticators = append(authenticators, middleware.AppAuthAuthenticator{
			Logger:              logger,
			RevaGatewaySelector: gatewaySelector,
			UserRoleAssigner:    roleAssigner,
		})
	}
	authenticators = append(authenticators, middleware.NewOIDCAuthenticator(
		middleware.Logger(logger),
		middleware.UserInfoCache(userInfoCache),
		middleware.DefaultAccessTokenTTL(cfg.OIDC.UserinfoCache.TTL),
		middleware.HTTPClient(oidcHTTPClient),
		middleware.OIDCIss(cfg.OIDC.Issuer),
		middleware.OIDCClient(oidc.NewOIDCClient(
			oidc.WithAccessTokenVerifyMethod(cfg.OIDC.AccessTokenVerifyMethod),
			oidc.WithLogger(logger),
			oidc.WithHTTPClient(oidcHTTPClient),
			oidc.WithOidcIssuer(cfg.OIDC.Issuer),
			oidc.WithJWKSOptions(cfg.OIDC.JWKS),
		)),
		middleware.SkipUserInfo(cfg.OIDC.SkipUserInfo),
	))
	authenticators = append(authenticators, middleware.PublicShareAuthenticator{
		Logger:              logger,
		RevaGatewaySelector: gatewaySelector,
	})

	var signURLVerifier signedurl.Verifier

	if cfg.PreSignedURL.JWTSigningSharedSecret != "" {
		var err error
		signURLVerifier, err = signedurl.NewJWTSignedURL(signedurl.WithSecret(cfg.PreSignedURL.JWTSigningSharedSecret))
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to initialize signed URL configuration.")
		}
	}
	authenticators = append(authenticators, middleware.SignedURLAuthenticator{
		Logger:             logger,
		PreSignedURLConfig: cfg.PreSignedURL,
		UserProvider:       userProvider,
		UserRoleAssigner:   roleAssigner,
		Store:              signingKeyStore,
		Now:                time.Now,
		URLVerifier:        signURLVerifier,
	})

	cspConfig, err := middleware.LoadCSPConfig(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load CSP configuration.")
	}

	return alice.New(
		chimiddleware.RealIP,
		chimiddleware.RequestID,
		// first make sure we log all requests and redirect to https if necessary
		otelhttp.NewMiddleware("proxy",
			otelhttp.WithTracerProvider(traceProvider),
			otelhttp.WithSpanNameFormatter(func(name string, r *http.Request) string {
				return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			}),
		),
		middleware.Tracer(traceProvider),
		pkgmiddleware.TraceContext,
		middleware.Instrumenter(metrics),
		middleware.AccessLog(logger),
		middleware.ContextLogger(logger),
		middleware.HTTPSRedirect,
		middleware.Security(cspConfig),
		router.Middleware(serviceSelector, cfg.PolicySelector, cfg.Policies, logger),
		middleware.Authentication(
			authenticators,
			middleware.CredentialsByUserAgent(cfg.AuthMiddleware.CredentialsByUserAgent),
			middleware.Logger(logger),
			middleware.OIDCIss(cfg.OIDC.Issuer),
			middleware.EnableBasicAuth(cfg.EnableBasicAuth || cfg.AuthMiddleware.AllowAppAuth),
			middleware.TraceProvider(traceProvider),
		),
		middleware.AccountResolver(
			middleware.Logger(logger),
			middleware.TraceProvider(traceProvider),
			middleware.UserProvider(userProvider),
			middleware.UserRoleAssigner(roleAssigner),
			middleware.SkipUserInfo(cfg.OIDC.SkipUserInfo),
			middleware.UserOIDCClaim(cfg.UserOIDCClaim),
			middleware.UserCS3Claim(cfg.UserCS3Claim),
			middleware.AutoprovisionAccounts(cfg.AutoprovisionAccounts),
			middleware.EventsPublisher(publisher),
		),
		middleware.SelectorCookie(
			middleware.Logger(logger),
			middleware.TraceProvider(traceProvider),
			middleware.PolicySelectorConfig(*cfg.PolicySelector),
		),
		middleware.Policies(
			cfg.PoliciesMiddleware.Query,
			middleware.Logger(logger),
			middleware.TraceProvider(traceProvider),
			middleware.WithRevaGatewaySelector(gatewaySelector),
			middleware.PoliciesProviderService(policiesProviderClient),
		),
		// finally, trigger home creation when a user logs in
		middleware.CreateHome(
			middleware.Logger(logger),
			middleware.TraceProvider(traceProvider),
			middleware.WithRevaGatewaySelector(gatewaySelector),
			middleware.RoleQuotas(cfg.RoleQuotas),
		),
	)
}
