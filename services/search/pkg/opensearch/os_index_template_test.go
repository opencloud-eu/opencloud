package opensearch_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	opensearchtest "github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestIndexTemplates(t *testing.T) {
	tc := opensearchtest.NewDefaultTestClient(t)
	t.Run("index templates plausibility", func(t *testing.T) {
		tests := []opensearchtest.TableTest[opensearch.IndexTemplate, struct{}]{
			{
				Name: "empty",
				Got:  opensearch.IndexTemplateResourceV1,
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				body, err := test.Got.MarshalJSON()
				require.NoError(t, err)
				require.NotEmpty(t, body)
				require.NotEmpty(t, test.Got.String())
				require.JSONEq(t, test.Got.String(), string(body))
				require.NotEmpty(t, test.Got.Name())
				require.NoError(t, test.Got.Apply(t.Context(), tc.Client()))
			})
		}
	})
}
