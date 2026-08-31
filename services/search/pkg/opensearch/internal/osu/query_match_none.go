package osu

import (
	"encoding/json"
)

// MatchNoneQuery matches no document.
type MatchNoneQuery struct{}

// NewMatchNoneQuery creates a query that matches no document.
func NewMatchNoneQuery() *MatchNoneQuery {
	return &MatchNoneQuery{}
}

// Map returns the query as a map.
func (q *MatchNoneQuery) Map() (map[string]any, error) {
	return map[string]any{
		"match_none": map[string]any{},
	}, nil
}

// MarshalJSON returns the query as JSON.
func (q *MatchNoneQuery) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}

	return json.Marshal(data)
}

// MatchAllQuery matches every document.
type MatchAllQuery struct{}

// NewMatchAllQuery creates a query that matches every document.
func NewMatchAllQuery() *MatchAllQuery {
	return &MatchAllQuery{}
}

// Map returns the query as a map.
func (q *MatchAllQuery) Map() (map[string]any, error) {
	return map[string]any{
		"match_all": map[string]any{},
	}, nil
}

// MarshalJSON returns the query as JSON.
func (q *MatchAllQuery) MarshalJSON() ([]byte, error) {
	data, err := q.Map()
	if err != nil {
		return nil, err
	}

	return json.Marshal(data)
}
