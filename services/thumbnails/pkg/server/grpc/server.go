package grpc

import (
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/service/grpc/handler/ratelimiter"
	"github.com/opencloud-eu/opencloud/pkg/version"
	thumbnailssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/thumbnails/v0"
	svc "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/service/grpc/v0"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/service/grpc/v0/decorators"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail/imgsource"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail/storage"
	"github.com/opencloud-eu/reva/v2/pkg/bytesize"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

// NewService initializes the grpc service and server.
func NewService(opts ...Option) grpc.Service {
	options := newOptions(opts...)

	service, err := grpc.NewServiceWithClient(
		options.Config.GrpcClient,
		grpc.TLSEnabled(options.Config.GRPC.TLS.Enabled),
		grpc.TLSCert(
			options.Config.GRPC.TLS.Cert,
			options.Config.GRPC.TLS.Key,
		),
		grpc.Logger(options.Logger),
		grpc.Namespace(options.Namespace),
		grpc.Name(options.Name),
		grpc.Version(version.GetString()),
		grpc.Address(options.Address),
		grpc.Context(options.Context),
		grpc.Version(version.GetString()),
		grpc.TraceProvider(options.TraceProvider),
		grpc.HandlerWrappers(ratelimiter.NewHandlerWrapper(options.MaxConcurrentRequests)),
	)
	if err != nil {
		options.Logger.Fatal().Err(err).Msg("Error creating thumbnail service")
		return grpc.Service{}
	}

	tconf := options.Config.Thumbnail
	tm, err := pool.StringToTLSMode(options.Config.GRPCClientTLS.Mode)
	if err != nil {
		options.Logger.Error().Err(err).Msg("could not get gateway client tls mode")
		return grpc.Service{}
	}

	gatewaySelector, err := pool.GatewaySelector(tconf.RevaGateway,
		pool.WithTLSCACert(options.Config.GRPCClientTLS.CACert),
		pool.WithTLSMode(tm),
		pool.WithRegistry(registry.GetRegistry()),
		pool.WithTracerProvider(options.TraceProvider),
	)
	if err != nil {
		options.Logger.Error().Err(err).Msg("could not get gateway selector")
		return grpc.Service{}
	}
	b, err := bytesize.Parse(tconf.MaxInputImageFileSize)
	if err != nil {
		options.Logger.Error().Err(err).Msg("could not parse MaxInputImageFileSize")
		return grpc.Service{}
	}

	var thumbnail decorators.DecoratedService
	{
		thumbnail = svc.NewService(
			svc.Config(options.Config),
			svc.Logger(options.Logger),
			svc.ThumbnailSource(imgsource.NewWebDavSource(tconf, b)),
			svc.ThumbnailStorage(
				storage.NewFileSystemStorage(
					tconf.FileSystemStorage,
					options.Logger,
				),
			),
			svc.CS3Source(imgsource.NewCS3Source(tconf, gatewaySelector, b)),
			svc.GatewaySelector(gatewaySelector),
		)
		thumbnail = decorators.NewInstrument(thumbnail, options.Metrics)
		thumbnail = decorators.NewLogging(thumbnail, options.Logger)
		thumbnail = decorators.NewTracing(thumbnail, options.TraceProvider)
	}

	_ = thumbnailssvc.RegisterThumbnailServiceHandler(
		service.Server(),
		thumbnail,
	)

	return service
}
