package opensearch_test

import (
	"fmt"
	"testing"

	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/require"

	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/engine"
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

		backend, err := opensearch.NewEngine("test-engine-new-engine", client)
		require.Nil(t, backend)
		require.ErrorIs(t, err, opensearch.ErrUnhealthyCluster)
	})
}

func TestEngine_Search(t *testing.T) {
	indexName := "opencloud-test-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, "", 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewEngine(indexName, tc.Client())
	require.NoError(t, err)

	document := opensearchtest.Testdata.Resources.File
	tc.Require.DocumentCreate(indexName, document.ID, opensearchtest.JSONMustMarshal(t, document))
	tc.Require.IndicesCount([]string{indexName}, "", 1)

	t.Run("most simple search", func(t *testing.T) {
		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s"`, document.Name),
		})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		require.Equal(t, int32(1), resp.TotalMatches)
		require.Equal(t, document.ID, fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId))
	})

	t.Run("ignores files that are marked as deleted", func(t *testing.T) {
		deletedDocument := opensearchtest.Testdata.Resources.File
		deletedDocument.ID = "1$2!4"
		deletedDocument.Deleted = true

		tc.Require.DocumentCreate(indexName, deletedDocument.ID, opensearchtest.JSONMustMarshal(t, deletedDocument))
		tc.Require.IndicesCount([]string{indexName}, "", 2)

		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s"`, document.Name),
		})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		require.Equal(t, int32(1), resp.TotalMatches)
		require.Equal(t, document.ID, fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId))
	})
}

func TestEngine_Upsert(t *testing.T) {
	indexName := "opencloud-test-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, "", 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewEngine(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("upsert with full document", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		require.NoError(t, backend.Upsert(document.ID, document))

		tc.Require.IndicesCount([]string{indexName}, "", 1)
	})
}

func TestEngine_Move(t *testing.T) {
	indexName := "opencloud-test-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, "", 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewEngine(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("moves the document to a new path", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, document.ID, opensearchtest.JSONMustMarshal(t, document))
		tc.Require.IndicesCount([]string{indexName}, "", 1)

		resources := opensearchtest.SearchHitsMustBeConverted[engine.Resource](t,
			tc.Require.Search(
				indexName,
				opensearch.NewRootQuery(
					opensearch.NewIDsQuery([]string{document.ID}),
				).String(),
			).Hits,
		)
		require.Len(t, resources, 1)
		require.Equal(t, document.Path, resources[0].Path)

		document.Path = "./new/path/to/resource"
		require.NoError(t, backend.Move(document.ID, document.ParentID, document.Path))

		resources = opensearchtest.SearchHitsMustBeConverted[engine.Resource](t,
			tc.Require.Search(
				indexName,
				opensearch.NewRootQuery(
					opensearch.NewIDsQuery([]string{document.ID}),
				).String(),
			).Hits,
		)
		require.Len(t, resources, 1)
		require.Equal(t, document.Path, resources[0].Path)
	})
}

func TestEngine_Delete(t *testing.T) {
	indexName := "opencloud-test-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, "", 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewEngine(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("mark document as deleted", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, document.ID, opensearchtest.JSONMustMarshal(t, document))
		tc.Require.IndicesCount([]string{indexName}, "", 1)

		tc.Require.IndicesCount([]string{indexName}, opensearch.NewRootQuery(
			opensearch.NewTermQuery[bool]("Deleted").Value(true),
		).String(), 0)

		require.NoError(t, backend.Delete(document.ID))
		tc.Require.IndicesCount([]string{indexName}, opensearch.NewRootQuery(
			opensearch.NewTermQuery[bool]("Deleted").Value(true),
		).String(), 1)
	})
}

func TestEngine_Restore(t *testing.T) {
	indexName := "opencloud-test-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, "", 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewEngine(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("mark document as not deleted", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		document.Deleted = true
		tc.Require.DocumentCreate(indexName, document.ID, opensearchtest.JSONMustMarshal(t, document))
		tc.Require.IndicesCount([]string{indexName}, "", 1)

		tc.Require.IndicesCount([]string{indexName}, opensearch.NewRootQuery(
			opensearch.NewTermQuery[bool]("Deleted").Value(true),
		).String(), 1)

		require.NoError(t, backend.Restore(document.ID))
		tc.Require.IndicesCount([]string{indexName}, opensearch.NewRootQuery(
			opensearch.NewTermQuery[bool]("Deleted").Value(true),
		).String(), 0)
	})
}

func TestEngine_Purge(t *testing.T) {
	indexName := "opencloud-test-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, "", 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewEngine(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("purge with full document", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, document.ID, opensearchtest.JSONMustMarshal(t, document))
		tc.Require.IndicesCount([]string{indexName}, "", 1)

		require.NoError(t, backend.Purge(document.ID))

		tc.Require.IndicesCount([]string{indexName}, "", 0)
	})
}

func TestEngine_DocCount(t *testing.T) {
	indexName := "opencloud-test-resource"
	tc := opensearchtest.NewDefaultTestClient(t)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, "", 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewEngine(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("ignore deleted documents", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, document.ID, opensearchtest.JSONMustMarshal(t, document))
		tc.Require.IndicesCount([]string{indexName}, "", 1)

		count, err := backend.DocCount()
		require.NoError(t, err)
		require.Equal(t, uint64(1), count)

		tc.Require.Update(indexName, document.ID, opensearchtest.JSONMustMarshal(t, map[string]any{
			"doc": map[string]any{
				"Deleted": true,
			},
		}))

		tc.Require.IndicesCount([]string{indexName}, "", 1)

		count, err = backend.DocCount()
		require.NoError(t, err)
		require.Equal(t, uint64(0), count)
	})
}
