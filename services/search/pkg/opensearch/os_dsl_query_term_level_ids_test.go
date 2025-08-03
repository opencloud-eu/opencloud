package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestIDsQuery(t *testing.T) {
	tests := []opensearchtest.TableTest[opensearch.Builder, map[string]any]{
		{
			Name: "empty",
			Got:  opensearch.NewIDsQuery(nil),
			Want: nil,
		},
		{
			Name: "ids",
			Got:  opensearch.NewIDsQuery([]string{"1", "2", "3", "3"}, opensearch.IDsQueryOptions{Boost: 1.0}),
			Want: map[string]any{
				"ids": map[string]any{
					"values": []string{"1", "2", "3"},
					"boost":  1.0,
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
