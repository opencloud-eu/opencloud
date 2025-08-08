package osu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestBoolQuery(t *testing.T) {
	tests := []opensearchtest.TableTest[osu.Builder, map[string]any]{
		{
			Name: "empty",
			Got:  osu.NewBoolQuery(),
			Want: nil,
		},
		{
			Name: "with-options",
			Got: osu.NewBoolQuery().Options(&osu.BoolQueryOptions{
				MinimumShouldMatch: 10,
				Boost:              10,
				Name:               "some-name",
			}),
			Want: map[string]any{
				"bool": map[string]any{
					"minimum_should_match": 10,
					"boost":                10,
					"_name":                "some-name",
				},
			},
		},
		{
			Name: "must",
			Got:  osu.NewBoolQuery().Must(osu.NewTermQuery[string]("name").Value("tom")),
			Want: map[string]any{
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
			Name: "must_not",
			Got:  osu.NewBoolQuery().MustNot(osu.NewTermQuery[string]("name").Value("tom")),
			Want: map[string]any{
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
			Name: "should",
			Got:  osu.NewBoolQuery().Should(osu.NewTermQuery[string]("name").Value("tom")),
			Want: map[string]any{
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
			Name: "filter",
			Got:  osu.NewBoolQuery().Filter(osu.NewTermQuery[string]("name").Value("tom")),
			Want: map[string]any{
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
			Name: "full",
			Got: osu.NewBoolQuery().
				Must(osu.NewTermQuery[string]("name").Value("tom")).
				MustNot(osu.NewTermQuery[bool]("deleted").Value(true)).
				Should(osu.NewTermQuery[string]("gender").Value("male")).
				Filter(osu.NewTermQuery[int]("age").Value(42)),
			Want: map[string]any{
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
		t.Run(test.Name, func(t *testing.T) {

			assert.JSONEq(t, opensearchtest.JSONMustMarshal(t, test.Want), opensearchtest.JSONMustMarshal(t, test.Got))
		})
	}
}
