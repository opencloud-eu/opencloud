package grpc

import (
	"github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/version"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	svc "github.com/opencloud-eu/opencloud/services/search/pkg/service/grpc/v0"
)

// Server initializes a new go-micro service ready to run
func Server(opts ...Option) (grpc.Service, func(), error) {
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
		options.Logger.Fatal().Err(err).Msg("Error creating search service")
		return grpc.Service{}, func() {}, err
	}

	handle, teardown, err := svc.NewHandler(
		svc.Config(options.Config),
		svc.Logger(options.Logger),
		svc.JWTSecret(options.JWTSecret),
		svc.TracerProvider(options.TraceProvider),
		svc.Metrics(options.Metrics),
	)
	if err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error initializing search service")
		return grpc.Service{}, teardown, err
	}

	if err := searchsvc.RegisterSearchProviderHandler(
		service.Server(),
		handle,
	); err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error registering search provider handler")
		return grpc.Service{}, teardown, err
	}

	return service, teardown, nil
}
