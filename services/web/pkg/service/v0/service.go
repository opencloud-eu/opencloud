package svc

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"

	"github.com/opencloud-eu/opencloud/pkg/account"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	consoleWebService "github.com/opencloud-eu/opencloud/services/console/pkg/web"
	"github.com/opencloud-eu/opencloud/services/web/pkg/assets"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
	"github.com/opencloud-eu/opencloud/services/web/pkg/theme"
)

// Service defines the service handlers.
type Service interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	Config(w http.ResponseWriter, r *http.Request)
}

// NewService returns a service implementation for Service.
func NewService(opts ...Option) (Service, error) {
	options := newOptions(opts...)

	m := chi.NewMux()
	m.Use(options.Middleware...)

	m.Use(
		otelchi.Middleware(
			"web",
			otelchi.WithChiRoutes(m),
			otelchi.WithTracerProvider(options.TraceProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	themeService, err := theme.NewService(
		theme.ServiceOptions{}.
			WithThemeFS(options.ThemeFS),
	)
	if err != nil {
		return Web{}, err
	}

	themeAPI, err := theme.NewHTTP(
		theme.HTTPOptions{}.
			WithService(themeService).
			WithLogger(options.Logger),
	)
	if err != nil {
		return Web{}, err
	}

	svc := Web{
		logger:       options.Logger,
		config:       options.Config,
		mux:          m,
		themeService: themeService,
	}

	m.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Get("/config.json", svc.Config)
		r.Route("/branding/logo", func(r chi.Router) {
			r.Use(middleware.ExtractAccountUUID(
				account.Logger(options.Logger),
				account.JWTSecret(options.Config.TokenManager.JWTSecret),
			))
		})
		r.Route("/themes", func(r chi.Router) {
			r.Get("/{id}/theme.json", themeAPI.Get)
			r.Mount("/", svc.Static(
				options.ThemeFS.IOFS(),
				path.Join(svc.config.HTTP.Root, "/themes"),
				options.Config.HTTP.CacheTTL,
			))
		})
		r.Mount(options.AppsHTTPEndpoint, svc.Static(
			options.AppFS,
			path.Join(svc.config.HTTP.Root, options.AppsHTTPEndpoint),
			options.Config.HTTP.CacheTTL,
		))
		r.Mount("/", svc.Static(
			options.CoreFS,
			svc.config.HTTP.Root,
			options.Config.HTTP.CacheTTL,
		))
	})
	_ = chi.Walk(m, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return svc, nil
}

// Web defines the handlers for the web service.
type Web struct {
	logger       log.Logger
	config       *config.Config
	mux          *chi.Mux
	themeService *theme.Service
}

// ServeHTTP implements the Service interface.
func (p Web) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mux.ServeHTTP(w, r)
}

// Config handles HTTP requests to provide the current configuration as a JSON response.
func (p Web) Config(w http.ResponseWriter, _ *http.Request) {
	// decouple theme-related config changes
	conf := *p.config
	// check if the console theme exists and apply it
	if conf.Web.ThemeServer == conf.Web.Config.Server && p.themeService.Exists(consoleWebService.ThemeID) {
		conf.Web.ThemePath = path.Join("themes", consoleWebService.ThemeID, "theme.json")
	}

	// build theme url
	if themeServer, err := url.Parse(conf.Web.ThemeServer); err == nil {
		themeServer.Path = conf.Web.ThemePath
		conf.Web.Config.Theme = themeServer.String()
	} else {
		conf.Web.Config.Theme = conf.Web.ThemePath
	}

	// make apps render as an empty array if it is empty
	// TODO remove once https://github.com/golang/go/issues/27589 is fixed
	if len(conf.Web.Config.Apps) == 0 {
		conf.Web.Config.Apps = make([]string, 0)
	}

	payload, err := json.Marshal(conf.Web.Config)
	if err != nil {
		msg := "Invalid or missing config"
		p.logger.Error().Err(err).Msg(msg)
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(payload); err != nil {
		p.logger.Error().Err(err).Msg("could not write config response")
	}
}

// Static simply serves all static files.
func (p Web) Static(f fs.FS, root string, ttl int) http.HandlerFunc {
	rootWithSlash := root

	if !strings.HasSuffix(rootWithSlash, "/") {
		rootWithSlash = rootWithSlash + "/"
	}

	static := http.StripPrefix(
		rootWithSlash,
		assets.FileServer(f),
	)

	lastModified := time.Now().UTC().Format(http.TimeFormat)
	expires := time.Now().Add(time.Second * time.Duration(ttl)).UTC().Format(http.TimeFormat)

	return func(w http.ResponseWriter, r *http.Request) {
		if rootWithSlash != "/" && r.URL.Path == p.config.HTTP.Root {
			http.Redirect(
				w,
				r,
				rootWithSlash,
				http.StatusMovedPermanently,
			)
			return
		}

		w.Header().Set("Cache-Control", "max-age="+strconv.Itoa(ttl))
		w.Header().Set("Expires", expires)
		w.Header().Set("Last-Modified", lastModified)
		w.Header().Set("SameSite", "Strict")

		static.ServeHTTP(w, r)
	}
}
