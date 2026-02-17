package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"go-micro.dev/v4/selector"
)

func TestProxyIntegration(t *testing.T) {
	var tests = []testCase{
		// Simple prefix route
		test("simple_prefix", withPolicy("opencloud", withRoutes{{
			Type:     config.PrefixRoute,
			Endpoint: "/api",
			Backend:  "http://api.example.com"},
		})).withRequest("GET", "https://example.com/api", nil).
			expectProxyTo("http://api.example.com/api"),

		// Complex prefix route, different method
		test("complex_prefix_post", withPolicy("opencloud", withRoutes{{
			Type:     config.PrefixRoute,
			Endpoint: "/api",
			Backend:  "http://api.example.com/service1/"},
		})).withRequest("POST", "https://example.com/api", nil).
			expectProxyTo("http://api.example.com/service1/api"),

		// Query route
		test("query_route", withPolicy("opencloud", withRoutes{{
			Type:     config.QueryRoute,
			Endpoint: "/api?format=json",
			Backend:  "http://backend/"},
		})).withRequest("GET", "https://example.com/api?format=json", nil).
			expectProxyTo("http://backend/api?format=json"),

		// Regex route
		test("regex_route", withPolicy("opencloud", withRoutes{{
			Type:     config.RegexRoute,
			Endpoint: `\/user\/(\d+)`,
			Backend:  "http://backend/"},
		})).withRequest("POST", "https://example.com/user/1234", nil).
			expectProxyTo("http://backend/user/1234"),

		// Multiple prefix routes 1
		test("multiple_prefix", withPolicy("opencloud", withRoutes{
			{
				Type:     config.PrefixRoute,
				Endpoint: "/api",
				Backend:  "http://api.example.com",
			},
			{
				Type:     config.PrefixRoute,
				Endpoint: "/payment",
				Backend:  "http://payment.example.com",
			},
		})).withRequest("GET", "https://example.com/payment", nil).
			expectProxyTo("http://payment.example.com/payment"),

		// Multiple prefix routes 2
		test("multiple_prefix", withPolicy("opencloud", withRoutes{
			{
				Type:     config.PrefixRoute,
				Endpoint: "/api",
				Backend:  "http://api.example.com",
			},
			{
				Type:     config.PrefixRoute,
				Endpoint: "/payment",
				Backend:  "http://payment.example.com",
			},
		})).withRequest("GET", "https://example.com/api", nil).
			expectProxyTo("http://api.example.com/api"),

		// Mixed route types
		test("mixed_types", withPolicy("opencloud", withRoutes{
			{
				Type:     config.PrefixRoute,
				Endpoint: "/api",
				Backend:  "http://api.example.com",
			},
			{
				Type:        config.RegexRoute,
				Endpoint:    `\/user\/(\d+)`,
				Backend:     "http://users.example.com",
				ApacheVHost: false,
			},
		})).withRequest("GET", "https://example.com/api", nil).
			expectProxyTo("http://api.example.com/api"),

		// Mixed route types
		test("mixed_types", withPolicy("opencloud", withRoutes{
			{
				Type:     config.PrefixRoute,
				Endpoint: "/api",
				Backend:  "http://api.example.com",
			},
			{
				Type:        config.RegexRoute,
				Endpoint:    `\/user\/(\d+)`,
				Backend:     "http://users.example.com",
				ApacheVHost: false,
			},
		})).withRequest("GET", "https://example.com/user/1234", nil).
			expectProxyTo("http://users.example.com/user/1234"),
	}

	reg := registry.GetRegistry()
	sel := selector.NewSelector(selector.Registry(reg))

	for k := range tests {
		t.Run(tests[k].id, func(t *testing.T) {
			t.Parallel()
			tc := tests[k]

			rt := router.Middleware(sel, nil, tc.conf, log.NewLogger())
			rp := newTestProxy(testConfig(tc.conf), func(req *http.Request) *http.Response {
				if got, want := req.URL.String(), tc.expect.String(); got != want {
					t.Errorf("Proxied url should be %v got %v", want, got)
				}

				if got, want := req.Method, tc.input.Method; got != want {
					t.Errorf("Proxied request method should be %v got %v", want, got)
				}

				if got, want := req.Proto, tc.input.Proto; got != want {
					t.Errorf("Proxied request proto should be %v got %v", want, got)
				}

				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(`OK`)),
					Header:     make(http.Header),
				}
			})

			rr := httptest.NewRecorder()
			rt(rp).ServeHTTP(rr, tc.input)
			rsp := rr.Result()

			if rsp.StatusCode != 200 {
				t.Errorf("Expected status 200 from proxy-response got %v", rsp.StatusCode)
			}

			resultBody, err := io.ReadAll(rsp.Body)
			if err != nil {
				t.Fatal("Error reading result body")
			}
			if err = rsp.Body.Close(); err != nil {
				t.Fatal("Error closing result body")
			}

			bodyString := string(resultBody)
			if bodyString != `OK` {
				t.Errorf("Result body of proxied response should be OK, got %v", bodyString)
			}

		})
	}
}

func newTestProxy(cfg *config.Config, fn RoundTripFunc) *MultiHostReverseProxy {
	rp, _ := NewMultiHostReverseProxy(Config(cfg))
	rp.Transport = fn
	return rp
}

type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

type withRoutes []config.Route

type testCase struct {
	id     string
	input  *http.Request
	expect *url.URL
	conf   []config.Policy
}

func test(id string, policies ...config.Policy) *testCase {
	tc := &testCase{
		id: id,
	}
	for k := range policies {
		tc.conf = append(tc.conf, policies[k])
	}

	return tc
}

func withPolicy(name string, r withRoutes) config.Policy {
	return config.Policy{Name: name, Routes: r}
}

func (tc *testCase) withRequest(method string, target string, body io.Reader) *testCase {
	tc.input = httptest.NewRequest(method, target, body)
	return tc
}

func (tc *testCase) expectProxyTo(strURL string) testCase {
	pu, err := url.Parse(strURL)
	if err != nil {
		panic(fmt.Sprintf("Error parsing %v", strURL))
	}

	tc.expect = pu
	return *tc
}

func testConfig(policy []config.Policy) *config.Config {
	return &config.Config{
		Debug:          config.Debug{},
		HTTP:           config.HTTP{},
		Policies:       policy,
		OIDC:           config.OIDC{},
		PolicySelector: nil,
	}
}
