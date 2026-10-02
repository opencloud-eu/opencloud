package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/webdav"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	spanAttrOutcome = "guest_auth.outcome"

	guestAuthOutcomeSuccess     = "success"
	guestAuthOutcomeExpired     = "expired"
	guestAuthOutcomeInvalid     = "invalid"
	guestAuthOutcomeUnavailable = "unavailable"
	guestAuthOutcomeInternal    = "internal"
)

var (
	guestLinkPathPrefixes = []string{
		"/graph/v1beta1/me/drive/sharedWithMe",
		"/dav/",
		"/remote.php/dav/",
		"/webdav/",
		"/remote.php/webdav/",
	}

	// unixEpoch is the zero time used for cookie expiration.
	unixEpoch = time.Unix(0, 0)

	// maxShareIDLength is the maximum allowed length for a share ID in InnerError.
	maxShareIDLength = 512
)

// guestLinkError represents the JSON error payload carried in Status.InnerError
// when a guest session has expired.
type guestLinkError struct {
	Type    string `json:"type"`
	Reason  string `json:"reason"`
	ShareID string `json:"share_id"`
}

// GuestLinkAuthConfig holds the guest-link authentication configuration.
type GuestLinkAuthConfig struct {
	CookieName string
}

// GuestLinkAuthenticator authenticates requests using a guest-session cookie.
type GuestLinkAuthenticator struct {
	Logger              log.Logger
	RevaGatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	Config              GuestLinkAuthConfig
	Tracer              trace.Tracer
}

// guestLinkCookieName returns the configured cookie name, defaulting to __Host-opencloud-guest.
func guestLinkCookieName(cfg GuestLinkAuthConfig) string {
	return cfg.CookieName
}

// isGuestLinkPath returns whether the request path falls within a supported guest-link prefix.
func isGuestLinkPath(path string) bool {
	for _, prefix := range guestLinkPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// readGuestCookie reads the guest-session cookie from the request.
// Returns empty string if the cookie is missing or empty.
func readGuestCookie(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", nil // Missing cookie is not an error; it means not applicable
	}
	return cookie.Value, nil
}

// clearGuestCookie returns a cookie that instructs the browser to delete the guest-session cookie.
func clearGuestCookie(name string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   "",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  unixEpoch,
		MaxAge:   -1,
	}
}

// SuppressAuthenticationChallenge prevents other authentication mechanisms from challenging guest-cookie requests.
func (m *GuestLinkAuthenticator) SuppressAuthenticationChallenge(req *http.Request) bool {
	name := guestLinkCookieName(m.Config)
	cookie, err := req.Cookie(name)
	if err != nil {
		return false
	}
	return cookie.Value != ""
}

// Authenticate implements the Authenticator interface for guest-link cookie authentication.
func (m *GuestLinkAuthenticator) Authenticate(r *http.Request) AuthenticationResult {
	_, span := m.tracer().Start(r.Context(), "guest_link_auth")
	defer span.End()

	// Check applicability: enabled, cookie present, path allowed.
	name := guestLinkCookieName(m.Config)

	rawCookie, err := readGuestCookie(r, name)
	if err != nil || rawCookie == "" {
		return NotApplicable()
	}

	if !isGuestLinkPath(r.URL.Path) {
		return NotApplicable()
	}

	// Call Reva gateway Authenticate API.
	gatewayClient, err := m.RevaGatewaySelector.Next()
	if err != nil {
		m.Logger.Error().Err(err).Str("authenticator", "guest_link").Msg("Failed to select gateway client")
		span.SetAttributes(attribute.String(spanAttrOutcome, guestAuthOutcomeUnavailable))
		return AuthenticationErrorResult(err)
	}

	authReq := &gateway.AuthenticateRequest{
		Type:         "guestlinks",
		ClientId:     "",
		ClientSecret: rawCookie,
	}

	authResp, err := gatewayClient.Authenticate(r.Context(), authReq)
	if err != nil {
		m.Logger.Error().Err(err).Str("authenticator", "guest_link").Msg("Gateway Authenticate call failed")
		span.SetAttributes(attribute.String(spanAttrOutcome, guestAuthOutcomeUnavailable))
		return AuthenticationErrorResult(err)
	}

	switch authResp.GetStatus().GetCode() {
	case rpcv1beta1.Code_CODE_OK:
		return m.handleOK(r, authResp, span)
	case rpcv1beta1.Code_CODE_UNAUTHENTICATED:
		return m.handleUnauthenticated(r, authResp, span)
	case rpcv1beta1.Code_CODE_UNAVAILABLE:
		span.SetAttributes(attribute.String(spanAttrOutcome, guestAuthOutcomeUnavailable))
		m.Logger.Debug().Str("authenticator", "guest_link").Str("message", authResp.GetStatus().GetMessage()).Msg("Guest auth unavailable")
		return AuthenticationErrorResult(errors.New(authResp.GetStatus().GetMessage()))
	default:
		span.SetAttributes(attribute.String(spanAttrOutcome, guestAuthOutcomeInternal))
		m.Logger.Error().Int32("status_code", int32(authResp.GetStatus().GetCode())).Str("authenticator", "guest_link").Msg("Unexpected guest auth status")
		return AuthenticationErrorResult(errors.New("unexpected authentication status"))
	}
}

func (m *GuestLinkAuthenticator) handleOK(r *http.Request, authResp *gateway.AuthenticateResponse, span trace.Span) AuthenticationResult {
	token := authResp.GetToken()
	user := authResp.GetUser()

	if token == "" {
		m.Logger.Error().Str("authenticator", "guest_link").Msg("Guest auth: CODE_OK but missing token")
		span.SetAttributes(attribute.String(spanAttrOutcome, guestAuthOutcomeInternal))
		return AuthenticationErrorResult(errors.New("CODE_OK but missing token"))
	}

	if user == nil {
		m.Logger.Error().Str("authenticator", "guest_link").Msg("Guest auth: CODE_OK but missing user")
		span.SetAttributes(attribute.String(spanAttrOutcome, guestAuthOutcomeInternal))
		return AuthenticationErrorResult(errors.New("CODE_OK but missing user"))
	}

	ctx := revactx.ContextSetUser(r.Context(), user)
	ctx = revactx.ContextSetToken(ctx, token)
	r = r.WithContext(ctx)
	r.Header.Set(revactx.TokenHeader, token)

	span.SetAttributes(
		attribute.String("guest_auth.outcome", guestAuthOutcomeSuccess),
		attribute.String("user.id", user.GetId().GetOpaqueId()),
	)
	span.SetAttributes(attribute.String(spanAttrOutcome, guestAuthOutcomeSuccess))

	m.Logger.Debug().Str("user.id", user.GetId().GetOpaqueId()).Str("authenticator", "guest_link").Msg("Guest auth succeeded")

	return Succeeded(r)
}

func (m *GuestLinkAuthenticator) handleUnauthenticated(r *http.Request, authResp *gateway.AuthenticateResponse, span trace.Span) AuthenticationResult {
	name := guestLinkCookieName(m.Config)

	innerError := authResp.GetStatus().GetInnerError()
	expired, shareID := m.parseInnerError(innerError)

	if expired {
		m.Logger.Debug().Str("share_id_hash", shareID).Str("authenticator", "guest_link").Msg("Guest session expired")
		span.SetAttributes(attribute.String(spanAttrOutcome, guestAuthOutcomeExpired))
		return AuthenticationResult{
			State:          AuthenticationFailed,
			Terminal:       true,
			CookiesToClear: []string{name},
			ErrorDetails: GuestSessionExpiredDetails{
				PermissionID: shareID,
				IsDAV:        webdav.IsWebdavRequest(r),
			},
		}
	}

	// Generic unauthenticated failure.
	span.SetAttributes(attribute.String(spanAttrOutcome, guestAuthOutcomeInvalid))
	m.Logger.Debug().Str("authenticator", "guest_link").Msg("Guest auth: invalid credentials")

	return AuthenticationResult{
		State:          AuthenticationFailed,
		Terminal:       true,
		CookiesToClear: []string{name},
	}
}

func (m *GuestLinkAuthenticator) parseInnerError(innerError *typesv1beta1.OpaqueEntry) (bool, string) {
	if innerError == nil {
		return false, ""
	}

	value := innerError.Value
	if len(value) == 0 {
		return false, ""
	}

	decoder := innerError.Decoder
	if decoder != "json" {
		return false, ""
	}

	var err guestLinkError
	if err := json.Unmarshal(value, &err); err != nil {
		return false, ""
	}

	if err.Type != "opencloud_guest_link_error" {
		return false, ""
	}

	if err.Reason != "session_expired" {
		return false, ""
	}

	if len(err.ShareID) == 0 || len(err.ShareID) > maxShareIDLength {
		return false, ""
	}

	return true, err.ShareID
}

func (m *GuestLinkAuthenticator) tracer() trace.Tracer {
	if m.Tracer != nil {
		return m.Tracer
	}
	return noop.NewTracerProvider().Tracer("proxy.middleware.guest_link")
}
