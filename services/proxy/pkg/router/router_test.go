package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config/defaults"
	"go-micro.dev/v4/selector"
)

type matchertest struct {
	method, endpoint, target string
	unprotected              bool
	matches                  bool
}

func TestPrefixRouteMatcher(t *testing.T) {
	cfg := defaults.DefaultConfig()
	cfg.Policies = defaults.DefaultPolicies()

	table := []matchertest{
		{endpoint: "/foobar", target: "/foobar/baz/some/url", matches: true},
		{endpoint: "/fobar", target: "/foobar/baz/some/url", matches: false},
	}

	for _, test := range table {
		u, _ := url.Parse(test.target)
		matched := prefixRouteMatcher(test.endpoint, *u)
		if matched != test.matches {
			t.Errorf("PrefixRouteMatcher returned %t expected %t for endpoint: %s and target %s",
				matched, test.matches, test.endpoint, u.String())
		}
	}
}

func TestQueryRouteMatcher(t *testing.T) {
	cfg := defaults.DefaultConfig()
	cfg.Policies = defaults.DefaultPolicies()

	table := []matchertest{
		{endpoint: "/foobar?parameter=true", target: "/foobar/baz/some/url?parameter=true", matches: true},
		{endpoint: "/foobar", target: "/foobar/baz/some/url?parameter=true", matches: false},
		{endpoint: "/foobar?parameter=false", target: "/foobar/baz/some/url?parameter=true", matches: false},
		{endpoint: "/foobar?parameter=false&other=true", target: "/foobar/baz/some/url?parameter=true", matches: false},
		{
			endpoint: "/foobar?parameter=false&other=true",
			target:   "/foobar/baz/some/url?parameter=false&other=true",
			matches:  true,
		},
		{endpoint: "/fobar", target: "/foobar", matches: false},
	}

	for _, test := range table {
		u, _ := url.Parse(test.target)
		matched := queryRouteMatcher(test.endpoint, *u)
		if matched != test.matches {
			t.Errorf("QueryRouteMatcher returned %t expected %t for endpoint: %s and target %s",
				matched, test.matches, test.endpoint, u.String())
		}
	}
}

func TestRegexRouteMatcher(t *testing.T) {
	cfg := defaults.DefaultConfig()
	cfg.Policies = defaults.DefaultPolicies()
	reg := registry.GetRegistry()
	sel := selector.NewSelector(selector.Registry(reg))
	rt := New(sel, cfg.PolicySelector, cfg.Policies, log.NewLogger())

	table := []matchertest{
		{endpoint: ".*some\\/url.*parameter=true", target: "/foobar/baz/some/url?parameter=true", matches: true},
		{endpoint: "([\\])\\w+", target: "/foobar/baz/some/url?parameter=true", matches: false},
	}

	for _, test := range table {
		u, _ := url.Parse(test.target)
		matched := rt.regexRouteMatcher(test.endpoint, *u)
		if matched != test.matches {
			t.Errorf("RegexRouteMatcher returned %t expected %t for endpoint: %s and target %s",
				matched, test.matches, test.endpoint, u.String())
		}
	}
}

func TestSingleJoiningSlash(t *testing.T) {
	type test struct {
		a, b, result string
	}

	table := []test{
		{a: "a", b: "b", result: "a/b"},
		{a: "a/", b: "b", result: "a/b"},
		{a: "a", b: "/b", result: "a/b"},
		{a: "a/", b: "/b", result: "a/b"},
	}

	for _, test := range table {
		p := singleJoiningSlash(test.a, test.b)
		if p != test.result {
			t.Errorf("SingleJoiningSlash got %s expected %s", p, test.result)
		}
	}
}

func TestRouter(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	}))
	defer svr.Close()

	policySelectorCfg := &config.PolicySelector{
		Static: &config.StaticSelectorConf{
			Policy: "default",
		},
	}

	policies := []config.Policy{
		{
			Name: "default",
			Routes: []config.Route{
				{Type: config.PrefixRoute, Endpoint: "/web/unprotected/demo/", Backend: "http://web", Unprotected: true},
				{Type: config.PrefixRoute, Endpoint: "/dav", Backend: "http://frontend"},
				{Type: config.PrefixRoute, Method: "REPORT", Endpoint: "/dav", Backend: "http://opencloud-webdav"},
			},
		},
	}

	reg := registry.GetRegistry()
	sel := selector.NewSelector(selector.Registry(reg))
	router := New(sel, policySelectorCfg, policies, log.NewLogger())

	table := []matchertest{
		{method: "PROPFIND", endpoint: "/dav/files/demo/", target: "frontend"},
		{method: "REPORT", endpoint: "/dav/files/demo/", target: "opencloud-webdav"},
		{method: "GET", endpoint: "/web/unprotected/demo/", target: "web", unprotected: true},
	}

	for _, test := range table {
		r := httptest.NewRequest(test.method, test.endpoint, nil)
		routingInfo, ok := router.Route(r)
		if !ok {
			t.Errorf("TestRouter router.Route failed to route the request.")
		}

		if routingInfo.IsRouteUnprotected() != test.unprotected {
			t.Errorf("TestRouter route flag unprotected expected to be %t got %t", test.unprotected, routingInfo.IsRouteUnprotected())
		}

		pr := &httputil.ProxyRequest{
			In:  r,
			Out: r.Clone(context.Background()),
		}
		routingInfo.Rewrite()(pr)

		if pr.Out.URL.Host != test.target {
			t.Errorf("TestRouter got host %s expected %s", pr.Out.URL.Host, test.target)
		}
	}
}
