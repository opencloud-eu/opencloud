package http

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/opencloud-eu/opencloud/pkg/cors"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	ohttp "github.com/opencloud-eu/opencloud/pkg/service/http"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	serviceErrors "github.com/opencloud-eu/opencloud/services/webfinger/pkg/service/v0"
	svc "github.com/opencloud-eu/opencloud/services/webfinger/pkg/service/v0"
	"github.com/pkg/errors"
	"github.com/riandyrn/otelchi"
	"go-micro.dev/v4"
)

// Server initializes the http service and server.
func Server(opts ...Option) (ohttp.Service, error) {
	options := newOptions(opts...)
	service := options.Service

	newService, err := ohttp.NewService(
		ohttp.TLSConfig(options.Config.HTTP.TLS),
		ohttp.Logger(options.Logger),
		ohttp.Namespace(options.Config.HTTP.Namespace),
		ohttp.Name(options.Config.Service.Name),
		ohttp.Version(version.GetString()),
		ohttp.Address(options.Config.HTTP.Addr),
		ohttp.Context(options.Context),
		ohttp.Flags(options.Flags...),
	)
	if err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error initializing http service")
		return ohttp.Service{}, err
	}

	mux := chi.NewMux()

	mux.Use(chimiddleware.RealIP)
	mux.Use(chimiddleware.RequestID)
	mux.Use(middleware.TraceContext)
	mux.Use(middleware.NoCache)
	mux.Use(
		middleware.Cors(
			cors.Logger(options.Logger),
			cors.AllowedOrigins(options.Config.HTTP.CORS.AllowedOrigins),
			cors.AllowedMethods(options.Config.HTTP.CORS.AllowedMethods),
			cors.AllowedHeaders(options.Config.HTTP.CORS.AllowedHeaders),
			cors.AllowCredentials(options.Config.HTTP.CORS.AllowCredentials),
		))

	mux.Use(middleware.Version(
		options.Name,
		version.String,
	))

	mux.Use(
		otelchi.Middleware(
			options.Name,
			otelchi.WithChiRoutes(mux),
			otelchi.WithTracerProvider(options.TraceProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	var oidcHTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: options.Config.Insecure, //nolint:gosec
			},
			DisableKeepAlives: true,
		},
		Timeout: time.Second * 10,
	}

	mux.Use(middleware.OidcAuth(
		middleware.WithLogger(options.Logger),
		middleware.WithOidcIssuer(options.Config.IDP),
		middleware.WithHttpClient(*oidcHTTPClient),
	))

	// this logs http request related data
	mux.Use(middleware.Logger(
		options.Logger,
	))

	mux.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Get("/.well-known/webfinger", WebfingerHandler(service))
	})

	err = micro.RegisterHandler(newService.Server(), mux)
	if err != nil {
		options.Logger.Fatal().Err(err).Msg("failed to register the handler")
	}

	newService.Init()
	return newService, nil
}

func WebfingerHandler(service svc.Service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// A WebFinger URI MUST contain a query component (see Section 3.4 of
		// RFC 3986).  The query component MUST contain a "resource" parameter
		// and MAY contain one or more "rel" parameters.
		resource := r.URL.Query().Get("resource")
		queryTarget, err := url.Parse(resource)
		if resource == "" || err != nil {
			// If the "resource" parameter is absent or malformed, the WebFinger
			// resource MUST indicate that the request is bad as per Section 10.4.1
			// of RFC 2616.
			render.Status(r, http.StatusBadRequest)
			render.PlainText(w, r, "absent or malformed 'resource' parameter")
			return
		}

		rels := r.URL.Query()["rel"]

		platform := r.URL.Query().Get("platform")

		jrd, err := service.Webfinger(ctx, queryTarget, rels, platform)
		if errors.Is(err, serviceErrors.ErrNotFound) {
			// from https://www.rfc-editor.org/rfc/rfc7033#section-4.2
			//
			// If the "resource" parameter is a value for which the server has no
			// information, the server MUST indicate that it was unable to match the
			// request as per Section 10.4.5 of RFC 2616.
			render.Status(r, http.StatusNotFound)
			render.PlainText(w, r, err.Error())
			return
		}
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.PlainText(w, r, err.Error())
			return
		}

		w.Header().Set("Content-type", "application/jrd+json")
		render.Status(r, http.StatusOK)
		render.JSON(w, r, jrd)
	}
}
