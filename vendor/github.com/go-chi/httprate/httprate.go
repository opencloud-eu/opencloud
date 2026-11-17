package httprate

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// LimitBy is the canonical entry point for rate-limiting by an explicit key.
//
// It is shorthand for Limit with keyFn installed as the rate-limit key. The
// key is a required positional argument, so every call site has to state, on
// purpose, what it rate-limits by.
//
// To rate-limit by a trusted client IP behind a proxy, resolve the IP with one
// of chi's middleware.ClientIPFrom* middlewares (chi v5.3.0+) and read it back
// in the KeyFunc; CanonicalizeIP buckets IPv6 clients by their /64:
//
//	r.Use(middleware.ClientIPFromXFF("10.0.0.0/8"))
//	r.Use(httprate.LimitBy(100, time.Minute, func(r *http.Request) (string, error) {
//		return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
//	}))
//
// Use JoinKeys to rate-limit by more than one dimension at once:
//
//	r.Use(httprate.LimitBy(100, time.Minute,
//		httprate.JoinKeys(clientIPKey, httprate.KeyByEndpoint)))
func LimitBy(requestLimit int, windowLength time.Duration, keyFn KeyFunc, options ...Option) func(next http.Handler) http.Handler {
	return NewRateLimiter(requestLimit, windowLength, append([]Option{WithKeyFuncs(keyFn)}, options...)...).Handler
}

type KeyFunc func(r *http.Request) (string, error)
type Option func(rl *RateLimiter)

// Set custom response headers. If empty, the header is omitted.
type ResponseHeaders struct {
	Limit      string // Default: X-RateLimit-Limit
	Remaining  string // Default: X-RateLimit-Remaining
	Increment  string // Default: X-RateLimit-Increment
	Reset      string // Default: X-RateLimit-Reset
	RetryAfter string // Default: Retry-After
}

func Key(key string) func(r *http.Request) (string, error) {
	return func(r *http.Request) (string, error) {
		return key, nil
	}
}

// CanonicalizeIP normalizes a client IP string for use as a rate-limit key:
//
//   - IPv4 addresses are returned unchanged. IPv4 mapped IPv6 addresses use
//     their dotted IPv4 form.
//   - IPv6 addresses are reduced to their /64 prefix. An IPv6 client typically
//     controls a whole /64 (2^64 addresses via SLAAC), so keying on the full
//     address would let it rotate within its own /64 to win a fresh bucket per
//     request and bypass a per-IP limit. Widen/narrow the prefix yourself if your
//     clients are delegated a larger block (e.g. a /56 or /48).
//   - Any other string, including "", is returned unchanged.
//
// httprate stays router-agnostic, so it does not resolve the client IP for you —
// pair CanonicalizeIP with whatever does. With chi's middleware.ClientIPFrom*
// (chi v5.3.0+) and middleware.GetClientIP:
//
//	r.Use(middleware.ClientIPFromXFF("10.0.0.0/8"))
//	r.Use(httprate.LimitBy(100, time.Minute, func(r *http.Request) (string, error) {
//		return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
//	}))
//
// WARNING: if the resolver returns "" (e.g. no ClientIPFrom* middleware is
// installed upstream), CanonicalizeIP returns "" and every request shares a
// single global rate-limit bucket. Strictly more restrictive, but a footgun —
// make sure the client IP is actually resolved upstream.
func CanonicalizeIP(ip string) string {
	isIPv6 := false
	// This is how net.ParseIP decides if an address is IPv6
	// https://cs.opensource.google/go/go/+/refs/tags/go1.17.7:src/net/ip.go;l=704
	for i := 0; !isIPv6 && i < len(ip); i++ {
		switch ip[i] {
		case '.':
			// IPv4
			return ip
		case ':':
			// IPv6
			isIPv6 = true
		}
	}
	if !isIPv6 {
		// Not an IP address at all
		return ip
	}

	ipv6 := net.ParseIP(ip)
	if ipv6 == nil {
		return ip
	}
	if ipv4 := ipv6.To4(); ipv4 != nil {
		return ipv4.String()
	}

	return ipv6.Mask(net.CIDRMask(64, 128)).String()
}

func KeyByEndpoint(r *http.Request) (string, error) {
	return r.URL.Path, nil
}

func WithKeyFuncs(keyFuncs ...KeyFunc) Option {
	return func(rl *RateLimiter) {
		if len(keyFuncs) > 0 {
			rl.keyFn = JoinKeys(keyFuncs...)
		}
	}
}

func WithLimitHandler(h http.HandlerFunc) Option {
	return func(rl *RateLimiter) {
		rl.onRateLimited = h
	}
}

func WithErrorHandler(h func(http.ResponseWriter, *http.Request, error)) Option {
	return func(rl *RateLimiter) {
		rl.onError = h
	}
}

func WithLimitCounter(c LimitCounter) Option {
	return func(rl *RateLimiter) {
		rl.limitCounter = c
	}
}

func WithResponseHeaders(headers ResponseHeaders) Option {
	return func(rl *RateLimiter) {
		rl.headers = headers
	}
}

func WithNoop() Option {
	return func(rl *RateLimiter) {}
}

// JoinKeys joins the results of several KeyFuncs into a single key with ":"
// separators, so they can be passed to LimitBy's positional key slot for
// multi-dimensional rate-limiting:
//
//	r.Use(httprate.LimitBy(100, time.Minute,
//		httprate.JoinKeys(clientIPKey, httprate.KeyByEndpoint)))
//
// where clientIPKey is your own client-IP KeyFunc (see LimitBy and
// CanonicalizeIP). It is the positional-argument equivalent of WithKeyFuncs. If any component
// KeyFunc returns an error, the joined key returns that error.
func JoinKeys(fns ...KeyFunc) KeyFunc {
	return func(r *http.Request) (string, error) {
		var key strings.Builder
		for i := 0; i < len(fns); i++ {
			k, err := fns[i](r)
			if err != nil {
				return "", err
			}
			key.WriteString(k)
			key.WriteRune(':')
		}
		return key.String(), nil
	}
}
