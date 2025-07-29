package opensearch

import (
	"encoding/json"
)

type MatchPhraseQuery struct {
	field   string
	query   string
	options MatchPhraseQueryOptions
}

type MatchPhraseQueryOptions struct {
	Analyzer       Analyzer `json:"analyzer,omitempty"`
	Slop           int      `json:"slop,omitempty"`
	ZeroTermsQuery string   `json:"zero_terms_query,omitempty"`
}

func NewMatchPhraseQuery(field string, o ...MatchPhraseQueryOptions) *MatchPhraseQuery {
	return &MatchPhraseQuery{field: field, options: merge(o...)}
}

func (q *MatchPhraseQuery) Query(v string) *MatchPhraseQuery {
	q.query = v
	return q
}

func (q *MatchPhraseQuery) Map() (map[string]any, error) {
	data, err := convert[map[string]any](q.options)
	if err != nil {
		return nil, err
	}

	if !isEmpty(q.query) {
		data["query"] = q.query
	}

	if isEmpty(data) {
		return nil, nil
	}

	return map[string]any{
		"match_phrase": map[string]any{
			q.field: data,
		},
	}, nil
}

func (q *MatchPhraseQuery) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func (q *MatchPhraseQuery) String() string {
	b, _ := q.MarshalJSON()
	return string(b)
}
