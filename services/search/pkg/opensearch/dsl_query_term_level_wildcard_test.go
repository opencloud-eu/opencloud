package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
)

func TestWildcardQuery(t *testing.T) {
	tests := []tableTest[opensearch.Builder, map[string]any]{
		{
			name: "empty",
			got:  opensearch.NewWildcardQuery("empty"),
			want: nil,
		},
		{
			name: "wildcard",
			got: opensearch.NewWildcardQuery("name", opensearch.WildcardQueryOptions{
				Boost:           1.0,
				CaseInsensitive: true,
				Rewrite:         opensearch.TopTermsBlendedFreqsN,
			}).Value("opencl*"),
			want: map[string]any{
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
		t.Run(test.name, func(t *testing.T) {
			gotJSON, err := test.got.MarshalJSON()
			assert.NoError(t, err)

			assert.JSONEq(t, toJSON(t, test.want), string(gotJSON))
		})
	}
}
