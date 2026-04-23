package policy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

func TestLoadSelector(t *testing.T) {
	type test struct {
		cfg         *config.PolicySelector
		expectedErr error
	}
	sCfg := &config.StaticSelectorConf{Policy: "reva"}
	ccfg := &config.ClaimsSelectorConf{}
	rcfg := &config.RegexSelectorConf{}

	table := []test{
		{cfg: &config.PolicySelector{Static: sCfg, Claims: ccfg, Regex: rcfg}, expectedErr: ErrMultipleSelectors},
		{cfg: &config.PolicySelector{}, expectedErr: ErrSelectorConfigIncomplete},
		{cfg: &config.PolicySelector{Static: sCfg}, expectedErr: nil},
		{cfg: &config.PolicySelector{Claims: ccfg}, expectedErr: nil},
		{cfg: &config.PolicySelector{Regex: rcfg}, expectedErr: nil},
	}

	for _, test := range table {
		_, err := LoadSelector(test.cfg)
		if err != test.expectedErr {
			t.Errorf("Unexpected error %v", err)
		}
	}
}

func TestStaticSelector(t *testing.T) {
	sel := NewStaticSelector(&config.StaticSelectorConf{Policy: "opencloud"})
	req := httptest.NewRequest("GET", "https://example.org/foo", nil)
	want := "opencloud"
	got, err := sel(req)
	if got != want {
		t.Errorf("Expected policy %v got %v", want, got)
	}

	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	sel = NewStaticSelector(&config.StaticSelectorConf{Policy: "foo"})

	want = "foo"
	got, err = sel(req)
	if got != want {
		t.Errorf("Expected policy %v got %v", want, got)
	}

	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
}

type testCase struct {
	Name     string
	Context  context.Context
	Cookie   *http.Cookie
	Expected string
}

func TestClaimsSelector(t *testing.T) {
	sel := NewClaimsSelector(&config.ClaimsSelectorConf{
		DefaultPolicy:         "default",
		UnauthenticatedPolicy: "unauthenticated",
		SelectorCookieName:    SelectorCookieName,
	})

	var tests = []testCase{
		{"unauthenticated", context.Background(), nil, "unauthenticated"},
		{"default", oidc.NewContext(context.Background(), map[string]any{oidc.OpenCloudRoutingPolicy: ""}), nil, "default"},
		{"claim-value", oidc.NewContext(context.Background(), map[string]any{oidc.OpenCloudRoutingPolicy: "opencloud.routing.policy-value"}), nil, "opencloud.routing.policy-value"},
		{"cookie-only", context.Background(), &http.Cookie{Name: SelectorCookieName, Value: "cookie"}, "cookie"},
		{"claim-can-override-cookie", oidc.NewContext(context.Background(), map[string]any{oidc.OpenCloudRoutingPolicy: "opencloud.routing.policy-value"}), &http.Cookie{Name: SelectorCookieName, Value: "cookie"}, "opencloud.routing.policy-value"},
	}
	for _, tc := range tests {
		r := httptest.NewRequest("GET", "https://example.com", nil)
		if tc.Cookie != nil {
			r.AddCookie(tc.Cookie)
		}
		nr := r.WithContext(tc.Context)
		got, err := sel(nr)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if got != tc.Expected {
			t.Errorf("Expected Policy %v got %v", tc.Expected, got)
		}
	}
}

func TestRegexSelector(t *testing.T) {
	sel := NewRegexSelector(&config.RegexSelectorConf{
		DefaultPolicy: "default",
		MatchesPolicies: []config.RegexRuleConf{
			{Priority: 10, Property: "mail", Match: "mary@example.org", Policy: "opencloud"},
			{Priority: 20, Property: "mail", Match: "[^@]+@example.org", Policy: "oc10"},
			{Priority: 30, Property: "username", Match: "(alan|feynman)", Policy: "opencloud"},
			{Priority: 40, Property: "username", Match: ".+", Policy: "oc10"},
			{Priority: 50, Property: "id", Match: "b1f74ec4-dd7e-11ef-a543-03775734d0f7", Policy: "opencloud"},
			{Priority: 60, Property: "id", Match: "056fc874-dd7f-11ef-ba84-af6fca4b7289", Policy: "oc10"},
		},
		UnauthenticatedPolicy: "unauthenticated",
	})

	var tests = []testCase{
		{"unauthenticated", context.Background(), nil, "unauthenticated"},
		{"default", revactx.ContextSetUser(context.Background(), &userv1beta1.User{}), nil, "default"},
		{"mail-opencloud", revactx.ContextSetUser(context.Background(), &userv1beta1.User{Mail: "mary@example.org"}), nil, "opencloud"},
		{"mail-oc10", revactx.ContextSetUser(context.Background(), &userv1beta1.User{Mail: "alan@example.org"}), nil, "oc10"},
		{"username-alan", revactx.ContextSetUser(context.Background(), &userv1beta1.User{Username: "alan"}), nil, "opencloud"},
		{"username-feynman", revactx.ContextSetUser(context.Background(), &userv1beta1.User{Username: "feynman"}), nil, "opencloud"},
		{"username-mary", revactx.ContextSetUser(context.Background(), &userv1beta1.User{Username: "mary"}), nil, "oc10"},
		{"id-nil", revactx.ContextSetUser(context.Background(), &userv1beta1.User{Id: &userv1beta1.UserId{}}), nil, "default"},
		{"id-1", revactx.ContextSetUser(context.Background(), &userv1beta1.User{Id: &userv1beta1.UserId{OpaqueId: "b1f74ec4-dd7e-11ef-a543-03775734d0f7"}}), nil, "opencloud"},
		{"id-2", revactx.ContextSetUser(context.Background(), &userv1beta1.User{Id: &userv1beta1.UserId{OpaqueId: "056fc874-dd7f-11ef-ba84-af6fca4b7289"}}), nil, "oc10"},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "https://example.com", nil)
			nr := r.WithContext(tc.Context)
			got, err := sel(nr)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if got != tc.Expected {
				t.Errorf("Expected Policy %v got %v", tc.Expected, got)
			}
		})
	}
}
