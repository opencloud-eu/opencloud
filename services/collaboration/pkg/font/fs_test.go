package font_test

import (
	"bytes"
	"errors"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	revaCtx "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/font"
)

// fontsOnDisk is a fonts directory on disk next to a directory outside of it,
// with the font service on top, set up as in the collaboration service.
type fontsOnDisk struct {
	root    string
	outside string
	font    []byte
	service font.Service
}

func newFontsOnDisk(t *testing.T, dir func(base string) string) fontsOnDisk {
	t.Helper()
	// resolve the temp dir itself: on macOS, it is behind a symlink (/var -> /private/var)
	base, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	d := fontsOnDisk{root: filepath.Join(base, "fonts"), outside: filepath.Join(base, "outside")}
	d.font, err = testdata.ReadFile("testdata/arimo-regular.ttf")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(d.outside, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(d.outside, "secret.ttf"), d.font, 0o644))

	configured := d.root
	if dir != nil {
		configured = dir(base)
	}
	fontFS, err := font.NewFS(configured, log.NopLogger())
	require.NoError(t, err)
	d.service, err = font.NewService(
		font.ServiceOptions{}.
			WithFontFS(fontFS).
			WithPreviewText("a").
			WithLogger(log.NopLogger()).
			WithRootURI("http://test.local").
			WithGatewaySelector(newGatewaySelector()),
	)
	require.NoError(t, err)
	return d
}

func (d fontsOnDisk) link(t *testing.T, target, name string) {
	t.Helper()
	require.NoError(t, os.Symlink(target, filepath.Join(d.root, name)))
}

func (d fontsOnDisk) get(id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()
	d.service.GetFont(w, req)
	return w
}

func (d fontsOnDisk) list() string {
	w := httptest.NewRecorder()
	d.service.ListFonts(w, httptest.NewRequest(http.MethodGet, "/", nil))
	return w.Body.String()
}

func (d fontsOnDisk) upload(t *testing.T, filename string) int {
	t.Helper()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	part, err := mw.CreateFormFile("font", filename)
	require.NoError(t, err)
	_, err = part.Write(d.font)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req = req.WithContext(revaCtx.ContextSetUser(req.Context(), &userpb.User{Id: &userpb.UserId{OpaqueId: "admin"}}))
	w := httptest.NewRecorder()
	d.service.UploadFont(w, req)
	return w.Code
}

func (d fontsOnDisk) delete(t *testing.T, id string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req.SetPathValue("id", id)
	req = req.WithContext(revaCtx.ContextSetUser(req.Context(), &userpb.User{Id: &userpb.UserId{OpaqueId: "admin"}}))
	w := httptest.NewRecorder()
	d.service.DeleteFont(w, req)
	return w.Code
}

func TestNewFS(t *testing.T) {
	t.Run("creates a missing directory", func(t *testing.T) {
		d := newFontsOnDisk(t, nil)
		info, err := os.Stat(d.root)
		require.NoError(t, err)
		require.True(t, info.IsDir())
	})

	t.Run("fails if the path is a file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))

		_, err := font.NewFS(path, log.NopLogger())
		require.Error(t, err)
		require.Contains(t, err.Error(), path)
	})
}

func TestService_OnDisk_GetFont(t *testing.T) {
	t.Run("symlink inside", func(t *testing.T) {
		d := newFontsOnDisk(t, nil)
		require.NoError(t, os.WriteFile(filepath.Join(d.root, "arimo.ttf"), d.font, 0o644))
		d.link(t, "arimo.ttf", "link.ttf")

		w := d.get("link.ttf")
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, d.font, w.Body.Bytes())
	})

	t.Run("configured path is a symlink", func(t *testing.T) {
		d := newFontsOnDisk(t, func(base string) string {
			mount := filepath.Join(base, "mount")
			require.NoError(t, os.MkdirAll(filepath.Join(base, "fonts"), 0o755))
			require.NoError(t, os.Symlink(filepath.Join(base, "fonts"), mount))
			return mount
		})
		require.NoError(t, os.WriteFile(filepath.Join(d.root, "arimo.ttf"), d.font, 0o644))

		w := d.get("arimo.ttf")
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, d.font, w.Body.Bytes())
	})

	for _, tt := range []struct {
		name   string
		target func(d fontsOnDisk) string
	}{
		{name: "relative symlink outside", target: func(fontsOnDisk) string { return "../outside/secret.ttf" }},
		{name: "absolute symlink outside", target: func(d fontsOnDisk) string { return filepath.Join(d.outside, "secret.ttf") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := newFontsOnDisk(t, nil)
			d.link(t, tt.target(d), "leak.ttf")

			// the same response as for a missing font
			leak, missing := d.get("leak.ttf"), d.get("missing.ttf")
			require.Equal(t, missing.Code, leak.Code)
			require.Equal(t, missing.Body.String(), leak.Body.String())
			require.NotEqual(t, d.font, leak.Body.Bytes())
		})
	}
}

func TestService_OnDisk_ListFonts(t *testing.T) {
	d := newFontsOnDisk(t, nil)
	require.NoError(t, os.WriteFile(filepath.Join(d.root, "arimo.ttf"), d.font, 0o644))
	d.link(t, "../outside/secret.ttf", "leak.ttf")

	fonts := gjson.Get(d.list(), "fonts.#.file").Array()
	require.Len(t, fonts, 1)
	require.Equal(t, "arimo.ttf", fonts[0].String())
}

func TestService_OnDisk_UploadFont(t *testing.T) {
	t.Run("regular upload", func(t *testing.T) {
		d := newFontsOnDisk(t, nil)

		require.Equal(t, http.StatusOK, d.upload(t, "arimo.ttf"))
		b, err := os.ReadFile(filepath.Join(d.root, "arimo.ttf"))
		require.NoError(t, err)
		require.Equal(t, d.font, b)
	})

	t.Run("through a symlink to a file outside", func(t *testing.T) {
		d := newFontsOnDisk(t, nil)
		require.NoError(t, os.WriteFile(filepath.Join(d.outside, "target.ttf"), []byte("outside content"), 0o644))
		d.link(t, "../outside/target.ttf", "arimo.ttf")

		require.Equal(t, http.StatusInternalServerError, d.upload(t, "arimo.ttf"))
		b, err := os.ReadFile(filepath.Join(d.outside, "target.ttf"))
		require.NoError(t, err)
		require.Equal(t, "outside content", string(b))
	})

	t.Run("through a dangling symlink to outside", func(t *testing.T) {
		d := newFontsOnDisk(t, nil)
		d.link(t, "../outside/new.ttf", "arimo.ttf")

		require.Equal(t, http.StatusInternalServerError, d.upload(t, "arimo.ttf"))
		_, err := os.Lstat(filepath.Join(d.outside, "new.ttf"))
		require.True(t, errors.Is(err, fs.ErrNotExist), "new.ttf must not exist outside")
	})
}

func TestService_OnDisk_DeleteFont(t *testing.T) {
	d := newFontsOnDisk(t, nil)
	d.link(t, "../outside/secret.ttf", "leak.ttf")

	// the symlink itself is removed, never its target
	require.Equal(t, http.StatusOK, d.delete(t, "leak.ttf"))
	_, err := os.Lstat(filepath.Join(d.root, "leak.ttf"))
	require.True(t, errors.Is(err, fs.ErrNotExist))
	_, err = os.Stat(filepath.Join(d.outside, "secret.ttf"))
	require.NoError(t, err)
}
