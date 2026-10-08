package opensearch_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestBatch_Push(t *testing.T) {
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)

	t.Run("reports the documents the bulk API rejected", func(t *testing.T) {
		indexName := "opencloud-test-batch-push-rejected"
		tc.Require.IndicesReset([]string{indexName})
		defer tc.Require.IndicesDelete([]string{indexName})

		// Name is a string, mapping it as a long makes every document fail to parse.
		tc.Require.IndicesCreate(indexName, strings.NewReader(`{"mappings":{"properties":{"Name":{"type":"long"}}}}`))

		batch, err := opensearch.NewBatch(tc.Client(), indexName, 10)
		require.NoError(t, err)

		document := opensearchtest.Testdata.Resources.File
		require.NoError(t, batch.Upsert(document.ID, document))

		err = batch.Push()
		require.Error(t, err)
		require.ErrorContains(t, err, document.ID)
		require.ErrorContains(t, err, "mapper_parsing_exception")
		tc.Require.IndicesCount([]string{indexName}, nil, 0)
	})

	t.Run("pushes the documents the bulk API accepted", func(t *testing.T) {
		indexName := "opencloud-test-batch-push-accepted"
		tc.Require.IndicesReset([]string{indexName})
		defer tc.Require.IndicesDelete([]string{indexName})

		tc.Require.IndicesCreate(indexName, strings.NewReader(opensearch.IndexManagerLatest.String()))

		batch, err := opensearch.NewBatch(tc.Client(), indexName, 10)
		require.NoError(t, err)

		document := opensearchtest.Testdata.Resources.File
		require.NoError(t, batch.Upsert(document.ID, document))
		require.NoError(t, batch.Push())

		tc.Require.IndicesCount([]string{indexName}, nil, 1)
	})
}
