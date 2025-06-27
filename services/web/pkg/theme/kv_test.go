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

	out, err := theme.LoadKV(fsys.IOFS(), "some.json")
	assert.Nil(t, err)
	assert.Equal(t, in, out)
}
