package osu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestNewMatchPhraseQuery(t *testing.T) {
	tests := []opensearchtest.TableTest[osu.Builder, map[string]any]{
		{
			Name: "empty",
			Got:  osu.NewMatchPhraseQuery("empty"),
			Want: nil,
		},
		{
			Name: "with params",
			Got: osu.NewMatchPhraseQuery("name").Params(&osu.MatchPhraseQueryParams{
				Analyzer:       "analyzer",
				Slop:           2,
				ZeroTermsQuery: "all",
			}),
			Want: map[string]any{
				"match_phrase": map[string]any{
					"name": map[string]any{
						"analyzer":         "analyzer",
						"slop":             2,
						"zero_terms_query": "all",
					},
				},
			},
		},
		{
			Name: "query",
			Got:  osu.NewMatchPhraseQuery("name").Query("some match query"),
			Want: map[string]any{
				"match_phrase": map[string]any{
					"name": map[string]any{
						"query": "some match query",
					},
				},
			},
		},
		{
			Name: "full",
			Got: osu.NewMatchPhraseQuery("name").Params(&osu.MatchPhraseQueryParams{
				Analyzer:       "analyzer",
				Slop:           2,
				ZeroTermsQuery: "all",
			}).Query("some match query"),
			Want: map[string]any{
				"match_phrase": map[string]any{
					"name": map[string]any{
						"query":            "some match query",
						"analyzer":         "analyzer",
						"slop":             2,
						"zero_terms_query": "all",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.JSONEq(t, opensearchtest.JSONMustMarshal(t, test.Want), opensearchtest.JSONMustMarshal(t, test.Got))
		})
	}
}
