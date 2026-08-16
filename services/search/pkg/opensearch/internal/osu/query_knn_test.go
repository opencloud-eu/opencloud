package osu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	opensearchtest "github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestKnnQuery(t *testing.T) {
	tests := []opensearchtest.TableTest[osu.Builder, map[string]any]{
		{
			Name: "vector and k",
			Got:  osu.NewKnnQuery("imageVector").Vector([]float32{0.5, 0.25}).K(10),
			Want: map[string]any{
				"knn": map[string]any{
					"imageVector": map[string]any{
						"vector": []any{0.5, 0.25},
						"k":      10,
					},
				},
			},
		},
		{
			Name: "with filter",
			Got: osu.NewKnnQuery("imageVector").Vector([]float32{1, 0}).K(3).Filter(
				osu.NewBoolQuery().Filter(osu.NewTermQuery[bool]("Deleted").Value(false)),
			),
			Want: map[string]any{
				"knn": map[string]any{
					"imageVector": map[string]any{
						"vector": []any{1.0, 0.0},
						"k":      3,
						"filter": map[string]any{
							"bool": map[string]any{
								"filter": []any{
									map[string]any{
										"term": map[string]any{
											"Deleted": map[string]any{"value": false},
										},
									},
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

	t.Run("incomplete queries error", func(t *testing.T) {
		_, err := osu.NewKnnQuery("imageVector").Map()
		assert.Error(t, err)
	})
}
