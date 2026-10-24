package svc

import (
	"context"
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
	"github.com/opencloud-eu/opencloud/services/web/pkg/announcement"
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
		logger:            options.Logger,
		config:            options.Config,
		mux:               m,
		gatewaySelector:   options.GatewaySelector,
		announcementStore: options.AnnouncementStore,
	}

	themeService, err := theme.NewService(
		theme.ServiceOptions{}.
			WithThemeFS(options.ThemeFS).
			WithGatewaySelector(options.GatewaySelector),
	)
	if err != nil {
		return svc, err
	}

	announcementService, err := announcement.NewService(
		announcement.ServiceOptions{}.
			WithLogger(options.Logger).
			WithStore(options.AnnouncementStore).
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
		r.Route("/announcement", func(r chi.Router) {
			r.Use(middleware.ExtractAccountUUID(
				account.Logger(options.Logger),
				account.JWTSecret(options.Config.TokenManager.JWTSecret),
			))
			r.Get("/", announcementService.Get)
			r.Put("/", announcementService.Set)
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
	logger            log.Logger
	config            *config.Config
	mux               *chi.Mux
	gatewaySelector   pool.Selectable[gateway.GatewayAPIClient]
	announcementStore *announcement.Store
}

// ServeHTTP implements the Service interface.
func (p Web) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mux.ServeHTTP(w, r)
}

func (p Web) getPayload(ctx context.Context) (payload []byte, err error) {
	// render dynamically using a copy of the config, so per-request values (e.g. the
	// announcement) are not written into the shared config concurrently.
	webConfig := p.config.Web.Config

	// build theme url
	if themeServer, err := url.Parse(p.config.Web.ThemeServer); err == nil {
		webConfig.Theme = themeServer.String() + p.config.Web.ThemePath
	} else {
		webConfig.Theme = p.config.Web.ThemePath
	}

	// make apps render as empty array if it is empty
	// TODO remove once https://github.com/golang/go/issues/27589 is fixed
	if len(webConfig.Apps) == 0 {
		webConfig.Apps = make([]string, 0)
	}

	// ensure that the server url has a trailing slash
	webConfig.Server = strings.TrimRight(webConfig.Server, "/") + "/"

	// the runtime store is the single source of truth for the announcement banner: expose it
	// when live, clear it otherwise. A statically configured value is not supported.
	webConfig.Options.Announcement = p.currentAnnouncement(ctx)

	return json.Marshal(webConfig)
}

// currentAnnouncement returns the stored announcement for config.json, or nil if unset, disabled
// or unavailable.
func (p Web) currentAnnouncement(ctx context.Context) *config.Announcement {
	if p.announcementStore == nil {
		return nil
	}

	a, err := p.announcementStore.Get(ctx)
	if err != nil {
		p.logger.Error().Err(err).Msg("could not read announcement from store")
		return nil
	}

	// only live (enabled) announcements with a banner line are exposed in the public config.json
	if !a.Enabled || a.BannerText == "" {
		return nil
	}

	return &config.Announcement{BannerText: a.BannerText, InfoText: a.InfoText}
}

// Config implements the Service interface.
func (p Web) Config(w http.ResponseWriter, r *http.Request) {
	payload, err := p.getPayload(r.Context())
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
