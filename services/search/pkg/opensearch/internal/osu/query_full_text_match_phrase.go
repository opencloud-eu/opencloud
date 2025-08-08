package osu

import (
	"encoding/json"
)

type MatchPhraseQuery struct {
	field   string
	query   string
	options *MatchPhraseQueryOptions
}

type MatchPhraseQueryOptions struct {
	Analyzer       string `json:"analyzer,omitempty"`
	Slop           int    `json:"slop,omitempty"`
	ZeroTermsQuery string `json:"zero_terms_query,omitempty"`
}

func NewMatchPhraseQuery(field string) *MatchPhraseQuery {
	return &MatchPhraseQuery{field: field}
}

func (q *MatchPhraseQuery) Options(v *MatchPhraseQueryOptions) *MatchPhraseQuery {
	q.options = v
	return q
}

func (q *MatchPhraseQuery) Query(v string) *MatchPhraseQuery {
	q.query = v
	return q
}

func (q *MatchPhraseQuery) Map() (map[string]any, error) {
	base, err := newBase(q.options)
	if err != nil {
		return nil, err
	}

	applyValue(base, "query", q.query)

	if isEmpty(base) {
		return nil, nil
	}

	return map[string]any{
		"match_phrase": map[string]any{
			q.field: base,
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
