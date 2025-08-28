package osu

import (
	"encoding/json"
	"slices"
)

type IDsQuery struct {
	values []string
	params *IDsQueryParams
}

type IDsQueryParams struct {
	Boost float32 `json:"boost,omitempty"`
}

func NewIDsQuery(v ...string) *IDsQuery {
	return &IDsQuery{values: slices.Compact(v)}
}

func (q *IDsQuery) Params(v *IDsQueryParams) *IDsQuery {
	q.params = v
	return q
}

func (q *IDsQuery) Map() (map[string]any, error) {
	base, err := newBase(q.params)
	if err != nil {
		return nil, err
	}

	applyValue(base, "values", q.values)

	if isEmpty(base) {
		return nil, nil
	}

	return map[string]any{
		"ids": base,
	}, nil
}

func (q *IDsQuery) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}
