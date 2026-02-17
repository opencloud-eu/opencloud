package relations

import (
	"context"

	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/config"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/service/v0"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/webfinger"
)

const (
	OpenIDConnectRel = "http://openid.net/specs/connect/1.0/issuer"
	clientIDProp     = "http://opencloud.eu/ns/oidc/client_id"
	scopesProp       = "http://opencloud.eu/ns/oidc/scopes"
)

type openIDDiscovery struct {
	Href        string
	OIDCClients map[string]config.OIDCClientConfig
}

// OpenIDDiscovery adds the Openid Connect issuer relation
func OpenIDDiscovery(href string, clients map[string]config.OIDCClientConfig) service.RelationProvider {
	return &openIDDiscovery{
		Href:        href,
		OIDCClients: clients,
	}
}

func (l *openIDDiscovery) Add(_ context.Context, platform string, jrd *webfinger.JSONResourceDescriptor) {
	if jrd == nil {
		jrd = &webfinger.JSONResourceDescriptor{}
	}
	jrd.Links = append(jrd.Links, webfinger.Link{
		Rel:  OpenIDConnectRel,
		Href: l.Href,
	})

	if platform != "" {
		if clientConfig, ok := l.OIDCClients[platform]; ok {
			jrd.Properties = make(map[string]any)
			jrd.Properties[clientIDProp] = clientConfig.ClientID
			jrd.Properties[scopesProp] = clientConfig.Scopes
		}
	}
}
