package grpc

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/version"
	websvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/web/v0"
	"github.com/opencloud-eu/opencloud/services/web/pkg/fs"
	svc "github.com/opencloud-eu/opencloud/services/web/pkg/service/grpc/v0"
)

// Server initializes a new go-micro service ready to run
func Server(opts ...Option) (grpc.Service, error) {
	options := newOptions(opts...)

	service, err := grpc.NewServiceWithClient(
		options.Config.GrpcClient,
		grpc.TLSEnabled(options.Config.GRPC.TLS.Enabled),
		grpc.TLSCert(
			options.Config.GRPC.TLS.Cert,
			options.Config.GRPC.TLS.Key,
		),
		grpc.Name(options.Config.Service.Name),
		grpc.Context(options.Context),
		grpc.Address(options.Config.GRPC.Addr),
		grpc.Namespace(options.Config.GRPC.Namespace),
		grpc.Logger(options.Logger),
		grpc.Version(version.GetString()),
		grpc.TraceProvider(options.TraceProvider),
	)
	if err != nil {
		options.Logger.Fatal().Err(err).Msg("Error creating web service")
		return grpc.Service{}, err
	}

	themeFS, err := fs.NewThemeFS(options.Config)
	if err != nil {
		return grpc.Service{}, fmt.Errorf("could not initialize theme filesystem: %w", err)
	}

	handle, err := svc.NewHandler(
		svc.Config(options.Config),
		svc.Logger(options.Logger),
		svc.JWTSecret(options.JWTSecret),
		svc.TracerProvider(options.TraceProvider),
		svc.ThemeFS(themeFS),
	)
	if err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error initializing web service")
		return grpc.Service{}, err
	}

	if err := websvc.RegisterWebServiceHandler(
		service.Server(),
		handle,
	); err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error registering web provider handler")
		return grpc.Service{}, err
	}

	return service, nil
}
