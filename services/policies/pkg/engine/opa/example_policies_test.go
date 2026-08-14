package opa_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/config"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine/opa"
)

const examplePolicyDir = "../../../../../devtools/deployments/service_policies/policies"

func gzipped(payload string) []byte {
	buf := new(bytes.Buffer)
	w := gzip.NewWriter(buf)
	_, err := w.Write([]byte(payload))
	Expect(err).ToNot(HaveOccurred())
	Expect(w.Close()).To(Succeed())

	return buf.Bytes()
}

var _ = Describe("the shipped example policies", func() {
	var (
		e    engine.Engine
		body []byte
		srv  *httptest.Server
	)

	BeforeEach(func() {
		var err error
		e, err = opa.NewOPA(10*time.Second, log.NopLogger(), config.Engine{Policies: []string{
			filepath.Join(examplePolicyDir, "proxy.rego"),
			filepath.Join(examplePolicyDir, "postprocessing.rego"),
			filepath.Join(examplePolicyDir, "utils.rego"),
		}})
		Expect(err).ToNot(HaveOccurred())

		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write(body)
			Expect(err).ToNot(HaveOccurred())
		}))
		DeferCleanup(srv.Close)
	})

	DescribeTable("data.proxy.granted judges the upload by its name",
		func(method, path, name string, expected bool) {
			granted, err := e.Evaluate(context.Background(), "data.proxy.granted", engine.Environment{
				Stage:    engine.StageHTTP,
				Request:  engine.Request{Method: method, Path: path},
				Resource: engine.Resource{Name: name},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(granted).To(Equal(expected))
		},
		Entry("allowed extension", http.MethodPut, "/remote.php/dav/files/alice/notes.txt", "notes.txt", true),
		Entry("denied extension", http.MethodPut, "/remote.php/dav/files/alice/virus.exe", "virus.exe", false),
		Entry("denied on tus post", http.MethodPost, "/data/upload", "virus.exe", false),
		Entry("unrestricted path", http.MethodPut, "/graph/v1.0/me", "virus.exe", true),
		Entry("unrestricted method", http.MethodGet, "/remote.php/dav/files/alice/virus.exe", "virus.exe", true),
	)

	DescribeTable("data.postprocessing.granted judges the upload by its content",
		func(name string, content func() []byte, expected bool) {
			body = content()

			granted, err := e.Evaluate(context.Background(), "data.postprocessing.granted", engine.Environment{
				Stage:    engine.StagePP,
				Resource: engine.Resource{Name: name, URL: srv.URL},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(granted).To(Equal(expected))
		},
		Entry("allowed mimetype", "image.png", func() []byte {
			buf := new(bytes.Buffer)
			Expect(png.Encode(buf, image.NewRGBA(image.Rect(0, 0, 1, 1)))).To(Succeed())
			return buf.Bytes()
		}, true),
		Entry("denied mimetype", "image.png", func() []byte { return gzipped("payload") }, false),
		Entry("denied extension short circuits before the download", "virus.exe", func() []byte { return nil }, false),
	)
})
