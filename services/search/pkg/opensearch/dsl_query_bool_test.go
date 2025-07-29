package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
)

func TestBoolQuery(t *testing.T) {
	tests := []tableTest[opensearch.Builder, map[string]any]{
		{
			name: "empty",
			got:  opensearch.NewBoolQuery(),
			want: nil,
		},
		{
			name: "naked",
			got: opensearch.NewBoolQuery(opensearch.BoolQueryOptions{
				MinimumShouldMatch: 10,
				Boost:              10,
				Name:               "some-name",
			}),
			want: map[string]any{
				"bool": map[string]any{
					"minimum_should_match": 10,
					"boost":                10,
					"_name":                "some-name",
				},
			},
		},
		{
			name: "must",
			got:  opensearch.NewBoolQuery().Must(opensearch.NewTermQuery[string]("name").Value("tom")),
			want: map[string]any{
				"bool": map[string]any{
					"must": []map[string]any{
						{
							"term": map[string]any{
								"name": map[string]any{
									"value": "tom",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "must_not",
			got:  opensearch.NewBoolQuery().MustNot(opensearch.NewTermQuery[string]("name").Value("tom")),
			want: map[string]any{
				"bool": map[string]any{
					"must_not": []map[string]any{
						{
							"term": map[string]any{
								"name": map[string]any{
									"value": "tom",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "should",
			got:  opensearch.NewBoolQuery().Should(opensearch.NewTermQuery[string]("name").Value("tom")),
			want: map[string]any{
				"bool": map[string]any{
					"should": []map[string]any{
						{
							"term": map[string]any{
								"name": map[string]any{
									"value": "tom",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "filter",
			got:  opensearch.NewBoolQuery().Filter(opensearch.NewTermQuery[string]("name").Value("tom")),
			want: map[string]any{
				"bool": map[string]any{
					"filter": []map[string]any{
						{
							"term": map[string]any{
								"name": map[string]any{
									"value": "tom",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "full",
			got: opensearch.NewBoolQuery().
				Must(opensearch.NewTermQuery[string]("name").Value("tom")).
				MustNot(opensearch.NewTermQuery[bool]("deleted").Value(true)).
				Should(opensearch.NewTermQuery[string]("gender").Value("male")).
				Filter(opensearch.NewTermQuery[int]("age").Value(42)),
			want: map[string]any{
				"bool": map[string]any{
					"must": []map[string]any{
						{
							"term": map[string]any{
								"name": map[string]any{
									"value": "tom",
								},
							},
						},
					},
					"must_not": []map[string]any{
						{
							"term": map[string]any{
								"deleted": map[string]any{
									"value": true,
								},
							},
						},
					},
					"should": []map[string]any{
						{
							"term": map[string]any{
								"gender": map[string]any{
									"value": "male",
								},
							},
						},
					},
					"filter": []map[string]any{
						{
							"term": map[string]any{
								"age": map[string]any{
									"value": 42,
								},
							},
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
