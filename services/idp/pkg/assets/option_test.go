package assets_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/assets"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/config"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/middleware"
)

func TestNew_OnDisk(t *testing.T) {
	g := gomega.NewWithT(t)
	// resolve the temp dir itself: on macOS, it is behind a symlink (/var -> /private/var)
	base, err := filepath.EvalSymlinks(t.TempDir())
	g.Expect(err).ToNot(gomega.HaveOccurred())
	root := filepath.Join(base, "idp")
	outside := filepath.Join(base, "outside")
	static := filepath.Join(root, "identifier", "static")

	write := func(name, content string) {
		g.Expect(os.MkdirAll(filepath.Dir(name), 0o755)).To(gomega.Succeed())
		g.Expect(os.WriteFile(name, []byte(content), 0o644)).To(gomega.Succeed())
	}
	link := func(target, name string) {
		g.Expect(os.MkdirAll(filepath.Dir(filepath.Join(static, name)), 0o755)).To(gomega.Succeed())
		g.Expect(os.Symlink(target, filepath.Join(static, name))).To(gomega.Succeed())
	}
	write(filepath.Join(outside, "secret.txt"), "outside content")
	write(filepath.Join(outside, "dir", "secret.txt"), "outside content")
	write(filepath.Join(static, "app.js"), "app content")
	link("app.js", "link-inside.js")
	link("../../../outside/secret.txt", "link-relative-outside.txt")
	link(filepath.Join(outside, "secret.txt"), "link-absolute-outside.txt")
	link("../../../outside/dir", "link-dir-outside")
	link("missing.txt", "link-dangling.txt")
	link("link-loop-b", "link-loop-a")
	link("link-loop-a", "link-loop-b")

	mount := filepath.Join(base, "mount")
	g.Expect(os.Symlink(root, mount)).To(gomega.Succeed())

	for _, dir := range []string{root, mount} {
		fSys, err := assets.New(
			assets.Logger(log.NopLogger()),
			assets.Config(&config.Config{Asset: config.Asset{Path: dir}}),
		)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
		handler := middleware.Static("/signin/v1/", fSys, noop.NewTracerProvider())(next)

		for _, tt := range []struct {
			name         string
			url          string
			expectedCode int
			expectedBody string
		}{
			{name: "regular file", url: "/signin/v1/static/app.js", expectedCode: http.StatusOK, expectedBody: "app content"},
			{name: "symlink inside", url: "/signin/v1/static/link-inside.js", expectedCode: http.StatusOK, expectedBody: "app content"},
			{name: "relative symlink outside", url: "/signin/v1/static/link-relative-outside.txt", expectedCode: http.StatusNotFound},
			{name: "absolute symlink outside", url: "/signin/v1/static/link-absolute-outside.txt", expectedCode: http.StatusNotFound},
			{name: "file in a symlinked directory outside", url: "/signin/v1/static/link-dir-outside/secret.txt", expectedCode: http.StatusNotFound},
			{name: "dangling symlink", url: "/signin/v1/static/link-dangling.txt", expectedCode: http.StatusNotFound},
			{name: "missing file", url: "/signin/v1/static/missing.txt", expectedCode: http.StatusNotFound},
			{name: "symlink loop", url: "/signin/v1/static/link-loop-a", expectedCode: http.StatusInternalServerError},
		} {
			t.Run(filepath.Base(dir)+"/"+tt.name, func(t *testing.T) {
				g := gomega.NewWithT(t)

				w := httptest.NewRecorder()
				handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.url, nil))

				g.Expect(w.Code).To(gomega.Equal(tt.expectedCode))
				g.Expect(w.Body.String()).ToNot(gomega.ContainSubstring("outside content"))
				if tt.expectedBody != "" {
					g.Expect(w.Body.String()).To(gomega.Equal(tt.expectedBody))
				}
			})
		}
	}
}

func TestNew_EmbeddedOnly(t *testing.T) {
	g := gomega.NewWithT(t)

	// without a path, only the embedded assets are used: there is no directory to open
	fSys, err := assets.New(assets.Logger(log.NopLogger()), assets.Config(&config.Config{}))
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(fSys).ToNot(gomega.BeNil())
}

func TestNew_UnusablePath(t *testing.T) {
	g := gomega.NewWithT(t)
	path := filepath.Join(t.TempDir(), "file")
	g.Expect(os.WriteFile(path, []byte("x"), 0o644)).To(gomega.Succeed())

	_, err := assets.New(assets.Logger(log.NopLogger()), assets.Config(&config.Config{Asset: config.Asset{Path: path}}))
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(err.Error()).To(gomega.ContainSubstring("not a directory"))
	g.Expect(err.Error()).To(gomega.ContainSubstring(path))
}
