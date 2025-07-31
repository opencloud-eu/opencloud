package opensearch

import (
	"encoding/json"
)

type TermQuery[T comparable] struct {
	field   string
	value   T
	options TermQueryOptions
}

type TermQueryOptions struct {
	Boost           float32 `json:"boost,omitempty"`
	CaseInsensitive bool    `json:"case_insensitive,omitempty"`
	Name            string  `json:"_name,omitempty"`
}

func NewTermQuery[T comparable](field string, o ...TermQueryOptions) *TermQuery[T] {
	return &TermQuery[T]{field: field, options: merge(o...)}
}

func (q *TermQuery[T]) Value(v T) *TermQuery[T] {
	q.value = v
	return q
}

func (q *TermQuery[T]) Map() (map[string]any, error) {
	data, err := convert[map[string]any](q.options)
	if err != nil {
		return nil, err
	}

	if !isEmpty(q.value) {
		data["value"] = q.value
	}

	if isEmpty(data) {
		return nil, nil
	}

	return map[string]any{
		"term": map[string]any{
			q.field: data,
		},
	}, nil
}

func (q *TermQuery[T]) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func (q *TermQuery[T]) String() string {
	b, _ := q.MarshalJSON()
	return string(b)
}
