package httprate

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// This file collects the deprecated public surface in one place. These are kept
// for backward compatibility and still compile/behave as before, but new code
// should migrate to the replacements called out in each doc comment. httprate
// has not reached a stable v1, so a future major release may remove them.

// Deprecated: Use LimitBy(requestLimit, windowLength, keyFn, options...) instead,
// which makes the rate-limit key an explicit, required argument rather than an
// optional WithKeyFuncs. Pass the key function directly (e.g. a KeyFunc that
// reads a trusted client IP — see CanonicalizeIP), or httprate.Key("*") for a
// single global bucket. The remaining options
// (WithLimitCounter, WithLimitHandler, WithResponseHeaders, ...) carry over
// unchanged as LimitBy's trailing variadic.
func Limit(requestLimit int, windowLength time.Duration, options ...Option) func(next http.Handler) http.Handler {
	// Key("*") is the default key; any WithKeyFuncs in options overrides it.
	return LimitBy(requestLimit, windowLength, Key("*"), options...)
}

// Deprecated: Use LimitBy(requestLimit, windowLength, Key("*")) instead — a
// single global rate-limit bucket keyed by a constant. (LimitAll already keys
// every request by "*" under the hood; this just makes that explicit.)
func LimitAll(requestLimit int, windowLength time.Duration) func(next http.Handler) http.Handler {
	return LimitBy(requestLimit, windowLength, Key("*"))
}

// Deprecated: LimitByIP keys off r.RemoteAddr (see KeyByIP). It is not
// spoofable, but behind a reverse proxy, load balancer, or CDN r.RemoteAddr is
// the proxy's address, so every client sharing that proxy lands in one bucket —
// usually the wrong key in production. State your trust model explicitly:
// install one of chi's middleware.ClientIPFrom* middlewares (chi v5.3.0+) and
// key off the resolved IP (CanonicalizeIP buckets IPv6 by /64):
//
//	// Directly exposed to clients (LimitByIP's exact behavior, made explicit):
//	r.Use(middleware.ClientIPFromRemoteAddr)
//	r.Use(httprate.LimitBy(requestLimit, windowLength, func(r *http.Request) (string, error) {
//		return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
//	}))
func LimitByIP(requestLimit int, windowLength time.Duration) func(next http.Handler) http.Handler {
	return LimitBy(requestLimit, windowLength, KeyByIP)
}

// Deprecated: LimitByRealIP is built on the spoofable KeyByRealIP and lets a
// remote attacker forge the rate-limit key — see GHSA-9g5q-2w5x-hmxf,
// GHSA-rjr7-jggh-pgcp, GHSA-3fxj-6jh8-hvhx for the equivalent flaw in chi's
// middleware.RealIP. Install one of chi's middleware.ClientIPFrom* middlewares
// (chi v5.3.0+) and key off the resolved IP instead (CanonicalizeIP buckets
// IPv6 by /64):
//
//	r.Use(middleware.ClientIPFromXFF("10.0.0.0/8"))
//	r.Use(httprate.LimitBy(requestLimit, windowLength, func(r *http.Request) (string, error) {
//		return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
//	}))
func LimitByRealIP(requestLimit int, windowLength time.Duration) func(next http.Handler) http.Handler {
	return LimitBy(requestLimit, windowLength, KeyByRealIP)
}

// Deprecated: KeyByRealIP trusts the client-supplied True-Client-IP,
// X-Real-IP, and X-Forwarded-For headers without verifying any proxy chain, so
// a remote attacker can forge the rate-limit key — see GHSA-9g5q-2w5x-hmxf,
// GHSA-rjr7-jggh-pgcp, GHSA-3fxj-6jh8-hvhx for the equivalent flaw in chi's
// middleware.RealIP. On a rate-limiter this is two-sided: an attacker can evade
// the limit by rotating the spoofed header (unbounded buckets) or lock a victim
// out by pinning the header to the victim's IP (exhausting their bucket).
//
// Install one of chi's middleware.ClientIPFrom* middlewares (chi v5.3.0+) and
// key off the resolved IP instead (CanonicalizeIP buckets IPv6 by /64):
//
//	r.Use(middleware.ClientIPFromXFF("10.0.0.0/8"))
//	r.Use(httprate.LimitBy(100, time.Minute, func(r *http.Request) (string, error) {
//		return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
//	}))
func KeyByRealIP(r *http.Request) (string, error) {
	var ip string

	if tcip := r.Header.Get("True-Client-IP"); tcip != "" {
		ip = tcip
	} else if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		ip = xrip
	} else if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		i := strings.Index(xff, ", ")
		if i == -1 {
			i = len(xff)
		}
		ip = xff[:i]
	} else {
		var err error
		ip, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
	}

	return CanonicalizeIP(ip), nil
}

// Deprecated: WithKeyByRealIP installs the spoofable KeyByRealIP and lets a
// remote attacker forge the rate-limit key — see GHSA-9g5q-2w5x-hmxf,
// GHSA-rjr7-jggh-pgcp, GHSA-3fxj-6jh8-hvhx for the equivalent flaw in chi's
// middleware.RealIP. Install one of chi's middleware.ClientIPFrom* middlewares
// (chi v5.3.0+) and key off the resolved IP with LimitBy instead (see
// CanonicalizeIP).
func WithKeyByRealIP() Option {
	return WithKeyFuncs(KeyByRealIP)
}

// Deprecated: KeyByIP keys off r.RemoteAddr, the TCP peer that opened the
// connection. Unlike KeyByRealIP it is NOT spoofable (RemoteAddr is set by
// net/http, never from a header) — but behind a reverse proxy, load balancer,
// or CDN, r.RemoteAddr is the proxy's address, so every client sharing that
// proxy lands in one rate-limit bucket. That's usually the wrong key in
// production, and there is no safe default IP source.
//
// State your trust model explicitly with one of chi's middleware.ClientIPFrom*
// middlewares (chi v5.3.0+) and key off the resolved IP (CanonicalizeIP buckets
// IPv6 by /64):
//
//	// Directly exposed to clients (KeyByIP's exact behavior, made explicit):
//	r.Use(middleware.ClientIPFromRemoteAddr)
//	r.Use(httprate.LimitBy(100, time.Minute, func(r *http.Request) (string, error) {
//		return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
//	}))
//
//	// Behind a proxy: use ClientIPFromXFF / ClientIPFromHeader / ... instead.
//
// KeyByIP returns the IPv4 address unchanged, or the /64 prefix for IPv6.
func KeyByIP(r *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return CanonicalizeIP(ip), nil
}

// Deprecated: WithKeyByIP installs KeyByIP, which keys off r.RemoteAddr — the
// proxy's address behind a reverse proxy, load balancer, or CDN, and usually
// the wrong key in production (see KeyByIP). It is not a spoofing issue, but
// there is no safe default IP source. State your trust model explicitly:
// install one of chi's middleware.ClientIPFrom* middlewares (chi v5.3.0+) and
// key off the resolved IP with LimitBy instead (see CanonicalizeIP).
func WithKeyByIP() Option {
	return WithKeyFuncs(KeyByIP)
}
