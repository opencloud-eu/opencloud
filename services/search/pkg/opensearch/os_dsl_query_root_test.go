package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestRootQuery(t *testing.T) {
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
		{
			Name: "highlight",
			Got: opensearch.NewRootQuery(
				opensearch.NewTermQuery[string]("content").Value("content"),
				opensearch.RootQueryOptions{
					Highlight: opensearch.RootQueryHighlight{
						PreTags:  []string{"<b>"},
						PostTags: []string{"</b>"},
						Fields: map[string]opensearch.RootQueryHighlight{
							"content": {},
						},
					},
				},
			),
			Want: map[string]any{
				"query": map[string]any{
					"term": map[string]any{
						"content": map[string]any{
							"value": "content",
						},
					},
				},
				"highlight": map[string]any{
					"pre_tags":  []string{"<b>"},
					"post_tags": []string{"</b>"},
					"fields": map[string]any{
						"content": map[string]any{},
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
