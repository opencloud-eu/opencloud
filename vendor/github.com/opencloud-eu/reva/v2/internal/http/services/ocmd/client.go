// Copyright 2018-2025 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package ocmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"

	"github.com/opencloud-eu/reva/v2/internal/http/services/wellknown"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/pkg/errors"
)

// OCMClient is the client for an OCM provider.
type OCMClient struct {
	client *http.Client
}

// newOCMTransport returns the HTTP transport used for outbound OCM requests.
// It honors the standard HTTP_PROXY, HTTPS_PROXY, and NO_PROXY environment
// variables and optionally skips TLS certificate verification.
func newOCMTransport(insecure bool) *http.Transport {
	var tr *http.Transport
	if dt, ok := http.DefaultTransport.(*http.Transport); ok {
		tr = dt.Clone()
	} else {
		tr = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		}
	}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}
	return tr
}

// NewClient returns a new OCMClient.
func NewClient(timeout time.Duration, insecure bool) *OCMClient {
	return &OCMClient{
		client: &http.Client{
			Transport: newOCMTransport(insecure),
			Timeout:   timeout,
		},
	}
}

// NewPublicOnlyClient returns an OCMClient that only dials public addresses, for
// discovery of hosts named by an untrusted caller. The check is in the dialer so
// it also covers redirects and DNS rebinding.
func NewPublicOnlyClient(timeout time.Duration, insecure bool) *OCMClient {
	tr := newOCMTransport(insecure)
	// with a proxy the dial goes to the proxy, so Control never sees the target
	tr.Proxy = nil
	tr.DialContext = (&net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
		Control:   refuseNonPublicAddr,
	}).DialContext
	return &OCMClient{
		client: &http.Client{
			Transport: tr,
			Timeout:   timeout,
		},
	}
}

func refuseNonPublicAddr(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || !isPublicIP(ip) {
		return errtypes.BadRequest("refusing to connect to non-public address " + address)
	}
	return nil
}

// 64:ff9b::/96 embeds an IPv4 address in its low 32 bits (RFC 6052), so on a
// NAT64 network it can still reach internal hosts.
var nat64WellKnownPrefix = net.IPNet{IP: net.ParseIP("64:ff9b::"), Mask: net.CIDRMask(96, 128)}

func isPublicIP(ip net.IP) bool {
	if nat64WellKnownPrefix.Contains(ip) {
		v6 := ip.To16()
		return isPublicIP(net.IPv4(v6[12], v6[13], v6[14], v6[15]))
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsInterfaceLocalMulticast() {
		return false
	}
	// 100.64.0.0/10 is carrier-grade NAT, which net.IP.IsPrivate does not cover
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1]&0xc0 == 64 {
		return false
	}
	return true
}

// Discover returns the OCM discovery information for a remote endpoint.
// It tries /.well-known/ocm first, then falls back to /ocm-provider (legacy).
// https://cs3org.github.io/OCM-API/docs.html?branch=develop&repo=OCM-API&user=cs3org#/paths/~1ocm-provider/get
func (c *OCMClient) Discover(ctx context.Context, endpoint string) (*wellknown.OcmDiscoveryData, error) {
	log := appctx.GetLogger(ctx)

	remoteurl, _ := url.JoinPath(endpoint, "/.well-known/ocm")
	body, err := c.discover(ctx, remoteurl)
	if err != nil || len(body) == 0 {
		log.Debug().Err(err).Str("sender", remoteurl).Str("response", string(body)).
			Msg("invalid or empty response, falling back to legacy discovery")
		remoteurl, _ := url.JoinPath(endpoint, "/ocm-provider") // legacy discovery endpoint

		body, err = c.discover(ctx, remoteurl)
		if err != nil || len(body) == 0 {
			log.Warn().Err(err).Str("sender", remoteurl).Str("response", string(body)).
				Msg("invalid or empty response")
			return nil, errtypes.InternalError("Invalid response on OCM discovery")
		}
	}

	var disco wellknown.OcmDiscoveryData
	err = json.Unmarshal(body, &disco)
	if err != nil {
		log.Warn().Err(err).Str("sender", remoteurl).Str("response", string(body)).
			Msg("malformed response")
		return nil, errtypes.InternalError("Invalid payload on OCM discovery")
	}

	log.Debug().Str("sender", remoteurl).Any("response", disco).Msg("discovery response")
	return &disco, nil
}

func (c *OCMClient) discover(ctx context.Context, url string) ([]byte, error) {
	log := appctx.GetLogger(ctx)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating OCM discovery request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if err != nil {
		return nil, errors.Wrap(err, "error doing OCM discovery request")
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Warn().Err(err).Msg("error closing response body")
		}
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Warn().Str("sender", url).Int("status", resp.StatusCode).Msg("discovery returned")
		return nil, errtypes.NewErrtypeFromHTTPStatusCode(resp.StatusCode, "Remote does not offer a valid OCM discovery endpoint")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "malformed remote OCM discovery")
	}
	return body, nil
}

// GetDirectoryService fetches a directory service listing from the given URL per OCM spec Appendix C.
func (c *OCMClient) GetDirectoryService(ctx context.Context, directoryURL string) (*DirectoryService, error) {
	log := appctx.GetLogger(ctx)

	// TODO(@MahdiBaghbani): the discover() should be changed into a generic function that can be used to fetch any OCM endpoint. I'll do it in the security PR to minimize conflicts.
	body, err := c.discover(ctx, directoryURL)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching directory service")
	}

	var dirService DirectoryService
	if err := json.Unmarshal(body, &dirService); err != nil {
		log.Warn().Err(err).Str("url", directoryURL).Str("response", string(body)).Msg("malformed directory service response")
		return nil, errors.Wrap(err, "invalid directory service payload")
	}

	// Validate required fields
	if dirService.Federation == "" {
		return nil, errtypes.InternalError("directory service missing required 'federation' field")
	}
	// Servers can be empty array, that's valid

	log.Debug().Str("url", directoryURL).Str("federation", dirService.Federation).Int("servers", len(dirService.Servers)).Msg("fetched directory service")
	return &dirService, nil
}
