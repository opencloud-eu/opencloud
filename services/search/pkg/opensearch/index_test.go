package opensearch_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/sjson"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestIndexManager(t *testing.T) {
	t.Run("index plausibility", func(t *testing.T) {
		tests := []opensearchtest.TableTest[opensearch.IndexManager, struct{}]{
			{
				Name: "empty",
				Got:  opensearch.IndexManagerLatest,
			},
		}
		tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				indexName := "opencloud-test-resource"
				tc.Require.IndicesReset([]string{indexName})

				body, err := test.Got.MarshalJSON()
				require.NoError(t, err)
				require.NotEmpty(t, body)
				require.NotEmpty(t, test.Got.String())
				require.JSONEq(t, test.Got.String(), string(body))
				require.NoError(t, test.Got.Apply(t.Context(), indexName, tc.Client()))
			})
		}
	})

	t.Run("does not create index if it already exists and is up to date", func(t *testing.T) {
		indexManager := opensearch.IndexManagerLatest
		indexName := "opencloud-test-resource"

		tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
		tc.Require.IndicesReset([]string{indexName})
		tc.Require.IndicesCreate(indexName, strings.NewReader(indexManager.String()))

		require.NoError(t, indexManager.Apply(t.Context(), indexName, tc.Client()))
	})

	t.Run("fails to create index if it already exists but is not up to date", func(t *testing.T) {
		indexManager := opensearch.IndexManagerLatest
		indexName := "opencloud-test-resource"

		tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
		tc.Require.IndicesReset([]string{indexName})

		body, err := sjson.Set(indexManager.String(), "settings.number_of_shards", "2")
		require.NoError(t, err)
		tc.Require.IndicesCreate(indexName, strings.NewReader(body))

		require.ErrorIs(t, indexManager.Apply(t.Context(), indexName, tc.Client()), opensearch.ErrManualActionRequired)
	})
}
