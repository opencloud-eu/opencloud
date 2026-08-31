package opa_test

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/config"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine/opa"
)

var _ = Describe("engine", func() {
	newEngine := func(source string) (engine.Engine, string) {
		path := filepath.Join(GinkgoT().TempDir(), "policy.rego")
		Expect(os.WriteFile(path, []byte(source), 0o600)).To(Succeed())

		e, err := opa.NewOPA(10*time.Second, log.NopLogger(), config.Engine{Policies: []string{path}}, nil)
		Expect(err).ToNot(HaveOccurred())

		return e, path
	}

	Context("when the policy files are read", func() {
		var path string

		install := func(name string) {
			source, err := os.ReadFile(filepath.Join("testdata", "rules", name))
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(path, source, 0o600)).To(Succeed())
		}

		start := func(policies ...string) engine.Engine {
			e, err := opa.NewOPA(10*time.Second, log.NopLogger(), config.Engine{Policies: policies}, nil)
			Expect(err).ToNot(HaveOccurred())

			return e
		}

		decide := func(e engine.Engine) bool {
			granted, err := e.Evaluate(context.Background(), "data.rules.granted", engine.Environment{})
			Expect(err).ToNot(HaveOccurred())

			return granted
		}

		BeforeEach(func() {
			path = filepath.Join(GinkgoT().TempDir(), "rules.rego")
			install("granted.rego")
		})

		It("keeps the rule set a rewrite replaced", func() {
			e := start(path)
			Expect(decide(e)).To(BeTrue())

			install("denied.rego")
			Expect(decide(e)).To(BeTrue(), "a rewrite only takes effect on restart")
		})

		It("keeps the rule set after the file disappeared", func() {
			e := start(path)
			Expect(os.Remove(path)).To(Succeed())

			Expect(decide(e)).To(BeTrue())
		})

		It("refuses to start on a broken policy", func() {
			install("broken.rego")

			_, err := opa.NewOPA(10*time.Second, log.NopLogger(), config.Engine{Policies: []string{path}}, nil)
			Expect(err).To(HaveOccurred())
		})

		It("refuses to start when a policy is missing", func() {
			Expect(os.Remove(path)).To(Succeed())

			_, err := opa.NewOPA(10*time.Second, log.NopLogger(), config.Engine{Policies: []string{path}}, nil)
			Expect(err).To(HaveOccurred())
		})

		It("takes a directory as a policy path", func() {
			dir := GinkgoT().TempDir()
			source, err := os.ReadFile(filepath.Join("testdata", "rules", "granted.rego"))
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(filepath.Join(dir, "policy.rego"), source, 0o600)).To(Succeed())

			Expect(decide(start(dir))).To(BeTrue())
		})
	})

	Context("across evaluations", func() {
		It("carries no downloaded content into the next one", func() {
			buf := new(bytes.Buffer)
			Expect(png.Encode(buf, image.NewRGBA(image.Rect(0, 0, 1, 1)))).To(Succeed())

			// same url, different content per request.
			var served atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				body := []byte("plain text, not an image")
				if served.Add(1) == 1 {
					body = buf.Bytes()
				}

				_, err := w.Write(body)
				Expect(err).ToNot(HaveOccurred())
			}))
			defer srv.Close()

			e, _ := newEngine("package isolation\n\nimport future.keywords.if\n\ndefault granted := true\n\ngranted = false if {\n    body := opencloud.resource.download(input.resource.url)\n    opencloud.mimetype.detect(body) == \"image/png\"\n}\n")
			env := engine.Environment{Resource: engine.Resource{URL: srv.URL}}

			first, err := e.Evaluate(context.Background(), "data.isolation.granted", env)
			Expect(err).ToNot(HaveOccurred())
			Expect(first).To(BeFalse(), "png is served first and has to be denied")

			second, err := e.Evaluate(context.Background(), "data.isolation.granted", env)
			Expect(err).ToNot(HaveOccurred())
			Expect(second).To(BeTrue(), "text is served second, seeing the png again means the memo leaked")

			Expect(served.Load()).To(BeEquivalentTo(2), "the second evaluation has to fetch again")
		})

		It("judges each one by its own input", func() {
			e, _ := newEngine("package isolation\n\nimport future.keywords.if\n\ndefault granted := true\n\ngranted = false if {\n    endswith(input.resource.name, \".exe\")\n}\n")

			for _, tc := range []struct {
				name string
				want bool
			}{
				{"notes.txt", true},
				{"virus.exe", false},
				{"notes.txt", true},
				{"virus.exe", false},
			} {
				granted, err := e.Evaluate(context.Background(), "data.isolation.granted", engine.Environment{
					Resource: engine.Resource{Name: tc.name},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(granted).To(Equal(tc.want), "for %s", tc.name)
			}
		})
	})
})
