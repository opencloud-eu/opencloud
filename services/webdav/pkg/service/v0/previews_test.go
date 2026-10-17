package svc

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/config"
)

var _ = Describe("with previews disabled", func() {
	// the routes stay registered, so a missing one cannot let the REPORT
	// patterns answer a thumbnail request with 405
	It("answers a thumbnail request with 404 before anything is parsed", func() {
		g := Webdav{config: &config.Config{DisablePreviews: true}}
		rec := httptest.NewRecorder()

		tr, ok := g.parseThumbnailRequest(rec, httptest.NewRequest(http.MethodGet, "/dav/files/einstein/photo.png?x=32&y=32&preview=1", nil), log.NopLogger())

		Expect(ok).To(BeFalse())
		Expect(tr).To(BeNil())
		Expect(rec.Code).To(Equal(http.StatusNotFound))
	})
})
