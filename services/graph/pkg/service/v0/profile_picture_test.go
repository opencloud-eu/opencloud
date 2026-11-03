package svc_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	svc "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

// minimalPNG is a valid 1×1 PNG used for testing image downloads.
var minimalPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // signature
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
	0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
	0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
	0x00, 0x00, 0x03, 0x00, 0x01, 0x5B, 0x42, 0xD3,
	0xBF, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
	0x44, 0xAE, 0x42, 0x60, 0x82,
}

// testPhotoService creates a UsersUserProfilePhotoService configured for
// profile picture sync with the given allowlist and issuer.
func testPhotoService(httpClient svc.HTTPClient, urlAllowlist []string, oidcIssuer string) svc.UsersUserProfilePhotoService {
	s, err := svc.NewUsersUserProfilePhotoService(nil, svc.WithProfilePictureSync(httpClient, urlAllowlist, oidcIssuer))
	Expect(err).To(Succeed())
	return s
}

var _ = Describe("profile pictures", func() {
	DescribeTable("profile picture URL pattern matching", func(pattern, target string, want bool) {
		parsed, err := url.Parse(target)
		Expect(err).To(Succeed())
		Expect(svc.URLPatternMatches(pattern, parsed)).To(Equal(want))
	},
		// Host glob patterns
		Entry("glob subdomain", "https://*.example.com", "https://cdn.example.com", true),
		Entry("glob exact host", "https://example.com", "https://example.com", true),
		Entry("glob no match host", "https://*.example.com", "https://evil.com", false),
		Entry("glob all", "*", "https://anything.com/path", true),
		Entry("glob all http", "*", "http://anything.com", true),

		// Scheme matching
		Entry("scheme mismatch", "https://example.com", "http://example.com", false),
		Entry("scheme match http", "http://example.com", "http://example.com", true),

		// Path matching
		Entry("path prefix wildcard", "https://example.com/img/*", "https://example.com/img/avatar.png", true),
		Entry("path exact", "https://example.com/avatar.png", "https://example.com/avatar.png", true),
		Entry("path no match", "https://example.com/avatar.png", "https://example.com/other.png", false),
		Entry("root path matches any path", "https://example.com", "https://example.com/any/path", true),
		Entry("root path slash matches any path", "https://example.com/", "https://example.com/any/path", true),

		// Only complete HTTP(S) URL patterns and the explicit wildcard are accepted.
		Entry("bare host rejected", "example.com", "https://example.com/avatar.png", false),
		Entry("bare host glob rejected", "*.example.com", "https://cdn.example.com/avatar.png", false),
		Entry("relative URL rejected", "//example.com", "https://example.com/avatar.png", false),
		Entry("invalid URL pattern", "https://example.com/%zz", "https://example.com/avatar.png", false),
		Entry("invalid host glob", "https://[*.example.com", "https://example.com/avatar.png", false),
		Entry("unsupported pattern scheme", "ftp://example.com", "https://example.com/avatar.png", false),
		Entry("unsupported target scheme", "*", "ftp://example.com/avatar.png", false),
		Entry("relative target rejected", "*", "/avatar.png", false),
		Entry("target credentials rejected", "*", "https://user:password@example.com/avatar.png", false),
		Entry("pattern credentials rejected", "https://user@example.com", "https://example.com/avatar.png", false),
		Entry("query in pattern rejected", "https://example.com?token=secret", "https://example.com/avatar.png", false),
		Entry("empty query in pattern rejected", "https://example.com?", "https://example.com/avatar.png", false),
		Entry("fragment in pattern rejected", "https://example.com#avatar", "https://example.com/avatar.png", false),
		Entry("empty fragment in pattern rejected", "https://example.com#", "https://example.com/avatar.png", false),
		Entry("target query ignored", "https://example.com/avatar.png", "https://example.com/avatar.png?size=128#photo", true),

		// Ports are part of the host pattern; paths are decoded and normalized.
		Entry("port match", "https://example.com:8443", "https://example.com:8443/avatar.png", true),
		Entry("port mismatch", "https://example.com:8443", "https://example.com/avatar.png", false),
		Entry("explicit default port differs", "https://example.com", "https://example.com:443/avatar.png", false),
		Entry("path prefix boundary", "https://example.com/img/*", "https://example.com/images/avatar.png", false),
		Entry("path case sensitive", "https://example.com/Avatar.png", "https://example.com/avatar.png", false),
		Entry("path is normalized", "https://example.com/avatar.png", "https://example.com/other/../avatar.png", true),
		Entry("encoded separator is normalized", "https://example.com/img/*", "https://example.com/img%2Favatar.png", true),
		Entry("path traversal rejected", "https://example.com/img/*", "https://example.com/img/../other.png", false),
		Entry("encoded path traversal rejected", "https://example.com/img/*", "https://example.com/img/%2e%2e/other.png", false),
		Entry("encoded separator traversal rejected", "https://example.com/img/*", "https://example.com/img%2f..%2fother.png", false),
		Entry("internal traversal normalized", "https://example.com/img/*", "https://example.com/img/sub/../avatar.png", true),
		Entry("encoded path exact", "https://example.com/my%20avatar.png", "https://example.com/my%20avatar.png", true),

		// IPv6 literals must not be interpreted as glob character classes.
		Entry("IPv6 literal", "https://[::1]", "https://[::1]/avatar.png", true),
		Entry("IPv6 port match", "https://[::1]:9200", "https://[::1]:9200/avatar.png", true),
		Entry("IPv6 port mismatch", "https://[::1]:9200", "https://[::1]:9201/avatar.png", false),
		Entry("IPv6 path match", "https://[2001:db8::1]:9200/img/*", "https://[2001:db8::1]:9200/img/avatar.png", true),
		Entry("IPv6 path mismatch", "https://[2001:db8::1]:9200/img/*", "https://[2001:db8::1]:9200/other/avatar.png", false),
		Entry("IPv6 host mismatch", "https://[::1]", "https://[::2]/avatar.png", false),
		Entry("IPv6 brackets do not match DNS host", "https://[::1]", "https://1/avatar.png", false),
		Entry("IPv6 case insensitive", "https://[2001:DB8::1]", "https://[2001:db8::1]/avatar.png", true),

		// Case insensitivity
		Entry("case insensitive host", "https://*.Example.COM", "https://cdn.example.com", true),
	)

	It("rejects a nil profile picture URL", func() {
		Expect(svc.URLPatternMatches("*", nil)).To(BeFalse())
	})

	DescribeTable("profile picture URL allowlist", func(allowlist []string, oidcIssuer, targetURL string, want bool) {
		s := testPhotoService(http.DefaultClient, allowlist, oidcIssuer)
		Expect(svc.IsProfilePictureURLAllowed(s, targetURL)).To(Equal(want))
	},
		Entry("default to issuer host", nil, "https://idp.example.com", "https://idp.example.com/avatar.png", true),
		Entry("default to issuer host rejects other host", nil, "https://idp.example.com", "https://evil.com/avatar.png", false),
		Entry("explicit allowlist with glob", []string{"https://*.cdn.example.com"}, "https://idp.example.com", "https://img.cdn.example.com/avatar.png", true),
		Entry("wildcard allows all", []string{"*"}, "", "https://any.host.com/path", true),
		Entry("rejects non-http scheme", []string{"*"}, "", "ftp://example.com/file", false),
		Entry("rejects invalid url", []string{"*"}, "", "://not-a-url", false),
		Entry("wildcard rejects credentials", []string{"*"}, "", "https://user:password@example.com/avatar.png", false),
		Entry("issuer path does not restrict photo paths", []string(nil), "https://idp.example.com/realms/opencloud", "https://idp.example.com/avatar.png", true),
		Entry("issuer scheme is enforced", []string(nil), "https://idp.example.com", "http://idp.example.com/avatar.png", false),
		Entry("no issuer and no allowlist rejects", nil, "", "https://example.com/avatar.png", false),
	)

	Describe("syncing a profile picture from a URL", func() {
		It("rejects disallowed URL", func() {
			s := testPhotoService(http.DefaultClient, nil, "https://idp.example.com")
			err := s.SyncPhotoFromURL(context.Background(), "user123", "https://evil.com/img.png")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not allowed"))
		})

		It("returns error for empty user ID", func() {
			s := testPhotoService(http.DefaultClient, []string{"*"}, "")
			err := s.SyncPhotoFromURL(context.Background(), "", "https://example.com/img.png")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("required argument is missing"))
		})

		It("rejects non-image content type", func() {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not an image", http.StatusOK)
			}))
			defer srv.Close()

			s := testPhotoService(srv.Client(), []string{"*"}, "")
			err := s.SyncPhotoFromURL(context.Background(), "user123", srv.URL)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("content type"))
		})

		It("rejects empty response", func() {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "image/png")
			}))
			defer srv.Close()

			s := testPhotoService(srv.Client(), []string{"*"}, "")
			err := s.SyncPhotoFromURL(context.Background(), "user123", srv.URL)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty"))
		})

		It("rejects error status code", func() {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			}))
			defer srv.Close()

			s := testPhotoService(srv.Client(), []string{"*"}, "")
			err := s.SyncPhotoFromURL(context.Background(), "user123", srv.URL)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("404"))
		})
	})

	It("rejects profile pictures larger than the download size limit", func() {
		oversized := bytes.Repeat(minimalPNG, (svc.MaxProfilePhotoBytes/len(minimalPNG))+1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = io.Copy(w, bytes.NewReader(oversized))
		}))
		defer srv.Close()

		s := testPhotoService(srv.Client(), []string{"*"}, "")
		_, err := svc.FetchProfilePicture(s, context.Background(), srv.URL)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exceeds"))
	})

})
