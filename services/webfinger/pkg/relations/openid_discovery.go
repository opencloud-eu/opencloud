package relations

import (
	"context"

	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/service/v0"
	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/webfinger"
)

const (
	OpenIDConnectRel        = "http://openid.net/specs/connect/1.0/issuer"
	OpenIDConnectDesktopRel = "http://openid.net/specs/connect/1.0/issuer/desktop"
	OpenIDConnectMobileRel  = "http://openid.net/specs/connect/1.0/issuer/mobile"
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

// ClientIDProperty is the property URI for the OIDC client ID
const ClientIDProperty = "http://openid.net/specs/connect/1.0/client_id"

type openIDDiscoveryDesktop struct {
	Href     string
	ClientID string
}

// OpenIDDiscoveryDesktop adds the OpenID Connect issuer relation for desktop clients.
// This allows identity providers that require separate OIDC clients per application type
// (like Authentik, Kanidm, Zitadel) to provide a distinct issuer URL for desktop clients.
// If clientID is provided, it will be included as a property in the link.
// See: https://github.com/opencloud-eu/desktop/issues/246
func OpenIDDiscoveryDesktop(href string, clientID string) service.RelationProvider {
	return &openIDDiscoveryDesktop{
		Href:     href,
		ClientID: clientID,
	}
}

func (l *openIDDiscoveryDesktop) Add(_ context.Context, jrd *webfinger.JSONResourceDescriptor) {
	if jrd == nil {
		jrd = &webfinger.JSONResourceDescriptor{}
	}
	link := webfinger.Link{
		Rel:  OpenIDConnectDesktopRel,
		Href: l.Href,
	}
	if l.ClientID != "" {
		link.Properties = map[string]string{
			ClientIDProperty: l.ClientID,
		}
	}
	jrd.Links = append(jrd.Links, link)
}

type openIDDiscoveryMobile struct {
	Href     string
	ClientID string
}

// OpenIDDiscoveryMobile adds the OpenID Connect issuer relation for mobile clients.
// This allows identity providers that require separate OIDC clients per application type
// (like Authentik, Kanidm, Zitadel) to provide a distinct issuer URL for mobile clients.
// If clientID is provided, it will be included as a property in the link.
// See: https://github.com/opencloud-eu/desktop/issues/246
func OpenIDDiscoveryMobile(href string, clientID string) service.RelationProvider {
	return &openIDDiscoveryMobile{
		Href:     href,
		ClientID: clientID,
	}
}

func (l *openIDDiscoveryMobile) Add(_ context.Context, jrd *webfinger.JSONResourceDescriptor) {
	if jrd == nil {
		jrd = &webfinger.JSONResourceDescriptor{}
	}
	link := webfinger.Link{
		Rel:  OpenIDConnectMobileRel,
		Href: l.Href,
	}
	if l.ClientID != "" {
		link.Properties = map[string]string{
			ClientIDProperty: l.ClientID,
		}
	}
	jrd.Links = append(jrd.Links, link)
}
