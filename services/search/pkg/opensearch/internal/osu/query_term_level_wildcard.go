package osu

import (
	"encoding/json"
)

type WildcardQuery struct {
	field   string
	value   string
	options *WildcardQueryOptions
}

type WildcardQueryOptions struct {
	Boost           float32 `json:"boost,omitempty"`
	CaseInsensitive bool    `json:"case_insensitive,omitempty"`
	Rewrite         string  `json:"rewrite,omitempty"`
}

func NewWildcardQuery(field string) *WildcardQuery {
	return &WildcardQuery{field: field}
}

func (q *WildcardQuery) Options(v *WildcardQueryOptions) *WildcardQuery {
	q.options = v
	return q
}

func (q *WildcardQuery) Value(v string) *WildcardQuery {
	q.value = v
	return q
}

func (q *WildcardQuery) Map() (map[string]any, error) {
	base, err := newBase(q.options)
	if err != nil {
		return nil, err
	}

	applyValue(base, "value", q.value)

	if isEmpty(base) {
		return nil, nil
	}

	return map[string]any{
		"wildcard": map[string]any{
			q.field: base,
		},
	}, nil
}

func (q *WildcardQuery) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}
