package opensearchtest

import (
	"encoding/json"
	"testing"
	"time"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
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

func SearchHitsMustBeConverted[T any](t *testing.T, hits []opensearchgoAPI.SearchHit) []T {
	ts := make([]T, len(hits))
	for i, hit := range hits {
		resource, err := conversions.To[T](hit.Source)
		require.NoError(t, err)
		ts[i] = resource
	}

	return ts
}
