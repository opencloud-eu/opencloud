package debug

import (
	"net/http"
	"strconv"

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
		options.Config.Notifications.Events.EnableTLS,
		options.Config.Notifications.Events.TLSInsecure,
		options.Config.Notifications.Events.TLSRootCACertificate,
	)
	readyHandlerConfiguration := handlers.NewCheckHandlerConfiguration().
		WithLogger(options.Logger).
		WithCheck("nats reachability", checks.NewNatsCheck(options.Config.Notifications.Events.Endpoint, secureOption)).
		WithCheck("smtp-check", checks.NewTCPCheck(options.Config.Notifications.SMTP.Host+":"+strconv.Itoa(options.Config.Notifications.SMTP.Port)))

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
