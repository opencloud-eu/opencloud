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
// its requests, so the request must also come from the loopback address.
func IsLocalHealthCheck(r *http.Request) bool {
	if r.UserAgent() != HealthCheckUserAgent {
		return false
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	a, err := netip.ParseAddr(host)
	return err == nil && a.Unmap().IsLoopback()
}
