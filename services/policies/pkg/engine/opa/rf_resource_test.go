package opa_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-policy-agent/opa/rego"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/config"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine/opa"
)

var _ = Describe("opa opencloud resource functions", func() {
	Describe("opencloud.resource.download", func() {
		It("downloads reva resources", func() {
			ts := []byte("Lorem Ipsum is simply dummy text of the printing and typesetting")
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write(ts)
			}))
			defer srv.Close()

			r := rego.New(rego.Query(`opencloud.resource.download("`+srv.URL+`")`), opa.RFResourceDownload(srv.Client()))
			rs, err := r.Eval(context.Background())
			Expect(err).ToNot(HaveOccurred())

			data, err := base64.StdEncoding.DecodeString(rs[0].Expressions[0].String())
			Expect(err).ToNot(HaveOccurred())

			Expect(data).To(Equal(ts))

		})

		It("is cut by the engine timeout", func() {
			const (
				timeout = 300 * time.Millisecond
				holds   = 3 * time.Second
			)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()

				select {
				case <-time.After(holds):
				case <-r.Context().Done():
				}
			}))
			defer srv.Close()

			path := filepath.Join(GinkgoT().TempDir(), "download.rego")
			source := "package download\n\nimport future.keywords.if\n\ndefault granted := true\n\ngranted = false if {\n    opencloud.resource.download(input.resource.url)\n}\n"
			Expect(os.WriteFile(path, []byte(source), 0o600)).To(Succeed())

			e, err := opa.NewOPA(timeout, log.NopLogger(), config.Engine{Policies: []string{path}})
			Expect(err).ToNot(HaveOccurred())

			start := time.Now()
			_, _ = e.Evaluate(context.Background(), "data.download.granted", engine.Environment{
				Resource: engine.Resource{URL: srv.URL},
			})
			Expect(time.Since(start)).To(BeNumerically("<", holds/2))
		})

		It("stays undefined on a non ok response", func() {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}))
			defer srv.Close()

			r := rego.New(rego.Query(`opencloud.resource.download("`+srv.URL+`")`), opa.RFResourceDownload(srv.Client()))

			rs, err := r.Eval(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(rs).To(BeEmpty())
		})
	})
})
