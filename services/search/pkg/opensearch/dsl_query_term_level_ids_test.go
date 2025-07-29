package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
)

func TestIDsQuery(t *testing.T) {
	tests := []tableTest[opensearch.Builder, map[string]any]{
		{
			name: "empty",
			got:  opensearch.NewIDsQuery(),
			want: nil,
		},
		{
			name: "ids",
			got: opensearch.NewIDsQuery(opensearch.IDsQueryOptions{Boost: 1.0}).
				Values("1", "2").
				Values("3", "3"),
			want: map[string]any{
				"ids": map[string]any{
					"values": []string{"1", "2", "3"},
					"boost":  1.0,
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
