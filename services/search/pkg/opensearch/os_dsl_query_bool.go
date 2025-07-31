package opensearch

import (
	"encoding/json"
)

type BoolQuery struct {
	must    []Builder
	mustNot []Builder
	should  []Builder
	filter  []Builder
	options BoolQueryOptions
}

type BoolQueryOptions struct {
	MinimumShouldMatch int16   `json:"minimum_should_match,omitempty"`
	Boost              float32 `json:"boost,omitempty"`
	Name               string  `json:"_name,omitempty"`
}

func NewBoolQuery(o ...BoolQueryOptions) *BoolQuery {
	return &BoolQuery{options: merge(o...)}
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
	data, err := convert[map[string]any](q.options)
	if err != nil {
		return nil, err
	}

	if err := applyBuilders(data, "must", q.must...); err != nil {
		return nil, err
	}

	if err := applyBuilders(data, "must_not", q.mustNot...); err != nil {
		return nil, err
	}

	if err := applyBuilders(data, "should", q.should...); err != nil {
		return nil, err
	}

	if err := applyBuilders(data, "filter", q.filter...); err != nil {
		return nil, err
	}

	if isEmpty(data) {
		return nil, nil
	}

	return map[string]any{
		"bool": data,
	}, nil
}

func (q *BoolQuery) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func (q *BoolQuery) String() string {
	b, _ := q.MarshalJSON()
	return string(b)
}
