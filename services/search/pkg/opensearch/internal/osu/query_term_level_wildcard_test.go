package osu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestWildcardQuery(t *testing.T) {
	tests := []opensearchtest.TableTest[osu.Builder, map[string]any]{
		{
			Name: "empty",
			Got:  osu.NewWildcardQuery("empty"),
			Want: nil,
		},
		{
			Name: "wildcard",
			Got: osu.NewWildcardQuery("name").Params(&osu.WildcardQueryParams{
				Boost:           1.0,
				CaseInsensitive: true,
				Rewrite:         "top_terms_blended_freqs_N",
			}).Value("opencl*"),
			Want: map[string]any{
				"wildcard": map[string]any{
					"name": map[string]any{
						"value":            "opencl*",
						"boost":            1.0,
						"case_insensitive": true,
						"rewrite":          "top_terms_blended_freqs_N",
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
