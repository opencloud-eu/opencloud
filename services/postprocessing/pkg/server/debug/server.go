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

	secureOption := nats.Secure(
		options.Config.Postprocessing.Events.EnableTLS,
		options.Config.Postprocessing.Events.TLSInsecure,
		options.Config.Postprocessing.Events.TLSRootCACertificate,
	)
	readyHandlerConfiguration := handlers.NewCheckHandlerConfiguration().
		WithLogger(options.Logger).
		WithCheck("nats reachability", checks.NewNatsCheck(options.Config.Postprocessing.Events.Endpoint, secureOption))

	return debug.NewService(
		debug.Logger(options.Logger),
		debug.Name(options.Config.Service.Name),
		debug.Version(version.GetString()),
		debug.Address(options.Config.Debug.Addr),
		debug.Token(options.Config.Debug.Token),
		debug.Pprof(options.Config.Debug.Pprof),
		debug.Zpages(options.Config.Debug.Zpages),
		debug.Ready(handlers.NewCheckHandler(readyHandlerConfiguration)),
	), nil
}
