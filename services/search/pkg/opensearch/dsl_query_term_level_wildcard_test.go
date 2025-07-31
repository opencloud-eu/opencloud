package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestWildcardQuery(t *testing.T) {
	tests := []opensearchtest.TableTest[opensearch.Builder, map[string]any]{
		{
			Name: "empty",
			Got:  opensearch.NewWildcardQuery("empty"),
			Want: nil,
		},
		{
			Name: "wildcard",
			Got: opensearch.NewWildcardQuery("name", opensearch.WildcardQueryOptions{
				Boost:           1.0,
				CaseInsensitive: true,
				Rewrite:         opensearch.TopTermsBlendedFreqsN,
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
			assert.JSONEq(t, opensearchtest.ToJSON(t, test.Want), opensearchtest.ToJSON(t, test.Got))
		})
	}
}
