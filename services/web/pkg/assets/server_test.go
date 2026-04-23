package assets

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/onsi/gomega"
)

func TestFileServer(t *testing.T) {
	g := gomega.NewWithT(t)
	recorderStatus := func(s int) string {
		return fmt.Sprintf("%03d %s", s, http.StatusText(s))
	}

	{
		s := FileServer(fstest.MapFS{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/foo", nil)
		//defer req.Body.Close()
		s.ServeHTTP(w, req)
		res := w.Result()
		defer res.Body.Close()

		g.Expect(res.Status).To(gomega.Equal(recorderStatus(http.StatusNotFound)))
	}

	for _, tt := range []struct {
		name     string
		url      string
		fs       fstest.MapFS
		expected string
	}{
		{
			name: "not found fallback",
			url:  "/index.txt",
			fs: fstest.MapFS{
				"index.html": &fstest.MapFile{
					Data: []byte("index file content"),
				},
			},
			expected: `<html><head><base href="/"/></head><body>index file content</body></html>`,
		},
		{
			name: "directory fallback",
			url:  "/some-folder",
			fs: fstest.MapFS{
				"some-folder": &fstest.MapFile{
					Mode: fs.ModeDir,
				},
				"index.html": &fstest.MapFile{
					Data: []byte("index file content"),
				},
			},
			expected: `<html><head><base href="/"/></head><body>index file content</body></html>`,
		},
		{
			name: "index.html",
			url:  "/index.html",
			fs: fstest.MapFS{
				"index.html": &fstest.MapFile{
					Data: []byte("index file content"),
				},
			},
			expected: `<html><head><base href="/"/></head><body>index file content</body></html>`,
		},
		{
			name: "oidc-callback.html",
			url:  "/oidc-callback.html",
			fs: fstest.MapFS{
				"index.html": &fstest.MapFile{
					Data: []byte("oidc-callback file content"),
				},
			},
			expected: `<html><head><base href="/"/></head><body>oidc-callback file content</body></html>`,
		},
		{
			name: "oidc-silent-redirect.html",
			url:  "/oidc-silent-redirect.html",
			fs: fstest.MapFS{
				"index.html": &fstest.MapFile{
					Data: []byte("oidc-silent-redirect file content"),
				},
			},
			expected: `<html><head><base href="/"/></head><body>oidc-silent-redirect file content</body></html>`,
		},
		{
			name: "some-file.txt",
			url:  "/some-file.txt",
			fs: fstest.MapFS{
				"some-file.txt": &fstest.MapFile{
					Data: []byte("some file content"),
				},
			},
			expected: "some file content",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.url, nil)
			FileServer(tt.fs).ServeHTTP(w, req)
			res := w.Result()
			defer res.Body.Close()

			g.Expect(res.Status).To(gomega.Equal(recorderStatus(http.StatusOK)))

			data, err := io.ReadAll(res.Body)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(string(data)).To(gomega.Equal(tt.expected))

		})
	}

}
