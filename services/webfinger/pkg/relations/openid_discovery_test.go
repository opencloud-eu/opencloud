package relations

import (
	"context"
	"testing"

	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/config"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/webfinger"
)

func TestOpenidDiscovery(t *testing.T) {
	clients := map[string]config.OIDCClientConfig{
		"web": {
			ClientID: "web",
			Scopes:   []string{"openid", "profile", "email"},
		},
		"test": {
			ClientID: "test",
			Scopes:   []string{"test"},
		},
	}

	provider := OpenIDDiscovery("http://issuer.url", clients)

	jrd := webfinger.JSONResourceDescriptor{}

	provider.Add(context.Background(), "", &jrd)

	if len(jrd.Links) != 1 {
		t.Errorf("provider returned wrong number of links: %v, expected 1", len(jrd.Links))
	}
	if jrd.Links[0].Href != "http://issuer.url" {
		t.Errorf("provider returned wrong issuer link href: %v, expected %v", jrd.Links[0].Href, "http://issuer.url")
	}
	if jrd.Links[0].Rel != "http://openid.net/specs/connect/1.0/issuer" {
		t.Errorf("provider returned wrong openid connect rel: %v, expected %v", jrd.Links[0].Href, OpenIDConnectRel)
	}
	if len(jrd.Properties) != 0 {
		t.Errorf("provider returned properties for empty platform: %v, expected 0", len(jrd.Properties))
	}

	jrd = webfinger.JSONResourceDescriptor{}
	provider.Add(context.Background(), "test", &jrd)
	if len(jrd.Properties) != 2 {
		t.Errorf("provider returned wrong number of properties for platform test: %v, expected 2", len(jrd.Properties))
	}
	if jrd.Properties["http://opencloud.eu/ns/oidc/client_id"] != "test" {
		t.Errorf("provider returned wrong client_id property: %v, expected %v", jrd.Properties["http://opencloud.eu/ns/oidc/client_id"], "test")
	}
	if scopes, ok := jrd.Properties["http://opencloud.eu/ns/oidc/scopes"].([]string); !ok || len(scopes) != 1 || scopes[0] != "test" {
		t.Errorf("provider returned wrong scopes property: %v, expected %v", jrd.Properties["http://opencloud.eu/ns/oidc/scopes"], []string{"test"})
	}
}
