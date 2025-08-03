package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestTermQuery(t *testing.T) {
	tests := []opensearchtest.TableTest[opensearch.Builder, map[string]any]{
		{
			Name: "empty",
			Got:  opensearch.NewTermQuery[string]("empty"),
			Want: nil,
		},
		{
			Name: "op-options",
			Got:  opensearch.NewTermQuery[bool]("deleted").Value(false),
			Want: map[string]any{
				"term": map[string]any{
					"deleted": map[string]any{
						"value": false,
					},
				},
			},
		},
		{
			Name: "with-options",
			Got: opensearch.NewTermQuery[bool]("deleted", opensearch.TermQueryOptions{
				Boost:           1.0,
				CaseInsensitive: true,
				Name:            "is-deleted",
			}).Value(true),
			Want: map[string]any{
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
		t.Run(test.Name, func(t *testing.T) {
			assert.JSONEq(t, opensearchtest.ToJSON(t, test.Want), opensearchtest.ToJSON(t, test.Got))
		})
	}
}
