package theme_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"

	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
	"github.com/opencloud-eu/opencloud/services/web/pkg/theme"
)

func TestHTTP_Get(t *testing.T) {
	primaryFS := fsx.NewMemMapFs()
	fallbackFS := fsx.NewFallbackFS(primaryFS, fsx.NewMemMapFs())

	add := func(filename string, content interface{}) {
		b, err := json.Marshal(content)
		assert.Nil(t, err)

		assert.Nil(t, afero.WriteFile(primaryFS, filename, b, 0644))
	}

	// baseTheme
	add("base/theme.json", map[string]interface{}{
		"base": "base",
	})
	// brandingTheme
	add("_branding/theme.json", map[string]interface{}{
		"_branding": "_branding",
	})

	service, err := theme.NewService(theme.ServiceOptions{}.WithThemeFS(fallbackFS))
	assert.NoError(t, err)

	handlers, err := theme.NewHTTP(theme.HTTPOptions{}.WithService(service))
	assert.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.SetPathValue("id", "base")

	w := httptest.NewRecorder()
	handlers.Get(w, r)

	jsonData := gjson.Parse(w.Body.String())
	// baseTheme
	assert.Equal(t, jsonData.Get("base").String(), "base")
	// brandingTheme
	assert.Equal(t, jsonData.Get("_branding").String(), "_branding")
	// themeDefaults
	assert.Equal(t, jsonData.Get("common.shareRoles."+unifiedrole.UnifiedRoleViewerID+".name").String(), "UnifiedRoleViewer")
}
