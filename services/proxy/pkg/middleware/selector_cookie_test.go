package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	revactx "github.com/opencloud-eu/reva/v3/pkg/ctx"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/proxy/policy"
)

func TestSelectorCookie(t *testing.T) {
	withClaims := func(ctx context.Context) context.Context {
		return oidc.NewContext(ctx, map[string]any{oidc.OpenCloudRoutingPolicy: "claims-policy"})
	}
	withClaimsAndUser := func(ctx context.Context) context.Context {
		ctx = revactx.ContextSetUser(ctx, &userpb.User{Username: "alice"})
		return oidc.NewContext(ctx, map[string]any{})
	}

	tests := []struct {
		name       string
		selector   config.PolicySelector
		ctx        func(context.Context) context.Context
		wantCookie bool
		wantName   string
		wantValue  string
	}{
		{
			name: "claims selector with a custom cookie name",
			selector: config.PolicySelector{Claims: &config.ClaimsSelectorConf{
				DefaultPolicy:      "default-policy",
				SelectorCookieName: "custom-selector",
			}},
			ctx:        withClaims,
			wantCookie: true,
			wantName:   "custom-selector",
			wantValue:  "claims-policy",
		},
		{
			name: "claims selector with the default cookie name",
			selector: config.PolicySelector{Claims: &config.ClaimsSelectorConf{
				DefaultPolicy: "default-policy",
			}},
			ctx:        withClaims,
			wantCookie: true,
			wantName:   policy.SelectorCookieName,
			wantValue:  "claims-policy",
		},
		{
			name: "regex selector",
			selector: config.PolicySelector{Regex: &config.RegexSelectorConf{
				DefaultPolicy: "default-policy",
				MatchesPolicies: []config.RegexRuleConf{
					{Priority: 10, Property: "username", Match: "^alice$", Policy: "regex-policy"},
				},
				SelectorCookieName: "custom-selector",
			}},
			ctx:        withClaimsAndUser,
			wantCookie: true,
			wantName:   "custom-selector",
			wantValue:  "regex-policy",
		},
		{
			name: "claims selector without oidc claims",
			selector: config.PolicySelector{Claims: &config.ClaimsSelectorConf{
				DefaultPolicy:      "default-policy",
				SelectorCookieName: "custom-selector",
			}},
			ctx:        func(ctx context.Context) context.Context { return ctx },
			wantCookie: false,
		},
		{
			name:       "static selector",
			selector:   config.PolicySelector{Static: &config.StaticSelectorConf{Policy: "default"}},
			ctx:        withClaims,
			wantCookie: false,
		},
		{
			name:       "no selector configured",
			selector:   config.PolicySelector{},
			ctx:        withClaims,
			wantCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.selector.Static != nil || tt.selector.Claims != nil || tt.selector.Regex != nil {
				// The proxy loads the selector once at startup, which fills in the
				// default cookie name. Do the same here.
				if _, err := policy.LoadSelector(&tt.selector); err != nil {
					t.Fatalf("loading the selector: %v", err)
				}
			}

			nextCalled := false
			handler := middleware.SelectorCookie(
				middleware.Logger(log.NopLogger()),
				middleware.PolicySelectorConfig(tt.selector),
			)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				nextCalled = true
			}))

			req := httptest.NewRequest(http.MethodGet, "https://cloud.example.com/", nil)
			req = req.WithContext(tt.ctx(req.Context()))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if !nextCalled {
				t.Error("the next handler was not called")
			}

			cookies := rec.Result().Cookies()
			if !tt.wantCookie {
				if len(cookies) != 0 {
					t.Fatalf("expected no cookie, got %v", cookies)
				}
				return
			}
			if len(cookies) != 1 {
				t.Fatalf("expected exactly one cookie, got %d: %v", len(cookies), cookies)
			}

			c := cookies[0]
			if c.Name != tt.wantName {
				t.Errorf("name: expected %q, got %q", tt.wantName, c.Name)
			}
			if c.Value != tt.wantValue {
				t.Errorf("value: expected %q, got %q", tt.wantValue, c.Value)
			}
			if c.Path != "/" {
				t.Errorf("path: expected %q, got %q", "/", c.Path)
			}
			if !c.Secure {
				t.Error("the cookie must only be sent over HTTPS (Secure)")
			}
			if !c.HttpOnly {
				t.Error("the cookie must not be readable by scripts (HttpOnly)")
			}
			if c.SameSite != http.SameSiteStrictMode {
				t.Errorf("the cookie must not be sent on cross-site requests: expected SameSite=Strict, got %v", c.SameSite)
			}
		})
	}
}
