package theme_test

import (
	"encoding/json"
	"strconv"
	"sync"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/web/pkg/theme"
)

func TestMergeKV(t *testing.T) {
	left := theme.KV{
		"left": "left",
		"both": "left",
	}
	right := theme.KV{
		"right": "right",
		"both":  "right",
	}

	result, err := theme.MergeKV(left, right)
	assert.Nil(t, err)
	assert.Equal(t, result, theme.KV{
		"left":  "left",
		"right": "right",
		"both":  "right",
	})
}

func TestMergeKVDoesNotMutateInputs(t *testing.T) {
	defaults := theme.KV{
		"common": theme.KV{
			"name": "default",
			"shareRoles": theme.KV{
				"viewer": theme.KV{"iconName": "eye"},
			},
		},
		"list": []any{theme.KV{"a": "a"}},
	}
	override := theme.KV{
		"common": map[string]any{
			"logo": "override.svg",
			"shareRoles": map[string]any{
				"viewer": map[string]any{"iconName": "glasses"},
			},
		},
	}

	result, err := theme.MergeKV(defaults, override)
	assert.Nil(t, err)
	assert.Equal(t, theme.KV{
		"common": theme.KV{
			"name": "default",
			"logo": "override.svg",
			"shareRoles": theme.KV{
				"viewer": theme.KV{"iconName": "glasses"},
			},
		},
		"list": []any{theme.KV{"a": "a"}},
	}, result)

	// the inputs must be left untouched by the merge
	assert.Equal(t, theme.KV{
		"common": theme.KV{
			"name": "default",
			"shareRoles": theme.KV{
				"viewer": theme.KV{"iconName": "eye"},
			},
		},
		"list": []any{theme.KV{"a": "a"}},
	}, defaults)

	// changing the result must not leak back into the inputs
	result["common"].(theme.KV)["name"] = "changed"
	result["list"].([]any)[0].(theme.KV)["a"] = "changed"
	assert.Equal(t, "default", defaults["common"].(theme.KV)["name"])
	assert.Equal(t, "a", defaults["list"].([]any)[0].(theme.KV)["a"])
	assert.Equal(t, "override.svg", override["common"].(map[string]any)["logo"])

	// a later merge must not see values from an earlier one
	result, err = theme.MergeKV(defaults, theme.KV{})
	assert.Nil(t, err)
	assert.NotContains(t, result["common"], "logo")
}

func TestMergeKVConcurrent(t *testing.T) {
	defaults := theme.KV{
		"common": theme.KV{
			"name": "default",
			"shareRoles": theme.KV{
				"viewer": theme.KV{"iconName": "eye"},
			},
		},
	}

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			name := strconv.Itoa(i)
			result, err := theme.MergeKV(defaults, theme.KV{
				"common": map[string]any{
					"name": name,
					"shareRoles": map[string]any{
						"viewer": map[string]any{"iconName": name},
					},
				},
			})
			assert.Nil(t, err)
			assert.Equal(t, name, result["common"].(theme.KV)["name"])
		}()
	}
	wg.Wait()

	assert.Equal(t, "default", defaults["common"].(theme.KV)["name"])
}

func TestPatchKV(t *testing.T) {
	in := theme.KV{
		"a": map[string]interface{}{
			"value": "a",
		},
		"b": map[string]interface{}{
			"value": "b",
		},
	}
	out := theme.PatchKV(in, theme.KV{
		"b.value":          "b-new",
		"c.value":          "c-new",
		"d":                "d-new",
		"e.value.subvalue": "e-new",
	})
	assert.Equal(t, theme.KV{
		"a": map[string]interface{}{
			"value": "a",
		},
		"b": map[string]interface{}{
			"value": "b-new",
		},
		"c": map[string]interface{}{
			"value": "c-new",
		},
		"d": "d-new",
		"e": map[string]interface{}{
			"value": map[string]interface{}{
				"subvalue": "e-new",
			},
		},
	}, out)
}

func TestPatchKVUnset(t *testing.T) {
	in := theme.KV{
		"a": map[string]interface{}{
			"value": "a",
		},
		"b": map[string]interface{}{
			"value": "b",
		},
	}
	out := theme.PatchKV(in, theme.KV{
		"a.value": nil,
		"b":       nil,
	})
	assert.Equal(t, theme.KV{
		"a": map[string]interface{}{},
	}, out)
}

func TestPatchKVwithNil(t *testing.T) {
	var in theme.KV
	out := theme.PatchKV(in, theme.KV{
		"b.value":          "b-new",
		"c.value":          "c-new",
		"d":                "d-new",
		"e.value.subvalue": "e-new",
	})
	assert.Equal(t, theme.KV{
		"b": map[string]interface{}{
			"value": "b-new",
		},
		"c": map[string]interface{}{
			"value": "c-new",
		},
		"d": "d-new",
		"e": map[string]interface{}{
			"value": map[string]interface{}{
				"subvalue": "e-new",
			},
		},
	}, out)
}

func TestLoadKV(t *testing.T) {
	in := theme.KV{
		"a": map[string]interface{}{
			"value": "a",
		},
		"b": map[string]interface{}{
			"value": "b",
		},
	}
	b, err := json.Marshal(in)
	assert.Nil(t, err)

	fsys := fsx.NewMemMapFs()
	assert.Nil(t, afero.WriteFile(fsys, "some.json", b, 0644))

	out, err := theme.LoadKV(fsys, "some.json")
	assert.Nil(t, err)
	assert.Equal(t, in, out)
}

func TestWriteKV(t *testing.T) {
	in := theme.KV{
		"a": map[string]interface{}{
			"value": "a",
		},
		"b": map[string]interface{}{
			"value": "b",
		},
	}

	fsys := fsx.NewMemMapFs()
	assert.Nil(t, theme.WriteKV(fsys, "some.json", in))

	f, err := fsys.Open("some.json")
	assert.Nil(t, err)

	var out theme.KV
	assert.Nil(t, json.NewDecoder(f).Decode(&out))
	assert.Equal(t, in, out)
}

func TestUpdateKV(t *testing.T) {
	fileKV := theme.KV{
		"a": map[string]interface{}{
			"value": "a",
		},
		"b": map[string]interface{}{
			"value": "b",
		},
	}

	wb, err := json.Marshal(fileKV)
	assert.Nil(t, err)

	fsys := fsx.NewMemMapFs()
	assert.Nil(t, afero.WriteFile(fsys, "some.json", wb, 0644))
	_ = theme.UpdateKV(fsys, "some.json", theme.KV{
		"b.value": "b-new",
		"c.value": "c-new",
	})

	f, err := fsys.Open("some.json")
	assert.Nil(t, err)

	var out theme.KV
	assert.Nil(t, json.NewDecoder(f).Decode(&out))
	assert.Equal(t, out, theme.KV{
		"a": map[string]interface{}{
			"value": "a",
		},
		"b": map[string]interface{}{
			"value": "b-new",
		},
		"c": map[string]interface{}{
			"value": "c-new",
		},
	})
}
