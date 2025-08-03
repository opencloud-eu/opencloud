package opensearch

import (
	"encoding/json"
)

type WildcardQuery struct {
	field   string
	value   string
	options WildcardQueryOptions
}

type WildcardQueryOptions struct {
	Boost           float32 `json:"boost,omitempty"`
	CaseInsensitive bool    `json:"case_insensitive,omitempty"`
	Rewrite         Rewrite `json:"rewrite,omitempty"`
}

func NewWildcardQuery(field string, o ...WildcardQueryOptions) *WildcardQuery {
	return &WildcardQuery{field: field, options: merge(o...)}
}

func (q *WildcardQuery) Value(v string) *WildcardQuery {
	q.value = v
	return q
}

func (q *WildcardQuery) Map() (map[string]any, error) {
	data, err := convert[map[string]any](q.options)
	if err != nil {
		return nil, err
	}

	applyValue(data, "value", q.value)

	if isEmpty(data) {
		return nil, nil
	}

	return map[string]any{
		"wildcard": map[string]any{
			q.field: data,
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

func (q *WildcardQuery) String() string {
	b, _ := q.MarshalJSON()
	return string(b)
}
