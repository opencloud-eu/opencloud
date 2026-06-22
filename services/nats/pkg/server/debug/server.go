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
		options.Config.Nats.EnableTLS,
		options.Config.Nats.TLSSkipVerifyClientCert,
		options.Config.Nats.TLSCert,
	)
	// For nats readiness and liveness checks are identical
	// the nats server will neither be healthy nor ready when it can not reach the nats server/cluster
	checkHandler := handlers.NewCheckHandler(
		handlers.NewCheckHandlerConfiguration().
			WithLogger(options.Logger).
			WithCheck("nats reachability", checks.NewNatsCheck(options.Config.Nats.Host+":"+strconv.Itoa(options.Config.Nats.Port), secureOption)),
	)

	return debug.NewService(
		debug.Logger(options.Logger),
		debug.Name(options.Config.Service.Name),
		debug.Version(version.GetString()),
		debug.Address(options.Config.Debug.Addr),
		debug.Token(options.Config.Debug.Token),
		debug.Pprof(options.Config.Debug.Pprof),
		debug.Zpages(options.Config.Debug.Zpages),
		debug.Health(checkHandler),
		debug.Ready(checkHandler),
	), nil
}
