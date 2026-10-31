package debug

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/handlers"
	"github.com/opencloud-eu/opencloud/pkg/service/debug"
	"github.com/opencloud-eu/opencloud/pkg/version"
)

// Server initializes the debug service and server.
func Server(opts ...Option) (*http.Server, error) {
	options := newOptions(opts...)

	readyHandlerConfiguration := handlers.NewCheckHandlerConfiguration().
		WithLogger(options.Logger)

	return debug.NewService(
		debug.Address(options.Config.Debug.Addr),
		debug.Token(options.Config.Debug.Token),
		debug.Pprof(options.Config.Debug.Pprof),
		debug.Zpages(options.Config.Debug.Zpages),
		debug.Logger(options.Logger),
		debug.Name(options.Config.Service.Name),
		debug.Version(version.GetString()),
		debug.Ready(handlers.NewCheckHandler(readyHandlerConfiguration)),
	), nil
}
