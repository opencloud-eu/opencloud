package debug

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/dutchcoders/go-clamd"

	"github.com/opencloud-eu/opencloud/pkg/checks"
	"github.com/opencloud-eu/opencloud/pkg/handlers"
	"github.com/opencloud-eu/opencloud/pkg/service/debug"
	"github.com/opencloud-eu/opencloud/pkg/version"
)

// Server initializes the debug service and server.
func Server(opts ...Option) (*http.Server, error) {
	options := newOptions(opts...)

	readyHandlerConfiguration := handlers.NewCheckHandlerConfiguration().
		WithLogger(options.Logger).
		WithCheck("nats reachability", checks.NewNatsCheck(options.Config.Events.Endpoint)).
		WithCheck("antivirus reachability", func(ctx context.Context) error {
			cfg := options.Config
			switch cfg.Scanner.Type {
			default:
				return errors.New("no antivirus configured")
			case "clamav":
				return clamd.NewClamd(cfg.Scanner.ClamAV.Socket).Ping()
			case "icap":
				u, err := url.Parse(cfg.Scanner.ICAP.URL)
				if err != nil {
					return err
				}
				return checks.NewTCPCheck(u.Host)(ctx)
			}
		})

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
