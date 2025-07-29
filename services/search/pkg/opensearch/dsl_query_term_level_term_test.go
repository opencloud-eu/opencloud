package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
)

func TestTermQuery(t *testing.T) {
	tests := []tableTest[opensearch.Builder, map[string]any]{
		{
			name: "empty",
			got:  opensearch.NewTermQuery[string]("empty"),
			want: nil,
		},
		{
			name: "naked",
			got:  opensearch.NewTermQuery[bool]("deleted").Value(false),
			want: map[string]any{
				"term": map[string]any{
					"deleted": map[string]any{
						"value": false,
					},
				},
			},
		},
		{
			name: "term",
			got: opensearch.NewTermQuery[bool]("deleted", opensearch.TermQueryOptions{
				Boost:           1.0,
				CaseInsensitive: true,
				Name:            "is-deleted",
			}).Value(true),
			want: map[string]any{
				"term": map[string]any{
					"deleted": map[string]any{
						"value":            true,
						"boost":            1.0,
						"case_insensitive": true,
						"_name":            "is-deleted",
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
