package middleware

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/webdav"
)

var (
	// SupportedAuthStrategies stores configured challenges.
	SupportedAuthStrategies []string

	// ProxyWwwAuthenticate is a list of endpoints that do not rely on reva underlying authentication, such as ocs.
	// services that fallback to reva authentication are declared in the "frontend" command on OpenCloud. It is a list of
	// regexp.Regexp which are safe to use concurrently.
	ProxyWwwAuthenticate = []regexp.Regexp{*regexp.MustCompile("/ocs/v[12].php/cloud/")}

	_publicPaths = [...]string{
		"/dav/public-files/",
		"/remote.php/dav/ocm/",
		"/dav/ocm/",
		"/ocm/",
		"/remote.php/dav/public-files/",
		"/ocs/v1.php/apps/files_sharing/api/v1/tokeninfo/unprotected",
		"/ocs/v2.php/apps/files_sharing/api/v1/tokeninfo/unprotected",
		"/ocs/v1.php/cloud/capabilities",
	}
)

const (
	// WwwAuthenticate captures the Www-Authenticate header string.
	WwwAuthenticate = "Www-Authenticate"
)

// AuthenticationState represents the outcome of an authentication attempt.
type AuthenticationState int

const (
	// AuthenticationNotApplicable means the authenticator does not apply to this request
	// (e.g., no relevant credentials present, or not the right path).
	AuthenticationNotApplicable AuthenticationState = iota
	// AuthenticationFailed means the authenticator found applicable credentials but
	// they were rejected (e.g., wrong password, invalid token).
	AuthenticationFailed
	// AuthenticationSucceeded means authentication was successful and the request
	// has been augmented with identity information.
	AuthenticationSucceeded
	// AuthenticationError means authentication encountered an internal/dependency failure
	// (e.g., backend unavailable, transport error).
	AuthenticationError
)

// AuthenticationResult represents the typed result of an authentication attempt.
type AuthenticationResult struct {
	Request      *http.Request
	State        AuthenticationState
	Err          error
	Terminal     bool
	ErrorDetails ErrorDetails
	// CookiesToClear lists cookie names that should be cleared on terminal failures.
	CookiesToClear []string
}

// ErrorDetails carries optional structured data for terminal failures.
type ErrorDetails interface {
	isErrorDetails()
}

// GuestSessionExpiredDetails carries data needed to render an expired-session response.
type GuestSessionExpiredDetails struct {
	PermissionID string
	IsDAV        bool
}

func (g GuestSessionExpiredDetails) isErrorDetails() {}

// NotApplicable returns a result indicating the authenticator does not apply to the request.
func NotApplicable() AuthenticationResult {
	return AuthenticationResult{State: AuthenticationNotApplicable}
}

// Failed returns a result indicating credentials were present but rejected.
func Failed() AuthenticationResult {
	return AuthenticationResult{State: AuthenticationFailed}
}

// FailedWithErr returns a result indicating credentials were present but rejected,
// with an associated error for logging/classification.
func FailedWithErr(err error) AuthenticationResult {
	return AuthenticationResult{State: AuthenticationFailed, Err: err}
}

// TerminalFailed returns a result indicating credentials were present but rejected,
// and no further authenticators should be tried.
func TerminalFailed() AuthenticationResult {
	return AuthenticationResult{State: AuthenticationFailed, Terminal: true}
}

// TerminalFailedWithErr returns a result indicating credentials were present but rejected,
// with an associated error for logging/classification, and no further authenticators should be tried.
func TerminalFailedWithErr(err error) AuthenticationResult {
	return AuthenticationResult{State: AuthenticationFailed, Err: err, Terminal: true}
}

// Succeeded returns a result indicating successful authentication with the augmented request.
func Succeeded(r *http.Request) AuthenticationResult {
	return AuthenticationResult{Request: r, State: AuthenticationSucceeded}
}

// AuthenticationErrorResult returns a result indicating an internal/dependency failure.
func AuthenticationErrorResult(err error) AuthenticationResult {
	return AuthenticationResult{State: AuthenticationError, Err: err}
}

// Authenticator is the common interface implemented by all request authenticators.
type Authenticator interface {
	// Authenticate is used to authenticate incoming HTTP requests.
	// The Authenticator returns a typed result indicating whether the request was
	// authenticated, not applicable, failed, or encountered an error.
	Authenticate(*http.Request) AuthenticationResult
}

type authenticationChallengeSuppressor interface {
	SuppressAuthenticationChallenge(*http.Request) bool
}

// Authentication is a higher order authentication middleware.
func Authentication(auths []Authenticator, opts ...Option) func(next http.Handler) http.Handler {
	options := newOptions(opts...)
	configureSupportedChallenges(options)
	tracer := getTraceProvider(options).Tracer("proxy.middleware.authentication")

	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path), spanOpts...)
			r = r.WithContext(ctx)
			defer span.End()

			ri := router.ContextRoutingInfo(ctx)
			if isOIDCTokenAuth(r) || ri.IsRouteUnprotected() || r.Method == "OPTIONS" {
				// Either this is a request that does not need any authentication or
				// the authentication for this request is handled by the IdP.
				span.SetAttributes(attribute.Bool("routeunprotected", true))
				span.End()
				next.ServeHTTP(w, r)
				return
			}

			suppressAuthenticationChallenge := false
			var terminalResult AuthenticationResult
			for _, a := range auths {
				result := a.Authenticate(r)
				if result.State == AuthenticationSucceeded {
					span.End()
					next.ServeHTTP(w, result.Request)
					return
				}
				if suppressor, ok := a.(authenticationChallengeSuppressor); ok && suppressor.SuppressAuthenticationChallenge(r) {
					suppressAuthenticationChallenge = true
				}
				if result.Terminal && result.State != AuthenticationSucceeded {
					terminalResult = result
					break
				}
			}

			// Handle terminal failures with error details (e.g., expired guest sessions).
			if terminalResult.State != AuthenticationNotApplicable {
				// Clear any cookies specified in the terminal result.
				for _, cookieName := range terminalResult.CookiesToClear {
					http.SetCookie(w, clearGuestCookie(cookieName))
				}
				if renderErr := renderTerminalFailure(w, r, terminalResult); renderErr != nil {
					options.Logger.Error().Err(renderErr).Str("authenticator", "terminal_failure_renderer").Msg("Failed to render terminal failure response")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				return
			}

			if !suppressAuthenticationChallenge && !isPublicPath(r.URL.Path) {
				// Failed basic authentication attempts receive the Www-Authenticate header in the response
				var touch bool
				caser := cases.Title(language.Und)
				for k, v := range options.CredentialsByUserAgent {
					if strings.Contains(k, r.UserAgent()) {
						removeSuperfluousAuthenticate(w)
						w.Header().Add("Www-Authenticate", fmt.Sprintf("%v realm=\"%s\", charset=\"UTF-8\"", caser.String(v), r.Host))
						touch = true
						break
					}
				}

				// if the request is not bound to any user agent, write all available challenges
				if !touch {
					writeSupportedAuthenticateHeader(w, r)
				}
			}

			if !suppressAuthenticationChallenge {
				for _, s := range SupportedAuthStrategies {
					userAgentAuthenticateLockIn(w, r, options.CredentialsByUserAgent, s)
				}
			}
			w.WriteHeader(http.StatusUnauthorized)
			// if the request is a PROPFIND return a WebDAV error code.
			// TODO: The proxy has to be smart enough to detect when a request is directed towards a webdav server
			// and react accordingly.
			if webdav.IsWebdavRequest(r) {
				b, err := webdav.Marshal(webdav.Exception{
					Code:    webdav.SabredavPermissionDenied,
					Message: "Authentication error",
				})

				webdav.HandleWebdavError(w, b, err)
			}

			if r.ProtoMajor == 1 {
				// https://github.com/owncloud/ocis/issues/5066
				// https://github.com/golang/go/blob/d5de62df152baf4de6e9fe81933319b86fd95ae4/src/net/http/server.go#L1357-L1417
				// https://github.com/golang/go/issues/15527

				defer r.Body.Close()
				_, _ = io.Copy(io.Discard, r.Body)
			}
		})
	}
}

// The token auth endpoint uses basic auth for clients, see https://openid.net/specs/openid-connect-basic-1_0.html#TokenRequest
// > The Client MUST authenticate to the Token Endpoint using the HTTP Basic method, as described in 2.3.1 of OAuth 2.0.
func isOIDCTokenAuth(req *http.Request) bool {
	return req.URL.Path == "/konnect/v1/token"
}

func isPublicPath(p string) bool {
	for _, pp := range _publicPaths {
		if strings.HasPrefix(p, pp) {
			return true
		}
	}
	return false
}

// configureSupportedChallenges adds known authentication challenges to the current session.
func configureSupportedChallenges(options Options) {
	if options.OIDCIss != "" {
		SupportedAuthStrategies = append(SupportedAuthStrategies, "bearer")
	}

	if options.EnableBasicAuth {
		SupportedAuthStrategies = append(SupportedAuthStrategies, "basic")
	}
}

func writeSupportedAuthenticateHeader(w http.ResponseWriter, r *http.Request) {
	caser := cases.Title(language.Und)
	for _, s := range SupportedAuthStrategies {
		if r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			w.Header().Add(WwwAuthenticate, fmt.Sprintf("%v realm=\"%s\", charset=\"UTF-8\"", caser.String(s), r.Host))
		}
	}
}

func removeSuperfluousAuthenticate(w http.ResponseWriter) {
	w.Header().Del(WwwAuthenticate)
}

// userAgentLocker aids in dependency injection for helper methods. The set of fields is arbitrary and the only relation
// they share is to fulfill their duty and lock a User-Agent to its correct challenge if configured.
type userAgentLocker struct {
	w        http.ResponseWriter
	r        *http.Request
	locks    map[string]string // locks represents a reva user-agent:challenge mapping.
	fallback string
}

// userAgentAuthenticateLockIn sets Www-Authenticate according to configured user agents. This is useful for the case of
// legacy clients that do not support protocols like OIDC or OAuth and want to lock a given user agent to a challenge
// such as basic. For more context check https://github.com/cs3org/reva/pull/1350
func userAgentAuthenticateLockIn(w http.ResponseWriter, r *http.Request, locks map[string]string, fallback string) {
	u := userAgentLocker{
		w:        w,
		r:        r,
		locks:    locks,
		fallback: fallback,
	}

	for _, r := range ProxyWwwAuthenticate {
		evalRequestURI(u, r)
	}
}

func evalRequestURI(l userAgentLocker, r regexp.Regexp) {
	if !r.MatchString(l.r.RequestURI) {
		return
	}
	caser := cases.Title(language.Und)
	for k, v := range l.locks {
		if strings.Contains(k, l.r.UserAgent()) {
			removeSuperfluousAuthenticate(l.w)
			l.w.Header().Add(WwwAuthenticate, fmt.Sprintf("%v realm=\"%s\", charset=\"UTF-8\"", caser.String(v), l.r.Host))
			return
		}
	}
}

func getTraceProvider(o Options) trace.TracerProvider {
	if o.TraceProvider != nil {
		return o.TraceProvider
	}
	return trace.NewNoopTracerProvider()
}

// renderTerminalFailure renders the appropriate response for a terminal authentication failure.
// Returns nil on success, or an error if rendering fails.
func renderTerminalFailure(w http.ResponseWriter, r *http.Request, result AuthenticationResult) error {
	// Check for structured error details (e.g., expired guest sessions).
	if result.ErrorDetails != nil {
		return renderErrorDetails(w, r, result.ErrorDetails)
	}

	// Default: generic 401 without challenges.
	if !webdav.IsWebdavRequest(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return nil
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	return webdav.Encode(w, webdav.Exception{
		Code:    webdav.SabredavNotAuthenticated,
		Message: "Authentication error",
	})
}

// renderErrorDetails renders a response based on structured error details.
func renderErrorDetails(w http.ResponseWriter, r *http.Request, details ErrorDetails) error {
	switch d := details.(type) {
	case GuestSessionExpiredDetails:
		if webdav.IsWebdavRequest(r) {
			return renderDAVExpired(w, d.PermissionID)
		}
		return renderJSONExpired(w, d.PermissionID)
	default:
		w.WriteHeader(http.StatusUnauthorized)
		return nil
	}
}

// renderJSONExpired writes a structured JSON response for expired guest sessions.
func renderJSONExpired(w http.ResponseWriter, permissionID string) error {
	resp := struct {
		ErrorType    string `json:"error_type"`
		Message      string `json:"message"`
		PermissionID string `json:"permissionId"`
	}{
		ErrorType:    "session_expired",
		Message:      "Your session has expired.",
		PermissionID: permissionID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	return json.NewEncoder(w).Encode(resp)
}

// renderDAVExpired writes a SabreDAV-compatible XML response for expired guest sessions.
func renderDAVExpired(w http.ResponseWriter, permissionID string) error {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	return webdav.EncodeXML(w, davExpiredResponse{
		XmlnsD:    "DAV",
		XmlnsS:    "http://sabredav.org/ns",
		Exception: "Sabre\\DAV\\Exception\\NotAuthenticated",
		Message:   "Guest session has expired.",
		Details: davExpiredDetails{
			XmlnsOpenCloud: "http://opencloud.org/ns",
			ErrorType:      "session_expired",
			ShareID:        permissionID,
		},
	})
}

type davExpiredResponse struct {
	XMLName   xml.Name          `xml:"d:error"`
	XmlnsD    string            `xml:"xmlns:d,attr"`
	XmlnsS    string            `xml:"xmlns:s,attr"`
	Exception string            `xml:"s:Exception"`
	Message   string            `xml:"s:Message"`
	Details   davExpiredDetails `xml:"opencloud:details"`
}

type davExpiredDetails struct {
	XmlnsOpenCloud string `xml:"xmlns:opencloud,attr"`
	ErrorType      string `xml:"opencloud:error_type"`
	ShareID        string `xml:"opencloud:share_id"`
}
