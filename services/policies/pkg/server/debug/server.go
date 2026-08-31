package debug

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/checks"
	"github.com/opencloud-eu/opencloud/pkg/handlers"
	"github.com/opencloud-eu/opencloud/pkg/nats"
	"github.com/opencloud-eu/opencloud/pkg/service/debug"
	"github.com/opencloud-eu/opencloud/pkg/version"
)

// Server initializes the debug service and server.
func Server(opts ...Option) (*http.Server, error) {
	options := newOptions(opts...)

	healthHandlerConfiguration := handlers.NewCheckHandlerConfiguration().
		WithLogger(options.Logger).
		WithCheck("grpc reachability", checks.NewGRPCCheck(options.Config.GRPC.Addr))

	natsCheckFactory := func() debug.Option {
		return func(o *debug.Options) { // do nothing
		}
	}
	if !options.Config.Events.Disabled {
		secureOption := nats.Secure(
			options.Config.Events.EnableTLS,
			options.Config.Events.TLSInsecure,
			options.Config.Events.TLSRootCACertificate,
		)
		readyHandlerConfiguration := healthHandlerConfiguration.
			WithCheck("nats reachability", checks.NewNatsCheck(options.Config.Events.Endpoint, secureOption))

		natsCheckFactory = func() debug.Option { return debug.Ready(handlers.NewCheckHandler(readyHandlerConfiguration)) }
	}

	return debug.NewService(
		debug.Logger(options.Logger),
		debug.Name(options.Config.Service.Name),
		debug.Version(version.GetString()),
		debug.Address(options.Config.Debug.Addr),
		debug.Token(options.Config.Debug.Token),
		debug.Pprof(options.Config.Debug.Pprof),
		debug.Zpages(options.Config.Debug.Zpages),
		debug.Health(handlers.NewCheckHandler(healthHandlerConfiguration)),
		natsCheckFactory(),
	), nil
}
