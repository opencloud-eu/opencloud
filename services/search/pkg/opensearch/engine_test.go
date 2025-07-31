package opensearch_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestEngine_Search(t *testing.T) {
	index := "test-engine-search"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	document := opensearchtest.Testdata.Resources.Full
	tc.Require.DocumentCreate(index, document.ID, opensearchtest.ToJSON(t, document))
	tc.Require.IndicesCount([]string{index}, "", 1)

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("most simple search", func(t *testing.T) {
		resp, err := engine.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s"`, document.Name),
		})
		assert.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		assert.Equal(t, int32(1), resp.TotalMatches)
		assert.Equal(t, document.ID, fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId))
	})

	t.Run("ignores files that are marked as deleted", func(t *testing.T) {
		deletedDocument := opensearchtest.Testdata.Resources.Full
		deletedDocument.ID = "1$2!4"
		deletedDocument.Deleted = true

		tc.Require.DocumentCreate(index, deletedDocument.ID, opensearchtest.ToJSON(t, deletedDocument))
		tc.Require.IndicesCount([]string{index}, "", 2)

		resp, err := engine.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s"`, document.Name),
		})
		assert.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		assert.Equal(t, int32(1), resp.TotalMatches)
		assert.Equal(t, document.ID, fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId))
	})
}

func TestEngine_Upsert(t *testing.T) {
	index := "test-engine-upsert"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("upsert with full document", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.Full
		assert.NoError(t, engine.Upsert(document.ID, document))

		tc.Require.IndicesCount([]string{index}, "", 1)
	})
}

func TestEngine_Move(t *testing.T) {}

func TestEngine_Delete(t *testing.T) {
	index := "test-engine-delete"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("mark document as deleted", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.Full
		tc.Require.DocumentCreate(index, document.ID, opensearchtest.ToJSON(t, document))
		tc.Require.IndicesCount([]string{index}, "", 1)

		tc.Require.IndicesCount([]string{index}, opensearch.NewRootQuery(
			opensearch.NewTermQuery[bool]("Deleted").Value(true),
		).String(), 0)

		assert.NoError(t, engine.Delete(document.ID))
		tc.Require.IndicesCount([]string{index}, opensearch.NewRootQuery(
			opensearch.NewTermQuery[bool]("Deleted").Value(true),
		).String(), 1)
	})
}

func TestEngine_Restore(t *testing.T) {
	index := "test-engine-restore"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("mark document as not deleted", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.Full
		document.Deleted = true
		tc.Require.DocumentCreate(index, document.ID, opensearchtest.ToJSON(t, document))
		tc.Require.IndicesCount([]string{index}, "", 1)

		tc.Require.IndicesCount([]string{index}, opensearch.NewRootQuery(
			opensearch.NewTermQuery[bool]("Deleted").Value(true),
		).String(), 1)

		assert.NoError(t, engine.Restore(document.ID))
		tc.Require.IndicesCount([]string{index}, opensearch.NewRootQuery(
			opensearch.NewTermQuery[bool]("Deleted").Value(true),
		).String(), 0)
	})
}

func TestEngine_Purge(t *testing.T) {
	index := "test-engine-purge"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("purge with full document", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.Full
		tc.Require.DocumentCreate(index, document.ID, opensearchtest.ToJSON(t, document))
		tc.Require.IndicesCount([]string{index}, "", 1)

		assert.NoError(t, engine.Purge(document.ID))

		tc.Require.IndicesCount([]string{index}, "", 0)
	})
}

func TestEngine_DocCount(t *testing.T) {
	index := "test-engine-doc-count"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("ignore deleted documents", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.Full
		tc.Require.DocumentCreate(index, document.ID, opensearchtest.ToJSON(t, document))
		tc.Require.IndicesCount([]string{index}, "", 1)

		count, err := engine.DocCount()
		assert.NoError(t, err)
		assert.Equal(t, uint64(1), count)

		tc.Require.Update(index, document.ID, opensearchtest.ToJSON(t, map[string]any{
			"doc": map[string]any{
				"Deleted": true,
			},
		}))

		tc.Require.IndicesCount([]string{index}, "", 1)

		count, err = engine.DocCount()
		assert.NoError(t, err)
		assert.Equal(t, uint64(0), count)
	})
}
