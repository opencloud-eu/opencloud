package http

import (
	"errors"
	"fmt"
	"path"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"go-micro.dev/v4"

	"github.com/opencloud-eu/opencloud/pkg/cors"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	natspkg "github.com/opencloud-eu/opencloud/pkg/nats"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/service/http"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/web"
	"github.com/opencloud-eu/opencloud/services/web/pkg/announcement"
	"github.com/opencloud-eu/opencloud/services/web/pkg/apps"
	svc "github.com/opencloud-eu/opencloud/services/web/pkg/service/v0"
)

var (
	// _customAppsEndpoint path is used to make app artifacts available by the web service.
	_customAppsEndpoint = "/assets/apps"
)

// Server initializes the http service and server.
func Server(opts ...Option) (http.Service, error) {
	options := newOptions(opts...)

	service, err := http.NewService(
		http.TLSConfig(options.Config.HTTP.TLS),
		http.Logger(options.Logger),
		http.Namespace(options.Namespace),
		http.Name("web"),
		http.Version(version.GetString()),
		http.Address(options.Config.HTTP.Addr),
		http.Context(options.Context),
		http.Flags(options.Flags...),
		http.TraceProvider(options.TraceProvider),
	)
	if err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error initializing http service")
		return http.Service{}, fmt.Errorf("could not initialize http service: %w", err)
	}

	gatewaySelector, err := pool.GatewaySelector(
		options.Config.GatewayAddress,
		pool.WithRegistry(registry.GetRegistry()),
		pool.WithTracerProvider(options.TraceProvider),
	)
	if err != nil {
		return http.Service{}, err
	}

	appsFS := fsx.NewFallbackFS(
		fsx.NewReadOnlyFs(fsx.NewBasePathFs(fsx.NewOsFs(), options.Config.Asset.AppsPath)),
		fsx.NewBasePathFs(fsx.FromIOFS(web.Assets), "assets/apps"),
	)
	// build and inject the list of applications into the config
	for _, application := range apps.List(options.Logger, options.Config.Apps, appsFS.Secondary().IOFS(), appsFS.Primary().IOFS()) {
		options.Config.Web.Config.ExternalApps = append(
			options.Config.Web.Config.ExternalApps,
			application.ToExternal(path.Join(options.Config.HTTP.Root, _customAppsEndpoint)),
		)
	}

	coreFS := fsx.NewFallbackFS(
		fsx.NewBasePathFs(fsx.NewOsFs(), options.Config.Asset.CorePath),
		fsx.NewBasePathFs(fsx.FromIOFS(web.Assets), "assets/core"),
	)
	themeFS := fsx.NewFallbackFS(
		fsx.NewBasePathFs(fsx.NewOsFs(), options.Config.Asset.ThemesPath),
		fsx.NewBasePathFs(fsx.FromIOFS(web.Assets), "assets/themes"),
	)

	// NATS JetStream key-value store for runtime managed web settings, e.g. the announcement banner.
	// Connect eagerly and fail fast: an unreachable store means the feature would be broken, so a
	// clear startup error is preferable to silently degrading.
	natsConn, err := nats.Connect(
		strings.Join(options.Config.Store.Nodes, ","),
		natspkg.Secure(options.Config.Store.EnableTLS, options.Config.Store.TLSInsecure, options.Config.Store.TLSRootCACertificate),
		nats.UserInfo(options.Config.Store.AuthUsername, options.Config.Store.AuthPassword),
	)
	if err != nil {
		return http.Service{}, fmt.Errorf("could not connect to nats for the announcement store: %w", err)
	}
	js, err := jetstream.New(natsConn)
	if err != nil {
		return http.Service{}, fmt.Errorf("could not create jetstream context for the announcement store: %w", err)
	}
	kv, err := js.KeyValue(options.Context, options.Config.Store.Database)
	if err != nil {
		if !errors.Is(err, jetstream.ErrBucketNotFound) {
			return http.Service{}, fmt.Errorf("could not open the announcement store bucket %q: %w", options.Config.Store.Database, err)
		}
		if kv, err = js.CreateKeyValue(options.Context, jetstream.KeyValueConfig{Bucket: options.Config.Store.Database}); err != nil {
			return http.Service{}, fmt.Errorf("could not create the announcement store bucket %q: %w", options.Config.Store.Database, err)
		}
	}
	announcementStore := announcement.NewStore(kv)

	handle, err := svc.NewService(
		svc.Logger(options.Logger),
		svc.CoreFS(coreFS.IOFS()),
		svc.AppFS(appsFS.IOFS()),
		svc.ThemeFS(themeFS),
		svc.AnnouncementStore(announcementStore),
		svc.AppsHTTPEndpoint(_customAppsEndpoint),
		svc.Config(options.Config),
		svc.GatewaySelector(gatewaySelector),
		svc.Middleware(
			chimiddleware.RealIP,
			chimiddleware.RequestID,
			chimiddleware.Compress(5),
			middleware.NoCache,
			middleware.Version(
				options.Config.Service.Name,
				version.GetString(),
			),
			middleware.Logger(
				options.Logger,
			),
			middleware.Cors(
				cors.Logger(options.Logger),
				cors.AllowedOrigins(options.Config.HTTP.CORS.AllowedOrigins),
				cors.AllowedMethods(options.Config.HTTP.CORS.AllowedMethods),
				cors.AllowedHeaders(options.Config.HTTP.CORS.AllowedHeaders),
				cors.AllowCredentials(options.Config.HTTP.CORS.AllowCredentials),
			),
		),
		svc.TraceProvider(options.TraceProvider),
	)

	if err != nil {
		return http.Service{}, err
	}

	{
		handle = svc.NewInstrument(handle, options.Metrics)
		handle = svc.NewLogging(handle, options.Logger)
	}

	if err := micro.RegisterHandler(service.Server(), handle); err != nil {
		return http.Service{}, err
	}

	return service, nil
}
