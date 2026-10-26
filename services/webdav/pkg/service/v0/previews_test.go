package svc

import (
	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/config/defaults"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/config"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
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

func TestThumbnailWorkflowValidatesSSEC(t *testing.T) {
	t.Setenv("WEBDAV_THUMBNAILS_CACHE_S3_SSE_C_KEY", "invalid-secret!")
	cfg := defaults.FullDefaultConfig()
	cfg.Thumbnails.CacheBackend = "s3"
	cfg.Thumbnails.CacheS3Bucket = "bucket"
	cfg.Thumbnails.CacheS3Region = "us-east-1"
	cfg.Thumbnails.CacheS3Endpoint = "https://localhost:9000"
	cfg.Thumbnails.CacheS3AccessKey = "ak"
	cfg.Thumbnails.CacheS3SecretKey = "sk"
	if err := envdecode.Decode(cfg); err != nil {
		t.Fatal(err)
	}
	_, err := newThumbnailWorkflow(cfg, thumbnail.NewSourceTypes(nil), log.NopLogger(), nil)
	if err == nil || !strings.Contains(err.Error(), "SSE-C key must be base64-encoded") {
		t.Fatalf("thumbnail workflow ignored invalid SSE-C key: %v", err)
	}
}
