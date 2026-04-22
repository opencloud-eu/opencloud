package theme_test

import (
	"encoding/json"
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

func TestPatchKV(t *testing.T) {
	in := theme.KV{
		"a": map[string]any{
			"value": "a",
		},
		"b": map[string]any{
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
		"a": map[string]any{
			"value": "a",
		},
		"b": map[string]any{
			"value": "b-new",
		},
		"c": map[string]any{
			"value": "c-new",
		},
		"d": "d-new",
		"e": map[string]any{
			"value": map[string]any{
				"subvalue": "e-new",
			},
		},
	}, out)
}

func TestPatchKVUnset(t *testing.T) {
	in := theme.KV{
		"a": map[string]any{
			"value": "a",
		},
		"b": map[string]any{
			"value": "b",
		},
	}
	out := theme.PatchKV(in, theme.KV{
		"a.value": nil,
		"b":       nil,
	})
	assert.Equal(t, theme.KV{
		"a": map[string]any{},
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
		"b": map[string]any{
			"value": "b-new",
		},
		"c": map[string]any{
			"value": "c-new",
		},
		"d": "d-new",
		"e": map[string]any{
			"value": map[string]any{
				"subvalue": "e-new",
			},
		},
	}, out)
}

func TestLoadKV(t *testing.T) {
	in := theme.KV{
		"a": map[string]any{
			"value": "a",
		},
		"b": map[string]any{
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
		"a": map[string]any{
			"value": "a",
		},
		"b": map[string]any{
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
		"a": map[string]any{
			"value": "a",
		},
		"b": map[string]any{
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
		"a": map[string]any{
			"value": "a",
		},
		"b": map[string]any{
			"value": "b-new",
		},
		"c": map[string]any{
			"value": "c-new",
		},
	})
}
