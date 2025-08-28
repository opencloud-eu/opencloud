package osu

import (
	"encoding/json"
)

type TermQuery[T comparable] struct {
	field  string
	value  T
	params *TermQueryParams
}

type TermQueryParams struct {
	Boost           float32 `json:"boost,omitempty"`
	CaseInsensitive bool    `json:"case_insensitive,omitempty"`
	Name            string  `json:"_name,omitempty"`
}

func NewTermQuery[T comparable](field string) *TermQuery[T] {
	return &TermQuery[T]{field: field}
}

func (q *TermQuery[T]) Params(v *TermQueryParams) *TermQuery[T] {
	q.params = v
	return q
}

func (q *TermQuery[T]) Value(v T) *TermQuery[T] {
	q.value = v
	return q
}

func (q *TermQuery[T]) Map() (map[string]any, error) {
	base, err := newBase(q.params)
	if err != nil {
		return nil, err
	}

	applyValue(base, "value", q.value)

	if isEmpty(base) {
		return nil, nil
	}

	return map[string]any{
		"term": map[string]any{
			q.field: base,
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
