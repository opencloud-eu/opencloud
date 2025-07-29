package opensearch_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

type tableTest[G any, W any] struct {
	name string
	got  G
	want W
}

func toJSON(t *testing.T, data any) string {
	jsonData, err := json.Marshal(data)
	assert.NoError(t, err, "failed to marshal data to JSON")
	return string(jsonData)
}
