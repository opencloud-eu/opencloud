package relations

import (
	"context"

	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/service/v0"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/webfinger"
)

const (
	OpenIDConnectRel        = "http://openid.net/specs/connect/1.0/issuer"
	OpenIDConnectDesktopRel = "http://openid.net/specs/connect/1.0/issuer/desktop"
)

type openIDDiscovery struct {
	Href string
}

// OpenIDDiscovery adds the Openid Connect issuer relation
func OpenIDDiscovery(href string) service.RelationProvider {
	return &openIDDiscovery{
		Href: href,
	}
}

func (l *openIDDiscovery) Add(_ context.Context, jrd *webfinger.JSONResourceDescriptor) {
	if jrd == nil {
		jrd = &webfinger.JSONResourceDescriptor{}
	}
	jrd.Links = append(jrd.Links, webfinger.Link{
		Rel:  OpenIDConnectRel,
		Href: l.Href,
	})
}

type openIDDiscoveryDesktop struct {
	Href string
}

// OpenIDDiscoveryDesktop adds the OpenID Connect issuer relation for desktop clients.
// This allows identity providers that require separate OIDC clients per application type
// (like Authentik, Kanidm, Zitadel) to provide a distinct issuer URL for desktop clients.
// See: https://github.com/opencloud-eu/desktop/issues/246
func OpenIDDiscoveryDesktop(href string) service.RelationProvider {
	return &openIDDiscoveryDesktop{
		Href: href,
	}
}

func (l *openIDDiscoveryDesktop) Add(_ context.Context, jrd *webfinger.JSONResourceDescriptor) {
	if jrd == nil {
		jrd = &webfinger.JSONResourceDescriptor{}
	}
	jrd.Links = append(jrd.Links, webfinger.Link{
		Rel:  OpenIDConnectDesktopRel,
		Href: l.Href,
	})
}
