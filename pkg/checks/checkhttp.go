package checks

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/handlers"
)

// HealthCheckUserAgent is the user agent of the requests NewHTTPCheck sends.
const HealthCheckUserAgent = "opencloud-health-check"

// NewHTTPCheck checks the reachability of a http server. Any answer counts,
// a server that doesn't answer within the timeout fails the check.
func NewHTTPCheck(url string) func(context.Context) error {
	return func(ctx context.Context) error {
		url, err := handlers.FailSaveAddress(url)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "http://" + url
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", HealthCheckUserAgent)

		c := http.Client{
			Timeout: 3 * time.Second,
		}
		resp, err := c.Do(req)
		if err != nil {
			return fmt.Errorf("could not connect to http server: %v", err)
		}
		_ = resp.Body.Close()
		return nil
	}
}

// IsLocalHealthCheck reports whether r is a request of NewHTTPCheck, so an
// access log can leave it out. The user agent alone would let any client hide
// its requests, so the connection must also come from the server's own
// address: the check calls the address the server listens on, for 0.0.0.0
// that is the IP of the host or pod, not the loopback address.
func IsLocalHealthCheck(r *http.Request) bool {
	if r.UserAgent() != HealthCheckUserAgent {
		return false
	}
	local, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		return false
	}
	peer, ok := addrIP(r.RemoteAddr)
	if !ok {
		return false
	}
	own, ok := addrIP(local.String())
	return ok && peer == own
}

func addrIP(hostport string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return netip.Addr{}, false
	}
	a, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	return a.Unmap().WithZone(""), true
}
