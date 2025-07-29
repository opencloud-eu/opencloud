package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
)

func TestNewMatchPhraseQuery(t *testing.T) {
	tests := []tableTest[opensearch.Builder, map[string]any]{
		{
			name: "empty",
			got:  opensearch.NewMatchPhraseQuery("empty"),
			want: nil,
		},
		{
			name: "options",
			got: opensearch.NewMatchPhraseQuery("name", opensearch.MatchPhraseQueryOptions{
				Analyzer:       "analyzer",
				Slop:           2,
				ZeroTermsQuery: "all",
			}),
			want: map[string]any{
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
			name: "query",
			got:  opensearch.NewMatchPhraseQuery("name").Query("some match query"),
			want: map[string]any{
				"match_phrase": map[string]any{
					"name": map[string]any{
						"query": "some match query",
					},
				},
			},
		},
		{
			name: "full",
			got: opensearch.NewMatchPhraseQuery("name", opensearch.MatchPhraseQueryOptions{
				Analyzer:       "analyzer",
				Slop:           2,
				ZeroTermsQuery: "all",
			}).Query("some match query"),
			want: map[string]any{
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
		t.Run(test.name, func(t *testing.T) {
			gotJSON, err := test.got.MarshalJSON()
			assert.NoError(t, err)

			assert.JSONEq(t, toJSON(t, test.want), string(gotJSON))
		})
	}
}
