package theme_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"

	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
	"github.com/opencloud-eu/opencloud/services/web"
	"github.com/opencloud-eu/opencloud/services/web/pkg/theme"
)

func TestNewService(t *testing.T) {
	t.Run("fails if the options are invalid", func(t *testing.T) {
		_, err := theme.NewService(theme.ServiceOptions{})
		assert.Error(t, err)
	})

	t.Run("success if the options are valid", func(t *testing.T) {
		_, err := theme.NewService(
			theme.ServiceOptions{}.
				WithThemeFS(fsx.NewFallbackFS(fsx.NewMemMapFs(), fsx.NewMemMapFs())).
				WithGatewaySelector(mocks.NewSelectable[gateway.GatewayAPIClient](t)),
		)
		assert.NoError(t, err)
	})
}

func TestService_Get(t *testing.T) {
	primaryFS := fsx.NewMemMapFs()
	fallbackFS := fsx.NewFallbackFS(primaryFS, fsx.NewMemMapFs())

	add := func(filename string, content any) {
		b, err := json.Marshal(content)
		assert.Nil(t, err)

		assert.Nil(t, afero.WriteFile(primaryFS, filename, b, 0644))
	}

	// baseTheme
	add("base/theme.json", map[string]any{
		"base": "base",
	})
	// brandingTheme
	add("_branding/theme.json", map[string]any{
		"_branding": "_branding",
	})

	service, _ := theme.NewService(
		theme.ServiceOptions{}.
			WithThemeFS(fallbackFS).
			WithGatewaySelector(mocks.NewSelectable[gateway.GatewayAPIClient](t)),
	)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.SetPathValue("id", "base")

	w := httptest.NewRecorder()
	service.Get(w, r)

	jsonData := gjson.Parse(w.Body.String())
	// baseTheme
	assert.Equal(t, jsonData.Get("base").String(), "base")
	// brandingTheme
	assert.Equal(t, jsonData.Get("_branding").String(), "_branding")
	// themeDefaults
	assert.Equal(t, "Viewer", jsonData.Get("common.shareRoles."+unifiedrole.UnifiedRoleViewerID+".label").String())
	assert.False(t, jsonData.Get("common.shareRoles."+unifiedrole.UnifiedRoleViewerID+".name").Exists())
}

func TestService_Get_BuiltInThemeShareRoleLabels(t *testing.T) {
	service, err := theme.NewService(
		theme.ServiceOptions{}.
			WithThemeFS(fsx.NewFallbackFS(fsx.NewMemMapFs(), fsx.NewBasePathFs(fsx.FromIOFS(web.Assets), "assets/themes"))).
			WithGatewaySelector(mocks.NewSelectable[gateway.GatewayAPIClient](t)),
	)
	assert.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.SetPathValue("id", "opencloud")
	w := httptest.NewRecorder()
	service.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	shareRoles := gjson.Get(w.Body.String(), "common.shareRoles")
	assert.True(t, shareRoles.IsObject())
	shareRoles.ForEach(func(id, role gjson.Result) bool {
		assert.NotEmpty(t, unifiedrole.GetUnifiedRoleLabel(id.String()), "unknown role %s", id.String())
		assert.Equal(t, unifiedrole.GetUnifiedRoleLabel(id.String()), role.Get("label").String(), "label of role %s", id.String())
		assert.False(t, role.Get("name").Exists(), "role %s has a name key", id.String())
		return true
	})
}

func TestService_GetConcurrent(t *testing.T) {
	primaryFS := fsx.NewMemMapFs()
	fallbackFS := fsx.NewFallbackFS(primaryFS, fsx.NewMemMapFs())

	add := func(filename string, content any) {
		b, err := json.Marshal(content)
		assert.Nil(t, err)

		assert.Nil(t, afero.WriteFile(primaryFS, filename, b, 0644))
	}

	// the base themes overlap with the defaults, like the shipped theme.json does
	add("a/theme.json", map[string]any{
		"common": map[string]any{
			"name": "a",
			"logo": "a.svg",
			"shareRoles": map[string]any{
				unifiedrole.UnifiedRoleViewerID: map[string]any{"iconName": "a"},
			},
		},
	})
	add("b/theme.json", map[string]any{
		"common": map[string]any{
			"name": "b",
		},
	})
	add("_branding/theme.json", map[string]any{
		"common": map[string]any{
			"slogan": "branding",
		},
	})

	service, _ := theme.NewService(
		theme.ServiceOptions{}.
			WithThemeFS(fallbackFS).
			WithGatewaySelector(mocks.NewSelectable[gateway.GatewayAPIClient](t)),
	)

	get := func(id string) gjson.Result {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.SetPathValue("id", id)
		w := httptest.NewRecorder()
		service.Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		return gjson.Parse(w.Body.String())
	}

	var wg sync.WaitGroup
	for i := range 50 {
		id := []string{"a", "b"}[i%2]
		wg.Add(1)
		go func() {
			defer wg.Done()
			jsonData := get(id)
			assert.Equal(t, id, jsonData.Get("common.name").String())
			assert.Equal(t, "branding", jsonData.Get("common.slogan").String())
		}()
	}
	wg.Wait()

	// values of theme a must never leak into the defaults and therefore into theme b
	jsonData := get("b")
	assert.Equal(t, "b", jsonData.Get("common.name").String())
	assert.False(t, jsonData.Get("common.logo").Exists())
	assert.Equal(t, "eye", jsonData.Get("common.shareRoles."+unifiedrole.UnifiedRoleViewerID+".iconName").String())
}
