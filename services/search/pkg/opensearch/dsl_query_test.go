package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
)

func TestQuery(t *testing.T) {
	tests := []tableTest[opensearch.Builder, map[string]any]{
		{
			name: "simple",
			got:  opensearch.NewRootQuery(opensearch.NewTermQuery[string]("name").Value("tom")),
			want: map[string]any{
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
		t.Run(test.name, func(t *testing.T) {
			gotJSON, err := test.got.MarshalJSON()
			assert.NoError(t, err)

			assert.JSONEq(t, toJSON(t, test.want), string(gotJSON))
		})
	}
}
