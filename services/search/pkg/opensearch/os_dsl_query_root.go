package opensearch

import (
	"encoding/json"
)

type RootQuery struct {
	query   Builder
	options RootQueryOptions
}

func NewRootQuery(builder Builder, o ...RootQueryOptions) *RootQuery {
	return &RootQuery{query: builder, options: merge(o...)}
}

func (q *RootQuery) Query(v Builder) *RootQuery {
	q.query = v
	return q
}

func (q *RootQuery) Map() (map[string]any, error) {
	data, err := convert[map[string]any](q.options)
	if err != nil {
		return nil, err
	}

	if err := applyBuilder(data, "query", q.query); err != nil {
		return nil, err
	}

	if isEmpty(data) {
		return nil, nil
	}

	return data, nil
}

func (q *RootQuery) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}

	return json.Marshal(data)
}

func (q *RootQuery) String() string {
	b, _ := q.MarshalJSON()
	return string(b)
}

type RootQueryOptions struct {
	Highlight *RootQueryHighlight `json:"highlight,omitempty"`
}

type RootQueryHighlight struct {
	PreTags  []string                      `json:"pre_tags,omitempty"`
	PostTags []string                      `json:"post_tags,omitempty"`
	Fields   map[string]RootQueryHighlight `json:"fields,omitempty"`
}
