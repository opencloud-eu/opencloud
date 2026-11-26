package audit

import (
	"context"
	"net/url"
	"strings"

	ctxpkg "github.com/opencloud-eu/reva/v3/pkg/ctx"

	"google.golang.org/grpc/metadata"
)

// Header carries the request context between services, as HTTP header and as
// gRPC metadata. Everything else the audit record needs (user, tenant,
// initiator, trace) already travels in its own header.
const Header = "x-audit-context"

// Short keys keep the header small, it is sent with every call.
const (
	keyClientAddress = "ip"
	keySessionID     = "sid"
	keyAuthMethod    = "am"
	keyClientID      = "cid"
	keyUserAgent     = "ua"
)

// RequestContext is what the proxy knows about a request and the audit
// record needs, but no other header carries.
type RequestContext struct {
	// ClientAddress is the real client address, resolved behind trusted proxies.
	ClientAddress string
	// SessionID identifies the login session.
	SessionID string
	// AuthMethod is how the actor authenticated, e.g. oidc, app_token, public_link, basic.
	AuthMethod string
	// ClientID is the OIDC client.
	ClientID string
	// UserAgent is the user agent of the client. The x-user-agent metadata
	// is not set by every service, and the gRPC user-agent is the one of
	// the calling service, not of the client.
	UserAgent string
}

// IsZero reports whether no field is set.
func (rc RequestContext) IsZero() bool {
	return rc == RequestContext{}
}

// Merge fills the identity fields rc lacks from initial, the request context
// of an earlier request of the same operation. A tus upload is created by an
// authenticated request, its data is sent with a transfer token, which has no
// login session or auth method. Address and user agent stay the ones of rc.
func (rc RequestContext) Merge(initial RequestContext) RequestContext {
	fill := func(v *string, from string) {
		if *v == "" {
			*v = from
		}
	}
	fill(&rc.SessionID, initial.SessionID)
	fill(&rc.AuthMethod, initial.AuthMethod)
	fill(&rc.ClientID, initial.ClientID)
	return rc
}

// PublicShareID returns the public link a request is authorized by, if any.
func PublicShareID(ctx context.Context) string {
	scopes, ok := ctxpkg.ContextGetScopes(ctx)
	if !ok {
		return ""
	}
	for k := range scopes {
		if id, found := strings.CutPrefix(k, "publicshare:"); found {
			return id
		}
	}
	return ""
}

// Encode returns the header value.
func (rc RequestContext) Encode() string {
	v := url.Values{}
	set := func(k, val string) {
		if val != "" {
			v.Set(k, val)
		}
	}
	set(keyClientAddress, rc.ClientAddress)
	set(keySessionID, rc.SessionID)
	set(keyAuthMethod, rc.AuthMethod)
	set(keyClientID, rc.ClientID)
	set(keyUserAgent, rc.UserAgent)
	return v.Encode()
}

// ParseRequestContext reads a header value. Unknown keys are ignored, a
// broken value returns an empty context.
func ParseRequestContext(s string) RequestContext {
	v, err := url.ParseQuery(s)
	if err != nil {
		return RequestContext{}
	}
	return RequestContext{
		ClientAddress: v.Get(keyClientAddress),
		SessionID:     v.Get(keySessionID),
		AuthMethod:    v.Get(keyAuthMethod),
		ClientID:      v.Get(keyClientID),
		UserAgent:     v.Get(keyUserAgent),
	}
}

type requestContextKey struct{}

// ContextSetRequest stores the request context in ctx and adds it to the
// outgoing gRPC metadata, so it reaches the next service.
func ContextSetRequest(ctx context.Context, rc RequestContext) context.Context {
	if rc.IsZero() {
		return ctx
	}
	ctx = context.WithValue(ctx, requestContextKey{}, rc)
	return metadata.AppendToOutgoingContext(ctx, Header, rc.Encode())
}

// ContextGetRequest returns the request context from ctx, falling back to the
// incoming gRPC metadata.
func ContextGetRequest(ctx context.Context) (RequestContext, bool) {
	if rc, ok := ctx.Value(requestContextKey{}).(RequestContext); ok {
		return rc, true
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if lst := md.Get(Header); len(lst) > 0 {
			rc := ParseRequestContext(lst[0])
			return rc, !rc.IsZero()
		}
	}
	return RequestContext{}, false
}
