package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestQuery(t *testing.T) {
	tests := []opensearchtest.TableTest[opensearch.Builder, map[string]any]{
		{
			Name: "simple",
			Got:  opensearch.NewRootQuery(opensearch.NewTermQuery[string]("name").Value("tom")),
			Want: map[string]any{
				"query": map[string]any{
					"term": map[string]any{
						"name": map[string]any{
							"value": "tom",
						},
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
