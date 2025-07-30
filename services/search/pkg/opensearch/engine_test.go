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
	tc := ostest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	document := ostest.Testdata.Resources.Full
	tc.Require.DocumentCreate(index, document.ID, toJSON(t, document))
	tc.Require.IndicesCount([]string{index}, "", 1)

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("most simple search", func(t *testing.T) {
		resp, err := engine.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s" Content:"%s"`, document.Name, document.Content),
		})
		assert.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		assert.Equal(t, int32(1), resp.TotalMatches)
		assert.Equal(t, document.Name, resp.Matches[0].Entity.Name)
	})
}

func TestEngine_Upsert(t *testing.T) {
	index := "test-engine-upsert"
	tc := ostest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("upsert with full document", func(t *testing.T) {
		document := ostest.Testdata.Resources.Full
		assert.NoError(t, engine.Upsert(document.ID, document))

		tc.Require.IndicesCount([]string{index}, "", 1)
	})
}

func TestEngine_Move(t *testing.T) {}

func TestEngine_Delete(t *testing.T) {
	index := "test-engine-delete"
	tc := ostest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("mark document as deleted", func(t *testing.T) {
		document := ostest.Testdata.Resources.Full
		tc.Require.DocumentCreate(index, document.ID, toJSON(t, document))
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
	tc := ostest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("mark document as not deleted", func(t *testing.T) {
		document := ostest.Testdata.Resources.Full
		document.Deleted = true
		tc.Require.DocumentCreate(index, document.ID, toJSON(t, document))
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
	tc := ostest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("purge with full document", func(t *testing.T) {
		document := ostest.Testdata.Resources.Full
		tc.Require.DocumentCreate(index, document.ID, toJSON(t, document))
		tc.Require.IndicesCount([]string{index}, "", 1)

		assert.NoError(t, engine.Purge(document.ID))

		tc.Require.IndicesCount([]string{index}, "", 0)
	})
}

func TestEngine_DocCount(t *testing.T) {
	index := "test-engine-doc-count"
	tc := ostest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	assert.NoError(t, err)

	t.Run("ignore deleted documents", func(t *testing.T) {
		document := ostest.Testdata.Resources.Full
		tc.Require.DocumentCreate(index, document.ID, toJSON(t, document))
		tc.Require.IndicesCount([]string{index}, "", 1)

		count, err := engine.DocCount()
		assert.NoError(t, err)
		assert.Equal(t, uint64(1), count)

		tc.Require.Update(index, document.ID, toJSON(t, map[string]any{
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
