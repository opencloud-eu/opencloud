package middleware

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

var _ = Describe("HTTPSRedirect", func() {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	serve := func(baseURL, target, host, proto string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.Host = host
		if proto != "" {
			req.Header.Set("X-Forwarded-Proto", proto)
		}
		rr := httptest.NewRecorder()
		HTTPSRedirect(log.NopLogger(), baseURL)(next).ServeHTTP(rr, req)
		return rr
	}

	It("redirects to the host of the base URL, not to the Host header", func() {
		rr := serve("https://cloud.example.com", "/remote.php/webdav?foo=bar", "evil.example.org", "http")
		Expect(rr.Code).To(Equal(http.StatusPermanentRedirect))
		Expect(rr.Header().Get("Location")).To(Equal("https://cloud.example.com/remote.php/webdav?foo=bar"))
	})

	It("keeps the port of the base URL", func() {
		rr := serve("https://localhost:9200", "/", "localhost:9200", "HTTP")
		Expect(rr.Code).To(Equal(http.StatusPermanentRedirect))
		Expect(rr.Header().Get("Location")).To(Equal("https://localhost:9200/"))
	})

	It("keeps the escaping of the path", func() {
		rr := serve("https://cloud.example.com", "/dav/files/a%2Fb%20c", "cloud.example.com", "http")
		Expect(rr.Code).To(Equal(http.StatusPermanentRedirect))
		Expect(rr.Header().Get("Location")).To(Equal("https://cloud.example.com/dav/files/a%2Fb%20c"))
	})

	It("stays on the host of the base URL for a path that starts with two slashes", func() {
		rr := serve("https://cloud.example.com", "//evil.example.org/x", "cloud.example.com", "http")
		Expect(rr.Code).To(Equal(http.StatusPermanentRedirect))
		Expect(rr.Header().Get("Location")).To(Equal("https://cloud.example.com//evil.example.org/x"))
	})

	It("redirects when the request has no Host header", func() {
		rr := serve("https://cloud.example.com", "/index.html", "", "http")
		Expect(rr.Code).To(Equal(http.StatusPermanentRedirect))
		Expect(rr.Header().Get("Location")).To(Equal("https://cloud.example.com/index.html"))
	})

	It("keeps a # in the request path as part of the path", func() {
		// clients don't send fragments: a # in the request is part of the path
		rr := serve("https://cloud.example.com", "/dav/files/a#b", "cloud.example.com", "http")
		Expect(rr.Code).To(Equal(http.StatusPermanentRedirect))
		Expect(rr.Header().Get("Location")).To(Equal("https://cloud.example.com/dav/files/a%23b"))
	})

	It("takes neither host nor user info from an absolute request URL", func() {
		// the user info would put the password into the Location header
		rr := serve("https://cloud.example.com", "http://user:secret@evil.example.org/x?y=1", "evil.example.org", "http")
		Expect(rr.Code).To(Equal(http.StatusPermanentRedirect))
		Expect(rr.Header().Get("Location")).To(Equal("https://cloud.example.com/x?y=1"))
	})

	DescribeTable("passes the request on",
		func(baseURL, proto string) {
			rr := serve(baseURL, "/", "cloud.example.com", proto)
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Header().Get("Location")).To(BeEmpty())
		},
		Entry("when it was forwarded as HTTPS", "https://cloud.example.com", "https"),
		Entry("when it has no X-Forwarded-Proto", "https://cloud.example.com", ""),
		Entry("when the base URL is empty", "", "http"),
		Entry("when the base URL has no host", "cloud.example.com", "http"),
		Entry("when the base URL has an empty host", "https://", "http"),
		Entry("when the base URL can't be parsed", "https://cloud example.com:abc", "http"),
	)
})
