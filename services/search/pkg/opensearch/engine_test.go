package opensearch_test

import (
	"fmt"
	"testing"

	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/require"

	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestNewEngine(t *testing.T) {
	t.Run("fails to create if the cluster is not healthy", func(t *testing.T) {
		client, err := opensearchgoAPI.NewClient(opensearchgoAPI.Config{
			Client: opensearchgo.Config{
				Addresses: []string{"http://localhost:1025"},
			},
		})
		require.NoError(t, err, "failed to create OpenSearch client")

		engine, err := opensearch.NewEngine("test-engine-new-engine", client)
		require.Nil(t, engine)
		require.ErrorIs(t, err, opensearch.ErrUnhealthyCluster)
	})
}

func TestEngine_Search(t *testing.T) {
	index := "opencloud-default-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	document := opensearchtest.Testdata.Resources.Full
	tc.Require.DocumentCreate(index, document.ID, opensearchtest.JSONMustMarshal(t, document))
	tc.Require.IndicesCount([]string{index}, "", 1)

	engine, err := opensearch.NewEngine(index, tc.Client())
	require.NoError(t, err)

	t.Run("most simple search", func(t *testing.T) {
		resp, err := engine.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s"`, document.Name),
		})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		require.Equal(t, int32(1), resp.TotalMatches)
		require.Equal(t, document.ID, fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId))
	})

	t.Run("ignores files that are marked as deleted", func(t *testing.T) {
		deletedDocument := opensearchtest.Testdata.Resources.Full
		deletedDocument.ID = "1$2!4"
		deletedDocument.Deleted = true

		tc.Require.DocumentCreate(index, deletedDocument.ID, opensearchtest.JSONMustMarshal(t, deletedDocument))
		tc.Require.IndicesCount([]string{index}, "", 2)

		resp, err := engine.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s"`, document.Name),
		})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		require.Equal(t, int32(1), resp.TotalMatches)
		require.Equal(t, document.ID, fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId))
	})
}

func TestEngine_Upsert(t *testing.T) {
	index := "opencloud-default-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	require.NoError(t, err)

	t.Run("upsert with full document", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.Full
		require.NoError(t, engine.Upsert(document.ID, document))

		tc.Require.IndicesCount([]string{index}, "", 1)
	})
}

func TestEngine_Move(t *testing.T) {}

func TestEngine_Delete(t *testing.T) {
	index := "opencloud-default-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	require.NoError(t, err)

	t.Run("mark document as deleted", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.Full
		tc.Require.DocumentCreate(index, document.ID, opensearchtest.JSONMustMarshal(t, document))
		tc.Require.IndicesCount([]string{index}, "", 1)

		tc.Require.IndicesCount([]string{index}, opensearch.NewRootQuery(
			opensearch.NewTermQuery[bool]("Deleted").Value(true),
		).String(), 0)

		require.NoError(t, engine.Delete(document.ID))
		tc.Require.IndicesCount([]string{index}, opensearch.NewRootQuery(
			opensearch.NewTermQuery[bool]("Deleted").Value(true),
		).String(), 1)
	})
}

func TestEngine_Restore(t *testing.T) {
	index := "opencloud-default-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	require.NoError(t, err)

	t.Run("mark document as not deleted", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.Full
		document.Deleted = true
		tc.Require.DocumentCreate(index, document.ID, opensearchtest.JSONMustMarshal(t, document))
		tc.Require.IndicesCount([]string{index}, "", 1)

		tc.Require.IndicesCount([]string{index}, opensearch.NewRootQuery(
			opensearch.NewTermQuery[bool]("Deleted").Value(true),
		).String(), 1)

		require.NoError(t, engine.Restore(document.ID))
		tc.Require.IndicesCount([]string{index}, opensearch.NewRootQuery(
			opensearch.NewTermQuery[bool]("Deleted").Value(true),
		).String(), 0)
	})
}

func TestEngine_Purge(t *testing.T) {
	index := "opencloud-default-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	require.NoError(t, err)

	t.Run("purge with full document", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.Full
		tc.Require.DocumentCreate(index, document.ID, opensearchtest.JSONMustMarshal(t, document))
		tc.Require.IndicesCount([]string{index}, "", 1)

		require.NoError(t, engine.Purge(document.ID))

		tc.Require.IndicesCount([]string{index}, "", 0)
	})
}

func TestEngine_DocCount(t *testing.T) {
	index := "opencloud-default-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{index})
	tc.Require.IndicesCount([]string{index}, "", 0)

	defer tc.Require.IndicesDelete([]string{index})

	engine, err := opensearch.NewEngine(index, tc.Client())
	require.NoError(t, err)

	t.Run("ignore deleted documents", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.Full
		tc.Require.DocumentCreate(index, document.ID, opensearchtest.JSONMustMarshal(t, document))
		tc.Require.IndicesCount([]string{index}, "", 1)

		count, err := engine.DocCount()
		require.NoError(t, err)
		require.Equal(t, uint64(1), count)

		tc.Require.Update(index, document.ID, opensearchtest.JSONMustMarshal(t, map[string]any{
			"doc": map[string]any{
				"Deleted": true,
			},
		}))

		tc.Require.IndicesCount([]string{index}, "", 1)

		count, err = engine.DocCount()
		require.NoError(t, err)
		require.Equal(t, uint64(0), count)
	})
}
