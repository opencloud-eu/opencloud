package font_test

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3Permissions "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	cs3RPC "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	revaCtx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/font"
)

//go:embed testdata/*
var testdata embed.FS

// Helper to create a gatewaySelector
func newGatewaySelector() pool.Selectable[gateway.GatewayAPIClient] {
	gatewayAPIClient := &cs3mocks.GatewayAPIClient{}
	gatewayAPIClient.On("CheckPermission", mock.Anything, mock.Anything).Return(
		&cs3Permissions.CheckPermissionResponse{
			Status: &cs3RPC.Status{
				Code: cs3RPC.Code_CODE_OK,
			},
		}, nil)
	gatewaySelector := pool.GetSelector[gateway.GatewayAPIClient](
		"GatewaySelector",
		"eu.opencloud.api.gateway",
		func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
			return gatewayAPIClient
		},
	)

	return gatewaySelector
}

func TestService_PreviewFont(t *testing.T) {
	testFS := afero.NewMemMapFs()
	svc, err := font.NewService(
		font.ServiceOptions{}.
			WithFontFS(testFS).
			WithPreviewText("a").
			WithLogger(log.NopLogger()).
			WithRootURI("http://test.local").
			WithGatewaySelector(newGatewaySelector()),
	)
	require.NoError(t, err)

	testDataFontB, err := testdata.ReadFile("testdata/arimo-regular.ttf")
	require.NoError(t, err)

	testDataFontPNG, err := testdata.ReadFile("testdata/arimo-regular.png")
	require.NoError(t, err)

	_ = afero.WriteFile(testFS, "arimo-regular.ttf", testDataFontB, 0644)
	defer func() {
		require.NoError(t, testFS.Remove("arimo-regular.ttf"))
	}()

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("id", "arimo-regular.ttf")
	resp := httptest.NewRecorder()
	svc.PreviewFont(resp, req)
	require.Equal(t, resp.Body.Bytes(), testDataFontPNG)
}

func TestService_DeleteFont(t *testing.T) {
	testFS := afero.NewMemMapFs()
	svc, err := font.NewService(
		font.ServiceOptions{}.
			WithFontFS(testFS).
			WithPreviewText("a").
			WithLogger(log.NopLogger()).
			WithRootURI("http://test.local").
			WithGatewaySelector(newGatewaySelector()),
	)
	require.NoError(t, err)

	testDataFontB, err := testdata.ReadFile("testdata/arimo-regular.ttf")
	require.NoError(t, err)

	_ = afero.WriteFile(testFS, "arimo-regular.ttf", testDataFontB, 0644)

	_, err = testFS.Stat("arimo-regular.ttf") // ensure the file exists before deletion
	require.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("id", "arimo-regular.ttf")
	req = req.WithContext(revaCtx.ContextSetUser(req.Context(), &userpb.User{
		Id: &userpb.UserId{
			OpaqueId: "user",
		},
	}))

	resp := httptest.NewRecorder()
	svc.DeleteFont(resp, req)

	_, err = testFS.Stat("arimo-regular.ttf") // ensure the file exists before deletion
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestService_GetFont(t *testing.T) {
	testFS := afero.NewMemMapFs()
	svc, err := font.NewService(
		font.ServiceOptions{}.
			WithFontFS(testFS).
			WithPreviewText("a").
			WithLogger(log.NopLogger()).
			WithRootURI("http://test.local").
			WithGatewaySelector(newGatewaySelector()),
	)
	require.NoError(t, err)

	testDataFontB, err := testdata.ReadFile("testdata/arimo-regular.ttf")
	require.NoError(t, err)

	_ = afero.WriteFile(testFS, "arimo-regular.ttf", testDataFontB, 0644)
	defer func() {
		require.NoError(t, testFS.Remove("arimo-regular.ttf"))
	}()

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("id", "arimo-regular.ttf")
	resp := httptest.NewRecorder()
	svc.GetFont(resp, req)
	require.Equal(t, resp.Body.Bytes(), testDataFontB)
}

func TestService_ListFonts(t *testing.T) {
	testFS := afero.NewMemMapFs()
	svc, err := font.NewService(
		font.ServiceOptions{}.
			WithFontFS(testFS).
			WithPreviewText("a").
			WithLogger(log.NopLogger()).
			WithRootURI("http://test.local").
			WithGatewaySelector(newGatewaySelector()),
	)
	require.NoError(t, err)

	t.Run("no fonts", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		resp := httptest.NewRecorder()
		svc.ListFonts(resp, req)

		jsonData := gjson.Parse(resp.Body.String())
		require.Equal(t, jsonData.Get("fonts").String(), "[]") // empty array, not `null`
	})

	t.Run("with fonts", func(t *testing.T) {
		testDataFontB, err := testdata.ReadFile("testdata/arimo-regular.ttf")
		require.NoError(t, err)

		fontconfigurationB, err := testdata.ReadFile("testdata/fontconfiguration.json")
		require.NoError(t, err)

		_ = afero.WriteFile(testFS, "arimo-regular.ttf", testDataFontB, 0644)
		defer func() {
			require.NoError(t, testFS.Remove("arimo-regular.ttf"))
		}()

		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		resp := httptest.NewRecorder()
		svc.ListFonts(resp, req)

		jsonData := gjson.Parse(resp.Body.String())
		require.JSONEq(t, jsonData.String(), string(fontconfigurationB))
	})
}

func TestService_UploadFont(t *testing.T) {
	testFS := afero.NewMemMapFs()
	svc, err := font.NewService(
		font.ServiceOptions{}.
			WithFontFS(testFS).
			WithPreviewText("a").
			WithLogger(log.NopLogger()).
			WithRootURI("http://test.local").
			WithGatewaySelector(newGatewaySelector()),
	)
	require.NoError(t, err)

	testDataFontB, err := testdata.ReadFile("testdata/arimo-regular.ttf")
	require.NoError(t, err)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, _ := w.CreateFormFile("font", "arimo-regular.ttf")
	_, _ = part.Write(testDataFontB)
	_ = w.Close()

	req, _ := http.NewRequest(http.MethodPost, "/", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req = req.WithContext(revaCtx.ContextSetUser(req.Context(), &userpb.User{
		Id: &userpb.UserId{
			OpaqueId: "user",
		},
	}))
	resp := httptest.NewRecorder()

	svc.UploadFont(resp, req)
	testFSFontF, err := testFS.Open("arimo-regular.ttf")
	require.NoError(t, err)

	testFSFontB, err := io.ReadAll(testFSFontF)
	require.NoError(t, err)
	require.Equal(t, testDataFontB, testFSFontB)
}
