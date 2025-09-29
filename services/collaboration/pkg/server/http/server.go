package http

import (
	"fmt"
	stdhttp "net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/opencloud-eu/opencloud/pkg/account"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/service/http"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/version"
	colabmiddleware "github.com/opencloud-eu/opencloud/services/collaboration/pkg/middleware"
	"github.com/riandyrn/otelchi"
	"go-micro.dev/v4"
)

// Server initializes the http service and server.
func Server(opts ...Option) (http.Service, error) {
	options := newOptions(opts...)

	service, err := http.NewService(
		http.TLSConfig(options.Config.HTTP.TLS),
		http.Logger(options.Logger),
		http.Namespace(options.Config.HTTP.Namespace),
		http.Name(options.Config.Service.Name),
		http.Version(version.GetString()),
		http.Address(options.Config.HTTP.Addr),
		http.Context(options.Context),
		http.TraceProvider(options.TracerProvider),
	)
	if err != nil {
		options.Logger.Error().
			Err(err).
			Msg("Error initializing http service")
		return http.Service{}, fmt.Errorf("could not initialize http service: %w", err)
	}

	middlewares := []func(stdhttp.Handler) stdhttp.Handler{
		chimiddleware.RequestID,
		middleware.Version(
			options.Config.Service.Name,
			version.GetString(),
		),
		colabmiddleware.AccessLog(
			options.Logger,
		),
		middleware.ExtractAccountUUID(
			account.Logger(options.Logger),
			account.JWTSecret(options.Config.TokenManager.JWTSecret),
		),
		/*
			// Need CORS? not in the original server
			// Also, CORS isn't part of the config right now
			middleware.Cors(
				cors.Logger(options.Logger),
				cors.AllowedOrigins(options.Config.HTTP.CORS.AllowedOrigins),
				cors.AllowedMethods(options.Config.HTTP.CORS.AllowedMethods),
				cors.AllowedHeaders(options.Config.HTTP.CORS.AllowedHeaders),
				cors.AllowCredentials(options.Config.HTTP.CORS.AllowCredentials),
			),
		*/
	}

	mux := chi.NewMux()
	mux.Use(middlewares...)

	mux.Use(
		otelchi.Middleware(
			options.Config.Service.Name,
			otelchi.WithChiRoutes(mux),
			otelchi.WithTracerProvider(options.TracerProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
			otelchi.WithRequestMethodInSpanName(true),
		),
	)

	prepareRoutes(mux, options)

	// in debug mode print out the actual routes
	_ = chi.Walk(mux, func(method string, route string, handler stdhttp.Handler, middlewares ...func(stdhttp.Handler) stdhttp.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	if err := micro.RegisterHandler(service.Server(), mux); err != nil {
		return http.Service{}, err
	}

	return service, nil
}

// prepareRoutes will prepare all the implemented routes
func prepareRoutes(r *chi.Mux, options Options) {
	adapter := options.Adapter
	logger := options.Logger
	// prepare basic logger for the request
	r.Use(func(h stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			ctx := logger.With().
				Str(log.RequestIDString, r.Header.Get("X-Request-ID")).
				Str("proto", r.Proto).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Logger().WithContext(r.Context())
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Route("/wopi", func(r chi.Router) {

		r.Get("/", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			stdhttp.Error(w, stdhttp.StatusText(stdhttp.StatusTeapot), stdhttp.StatusTeapot)
		})

		r.Route("/files/{fileid}", func(r chi.Router) {

			r.Use(
				func(h stdhttp.Handler) stdhttp.Handler {
					// authentication and wopi context
					return colabmiddleware.WopiContextAuthMiddleware(options.Config, options.Store, h)
				},
				colabmiddleware.CollaborationTracingMiddleware,
			)

			// check whether we should check for proof keys
			if !options.Config.App.ProofKeys.Disable {
				r.Use(func(h stdhttp.Handler) stdhttp.Handler {
					return colabmiddleware.ProofKeysMiddleware(options.Config, h)
				})
			}

			r.Get("/", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				adapter.CheckFileInfo(w, r)
			})

			r.Post("/", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				action := r.Header.Get("X-WOPI-Override")
				switch action {

				case "LOCK":
					// "UnlockAndRelock" operation goes through here
					adapter.Lock(w, r)
				case "GET_LOCK":
					adapter.GetLock(w, r)
				case "REFRESH_LOCK":
					adapter.RefreshLock(w, r)
				case "UNLOCK":
					adapter.UnLock(w, r)

				case "PUT_USER_INFO":
					// https://docs.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/putuserinfo
					stdhttp.Error(w, stdhttp.StatusText(stdhttp.StatusNotImplemented), stdhttp.StatusNotImplemented)
				case "PUT_RELATIVE":
					adapter.PutRelativeFile(w, r)
				case "RENAME_FILE":
					adapter.RenameFile(w, r)
				case "DELETE":
					adapter.DeleteFile(w, r)

				default:
					stdhttp.Error(w, stdhttp.StatusText(stdhttp.StatusInternalServerError), stdhttp.StatusInternalServerError)
				}
			})

			r.Route("/contents", func(r chi.Router) {
				r.Get("/", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
					adapter.GetFile(w, r)
				})

				r.Post("/", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
					action := r.Header.Get("X-WOPI-Override")
					switch action {

					case "PUT":
						adapter.PutFile(w, r)

					default:
						stdhttp.Error(w, stdhttp.StatusText(stdhttp.StatusInternalServerError), stdhttp.StatusInternalServerError)
					}
				})
			})
		})
		r.Route("/templates/{templateID}", func(r chi.Router) {
			r.Use(
				func(h stdhttp.Handler) stdhttp.Handler {
					// authentication and wopi context
					return colabmiddleware.WopiContextAuthMiddleware(options.Config, options.Store, h)
				},
				colabmiddleware.CollaborationTracingMiddleware,
			)
			r.Get("/", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
				adapter.GetFile(w, r)
			})
		})
	})
}
