package osu

import (
	"encoding/json"
	"errors"
	"time"
)

type RangeQuery[T time.Time | string] struct {
	field   string
	gt      T
	gte     T
	lt      T
	lte     T
	options *RangeQueryOptions
}

type RangeQueryOptions struct {
	Format   string  `json:"format,omitempty"`
	Relation string  `json:"relation,omitempty"`
	Boost    float32 `json:"boost,omitempty"`
	TimeZone string  `json:"time_zone,omitempty"`
}

func NewRangeQuery[T time.Time | string](field string) *RangeQuery[T] {
	return &RangeQuery[T]{field: field}
}

func (q *RangeQuery[T]) Options(v *RangeQueryOptions) *RangeQuery[T] {
	q.options = v
	return q
}

func (q *RangeQuery[T]) Gt(v T) *RangeQuery[T] {
	q.gt = v
	return q
}

func (q *RangeQuery[T]) Gte(v T) *RangeQuery[T] {
	q.gte = v
	return q
}

func (q *RangeQuery[T]) Lt(v T) *RangeQuery[T] {
	q.lt = v
	return q
}

func (q *RangeQuery[T]) Lte(v T) *RangeQuery[T] {
	q.lte = v
	return q
}

func (q *RangeQuery[T]) Map() (map[string]any, error) {
	if !isEmpty(q.gt) && !isEmpty(q.gte) {
		return nil, errors.New("cannot set both gt and gte in RangeQuery")
	}

	if !isEmpty(q.lt) && !isEmpty(q.lte) {
		return nil, errors.New("cannot set both lt and lte in RangeQuery")
	}

	base, err := newBase(q.options)
	if err != nil {
		return nil, err
	}

	applyValues(base, map[string]T{
		"gt":  q.gt,
		"gte": q.gte,
		"lt":  q.lt,
		"lte": q.lte,
	})

	if isEmpty(base) {
		return nil, nil
	}

	return map[string]any{
		"range": map[string]any{
			q.field: base,
		},
	}, nil
}

func (q *RangeQuery[T]) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}
