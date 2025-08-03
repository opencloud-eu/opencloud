package opensearchtest

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var TimeMustParse = func(t *testing.T, ts string) time.Time {
	tp, err := time.Parse(time.RFC3339Nano, ts)
	require.NoError(t, err, "failed to parse time %s", ts)

	return tp
}

func JSONMustMarshal(t *testing.T, data any) string {
	jsonData, err := json.Marshal(data)
	require.NoError(t, err, "failed to marshal data to JSON")
	return string(jsonData)
}
