package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestNewMatchPhraseQuery(t *testing.T) {
	tests := []opensearchtest.TableTest[opensearch.Builder, map[string]any]{
		{
			Name: "empty",
			Got:  opensearch.NewMatchPhraseQuery("empty"),
			Want: nil,
		},
		{
			Name: "options",
			Got: opensearch.NewMatchPhraseQuery("name", opensearch.MatchPhraseQueryOptions{
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
			Got:  opensearch.NewMatchPhraseQuery("name").Query("some match query"),
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
			Got: opensearch.NewMatchPhraseQuery("name", opensearch.MatchPhraseQueryOptions{
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
			assert.JSONEq(t, opensearchtest.ToJSON(t, test.Want), opensearchtest.ToJSON(t, test.Got))
		})
	}
}
