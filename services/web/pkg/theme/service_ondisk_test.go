package theme_test

import (
	"bytes"
	"errors"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3permissions "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	revactx "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v3/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/web/pkg/theme"
)

// themesOnDisk is a themes directory on disk next to a directory outside of it,
// with the theme service on top, layered as in the web service.
type themesOnDisk struct {
	root    string
	outside string
	service theme.Service
}

func newThemesOnDisk(t *testing.T) themesOnDisk {
	t.Helper()
	// resolve the temp dir itself: on macOS, it is behind a symlink (/var -> /private/var)
	base, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	d := themesOnDisk{root: filepath.Join(base, "themes"), outside: filepath.Join(base, "outside")}
	require.NoError(t, os.MkdirAll(d.root, 0o755))
	require.NoError(t, os.MkdirAll(d.outside, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(d.outside, "logo.png"), []byte("outside logo"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(d.outside, "theme.json"), []byte(`{"common":{"name":"outside theme"}}`), 0o644))

	gatewayClient := &cs3mocks.GatewayAPIClient{}
	gatewayClient.On("CheckPermission", mock.Anything, mock.Anything).Return(
		&cs3permissions.CheckPermissionResponse{Status: &cs3rpc.Status{Code: cs3rpc.Code_CODE_OK}}, nil)
	gatewaySelector := pool.GetSelector(
		"GatewaySelector",
		"eu.opencloud.api.gateway",
		func(grpc.ClientConnInterface) gateway.GatewayAPIClient { return gatewayClient },
	)

	rootFS, err := fsx.NewRootFs(d.root, log.NopLogger())
	require.NoError(t, err)
	d.service, err = theme.NewService(
		theme.ServiceOptions{}.
			WithThemeFS(fsx.NewFallbackFS(rootFS, fsx.NewMemMapFs())).
			WithGatewaySelector(gatewaySelector),
	)
	require.NoError(t, err)
	return d
}

func (d themesOnDisk) link(t *testing.T, target, name string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(d.root, name)), 0o755))
	require.NoError(t, os.Symlink(target, filepath.Join(d.root, name)))
}

func (d themesOnDisk) upload(t *testing.T, filename, content string) int {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="logo"; filename="`+filename+`"`)
	h.Set("Content-Type", "image/png")
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req = req.WithContext(revactx.ContextSetUser(req.Context(), &userpb.User{Id: &userpb.UserId{OpaqueId: "admin"}}))
	rec := httptest.NewRecorder()
	d.service.LogoUpload(rec, req)
	return rec.Code
}

func (d themesOnDisk) reset(t *testing.T) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(revactx.ContextSetUser(req.Context(), &userpb.User{Id: &userpb.UserId{OpaqueId: "admin"}}))
	rec := httptest.NewRecorder()
	d.service.LogoReset(rec, req)
	return rec.Code
}

func (d themesOnDisk) get(t *testing.T, id string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	d.service.Get(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	return rec.Body.String()
}

func (d themesOnDisk) readOutside(t *testing.T, name string) string {
	t.Helper()
	b, err := fs.ReadFile(os.DirFS(d.outside), name)
	require.NoError(t, err)
	return string(b)
}

func (d themesOnDisk) requireNoOutside(t *testing.T, name string) {
	t.Helper()
	_, err := os.Lstat(filepath.Join(d.outside, name))
	require.True(t, errors.Is(err, fs.ErrNotExist), "%s must not exist outside", name)
}

func TestService_OnDisk_LogoUpload(t *testing.T) {
	t.Run("regular upload", func(t *testing.T) {
		d := newThemesOnDisk(t)

		require.Equal(t, http.StatusOK, d.upload(t, "logo.png", "new logo"))
		b, err := fs.ReadFile(os.DirFS(d.root), "_branding/logo.png")
		require.NoError(t, err)
		require.Equal(t, "new logo", string(b))
		require.Equal(t, "themes/_branding/logo.png", gjson.Get(d.get(t, "opencloud"), "common.logo").String())
	})

	t.Run("through a symlink inside", func(t *testing.T) {
		d := newThemesOnDisk(t)
		require.NoError(t, os.MkdirAll(filepath.Join(d.root, "shared"), 0o755))
		d.link(t, "../shared/logo.png", "_branding/logo.png")

		require.Equal(t, http.StatusOK, d.upload(t, "logo.png", "new logo"))
		b, err := fs.ReadFile(os.DirFS(d.root), "shared/logo.png")
		require.NoError(t, err)
		require.Equal(t, "new logo", string(b))
	})

	t.Run("through a symlink to a file outside", func(t *testing.T) {
		d := newThemesOnDisk(t)
		d.link(t, "../../outside/logo.png", "_branding/logo.png")

		require.Equal(t, http.StatusInternalServerError, d.upload(t, "logo.png", "new logo"))
		require.Equal(t, "outside logo", d.readOutside(t, "logo.png"))
	})

	t.Run("through a dangling symlink to outside", func(t *testing.T) {
		d := newThemesOnDisk(t)
		d.link(t, "../../outside/new.png", "_branding/logo.png")

		require.Equal(t, http.StatusInternalServerError, d.upload(t, "logo.png", "new logo"))
		d.requireNoOutside(t, "new.png")
	})

	t.Run("into a symlinked directory outside", func(t *testing.T) {
		d := newThemesOnDisk(t)
		d.link(t, "../outside", "_branding")

		require.Equal(t, http.StatusInternalServerError, d.upload(t, "new.png", "new logo"))
		d.requireNoOutside(t, "new.png")
		require.Equal(t, `{"common":{"name":"outside theme"}}`, d.readOutside(t, "theme.json"))
	})

	t.Run("with theme.json as a symlink to outside", func(t *testing.T) {
		d := newThemesOnDisk(t)
		d.link(t, "../../outside/theme.json", "_branding/theme.json")

		require.Equal(t, http.StatusInternalServerError, d.upload(t, "logo.png", "new logo"))
		require.Equal(t, `{"common":{"name":"outside theme"}}`, d.readOutside(t, "theme.json"))

		require.Equal(t, http.StatusInternalServerError, d.reset(t))
		require.Equal(t, `{"common":{"name":"outside theme"}}`, d.readOutside(t, "theme.json"))
	})

	t.Run("with a parent directory in the file name", func(t *testing.T) {
		d := newThemesOnDisk(t)

		require.Equal(t, http.StatusOK, d.upload(t, "../../outside/new.png", "new logo"))
		d.requireNoOutside(t, "new.png")
		// mime/multipart drops the directories of the file name
		_, err := os.Stat(filepath.Join(d.root, "_branding", "new.png"))
		require.NoError(t, err)
	})
}

func TestService_OnDisk_Get(t *testing.T) {
	t.Run("configmap symlink chain inside", func(t *testing.T) {
		d := newThemesOnDisk(t)
		data := filepath.Join(d.root, "custom", "..2026_11_27_00_00_00.000000000")
		require.NoError(t, os.MkdirAll(data, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(data, "theme.json"), []byte(`{"common":{"name":"custom theme"}}`), 0o644))
		d.link(t, "..2026_11_27_00_00_00.000000000", "custom/..data")
		d.link(t, "..data/theme.json", "custom/theme.json")

		require.Equal(t, "custom theme", gjson.Get(d.get(t, "custom"), "common.name").String())
	})

	t.Run("theme.json as a symlink to outside", func(t *testing.T) {
		d := newThemesOnDisk(t)
		d.link(t, "../../outside/theme.json", "custom/theme.json")
		d.link(t, "../../outside/theme.json", "_branding/theme.json")

		require.NotContains(t, d.get(t, "custom"), "outside theme")
	})
}
