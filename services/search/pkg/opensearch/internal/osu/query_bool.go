package osu

import (
	"encoding/json"
)

type BoolQuery struct {
	must    []Builder
	mustNot []Builder
	should  []Builder
	filter  []Builder
	params  *BoolQueryParams
}

type BoolQueryParams struct {
	MinimumShouldMatch int16   `json:"minimum_should_match,omitempty"`
	Boost              float32 `json:"boost,omitempty"`
	Name               string  `json:"_name,omitempty"`
}

func NewBoolQuery() *BoolQuery {
	return &BoolQuery{}
}

func (q *BoolQuery) Params(v *BoolQueryParams) *BoolQuery {
	q.params = v
	return q
}

func (q *BoolQuery) Must(v ...Builder) *BoolQuery {
	q.must = append(q.must, v...)
	return q
}

func (q *BoolQuery) MustNot(v ...Builder) *BoolQuery {
	q.mustNot = append(q.mustNot, v...)
	return q
}

func (q *BoolQuery) Should(v ...Builder) *BoolQuery {
	q.should = append(q.should, v...)
	return q
}

func (q *BoolQuery) Filter(v ...Builder) *BoolQuery {
	q.filter = append(q.filter, v...)
	return q
}

func (q *BoolQuery) Map() (map[string]any, error) {
	base, err := newBase(q.params)
	if err != nil {
		return nil, err
	}

	if err := applyBuilders(base, "must", q.must...); err != nil {
		return nil, err
	}

	if err := applyBuilders(base, "must_not", q.mustNot...); err != nil {
		return nil, err
	}

	if err := applyBuilders(base, "should", q.should...); err != nil {
		return nil, err
	}

	if err := applyBuilders(base, "filter", q.filter...); err != nil {
		return nil, err
	}

	if isEmpty(base) {
		return nil, nil
	}

	return map[string]any{
		"bool": base,
	}, nil
}

func (q *BoolQuery) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}
