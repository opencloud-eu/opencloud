package middleware

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Context key for passing the datagateway target URL to the proxy
type ctxKey string

const DatagatewaySkipRoutingKey ctxKey = "datagateway_skip_routing"

const TokenTransportHeader = "X-Reva-Transfer"

// transferClaims matches the structure expected by Reva's storage provider
type transferClaims struct {
	jwt.RegisteredClaims
	Target string `json:"target"`
}

// DataGatewayMiddleware handles both routing of /data requests and rewriting TUS Location headers.
type DataGatewayMiddleware struct {
	Prefix  string
	Secret  string
	Timeout time.Duration
}

func NewDataGatewayMiddleware(prefix, secret string, timeout time.Duration) *DataGatewayMiddleware {
	return &DataGatewayMiddleware{
		Prefix:  prefix,
		Secret:  secret,
		Timeout: timeout,
	}
}

// Handler returns the middleware handler function.
func (m *DataGatewayMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// --- REQUEST PHASE: Handle /data requests ---
		// Not only the TUS PATCH requests by clients need to be rewritten.
		// Services like antivirus make GET requests to check the file content
		// and a DELETE reuquest is used to cancel an upload.
		// All of these requests need to be routed through the datagateway.
		isDataGatewayRequest := strings.HasPrefix(r.URL.Path, m.Prefix)

		if isDataGatewayRequest {
			token := r.Header.Get(TokenTransportHeader)
			if token == "" {
				token = path.Base(r.URL.Path)
			}

			j, err := jwt.ParseWithClaims(token, &transferClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(m.Secret), nil
			})
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			claims, ok := j.Claims.(*transferClaims)
			if !ok || !j.Valid {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			targetURL, err := url.Parse(claims.Target)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			r.URL.Scheme = targetURL.Scheme
			r.URL.Host = targetURL.Host
			r.URL.Path = targetURL.Path
			r.Host = targetURL.Host
			r.Header.Del(TokenTransportHeader)

			ctx := context.WithValue(r.Context(), DatagatewaySkipRoutingKey, true)
			ctx, cancel := context.WithTimeout(ctx, m.Timeout)
			defer cancel()

			r = r.WithContext(ctx)
		}

		// --- RESPONSE PHASE: Intercept TUS responses ---
		rw := &responseWriter{ResponseWriter: w}

		next.ServeHTTP(rw, r)

		// Rewrite Location headers if the response is a
		// TUS initiation response (201 Created + Tus-Resumable header in request) to a POST request
		isTusPostResponse := r.Method == http.MethodPost && rw.statusCode == 201 && r.Header.Get("Tus-Resumable") == "1.0.0"
		if isTusPostResponse {
			loc := rw.Header().Get("Location")
			newLoc := rewriteInternalDataURL(loc, getExternalBaseURL(r), m.Secret)
			if newLoc != loc {
				rw.Header().Set("Location", newLoc)
			}
		}

		rw.flush()
	})
}

// responseWriter captures the status code and allows header modification before flush
type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
	flushed     bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
	}
}

// flush actually sends the buffered status code (and thus the headers).
func (rw *responseWriter) flush() {
	if rw.wroteHeader && !rw.flushed {
		rw.flushed = true
		rw.ResponseWriter.WriteHeader(rw.statusCode)
	}
}

// Write ensures headers are flushed before body bytes go out.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	rw.flush()
	return rw.ResponseWriter.Write(b)
}

// Flush satisfies http.Flusher so the reverse proxy can stream properly.
func (rw *responseWriter) Flush() {
	rw.flush()
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the real ResponseWriter
// for optional interfaces (Flusher, Hijacker, etc.) that we don't proxy ourselves.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// getExternalBaseURL constructs the external URL (scheme + host) from the request.
func getExternalBaseURL(r *http.Request) string {
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = strings.ToLower(proto)
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// rewriteInternalDataURL converts internal URLs to signed external gateway paths.
func rewriteInternalDataURL(loc string, gatewayBaseURL string, secret string) string {
	u, err := url.Parse(loc)
	if err != nil || !strings.HasPrefix(u.Path, "/data/") {
		return loc // Only intercept Reva data endpoints
	}

	token := signURL(loc, secret)
	extU, _ := url.Parse(gatewayBaseURL)
	u.Scheme = extU.Scheme
	u.Host = extU.Host
	u.Path = path.Join(u.Path, token)
	return u.String()
}

// signURL creates a JWT containing the target internal URL
func signURL(target string, secret string) string {
	claims := &transferClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24h TTL for resumable uploads
		},
		Target: target,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "" // Should not happen with valid secret
	}
	return signedToken
}
