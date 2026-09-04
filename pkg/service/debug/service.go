package debug

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/justinas/alice"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/zpages"

	"github.com/opencloud-eu/opencloud/pkg/cors"
	"github.com/opencloud-eu/opencloud/pkg/handlers"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	graphMiddleware "github.com/opencloud-eu/opencloud/services/graph/pkg/middleware"
)

var handleProbe = func(mux *http.ServeMux, pattern string, h http.Handler, logger log.Logger) {
	if h == nil {
		h = handlers.NewCheckHandler(handlers.NewCheckHandlerConfiguration())
		logger.Info().
			Str("endpoint", pattern).
			Msg("no probe provided, reverting to default (OK)")
	}

	mux.Handle(pattern, h)
}

// NewService initializes a new debug service.
func NewService(opts ...Option) *http.Server {
	dopts := newOptions(opts...)
	mux := http.NewServeMux()

	mux.Handle("/metrics", alice.New(
		graphMiddleware.Token(
			dopts.Token,
		),
	).Then(
		promhttp.Handler(),
	))

	handleProbe(mux, "/healthz", dopts.Health, dopts.Logger) // healthiness check
	handleProbe(mux, "/readyz", dopts.Ready, dopts.Logger)   // readiness check

	if dopts.ConfigDump != nil {
		mux.Handle("/config", dopts.ConfigDump)
	}

	if dopts.Pprof {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	if dopts.Zpages {
		h := zpages.NewTracezHandler(zpages.NewSpanProcessor())
		mux.Handle("/debug", h)
	}

	baseCtx := dopts.Context
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	return &http.Server{
		Addr: dopts.Address,
		BaseContext: func(_ net.Listener) context.Context {
			return baseCtx
		},
		Handler: alice.New(
			chimiddleware.ClientIPFromRemoteAddr,
			chimiddleware.RequestID,
			middleware.NoCache,
			middleware.Cors(
				cors.AllowedOrigins(dopts.CorsAllowedOrigins),
				cors.AllowedMethods(dopts.CorsAllowedMethods),
				cors.AllowedHeaders(dopts.CorsAllowedHeaders),
				cors.AllowCredentials(dopts.CorsAllowCredentials),
			),
			middleware.Version(
				dopts.Name,
				dopts.Version,
			),
		).Then(
			mux,
		),
	}
}
