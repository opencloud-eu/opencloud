package opensearch

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddOpenExtensionSource(t *testing.T) {
	const prefix = "http://opencloud.eu/ns/extensions/com.example.project/"
	body := map[string]any{"Name": "a.txt"}
	addOpenExtensionSource(body, map[string]string{
		prefix + "state":    "s:Open",
		prefix + "priority": "n:3",
		prefix + "tags":     `S:["A","b"]`,
		prefix + "due":      "d:2026-10-01T00:00:00Z",
		prefix + "site":     "g:52.5,13.4",
	})

	got, err := json.Marshal(body)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"Name": "a.txt",
		"OpenExtensions": {
			"`+prefix+`state":    "s:Open",
			"`+prefix+`priority": "n:3",
			"`+prefix+`tags":     "S:[\"A\",\"b\"]",
			"`+prefix+`due":      "d:2026-10-01T00:00:00Z",
			"`+prefix+`site":     "g:52.5,13.4"
		},
		"ext": {"com": {"example": {"project": {
			"state":    {"@keyword": "Open", "@lower": "open"},
			"priority": {"@number": 3},
			"tags":     {"@keyword": ["A", "b"], "@lower": ["a", "b"]},
			"due":      {"@date": "2026-10-01T00:00:00Z"},
			"site":     {"@geo": {"lat": 52.5, "lon": 13.4}}
		}}}}
	}`, string(got))
}

func TestAddOpenExtensionSourceWithoutExtensions(t *testing.T) {
	body := map[string]any{"Name": "a.txt"}
	addOpenExtensionSource(body, nil)
	require.Equal(t, map[string]any{"Name": "a.txt"}, body)
}

func TestClassifyOpenExtensionMapping(t *testing.T) {
	templates := func() string {
		b, err := json.Marshal(openExtensionTemplates())
		require.NoError(t, err)
		return string(b)
	}
	local := func() map[string]any {
		return map[string]any{"Name": map[string]any{"type": "keyword"}, "ext": map[string]any{"type": "object", "dynamic": true}}
	}
	// what GET _mapping returns once documents were indexed: the flag as a
	// string and the concrete fields OpenSearch added under ext
	remote := func() map[string]any {
		return map[string]any{
			"Name": map[string]any{"type": "keyword"},
			"ext": map[string]any{"dynamic": "true", "properties": map[string]any{
				"com": map[string]any{"properties": map[string]any{"example": map[string]any{"properties": map[string]any{
					"project": map[string]any{"properties": map[string]any{"priority": map[string]any{"properties": map[string]any{"@number": map[string]any{"type": "double"}}}}},
				}}}},
			}},
		}
	}

	t.Run("concrete fields under ext are not a difference", func(t *testing.T) {
		l, r := local(), remote()
		reasons := classifyOpenExtensionMapping(l, r, templates(), templates())
		require.Empty(t, reasons)
		require.NotContains(t, l, "ext")
		require.NotContains(t, r, "ext")
		require.Equal(t, l, r, "what is left is compared by Classify")
	})

	t.Run("a changed template is breaking", func(t *testing.T) {
		reasons := classifyOpenExtensionMapping(local(), remote(), templates(), `[{"ext_number":{"path_match":"ext.*","match":"@number","mapping":{"type":"long"}}}]`)
		require.Len(t, reasons, 1)
		require.Contains(t, reasons[0], "dynamic_templates changed")
	})

	t.Run("a changed dynamic flag is breaking", func(t *testing.T) {
		r := remote()
		r["ext"].(map[string]any)["dynamic"] = "strict"
		reasons := classifyOpenExtensionMapping(local(), r, templates(), templates())
		require.Len(t, reasons, 1)
		require.Contains(t, reasons[0], "dynamic changed")
	})

	t.Run("an index without ext leaves the addition to Classify", func(t *testing.T) {
		l, r := local(), map[string]any{"Name": map[string]any{"type": "keyword"}}
		reasons := classifyOpenExtensionMapping(l, r, templates(), "")
		require.Empty(t, reasons)
		require.Contains(t, l, "ext")
	})
}
