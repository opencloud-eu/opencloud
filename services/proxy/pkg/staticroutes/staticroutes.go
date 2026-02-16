package staticroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	microstore "go-micro.dev/v4/store"
)

// StaticRouteHandler defines a Route Handler for static routes
type StaticRouteHandler struct {
	Prefix          string
	Proxy           http.Handler
	UserInfoCache   microstore.Store
	Logger          log.Logger
	Config          config.Config
	OidcClient      oidc.OIDCClient
	OidcHttpClient  *http.Client
	EventsPublisher events.Publisher
	UserProvider    backend.UserBackend
}

type jse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (s *StaticRouteHandler) Handler() http.Handler {
	m := chi.NewMux()
	m.Route(s.Prefix, func(r chi.Router) {

		// Wrapper for backchannel logout
		r.Post("/backchannel_logout", s.backchannelLogout)

		// openid .well-known
		if s.Config.OIDC.RewriteWellKnown {
			r.Get("/.well-known/openid-configuration", s.oIDCWellKnownRewrite(s.Config.OIDC.Issuer))
		}

		// Send all requests to the proxy handler
		r.HandleFunc("/*", s.Proxy.ServeHTTP)
	})

	// Also send requests for methods unknown to chi to the proxy handler as well
	m.MethodNotAllowed(s.Proxy.ServeHTTP)

	return m
}
