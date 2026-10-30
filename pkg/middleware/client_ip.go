package middleware

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// DefaultClientIPHeader is the header a trusted reverse proxy sends the client
// address in, unless configured otherwise.
const DefaultClientIPHeader = "X-Forwarded-For"

// DefaultTrustedProxies returns the loopback and the private network ranges.
// They cover a reverse proxy in the same Docker network or a Kubernetes
// ingress, while clients connecting from the internet can't set their address.
func DefaultTrustedProxies() []string {
	return []string{"127.0.0.0/8", "::1/128", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fc00::/7"}
}

// ParseTrustedProxies parses IP addresses and CIDR ranges.
func ParseTrustedProxies(list []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(list))
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if strings.Contains(s, "/") {
			p, err := netip.ParsePrefix(s)
			if err != nil {
				return nil, fmt.Errorf("invalid trusted proxy %q: %w", s, err)
			}
			prefixes = append(prefixes, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(s)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy %q: %w", s, err)
		}
		a = a.Unmap()
		prefixes = append(prefixes, netip.PrefixFrom(a, a.BitLen()))
	}
	return prefixes, nil
}

// ClientIP determines the address of the client with chi's ClientIPFrom*
// middlewares; read it with ClientAddress. None of chi's header based
// middlewares look at who sent the header, so the header is only read when the
// request comes from a trusted proxy. Everyone else gets their network
// address, whatever headers they send.
//
// From a trusted proxy the address is taken from header. For X-Forwarded-For
// it is the first address from the right that is not a trusted proxy, for any
// other header, e.g. X-Real-IP or CF-Connecting-IP, its value. Without a
// usable header, e.g. on health checks, it is the address of the proxy.
func ClientIP(trusted []netip.Prefix, header string) func(http.Handler) http.Handler {
	cidrs := make([]string, len(trusted))
	for i, p := range trusted {
		cidrs[i] = p.String()
	}
	fromHeader := chimiddleware.ClientIPFromXFF(cidrs...)
	if header != "" && http.CanonicalHeaderKey(header) != DefaultClientIPHeader {
		fromHeader = chimiddleware.ClientIPFromHeader(header)
	}

	return func(next http.Handler) http.Handler {
		// the header only overwrites the network address when it has a usable one
		fromProxy := chimiddleware.ClientIPFromRemoteAddr(fromHeader(next))
		direct := chimiddleware.ClientIPFromRemoteAddr(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if peer, ok := parseAddr(r.RemoteAddr); ok && isTrusted(peer, trusted) {
				fromProxy.ServeHTTP(w, r)
				return
			}
			direct.ServeHTTP(w, r)
		})
	}
}

// ClientAddress returns the client address ClientIP determined, the network
// address of the peer if it didn't run.
func ClientAddress(r *http.Request) string {
	if ip := chimiddleware.GetClientIP(r.Context()); ip != "" {
		return ip
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// parseAddr reads an address with or without port.
func parseAddr(s string) (netip.Addr, bool) {
	s = strings.TrimSpace(s)
	if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}
	a, err := netip.ParseAddr(strings.Trim(s, "[]"))
	if err != nil {
		return netip.Addr{}, false
	}
	return a.Unmap().WithZone(""), true
}

func isTrusted(a netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}
