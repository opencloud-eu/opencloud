package opensearch

import (
	"encoding/json"
	"slices"
)

type IDsQuery struct {
	values  []string
	options IDsQueryOptions
}

type IDsQueryOptions struct {
	Boost float32 `json:"boost,omitempty"`
}

func NewIDsQuery(v []string, o ...IDsQueryOptions) *IDsQuery {
	return &IDsQuery{values: slices.Compact(v), options: merge(o...)}
}

func (q *IDsQuery) Map() (map[string]any, error) {
	data, err := convert[map[string]any](q.options)
	if err != nil {
		return nil, err
	}

	applyValue(data, "values", q.values)

	if isEmpty(data) {
		return nil, nil
	}

	return map[string]any{
		"ids": data,
	}, nil
}

func (q *IDsQuery) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func (q *IDsQuery) String() string {
	b, _ := q.MarshalJSON()
	return string(b)
}
