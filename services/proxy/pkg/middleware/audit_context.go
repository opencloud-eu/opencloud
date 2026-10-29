package middleware

import (
	"context"
	"net/http"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocmiddleware "github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/reva/v2/pkg/audit"
)

// Values of audit.RequestContext.AuthMethod, one per authenticator.
const (
	AuthMethodOIDC       = "oidc"
	AuthMethodBasic      = "basic"
	AuthMethodAppToken   = "app_token"
	AuthMethodPublicLink = "public_link"
	AuthMethodGuestLink  = "guest_link"
	AuthMethodSignedURL  = "signed_url"
)

type authMethodKey struct{}

// withAuthMethod remembers which authenticator accepted the request.
func withAuthMethod(r *http.Request, a Authenticator) *http.Request {
	m := authMethodOf(a)
	if m == "" {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), authMethodKey{}, m))
}

func authMethodFromContext(ctx context.Context) string {
	m, _ := ctx.Value(authMethodKey{}).(string)
	return m
}

// authMethoder lets an authenticator name its auth method itself.
type authMethoder interface {
	AuthMethod() string
}

func authMethodOf(a Authenticator) string {
	if m, ok := a.(authMethoder); ok {
		return m.AuthMethod()
	}
	switch a.(type) {
	case *OIDCAuthenticator:
		return AuthMethodOIDC
	case BasicAuthenticator, *BasicAuthenticator:
		return AuthMethodBasic
	case AppAuthAuthenticator, *AppAuthAuthenticator:
		return AuthMethodAppToken
	case PublicShareAuthenticator, *PublicShareAuthenticator:
		return AuthMethodPublicLink
	case *GuestLinkAuthenticator:
		return AuthMethodGuestLink
	case SignedURLAuthenticator, *SignedURLAuthenticator:
		return AuthMethodSignedURL
	default:
		return ""
	}
}

// AuditContext collects what the audit records need about the request and
// no other header carries: client address, auth method, session and OIDC
// client. It stores them in the context, which also adds them to outgoing
// gRPC calls, and sets the X-Audit-Context header for the upstream HTTP
// services. A value sent by the client is always dropped.
//
// It must run after ClientIP, so the client address is resolved, and after
// Authentication, so the auth method and the OIDC claims are known.
func AuditContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del(audit.Header)

		ctx := r.Context()
		rc := audit.RequestContext{
			ClientAddress: ocmiddleware.ClientAddress(r),
			AuthMethod:    authMethodFromContext(ctx),
		}
		if rc.AuthMethod == AuthMethodOIDC {
			claims := oidc.FromContext(ctx)
			rc.SessionID = stringClaim(claims, "sid")
			rc.ClientID = stringClaim(claims, "azp")
			if rc.ClientID == "" {
				rc.ClientID = stringClaim(claims, "client_id")
			}
		}

		if !rc.IsZero() {
			ctx = audit.ContextSetRequest(ctx, rc)
			r = r.WithContext(ctx)
			r.Header.Set(audit.Header, rc.Encode())
		}
		next.ServeHTTP(w, r)
	})
}

func stringClaim(claims map[string]any, name string) string {
	s, _ := claims[name].(string)
	return s
}

// auditLogin reports a new login session. ctx carries the user and the
// request context; app tokens and basic auth authenticate every request, so
// only new OIDC sessions count as logins.
func auditLogin(ctx context.Context, emitter audit.Emitter, user *userpb.User) {
	if emitter == nil {
		return
	}
	_ = emitter.Emit(ctx, audit.Event{
		Definition: audit.AuthLogin,
		Target:     audit.Target{ID: user.GetId().GetOpaqueId(), Name: user.GetUsername()},
	})
}

// auditLoginFailure reports rejected credentials, once per request. The
// AuditContext middleware has not run yet, so the request context is built
// here. Expired OIDC tokens are part of the normal token renewal and public
// link access is audited on its own, both are left out.
func auditLoginFailure(r *http.Request, emitter audit.Emitter, a Authenticator, res AuthenticationResult) {
	if emitter == nil {
		return
	}
	method := authMethodOf(a)
	reason := "invalid_credentials"
	actor := &audit.Actor{Type: audit.ActorAnonymous}
	switch method {
	case "", AuthMethodPublicLink:
		return
	case AuthMethodOIDC:
		if res.Err != nil && isExpectedTokenTimeError(res.Err) {
			return
		}
		reason = "invalid_token"
	case AuthMethodSignedURL:
		reason = "invalid_signature"
	case AuthMethodGuestLink:
		actor.Type = audit.ActorGuest
	}

	target := audit.Target{ID: "unknown"}
	// the login name someone tried, what a SIEM needs to spot password guessing
	if login, _, ok := r.BasicAuth(); ok && login != "" {
		actor = &audit.Actor{Type: audit.ActorUser, Name: login}
		target = audit.Target{ID: login, Name: login}
	}

	ctx := audit.ContextSetRequest(r.Context(), audit.RequestContext{ClientAddress: ocmiddleware.ClientAddress(r), AuthMethod: method})
	_ = emitter.Emit(ctx, audit.Event{
		Definition: audit.AuthLogin,
		Outcome:    audit.OutcomeFailure,
		Reason:     reason,
		Actor:      actor,
		Target:     target,
	})
}
