package assets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"testing/fstest"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/onsi/gomega"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
)

func TestFileServer(t *testing.T) {
	g := gomega.NewWithT(t)
	recorderStatus := func(s int) string {
		return fmt.Sprintf("%03d %s", s, http.StatusText(s))
	}

	{
		s := FileServer(log.NopLogger(), fstest.MapFS{})
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
			FileServer(log.NopLogger(), tt.fs).ServeHTTP(w, req)
			res := w.Result()
			defer res.Body.Close()

			g.Expect(res.Status).To(gomega.Equal(recorderStatus(http.StatusOK)))

			data, err := io.ReadAll(res.Body)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(string(data)).To(gomega.Equal(tt.expected))

		})
	}

}

// limitedFS refuses to open more than max files, like a process that has
// run out of file descriptors, and counts how often Open was called.
type limitedFS struct {
	http.FileSystem
	max   int
	opens int
}

func (l *limitedFS) Open(name string) (http.File, error) {
	l.opens++
	if l.opens > l.max {
		return nil, fs.ErrPermission
	}
	return l.FileSystem.Open(name)
}

func TestFileServerIndexIsDirectory(t *testing.T) {
	g := gomega.NewWithT(t)

	fsys := &limitedFS{
		FileSystem: http.FS(fstest.MapFS{
			"index.html": &fstest.MapFile{Mode: fs.ModeDir},
		}),
		max: 100,
	}

	w := httptest.NewRecorder()
	(&fileServer{fsys: fsys, logger: log.NopLogger()}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/missing", nil))

	g.Expect(w.Code).To(gomega.Equal(http.StatusNotFound))
	g.Expect(fsys.opens).To(gomega.BeNumerically("<=", 3), "the index fallback must not call itself again")
}

// faultyFS returns files whose Stat or Read fail, for the names given.
type faultyFS struct {
	http.FileSystem
	openErr map[string]error
	statErr map[string]bool
	readErr map[string]bool
}

type faultyFile struct {
	http.File
	statErr bool
	readErr bool
}

func (f faultyFile) Stat() (fs.FileInfo, error) {
	if f.statErr {
		return nil, syscall.EIO
	}
	return f.File.Stat()
}

func (f faultyFile) Read(p []byte) (int, error) {
	if f.readErr {
		n, _ := f.File.Read(p[:min(len(p), 4)])
		if n > 0 {
			return n, nil
		}
		return 0, io.ErrUnexpectedEOF
	}
	return f.File.Read(p)
}

func (f faultyFS) Open(name string) (http.File, error) {
	if err := f.openErr[name]; err != nil {
		return nil, err
	}
	file, err := f.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}
	return faultyFile{File: file, statErr: f.statErr[name], readErr: f.readErr[name]}, nil
}

type logEntry struct {
	level    string
	path     string
	filename string
}

// logEntries returns the level and the path of each log entry, and checks
// that each one carries the request id.
func logEntries(g *gomega.WithT, buf *bytes.Buffer, requestID string) []logEntry {
	var entries []logEntry
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var e map[string]any
		g.Expect(json.Unmarshal([]byte(line), &e)).To(gomega.Succeed())
		g.Expect(e).To(gomega.HaveKeyWithValue(log.RequestIDString, requestID), line)
		entry := logEntry{level: fmt.Sprint(e["level"])}
		if p, ok := e["path"]; ok {
			entry.path = fmt.Sprint(p)
		}
		if f, ok := e["filename"]; ok {
			entry.filename = fmt.Sprint(f)
		}
		entries = append(entries, entry)
	}
	return entries
}

func TestFileServerFaultyFiles(t *testing.T) {
	// Match log.NewLogger's global level, the loggers of the test cases filter on their own.
	previousLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(previousLevel) })

	mapFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("index file content")},
		"app.js":     &fstest.MapFile{Data: []byte("console.log('complete file')")},
	}
	index := `<html><head><base href="/"/></head><body>index file content</body></html>`

	for _, tt := range []struct {
		name         string
		url          string
		openErr      map[string]error
		statErr      []string
		readErr      []string
		expectedCode int
		expectedBody string
		// the cache headers after the request, when Static has set them before
		expectedCacheControl string
		expectedLogs         []logEntry
	}{
		{
			name:                 "missing file falls back to the index without a log entry",
			url:                  "/missing",
			expectedCode:         http.StatusOK,
			expectedBody:         index,
			expectedCacheControl: "no-cache",
		},
		{
			name:                 "name too long falls back to the index without a log entry",
			url:                  "/long",
			openErr:              map[string]error{"/long": &fs.PathError{Op: "open", Path: "long", Err: syscall.ENAMETOOLONG}},
			expectedCode:         http.StatusOK,
			expectedBody:         index,
			expectedCacheControl: "no-cache",
		},
		{
			// a NUL byte in the name is checked before opening; EINVAL from the filesystem is a server error.
			name:                 "EINVAL without a NUL byte in the name",
			url:                  "/invalid",
			openErr:              map[string]error{"/invalid": &fs.PathError{Op: "open", Path: "invalid", Err: syscall.EINVAL}},
			expectedCode:         http.StatusInternalServerError,
			expectedBody:         "Internal Server Error\n",
			expectedCacheControl: "no-store",
			expectedLogs:         []logEntry{{level: "error", path: "/invalid", filename: "/invalid"}},
		},
		{
			name:                 "invalid fs path falls back to the index without a log entry",
			url:                  "/invalid",
			openErr:              map[string]error{"/invalid": &fs.PathError{Op: "open", Path: "invalid", Err: fs.ErrInvalid}},
			expectedCode:         http.StatusOK,
			expectedBody:         index,
			expectedCacheControl: "no-cache",
		},
		{
			name:         "missing index",
			url:          "/missing",
			openErr:      map[string]error{"/index.html": &fs.PathError{Op: "open", Path: "index.html", Err: fs.ErrNotExist}},
			expectedCode: http.StatusNotFound,
			expectedLogs: []logEntry{{level: "debug", path: "/missing"}},
		},
		{
			name:                 "open error on an asset",
			url:                  "/app.js",
			openErr:              map[string]error{"/app.js": &fs.PathError{Op: "open", Path: "app.js", Err: fs.ErrPermission}},
			expectedCode:         http.StatusInternalServerError,
			expectedBody:         "Internal Server Error\n",
			expectedCacheControl: "no-store",
			expectedLogs:         []logEntry{{level: "error", path: "/app.js", filename: "/app.js"}},
		},
		{
			name:                 "open error on the index fallback",
			url:                  "/missing",
			openErr:              map[string]error{"/index.html": &fs.PathError{Op: "open", Path: "index.html", Err: fs.ErrPermission}},
			expectedCode:         http.StatusInternalServerError,
			expectedBody:         "Internal Server Error\n",
			expectedCacheControl: "no-store",
			expectedLogs:         []logEntry{{level: "error", path: "/missing", filename: "/index.html"}},
		},
		{
			name:                 "stat error on an asset",
			url:                  "/app.js",
			statErr:              []string{"/app.js"},
			expectedCode:         http.StatusInternalServerError,
			expectedBody:         "Internal Server Error\n",
			expectedCacheControl: "no-store",
			expectedLogs:         []logEntry{{level: "error", path: "/app.js", filename: "/app.js"}},
		},
		{
			name:                 "stat error on the index fallback",
			url:                  "/missing",
			statErr:              []string{"/index.html"},
			expectedCode:         http.StatusInternalServerError,
			expectedBody:         "Internal Server Error\n",
			expectedCacheControl: "no-store",
			expectedLogs:         []logEntry{{level: "error", path: "/missing", filename: "/index.html"}},
		},
		{
			name:                 "read error on an asset",
			url:                  "/app.js",
			readErr:              []string{"/app.js"},
			expectedCode:         http.StatusInternalServerError,
			expectedBody:         "Internal Server Error\n",
			expectedCacheControl: "no-store",
			expectedLogs:         []logEntry{{level: "error", path: "/app.js", filename: "/app.js"}},
		},
		{
			name:                 "read error on the index",
			url:                  "/index.html",
			readErr:              []string{"/index.html"},
			expectedCode:         http.StatusInternalServerError,
			expectedBody:         "Internal Server Error\n",
			expectedCacheControl: "no-store",
			expectedLogs:         []logEntry{{level: "error", path: "/index.html", filename: "/index.html"}},
		},
		{
			name:                 "read error on the index fallback",
			url:                  "/missing",
			readErr:              []string{"/index.html"},
			expectedCode:         http.StatusInternalServerError,
			expectedBody:         "Internal Server Error\n",
			expectedCacheControl: "no-store",
			expectedLogs:         []logEntry{{level: "error", path: "/missing", filename: "/index.html"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			fsys := faultyFS{FileSystem: http.FS(mapFS), openErr: tt.openErr, statErr: map[string]bool{}, readErr: map[string]bool{}}
			for _, n := range tt.statErr {
				fsys.statErr[n] = true
			}
			for _, n := range tt.readErr {
				fsys.readErr[n] = true
			}

			buf := &bytes.Buffer{}
			server := &fileServer{fsys: fsys, logger: log.Logger{Logger: zerolog.New(buf).Level(zerolog.DebugLevel)}}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req = req.WithContext(context.WithValue(req.Context(), chimiddleware.RequestIDKey, "test-request-id"))

			w := httptest.NewRecorder()
			w.Header().Set("Cache-Control", "max-age=60")
			w.Header().Set("Expires", "Thu, 01 Jan 2099 00:00:00 GMT")
			g.Expect(func() { server.ServeHTTP(w, req) }).ToNot(gomega.Panic())

			g.Expect(w.Code).To(gomega.Equal(tt.expectedCode))
			if tt.expectedBody != "" {
				g.Expect(w.Body.String()).To(gomega.Equal(tt.expectedBody))
			}
			if tt.expectedCacheControl != "" {
				g.Expect(w.Header().Get("Cache-Control")).To(gomega.Equal(tt.expectedCacheControl))
				g.Expect(w.Header().Get("Expires")).To(gomega.BeEmpty())
			}
			g.Expect(logEntries(g, buf, "test-request-id")).To(gomega.Equal(tt.expectedLogs))
		})
	}
}

func TestFileServerOnDisk(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "core")
	g := gomega.NewWithT(t)
	g.Expect(os.MkdirAll(filepath.Join(root, "js"), 0o755)).To(gomega.Succeed())
	g.Expect(os.WriteFile(filepath.Join(base, "outside.txt"), []byte("outside content"), 0o644)).To(gomega.Succeed())
	g.Expect(os.WriteFile(filepath.Join(root, "index.html"), []byte("index file content"), 0o644)).To(gomega.Succeed())
	g.Expect(os.WriteFile(filepath.Join(root, "js", "app.js"), []byte("app content"), 0o644)).To(gomega.Succeed())

	// the same layering as the web service: a directory on disk with the embedded assets as fallback
	server := FileServer(log.NopLogger(), fsx.NewFallbackFS(
		fsx.NewBasePathFs(fsx.NewOsFs(), root),
		fsx.NewMemMapFs(),
	).IOFS())

	index := `<html><head><base href="/"/></head><body>index file content</body></html>`
	for _, tt := range []struct {
		name         string
		url          string
		expectedBody string
	}{
		{name: "regular file", url: "/js/app.js", expectedBody: "app content"},
		{name: "deep link", url: "/files/spaces/personal/admin", expectedBody: index},
		{name: "directory", url: "/js", expectedBody: index},
		{name: "parent directory", url: "/../outside.txt", expectedBody: index},
		{name: "encoded parent directory", url: "/%2e%2e/outside.txt", expectedBody: index},
		{name: "relative path", url: "js/app.js", expectedBody: "app content"},
		{name: "name too long", url: "/files/spaces/personal/admin/" + strings.Repeat("文", 100), expectedBody: index},
		{name: "nul byte", url: "/files/a%00b", expectedBody: index},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			// set the path directly: a path without a leading slash is no valid request target
			u, err := url.Parse(tt.url)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			req.URL.Path = u.Path
			server.ServeHTTP(w, req)

			g.Expect(w.Code).To(gomega.Equal(http.StatusOK))
			g.Expect(w.Body.String()).To(gomega.Equal(tt.expectedBody))
		})
	}
}

func TestFileServerOnDiskUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read files without read permission")
	}
	g := gomega.NewWithT(t)

	root := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(root, "index.html"), []byte("index file content"), 0o644)).To(gomega.Succeed())
	g.Expect(os.WriteFile(filepath.Join(root, "app.js"), []byte("app content"), 0o000)).To(gomega.Succeed())

	server := FileServer(log.NopLogger(), fsx.NewFallbackFS(
		fsx.NewBasePathFs(fsx.NewOsFs(), root),
		fsx.NewMemMapFs(),
	).IOFS())

	w := httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/app.js", nil))

	g.Expect(w.Code).To(gomega.Equal(http.StatusInternalServerError))
	g.Expect(w.Body.String()).ToNot(gomega.ContainSubstring("index file content"))
}
