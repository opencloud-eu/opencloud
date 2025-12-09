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

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/riandyrn/otelchi"

	"github.com/opencloud-eu/opencloud/pkg/account"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/web/pkg/assets"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
	"github.com/opencloud-eu/opencloud/services/web/pkg/theme"
)

// ErrConfigInvalid is returned when the config parse is invalid.
var ErrConfigInvalid = `Invalid or missing config`

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

	svc := Web{
		logger:          options.Logger,
		config:          options.Config,
		mux:             m,
		coreFS:          options.CoreFS,
		themeFS:         options.ThemeFS,
		gatewaySelector: options.GatewaySelector,
	}

	themeService, err := theme.NewService(
		theme.ServiceOptions{}.
			WithThemeFS(options.ThemeFS).
			WithGatewaySelector(options.GatewaySelector),
	)
	if err != nil {
		return svc, err
	}

	m.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Get("/config.json", svc.Config)
		r.Route("/branding/logo", func(r chi.Router) {
			r.Use(middleware.ExtractAccountUUID(
				account.Logger(options.Logger),
				account.JWTSecret(options.Config.TokenManager.JWTSecret),
			))
			r.Post("/", themeService.LogoUpload)
			r.Delete("/", themeService.LogoReset)
		})
		r.Route("/themes", func(r chi.Router) {
			r.Get("/{id}/theme.json", themeService.Get)
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
			svc.coreFS,
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
	logger          log.Logger
	config          *config.Config
	mux             *chi.Mux
	coreFS          fs.FS
	themeFS         *fsx.FallbackFS
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
}

// ServeHTTP implements the Service interface.
func (p Web) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mux.ServeHTTP(w, r)
}

func (p Web) getPayload() (payload []byte, err error) {
	// render dynamically using config

	// build theme url
	if themeServer, err := url.Parse(p.config.Web.ThemeServer); err == nil {
		p.config.Web.Config.Theme = themeServer.String() + p.config.Web.ThemePath
	} else {
		p.config.Web.Config.Theme = p.config.Web.ThemePath
	}

	// make apps render as empty array if it is empty
	// TODO remove once https://github.com/golang/go/issues/27589 is fixed
	if len(p.config.Web.Config.Apps) == 0 {
		p.config.Web.Config.Apps = make([]string, 0)
	}

	// ensure that the server url has a trailing slash
	p.config.Web.Config.Server = strings.TrimRight(p.config.Web.Config.Server, "/") + "/"

	return json.Marshal(p.config.Web.Config)
}

// Config implements the Service interface.
func (p Web) Config(w http.ResponseWriter, _ *http.Request) {
	payload, err := p.getPayload()
	if err != nil {
		http.Error(w, ErrConfigInvalid, http.StatusUnprocessableEntity)
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
