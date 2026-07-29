package opensearch_test

import (
	"fmt"
	"strings"
	"testing"

	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// TestVersionedIndexName guards that the index name and the generator identity
// carry the same schema version.
func TestVersionedIndexName(t *testing.T) {
	require.Equal(t,
		fmt.Sprintf("opencloud-resource-v%d", search.SchemaVersion),
		opensearch.VersionedIndexName("opencloud-resource"),
	)
	require.Equal(t,
		fmt.Sprintf("resource_v%d", search.SchemaVersion),
		string(opensearch.IndexManagerLatest),
	)
}

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
				require.NoError(t, test.Got.Apply(t.Context(), indexName, tc.Client(), log.NopLogger()))
			})
		}
	})

	t.Run("does not create index if it already exists and is up to date", func(t *testing.T) {
		indexManager := opensearch.IndexManagerLatest
		indexName := "opencloud-test-resource"

		tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
		tc.Require.IndicesReset([]string{indexName})
		tc.Require.IndicesCreate(indexName, strings.NewReader(indexManager.String()))

		require.NoError(t, indexManager.Apply(t.Context(), indexName, tc.Client(), log.NopLogger()))
	})

	t.Run("fails when the analysis settings drift", func(t *testing.T) {
		indexManager := opensearch.IndexManagerLatest
		indexName := "opencloud-test-resource"

		tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
		tc.Require.IndicesReset([]string{indexName})

		body, err := sjson.Set(indexManager.String(), "settings.analysis.analyzer.lowercaseKeyword.tokenizer", "standard")
		require.NoError(t, err)
		tc.Require.IndicesCreate(indexName, strings.NewReader(body))

		require.ErrorIs(t, indexManager.Apply(t.Context(), indexName, tc.Client(), log.NopLogger()), opensearch.ErrManualActionRequired)
	})

	t.Run("tolerates replica drift", func(t *testing.T) {
		indexManager := opensearch.IndexManagerLatest
		indexName := "opencloud-test-resource"

		tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
		tc.Require.IndicesReset([]string{indexName})

		body, err := sjson.Set(indexManager.String(), "settings.number_of_replicas", "2")
		require.NoError(t, err)
		tc.Require.IndicesCreate(indexName, strings.NewReader(body))

		require.NoError(t, indexManager.Apply(t.Context(), indexName, tc.Client(), log.NopLogger()))
	})

	t.Run("tolerates shard drift", func(t *testing.T) {
		indexManager := opensearch.IndexManagerLatest
		indexName := "opencloud-test-resource"

		tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
		tc.Require.IndicesReset([]string{indexName})

		body, err := sjson.Set(indexManager.String(), "settings.number_of_shards", "2")
		require.NoError(t, err)
		tc.Require.IndicesCreate(indexName, strings.NewReader(body))

		require.NoError(t, indexManager.Apply(t.Context(), indexName, tc.Client(), log.NopLogger()))
	})

	t.Run("is idempotent", func(t *testing.T) {
		indexManager := opensearch.IndexManagerLatest
		indexName := "opencloud-test-resource"

		tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
		tc.Require.IndicesReset([]string{indexName})

		require.NoError(t, indexManager.Apply(t.Context(), indexName, tc.Client(), log.NopLogger()))
		require.NoError(t, indexManager.Apply(t.Context(), indexName, tc.Client(), log.NopLogger()))
	})

	t.Run("adds a new field to an existing index in place", func(t *testing.T) {
		indexManager := opensearch.IndexManagerLatest
		indexName := "opencloud-test-resource"

		tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
		tc.Require.IndicesReset([]string{indexName})

		body, err := sjson.Delete(indexManager.String(), "mappings.properties.Title")
		require.NoError(t, err)
		tc.Require.IndicesCreate(indexName, strings.NewReader(body))

		require.NoError(t, indexManager.Apply(t.Context(), indexName, tc.Client(), log.NopLogger()))

		resp, err := tc.Client().Indices.Mapping.Get(t.Context(), &opensearchgoAPI.MappingGetReq{Indices: []string{indexName}})
		require.NoError(t, err)
		require.True(t, gjson.GetBytes(resp.Indices[indexName].Mappings, "properties.Title").Exists())
	})

	t.Run("adds a new nested field to an existing index in place", func(t *testing.T) {
		indexManager := opensearch.IndexManagerLatest
		indexName := "opencloud-test-resource"

		tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
		tc.Require.IndicesReset([]string{indexName})

		body, err := sjson.Delete(indexManager.String(), "mappings.properties.photo.properties.cameraMake")
		require.NoError(t, err)
		tc.Require.IndicesCreate(indexName, strings.NewReader(body))

		require.NoError(t, indexManager.Apply(t.Context(), indexName, tc.Client(), log.NopLogger()))

		resp, err := tc.Client().Indices.Mapping.Get(t.Context(), &opensearchgoAPI.MappingGetReq{Indices: []string{indexName}})
		require.NoError(t, err)
		require.True(t, gjson.GetBytes(resp.Indices[indexName].Mappings, "properties.photo.properties.cameraMake").Exists())
	})

	t.Run("fails when an existing field changed its definition", func(t *testing.T) {
		indexManager := opensearch.IndexManagerLatest
		indexName := "opencloud-test-resource"

		tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
		tc.Require.IndicesReset([]string{indexName})

		body, err := sjson.Set(indexManager.String(), "mappings.properties.Deleted.type", "keyword")
		require.NoError(t, err)
		tc.Require.IndicesCreate(indexName, strings.NewReader(body))

		require.ErrorIs(t, indexManager.Apply(t.Context(), indexName, tc.Client(), log.NopLogger()), opensearch.ErrManualActionRequired)
	})

	t.Run("fails when the index contains a field the code schema does not know", func(t *testing.T) {
		indexManager := opensearch.IndexManagerLatest
		indexName := "opencloud-test-resource"

		tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
		tc.Require.IndicesReset([]string{indexName})

		body, err := sjson.Set(indexManager.String(), "mappings.properties.legacyField.type", "keyword")
		require.NoError(t, err)
		tc.Require.IndicesCreate(indexName, strings.NewReader(body))

		require.ErrorIs(t, indexManager.Apply(t.Context(), indexName, tc.Client(), log.NopLogger()), opensearch.ErrManualActionRequired)
	})

	t.Run("transport errors do not demand manual action", func(t *testing.T) {
		client, err := opensearchgoAPI.NewClient(opensearchgoAPI.Config{
			Client: opensearchgo.Config{
				Addresses: []string{"http://localhost:1025"},
			},
		})
		require.NoError(t, err)

		err = opensearch.IndexManagerLatest.Apply(t.Context(), "opencloud-test-resource", client, log.NopLogger())
		require.Error(t, err)
		require.NotErrorIs(t, err, opensearch.ErrManualActionRequired)
	})
}
