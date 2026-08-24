package grpc

import (
	"github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/version"
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
	)
	if err != nil {
		options.Logger.Fatal().Err(err).Msg("Error creating event history service")
		return grpc.Service{}
	}

	return service
}
