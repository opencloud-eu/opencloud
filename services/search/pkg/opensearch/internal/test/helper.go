package opensearchtest

import (
	"encoding/json"
	"testing"
	"time"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/samber/lo"
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

func SearchHitsMustBeConverted[T any](t *testing.T, hits []opensearchgoAPI.SearchHit) []T {
	return lo.ReduceRight(hits, func(agg []T, item opensearchgoAPI.SearchHit, _ int) []T {
		resource, err := convert[T](item.Source)
		require.NoError(t, err)
		return append(agg, resource)
	}, []T{})
}

func convert[T any](v any) (T, error) {
	var t T

	if v == nil {
		return t, nil
	}

	j, err := json.Marshal(v)
	if err != nil {
		return t, err
	}

	if err := json.Unmarshal(j, &t); err != nil {
		return t, err
	}

	return t, nil
}
