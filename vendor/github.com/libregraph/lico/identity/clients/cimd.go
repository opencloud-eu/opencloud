/*
 * Copyright 2026 Kopano and its licensors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package clients

// Client ID Metadata Documents, as specified at
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-client-id-metadata-document-02.
// A client uses the https URL of a JSON document describing itself as its
// client_id, and the document is fetched instead of a registration being kept.

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/libregraph/oidc-go"
	"github.com/pquerna/cachecontrol"
	"github.com/sirupsen/logrus"
)

const (
	// MetadataDocumentHostAny allows the metadata documents of any public host.
	MetadataDocumentHostAny = "*"

	// The only scheme a metadata document is ever fetched with.
	metadataDocumentScheme = "https"

	// The draft recommends reading no more than 5 KB of a document.
	metadataDocumentSizeLimit = 5 * 1024

	// Short, since a fetch can sit in the middle of an authorization or a
	// token request, which the client is waiting on.
	metadataDocumentFetchTimeout = 5 * time.Second

	// Lifetime of a document which states no caching of its own, and the bounds
	// any stated lifetime is held to.
	metadataDocumentDefaultTTL = 5 * time.Minute
	metadataDocumentMinTTL     = time.Minute
	metadataDocumentMaxTTL     = 24 * time.Hour

	// How long an expired document keeps being used while its host cannot be
	// reached. The token endpoint looks the client up on every refresh, which
	// must not depend on the client's host being up at that moment.
	metadataDocumentStaleTTL = 24 * time.Hour

	// Longest claimed client name shown to a visitor.
	metadataDocumentClaimedNameLimit = 100
)

// Domains which only resolve, or only mean something, inside a local network,
// and so never qualify for a wildcard host entry.
var metadataDocumentLocalDomains = []string{
	"arpa",
	"corp",
	"example",
	"home",
	"internal",
	"intranet",
	"invalid",
	"lan",
	"local",
	"localdomain",
	"localhost",
	"onion",
	"private",
	"test",
}

// Special-use address blocks of RFC 6890 and the IANA special-purpose address
// registries, which a metadata document fetch must never reach.
var metadataDocumentSpecialUsePrefixes = mustParsePrefixes(
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.31.196.0/24",
	"192.52.193.0/24",
	"192.88.99.0/24",
	"192.168.0.0/16",
	"192.175.48.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"::/128",
	"::1/128",
	"::ffff:0:0/96",
	"64:ff9b::/96",
	"64:ff9b:1::/48",
	"100::/64",
	"2001::/23",
	"2001:db8::/32",
	"2002::/16",
	"3fff::/20",
	"5f00::/16",
	"fc00::/7",
	"fe80::/10",
	"ff00::/8",
)

func mustParsePrefixes(prefixes ...string) []netip.Prefix {
	parsed := make([]netip.Prefix, 0, len(prefixes))
	for _, prefix := range prefixes {
		parsed = append(parsed, netip.MustParsePrefix(prefix))
	}
	return parsed
}

// errMetadataDocumentUnavailable marks a fetch which failed for a reason that
// may go away by itself, as opposed to an answer which says the document is
// gone or invalid.
var errMetadataDocumentUnavailable = errors.New("client metadata document unavailable")

// IsMetadataDocumentClientID returns true if the provided client ID has the
// shape of a Client ID Metadata Document URL. Only https is ever fetched.
func IsMetadataDocumentClientID(clientID string) bool {
	return strings.HasPrefix(clientID, metadataDocumentScheme+"://")
}

// metadataDocumentHosts is the operator's list of the hosts whose metadata
// documents may be fetched.
type metadataDocumentHosts struct {
	any      bool
	exact    map[string]bool
	suffixes []string
}

func newMetadataDocumentHosts(entries []string) (*metadataDocumentHosts, error) {
	hosts := &metadataDocumentHosts{
		exact: make(map[string]bool),
	}
	for _, entry := range entries {
		entry = strings.ToLower(strings.TrimSpace(entry))
		switch {
		case entry == "":
			continue
		case entry == MetadataDocumentHostAny:
			hosts.any = true
		case strings.HasPrefix(entry, "*."):
			suffix := entry[1:]
			if !isPublicHostname(suffix[1:]) {
				return nil, fmt.Errorf("invalid client metadata document host %q: a wildcard needs a public domain with at least two labels", entry)
			}
			hosts.suffixes = append(hosts.suffixes, suffix)
		default:
			if strings.ContainsAny(entry, "/@*?#") {
				return nil, fmt.Errorf("invalid client metadata document host %q: expected a host name, optionally with a port", entry)
			}
			hosts.exact[entry] = true
		}
	}

	return hosts, nil
}

// allows returns true if the metadata document at the provided URL may be
// fetched. An exact entry allows its host as listed. A wildcard allows public
// host names on the default port only, never an address, a single label name
// or a name which only means something inside a local network.
func (h *metadataDocumentHosts) allows(u *url.URL) bool {
	if h.exact[strings.ToLower(u.Host)] {
		return true
	}
	if !h.any && len(h.suffixes) == 0 {
		return false
	}
	if u.Port() != "" {
		return false
	}
	hostname := strings.ToLower(u.Hostname())
	if !isPublicHostname(hostname) {
		return false
	}
	if h.any {
		return true
	}
	for _, suffix := range h.suffixes {
		if strings.HasSuffix(hostname, suffix) {
			return true
		}
	}

	return false
}

func (h *metadataDocumentHosts) empty() bool {
	return !h.any && len(h.exact) == 0 && len(h.suffixes) == 0
}

// isPublicHostname returns true for a host name which can name a host on the
// internet: not an address, with at least two labels, and not below a domain
// which is only meaningful inside a local network.
func isPublicHostname(hostname string) bool {
	if hostname == "" || strings.HasSuffix(hostname, ".") {
		return false
	}
	if _, err := netip.ParseAddr(strings.Trim(hostname, "[]")); err == nil {
		return false
	}
	if !strings.Contains(hostname, ".") {
		return false
	}
	for _, domain := range metadataDocumentLocalDomains {
		if hostname == domain || strings.HasSuffix(hostname, "."+domain) {
			return false
		}
	}

	return true
}

// checkMetadataDocumentAddress refuses a connection to a special-use address.
// It runs on the address actually dialed, after name resolution, so a listed
// name which resolves inward is refused just the same.
func checkMetadataDocumentAddress(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return err
	}
	ip = ip.Unmap()
	for _, prefix := range metadataDocumentSpecialUsePrefixes {
		if prefix.Contains(ip) {
			return fmt.Errorf("refusing to fetch client metadata document from special-use address %s", ip)
		}
	}

	return nil
}

// parseMetadataDocumentClientID returns the URL of the provided client ID if
// it is a valid Client Identifier URL as specified in section 3 of the draft.
func parseMetadataDocumentClientID(clientID string) (*url.URL, error) {
	u, err := url.Parse(clientID)
	if err != nil {
		return nil, fmt.Errorf("client_id is not a URL: %w", err)
	}
	switch {
	case u.Scheme != metadataDocumentScheme:
		return nil, errors.New("client_id URL must use https")
	case u.Opaque != "" || u.Host == "":
		return nil, errors.New("client_id URL has no host")
	case u.User != nil:
		return nil, errors.New("client_id URL must not contain user info")
	case strings.ContainsAny(clientID, "#?"):
		return nil, errors.New("client_id URL must not contain a query or a fragment")
	case u.Path == "":
		return nil, errors.New("client_id URL must contain a path")
	case strings.HasSuffix(u.Hostname(), "."):
		return nil, errors.New("client_id URL host must not end with a dot")
	case hasDotSegment(u.Path):
		return nil, errors.New("client_id URL must not contain dot segments")
	}

	return u, nil
}

func hasDotSegment(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

// metadataDocument holds the members of a Client ID Metadata Document which
// are used here. Everything else in it is ignored.
type metadataDocument struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// parseMetadataDocument validates the provided document fetched from the
// provided client ID URL and returns the client registration it describes.
func parseMetadataDocument(clientID string, u *url.URL, body []byte) (*ClientRegistration, error) {
	var members map[string]json.RawMessage
	if err := json.Unmarshal(body, &members); err != nil {
		return nil, fmt.Errorf("client metadata document is not a JSON object: %w", err)
	}
	doc := &metadataDocument{}
	if err := json.Unmarshal(body, doc); err != nil {
		return nil, fmt.Errorf("client metadata document is invalid: %w", err)
	}

	if doc.ClientID != clientID {
		return nil, fmt.Errorf("client metadata document client_id %q does not match its URL", doc.ClientID)
	}

	if err := checkMetadataDocumentCredentials(members, doc); err != nil {
		return nil, err
	}
	grantTypes, err := metadataDocumentGrantTypes(doc)
	if err != nil {
		return nil, err
	}
	applicationType, err := metadataDocumentApplicationType(u, doc.RedirectURIs)
	if err != nil {
		return nil, err
	}

	registration := &ClientRegistration{
		ID: clientID,
		// The name shown for the client is the host vouching for it. The name
		// the document gives is self-asserted and only travels beside it.
		Name:                       u.Host,
		ClaimedName:                truncateRunes(strings.TrimSpace(doc.ClientName), metadataDocumentClaimedNameLimit),
		ApplicationType:            applicationType,
		RedirectURIs:               doc.RedirectURIs,
		GrantTypes:                 grantTypes,
		RawTokenEndpointAuthMethod: oidc.AuthMethodNone,
		RequirePKCE:                true,
		MetadataDocument:           true,
	}
	if err := prepareRedirectURIs(registration); err != nil {
		return nil, fmt.Errorf("client metadata document: %w", err)
	}

	return registration, nil
}

// checkMetadataDocumentCredentials refuses a document which asks for anything
// but a public client. A document is public, so it can hold no shared secret,
// and a key based client authentication is not implemented. Anything but none
// is refused rather than silently downgraded.
func checkMetadataDocumentCredentials(members map[string]json.RawMessage, doc *metadataDocument) error {
	for _, member := range []string{"client_secret", "client_secret_expires_at"} {
		if _, ok := members[member]; ok {
			return fmt.Errorf("client metadata document must not contain %s", member)
		}
	}
	if doc.TokenEndpointAuthMethod != "" && doc.TokenEndpointAuthMethod != oidc.AuthMethodNone {
		return fmt.Errorf("client metadata document token_endpoint_auth_method %q is not supported", doc.TokenEndpointAuthMethod)
	}

	return nil
}

// metadataDocumentGrantTypes returns the grant types of the provided document
// which are implemented here. Grant types are a request, not a contract, see
// RFC 7591 section 3.2.1, so the others are dropped, and only a document which
// cannot use the authorization code grant is refused. Their absence means
// authorization_code, as it does for a registration.
func metadataDocumentGrantTypes(doc *metadataDocument) ([]string, error) {
	var grantTypes []string
	if len(doc.GrantTypes) > 0 {
		for _, grantType := range doc.GrantTypes {
			switch grantType {
			case oidc.GrantTypeAuthorizationCode, oidc.GrantTypeRefreshToken:
				grantTypes = append(grantTypes, grantType)
			}
		}
		if !slices.Contains(grantTypes, oidc.GrantTypeAuthorizationCode) {
			return nil, errors.New("client metadata document does not use the authorization_code grant")
		}
	}
	if len(doc.ResponseTypes) > 0 && !slices.Contains(doc.ResponseTypes, oidc.ResponseTypeCode) {
		return nil, errors.New("client metadata document does not use the code response type")
	}

	return grantTypes, nil
}

// metadataDocumentApplicationType checks the redirect URIs of a document from
// the provided client ID URL and returns the application type they make. Every
// one which is not loopback has to be https on the host of the client ID, so a
// document cannot claim a callback on a host it does not speak for. The
// registry ties a client to a single application type, web returning to https
// only and native to loopback only, so a mix is refused.
func metadataDocumentApplicationType(u *url.URL, redirectURIs []string) (string, error) {
	if len(redirectURIs) == 0 {
		return "", errors.New("client metadata document has no redirect_uris")
	}

	loopback := 0
	for _, redirectURIString := range redirectURIs {
		redirectURI, err := url.Parse(redirectURIString)
		if err != nil || !isPlainRedirectURI(redirectURI) {
			return "", fmt.Errorf("client metadata document redirect_uri %q is invalid", redirectURIString)
		}
		switch {
		case IsLocalNativeHTTPURI(redirectURI):
			loopback++
		case redirectURI.Scheme == u.Scheme && strings.EqualFold(redirectURI.Host, u.Host):
		default:
			return "", fmt.Errorf("client metadata document redirect_uri %q is neither loopback nor on the host of the client_id", redirectURIString)
		}
	}

	switch loopback {
	case 0:
		return oidc.ApplicationTypeWeb, nil
	case len(redirectURIs):
		return oidc.ApplicationTypeNative, nil
	default:
		return "", errors.New("client metadata document mixes loopback and https redirect_uris")
	}
}

// isPlainRedirectURI returns true for a redirect URI with a host and without
// user info or a fragment.
func isPlainRedirectURI(uri *url.URL) bool {
	return uri.Host != "" && uri.User == nil && uri.Fragment == ""
}

func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}

// metadataDocumentEntry is a validated document in the cache.
type metadataDocumentEntry struct {
	registration *ClientRegistration
	expires      time.Time
}

// metadataDocumentCall is a fetch in flight, shared by everyone asking for the
// same client ID meanwhile.
type metadataDocumentCall struct {
	done         chan struct{}
	registration *ClientRegistration
	err          error
}

// metadataDocumentFetcher fetches, validates and caches Client ID Metadata
// Documents from the hosts the operator allowed.
type metadataDocumentFetcher struct {
	hosts  *metadataDocumentHosts
	client *http.Client
	logger logrus.FieldLogger
	now    func() time.Time

	mutex   sync.Mutex
	entries map[string]*metadataDocumentEntry
	calls   map[string]*metadataDocumentCall
}

func newMetadataDocumentFetcher(hostEntries []string, logger logrus.FieldLogger) (*metadataDocumentFetcher, error) {
	hosts, err := newMetadataDocumentHosts(hostEntries)
	if err != nil {
		return nil, err
	}
	if hosts.empty() {
		return nil, nil
	}

	return &metadataDocumentFetcher{
		hosts:  hosts,
		client: newMetadataDocumentClient(checkMetadataDocumentAddress, nil),
		logger: logger,
		now:    time.Now,

		entries: make(map[string]*metadataDocumentEntry),
		calls:   make(map[string]*metadataDocumentCall),
	}, nil
}

// newMetadataDocumentClient returns the HTTP client documents are fetched
// with. Every address it dials is checked with the provided function first.
func newMetadataDocumentClient(checkAddress func(address string) error, tlsConfig *tls.Config) *http.Client {
	dialer := &net.Dialer{
		Timeout: metadataDocumentFetchTimeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			return checkAddress(address)
		},
	}
	transport := &http.Transport{
		// No proxy: a proxy dials on our behalf, past the address check.
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		TLSClientConfig:       tlsConfig,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   metadataDocumentFetchTimeout,
		ResponseHeaderTimeout: metadataDocumentFetchTimeout,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   metadataDocumentFetchTimeout,
		// The draft forbids following redirects, and the answer to the
		// request itself is then a non-200 status, which is an error.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// Get returns the client registration described by the metadata document at
// the provided client ID.
func (f *metadataDocumentFetcher) Get(ctx context.Context, clientID string) (*ClientRegistration, error) {
	u, err := parseMetadataDocumentClientID(clientID)
	if err != nil {
		return nil, err
	}
	if !f.hosts.allows(u) {
		return nil, fmt.Errorf("client metadata documents of host %s are not allowed", u.Host)
	}

	registration, call := f.cachedOrFetch(clientID, u)
	if registration != nil {
		return registration, nil
	}

	select {
	case <-call.done:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if call.err == nil {
		return call.registration, nil
	}

	if errors.Is(call.err, errMetadataDocumentUnavailable) {
		if registration := f.stale(clientID); registration != nil {
			f.logger.WithError(call.err).WithField("client_id", clientID).Warnln("using expired client metadata document")
			return registration, nil
		}
	}

	return nil, call.err
}

// cachedOrFetch returns the cached registration of the provided client ID if
// it has not expired, and the fetch to wait for otherwise, starting one unless
// there is one in flight already.
func (f *metadataDocumentFetcher) cachedOrFetch(clientID string, u *url.URL) (*ClientRegistration, *metadataDocumentCall) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if entry, ok := f.entries[clientID]; ok && f.now().Before(entry.expires) {
		return entry.registration, nil
	}
	call, ok := f.calls[clientID]
	if !ok {
		call = &metadataDocumentCall{
			done: make(chan struct{}),
		}
		f.calls[clientID] = call
		// Detached from the request, so a caller which gives up does not fail
		// the fetch for everyone else waiting on it.
		go f.fetch(clientID, u, call)
	}

	return nil, call
}

// stale returns the cached registration of the provided client ID if it is
// expired by no more than the stale limit.
func (f *metadataDocumentFetcher) stale(clientID string) *ClientRegistration {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if entry, ok := f.entries[clientID]; ok && f.now().Before(entry.expires.Add(metadataDocumentStaleTTL)) {
		return entry.registration
	}
	return nil
}

func (f *metadataDocumentFetcher) fetch(clientID string, u *url.URL, call *metadataDocumentCall) {
	registration, expires, err := f.request(clientID, u)

	f.mutex.Lock()
	delete(f.calls, clientID)
	switch {
	case err == nil:
		f.entries[clientID] = &metadataDocumentEntry{
			registration: registration,
			expires:      expires,
		}
	case !errors.Is(err, errMetadataDocumentUnavailable):
		// The host answered, and the answer is that there is no valid
		// document. Errors are never cached, and neither is the last good
		// document once its host says it is gone.
		delete(f.entries, clientID)
	}
	call.registration = registration
	call.err = err
	f.mutex.Unlock()
	close(call.done)

	if err != nil {
		f.logger.WithError(err).WithField("client_id", clientID).Debugln("client metadata document fetch failed")
	} else {
		f.logger.WithFields(logrus.Fields{
			"client_id":        clientID,
			"application_type": registration.ApplicationType,
			"redirect_uris":    registration.RedirectURIs,
			"expires":          expires,
		}).Debugln("client metadata document fetched")
	}
}

func (f *metadataDocumentFetcher) request(clientID string, u *url.URL) (*ClientRegistration, time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), metadataDocumentFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientID, http.NoBody)
	if err != nil {
		return nil, time.Time{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("%w: %w", errMetadataDocumentUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("client metadata document answered with status %d", resp.StatusCode)
		if resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests {
			err = fmt.Errorf("%w: %w", errMetadataDocumentUnavailable, err)
		}
		return nil, time.Time{}, err
	}

	body, err := readMetadataDocument(resp)
	if err != nil {
		return nil, time.Time{}, err
	}

	registration, err := parseMetadataDocument(clientID, u, body)
	if err != nil {
		return nil, time.Time{}, err
	}

	return registration, f.expires(req, resp), nil
}

// readMetadataDocument returns the body of the provided response if it is a
// JSON document within the size limit.
func readMetadataDocument(resp *http.Response) ([]byte, error) {
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaType != "application/json" && !strings.HasSuffix(mediaType, "+json") {
		return nil, fmt.Errorf("client metadata document has content type %q, not JSON", mediaType)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, metadataDocumentSizeLimit+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errMetadataDocumentUnavailable, err)
	}
	if len(body) > metadataDocumentSizeLimit {
		return nil, fmt.Errorf("client metadata document is larger than %d bytes", metadataDocumentSizeLimit)
	}

	return body, nil
}

// expires returns until when a document may be used without a refetch, from
// its caching headers and held to the bounds.
func (f *metadataDocumentFetcher) expires(req *http.Request, resp *http.Response) time.Time {
	now := f.now()
	ttl := metadataDocumentDefaultTTL
	if _, expires, err := cachecontrol.CachableResponse(req, resp, cachecontrol.Options{}); err == nil && !expires.IsZero() {
		ttl = expires.Sub(now)
	}
	if ttl < metadataDocumentMinTTL {
		ttl = metadataDocumentMinTTL
	}
	if ttl > metadataDocumentMaxTTL {
		ttl = metadataDocumentMaxTTL
	}

	return now.Add(ttl)
}
