package debug

import (
	"net/http"
	"net/url"

	"github.com/opencloud-eu/opencloud/pkg/checks"
	"github.com/opencloud-eu/opencloud/pkg/handlers"
	"github.com/opencloud-eu/opencloud/pkg/service/debug"
	"github.com/opencloud-eu/opencloud/pkg/version"
)

// Server initializes the debug service and server.
func Server(opts ...Option) (*http.Server, error) {
	options := newOptions(opts...)

	healthHandlerConfiguration := handlers.NewCheckHandlerConfiguration().
		WithLogger(options.Logger).
		WithCheck("web reachability", checks.NewHTTPCheck(options.Config.HTTP.Addr))

	readyHandlerConfiguration := healthHandlerConfiguration

	// Check for LDAP reachability, when we're using the LDAP backend
	if options.Config.Identity.Backend == "ldap" {
		u, err := url.Parse(options.Config.Identity.LDAP.URI)
		if err != nil {
			return nil, err
		}
		readyHandlerConfiguration = readyHandlerConfiguration.
			WithCheck("ldap reachability", checks.NewTCPCheck(u.Host))
	}

	// only check nats if really needed
	if options.Config.Events.Endpoint != "" {
		readyHandlerConfiguration = readyHandlerConfiguration.
			WithCheck("nats reachability", checks.NewNatsCheck(options.Config.Events.Endpoint))
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
		debug.Ready(handlers.NewCheckHandler(readyHandlerConfiguration)),
	), nil
}
