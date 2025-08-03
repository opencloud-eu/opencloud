package opensearchtest

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type TableTest[G any, W any] struct {
	Name string
	Got  G
	Want W
	Err  error
}

func ToJSON(t *testing.T, data any) string {
	jsonData, err := json.Marshal(data)
	require.NoError(t, err, "failed to marshal data to JSON")
	return string(jsonData)
}
