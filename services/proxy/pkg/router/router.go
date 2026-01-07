package router

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/proxy/policy"
	"go-micro.dev/v4/selector"
)

type routingInfoCtxKey struct{}

var noInfo = RoutingInfo{}

// Middleware returns a HTTP middleware containing the router.
func Middleware(serviceSelector selector.Selector, policySelectorCfg *config.PolicySelector, policies []config.Policy, logger log.Logger) func(http.Handler) http.Handler {
	router := New(serviceSelector, policySelectorCfg, policies, logger)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ri, ok := router.Route(r)
			if !ok {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			next.ServeHTTP(w, r.WithContext(SetRoutingInfo(r.Context(), ri)))
		})
	}
}

// New creates a new request router.
// It initializes the routes before returning the router.
func New(serviceSelector selector.Selector, policySelectorCfg *config.PolicySelector, policies []config.Policy, logger log.Logger) Router {
	if policySelectorCfg == nil {
		firstPolicy := policies[0].Name
		logger.Warn().Str("policy", firstPolicy).Msg("policy-selector not configured. Will always use first policy")
		policySelectorCfg = &config.PolicySelector{
			Static: &config.StaticSelectorConf{
				Policy: firstPolicy,
			},
		}
	}

	logger.Debug().
		Interface("selector_config", policySelectorCfg).
		Msg("loading policy-selector")

	policySelector, err := policy.LoadSelector(policySelectorCfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Could not load policy-selector")
	}

	r := Router{
		logger:          logger,
		rewriters:       make(map[string]map[config.RouteType]map[string][]RoutingInfo),
		policySelector:  policySelector,
		serviceSelector: serviceSelector,
	}
	for _, pol := range policies {
		for _, route := range pol.Routes {
			logger.Debug().Str("fwd: ", route.Endpoint)

			if route.Backend == "" && route.Service == "" {
				logger.Fatal().Interface("route", route).Msg("neither Backend nor Service is set")
			}
			uri, err2 := url.Parse(route.Backend)
			if err2 != nil {
				logger.
					Fatal(). // fail early on misconfiguration
					Err(err2).
					Str("backend", route.Backend).
					Msg("malformed url")
			}

			// here the backend is used as a uri
			r.addHost(pol.Name, uri, route)
		}
	}
	return r
}

// RoutingInfo contains the proxy rewrite hook and some information about the route.
type RoutingInfo struct {
	rewrite          func(*httputil.ProxyRequest)
	endpoint         string
	unprotected      bool
	remoteUserHeader string
	skipXAccessToken bool
}

// Rewrite returns the proxy rewrite hook.
func (r RoutingInfo) Rewrite() func(*httputil.ProxyRequest) {
	return r.rewrite
}

// IsRouteUnprotected returns true if the route doesn't need to be authenticated.
func (r RoutingInfo) IsRouteUnprotected() bool {
	return r.unprotected
}

// RemoteUserHeader returns the name of Header for setting the remote user value
func (r RoutingInfo) RemoteUserHeader() string {
	return r.remoteUserHeader
}

// SkipXAccessToken return true if the reva access token should not be added to the
// outgoing request
func (r RoutingInfo) SkipXAccessToken() bool {
	return r.skipXAccessToken
}

// Router handles the routing of HTTP requests according to the given policies.
type Router struct {
	logger          log.Logger
	rewriters       map[string]map[config.RouteType]map[string][]RoutingInfo
	policySelector  policy.Selector
	serviceSelector selector.Selector
}

func (rt Router) addHost(policy string, target *url.URL, route config.Route) {
	targetQuery := target.RawQuery
	if rt.rewriters[policy] == nil {
		rt.rewriters[policy] = make(map[config.RouteType]map[string][]RoutingInfo)
	}
	routeType := config.DefaultRouteType
	if route.Type != "" {
		routeType = route.Type
	}
	if rt.rewriters[policy][routeType] == nil {
		rt.rewriters[policy][routeType] = make(map[string][]RoutingInfo)
	}
	if rt.rewriters[policy][routeType][route.Method] == nil {
		rt.rewriters[policy][routeType][route.Method] = make([]RoutingInfo, 0)
	}

	rt.rewriters[policy][routeType][route.Method] = append(rt.rewriters[policy][routeType][route.Method], RoutingInfo{
		endpoint:         route.Endpoint,
		unprotected:      route.Unprotected,
		remoteUserHeader: route.RemoteUserHeader,
		skipXAccessToken: route.SkipXAccessToken,
		rewrite: func(req *httputil.ProxyRequest) {
			if route.Service != "" {
				// select next node
				next, err := rt.serviceSelector.Select(route.Service)
				if err != nil {
					rt.logger.Error().Err(err).
						Str("service", route.Service).
						Msg("could not select service from the registry")
					return // TODO error? fallback to target.Host & Scheme?
				}
				node, err := next()
				if err != nil {
					rt.logger.Error().Err(err).
						Str("service", route.Service).
						Msg("could not select next node")
					return // TODO error? fallback to target.Host & Scheme?
				}
				req.Out.URL.Host = node.Address
				req.Out.URL.Scheme = node.Metadata["protocol"] // TODO check property exists?
				if node.Metadata["use_tls"] == "true" {
					req.Out.URL.Scheme = "https"
				}
			} else {
				req.Out.URL.Host = target.Host
				req.Out.URL.Scheme = target.Scheme
			}

			// Apache deployments host addresses need to match on req.Out.Host and req.Out.URL.Host
			// see https://stackoverflow.com/questions/34745654/golang-reverseproxy-with-apache2-sni-hostname-error
			if route.ApacheVHost {
				req.Out.Host = target.Host
			}

			for k, v := range route.AdditionalHeaders {
				req.Out.Header.Set(k, v)
			}

			req.Out.URL.Path = singleJoiningSlash(target.Path, req.Out.URL.Path)
			if targetQuery == "" || req.Out.URL.RawQuery == "" {
				req.Out.URL.RawQuery = targetQuery + req.Out.URL.RawQuery
			} else {
				req.Out.URL.RawQuery = targetQuery + "&" + req.Out.URL.RawQuery
			}
			if _, ok := req.Out.Header["User-Agent"]; !ok {
				// explicitly disable User-Agent so it's not set to default value
				req.Out.Header.Set("User-Agent", "")
			}
			req.SetXForwarded()
		},
	})
}

// Route is evaluating the policies on the request and returns the RoutingInfo if successful.
func (rt Router) Route(r *http.Request) (RoutingInfo, bool) {
	pol, err := rt.policySelector(r)
	if err != nil {
		rt.logger.Error().Err(err).Msg("Error while selecting pol")
		return noInfo, false
	}

	if _, ok := rt.rewriters[pol]; !ok {
		rt.logger.
			Error().
			Str("policy", pol).
			Msg("policy is not configured")
		return noInfo, false
	}

	method := ""
	// find matching rewrite hook
	for _, rtype := range config.RouteTypes {
		var handler func(string, url.URL) bool
		switch rtype {
		case config.QueryRoute:
			handler = queryRouteMatcher
		case config.RegexRoute:
			handler = rt.regexRouteMatcher
		case config.PrefixRoute:
			fallthrough
		default:
			handler = prefixRouteMatcher
		}
		if rt.rewriters[pol][rtype][r.Method] != nil {
			// use specific method
			method = r.Method
		} else {
			method = ""
		}

		for _, ri := range rt.rewriters[pol][rtype][method] {
			if handler(ri.endpoint, *r.URL) {
				rt.logger.Debug().
					Str("policy", pol).
					Str("method", r.Method).
					Str("prefix", ri.endpoint).
					Str("path", r.URL.Path).
					Str("routeType", string(rtype)).
					Msg("rewrite hook found")

				return ri, true
			}
		}
	}

	// override default rewrite hook with root. If any
	if ri := rt.rewriters[pol][config.PrefixRoute][method][0]; ri.endpoint == "/" { // try specific method
		return ri, true
	} else if ri := rt.rewriters[pol][config.PrefixRoute][""][0]; ri.endpoint == "/" { // fallback to unspecific method
		return ri, true
	}

	rt.logger.
		Warn().
		Str("policy", pol).
		Str("path", r.URL.Path).
		Msg("no rewrite hook found")
	return noInfo, false
}

func (rt Router) regexRouteMatcher(pattern string, target url.URL) bool {
	matched, err := regexp.MatchString(pattern, target.String())
	if err != nil {
		rt.logger.Warn().Err(err).Str("pattern", pattern).Msg("regex with pattern failed")
	}
	return matched
}

func prefixRouteMatcher(prefix string, target url.URL) bool {
	cleanTarget := path.Clean(target.Path)
	if strings.HasSuffix(target.Path, "/") {
		cleanTarget += "/"
	}
	return strings.HasPrefix(cleanTarget, prefix) && prefix != "/"
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func queryRouteMatcher(endpoint string, target url.URL) bool {
	u, _ := url.Parse(endpoint)
	if !strings.HasPrefix(path.Clean(target.Path), u.Path) || endpoint == "/" {
		return false
	}
	q := u.Query()
	if len(q) == 0 {
		return false
	}
	tq := target.Query()
	for k := range q {
		if q.Get(k) != tq.Get(k) {
			return false
		}
	}
	return true
}

// SetRoutingInfo puts the routing info in the context.
func SetRoutingInfo(parent context.Context, ri RoutingInfo) context.Context {
	return context.WithValue(parent, routingInfoCtxKey{}, ri)
}

// ContextRoutingInfo gets the routing information from the context.
func ContextRoutingInfo(ctx context.Context) RoutingInfo {
	val := ctx.Value(routingInfoCtxKey{})
	return val.(RoutingInfo)
}
