package opensearch_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/require"

	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func TestNewBackend(t *testing.T) {
	t.Run("fails to create if the cluster is not healthy", func(t *testing.T) {
		client, err := opensearchgoAPI.NewClient(opensearchgoAPI.Config{
			Client: opensearchgo.Config{
				Addresses: []string{"http://localhost:1025"},
			},
		})
		require.NoError(t, err, "failed to create OpenSearch client")

		backend, err := opensearch.NewBackend("test-engine-new-engine", client)
		require.Nil(t, backend)
		require.ErrorIs(t, err, opensearch.ErrUnhealthyCluster)
	})
}

func TestEngine_Search(t *testing.T) {
	indexName := "opencloud-test-engine-search"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	// Upsert (not DocumentCreate) so PrepareForIndex writes the _lowercase
	// siblings the case-insensitive Name search relies on.
	document := opensearchtest.Testdata.Resources.File
	require.NoError(t, backend.Upsert(document.ID, document))
	tc.Require.IndicesRefresh([]string{indexName}, nil)
	tc.Require.IndicesCount([]string{indexName}, nil, 1)

	t.Run("most simple search", func(t *testing.T) {
		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s"`, document.Name),
		})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		require.Equal(t, int32(1), resp.TotalMatches)
		require.Equal(t, document.ID, fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId))
	})

	t.Run("path scope restricts hits and totals at query level", func(t *testing.T) {
		outside := opensearchtest.Testdata.Resources.File
		outside.ID = "1$1!5"
		outside.Path = "./other folder/else.jpg"
		require.NoError(t, backend.Upsert(outside.ID, outside))
		tc.Require.IndicesRefresh([]string{indexName}, nil)

		scoped := &searchMessage.Reference{
			ResourceId: &searchMessage.ResourceID{StorageId: "1", SpaceId: "1", OpaqueId: "1"},
			Path:       "./parent d!r",
		}
		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s"`, document.Name),
			Ref:   scoped,
		})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		require.Equal(t, int32(1), resp.TotalMatches)
		require.Equal(t, "./parent d!r/child.jpg", resp.Matches[0].Entity.Ref.Path)

		// the scope is a reference and matches case-sensitively
		wrongCase := &searchMessage.Reference{
			ResourceId: &searchMessage.ResourceID{StorageId: "1", SpaceId: "1", OpaqueId: "1"},
			Path:       "./PARENT D!R",
		}
		respWrongCase, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s"`, document.Name),
			Ref:   wrongCase,
		})
		require.NoError(t, err)
		require.Len(t, respWrongCase.Matches, 0)
		require.Equal(t, int32(0), respWrongCase.TotalMatches)

		require.NoError(t, backend.Purge(outside.ID, false))
		tc.Require.IndicesRefresh([]string{indexName}, nil)
	})

	t.Run("ignores files that are marked as deleted", func(t *testing.T) {
		deletedDocument := opensearchtest.Testdata.Resources.File
		deletedDocument.ID = "1$2!4"
		deletedDocument.Deleted = true

		require.NoError(t, backend.Upsert(deletedDocument.ID, deletedDocument))
		tc.Require.IndicesRefresh([]string{indexName}, nil)
		tc.Require.IndicesCount([]string{indexName}, nil, 2)

		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: fmt.Sprintf(`"%s"`, document.Name),
		})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		require.Equal(t, int32(1), resp.TotalMatches)
		require.Equal(t, document.ID, fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId))
	})
}

func TestEngine_FullTextSearch(t *testing.T) {
	indexName := "opencloud-test-engine-fulltext"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	document := opensearchtest.Testdata.Resources.File
	document.Content = "Running Foxes"
	require.NoError(t, backend.Upsert(document.ID, document))
	tc.Require.IndicesRefresh([]string{indexName}, nil)

	t.Run("content search is case-insensitive and stemmed, like bleve", func(t *testing.T) {
		// case-folded and porter-stemmed by the fulltext analyzer; the match
		// query analyzes the query value the same way.
		// "content:run*" is an unanalyzed wildcard over the stemmed term "run", so
		// it must still route to a wildcard query (not degrade to a phrase match).
		for _, q := range []string{"content:running", "content:RUNNING", "content:run", "content:run*"} {
			resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{Query: q})
			require.NoError(t, err, q)
			require.Len(t, resp.Matches, 1, q)
		}

		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{Query: "content:cat"})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 0)
	})
}

func TestEngine_CaseInsensitiveSearch(t *testing.T) {
	indexName := "opencloud-test-engine-ci"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	folder := opensearchtest.Testdata.Resources.Folder
	folder.ID = "1$2!cifolder"
	folder.Path = "./My Dir"
	folder.Tags = []string{"Work", "Urgent"}
	require.NoError(t, backend.Upsert(folder.ID, folder))

	child := opensearchtest.Testdata.Resources.File
	child.ID = "1$2!cichild"
	child.ParentID = folder.ID
	child.Path = "./My Dir/report.pdf"
	child.Tags = nil
	require.NoError(t, backend.Upsert(child.ID, child))

	// a doc outside the folder, so the path assertions below discriminate: a
	// phrase-matched path query would analyze into the "." prefix and match
	// this one too
	outside := opensearchtest.Testdata.Resources.File
	outside.ID = "1$2!cioutside"
	outside.Path = "./other.pdf"
	outside.Tags = nil
	require.NoError(t, backend.Upsert(outside.ID, outside))
	tc.Require.IndicesRefresh([]string{indexName}, nil)

	t.Run("tags match case-insensitively (array sibling)", func(t *testing.T) {
		for _, q := range []string{"tag:work", "tag:WORK", "Tags:Urgent"} {
			resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{Query: q})
			require.NoError(t, err, q)
			require.Len(t, resp.Matches, 1, q)
		}
	})

	t.Run("path with spaces matches folder and descendants case-sensitively", func(t *testing.T) {
		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{Query: `path:"./My Dir"`})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 2) // folder itself + the descendant, not the outside doc

		// paths act as references, a wrong-cased path must not match
		respWrongCase, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{Query: `path:"./MY DIR"`})
		require.NoError(t, err)
		require.Len(t, respWrongCase.Matches, 0)
	})

	t.Run("path with spaces matches a descendant only itself", func(t *testing.T) {
		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{Query: `path:"./My Dir/report.pdf"`})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 1)
	})
}

func TestEngine_MediaTypeSearch(t *testing.T) {
	indexName := "opencloud-test-engine-mediatype"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	svg := opensearchtest.Testdata.Resources.File
	svg.ID = "1$2!svg"
	svg.MimeType = "image/svg+xml"
	require.NoError(t, backend.Upsert(svg.ID, svg))

	png := opensearchtest.Testdata.Resources.File
	png.ID = "1$2!png"
	png.MimeType = "image/png"
	require.NoError(t, backend.Upsert(png.ID, png))

	folder := opensearchtest.Testdata.Resources.Folder
	folder.ID = "1$2!dir"
	folder.MimeType = "httpd/unix-directory"
	require.NoError(t, backend.Upsert(folder.ID, folder))
	tc.Require.IndicesRefresh([]string{indexName}, nil)

	cases := []struct {
		query string
		want  int
	}{
		{"mediatype:image", 2},         // image/* wildcard -> both files
		{"mediatype:IMAGE", 2},         // categories are case-insensitive
		{"mediatype:image/svg+xml", 1}, // literal MIME (+ and /) via mediatype
		{"MimeType:image/svg+xml", 1},  // same literal via the raw field name
		{"mediatype:image/png", 1},
		{"mediatype:pdf", 0},
		{"mediatype:folder", 1},                      // the directory only
		{"mediatype:file", 2},                        // both files, not the directory
		{"mediatype:file AND MimeType:image/png", 1}, // file NOT-dir combined with a term
	}
	for _, c := range cases {
		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{Query: c.query})
		require.NoError(t, err, c.query)
		require.Len(t, resp.Matches, c.want, c.query)
	}
}

func TestEngine_Upsert(t *testing.T) {
	indexName := "opencloud-test-engine-upsert"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("upsert with full document", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		require.NoError(t, backend.Upsert(document.ID, document))

		tc.Require.IndicesCount([]string{indexName}, nil, 1)
	})

	t.Run("upsert without mtime", func(t *testing.T) {
		// content.Extract leaves Mtime nil when the resource info carries none
		document := opensearchtest.Testdata.Resources.File
		document.ID = "1$1!4"
		document.Mtime = nil
		require.NoError(t, backend.Upsert(document.ID, document))

		tc.Require.IndicesCount([]string{indexName}, nil, 2)
	})
}

func TestEngine_Move(t *testing.T) {
	indexName := "opencloud-test-engine-move"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("moves the document to a new path", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, document)))
		tc.Require.IndicesCount([]string{indexName}, nil, 1)

		body := opensearchtest.JSONMustMarshal(t, map[string]any{
			"query": map[string]any{
				"ids": map[string]any{
					"values": []string{document.ID},
				},
			},
		})

		resources := opensearchtest.SearchHitsMustBeConverted[search.Resource](t, tc.Require.Search(indexName, strings.NewReader(body)).Hits)
		require.Len(t, resources, 1)
		require.Equal(t, document.Path, resources[0].Path)

		document.Path = "./new/path/to/resource"
		require.NoError(t, backend.Move(document.ID, document.ParentID, document.Path))

		resources = opensearchtest.SearchHitsMustBeConverted[search.Resource](t, tc.Require.Search(indexName, strings.NewReader(body)).Hits)
		require.Len(t, resources, 1)
		require.Equal(t, document.Path, resources[0].Path)
	})

	t.Run("keeps case-sensitive path search working after a move", func(t *testing.T) {
		// Spaced paths so the queries only stay exact as term queries; a phrase
		// match would analyze into the "." prefix and match regardless.
		document := opensearchtest.Testdata.Resources.File
		document.ID = "1$2!cimove"
		document.Path = "./Foo Dir/Bar"
		require.NoError(t, backend.Upsert(document.ID, document))
		tc.Require.IndicesRefresh([]string{indexName}, nil)

		document.Path = "./Moved Dir/Bar"
		require.NoError(t, backend.Move(document.ID, document.ParentID, document.Path))
		tc.Require.IndicesRefresh([]string{indexName}, nil)

		// Path is case-sensitive by design: the exact new path matches, a
		// wrong-cased query does not, and the old path no longer matches.
		respNew, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{Query: `path:"./Moved Dir/Bar"`})
		require.NoError(t, err)
		require.Len(t, respNew.Matches, 1)

		respWrongCase, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{Query: `path:"./MOVED DIR/BAR"`})
		require.NoError(t, err)
		require.Len(t, respWrongCase.Matches, 0)

		respOld, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{Query: `path:"./Foo Dir/Bar"`})
		require.NoError(t, err)
		require.Len(t, respOld.Matches, 0)
	})
}

func TestEngine_Delete(t *testing.T) {
	indexName := "opencloud-test-engine-delete"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("mark document as deleted", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, document)))
		tc.Require.IndicesCount([]string{indexName}, nil, 1)

		body := opensearchtest.JSONMustMarshal(t, map[string]any{
			"query": map[string]any{
				"term": map[string]any{
					"Deleted": map[string]any{
						"value": true,
					},
				},
			},
		})

		tc.Require.IndicesCount([]string{indexName}, strings.NewReader(body), 0)

		require.NoError(t, backend.Delete(document.ID))
		tc.Require.IndicesCount([]string{indexName}, strings.NewReader(body), 1)
	})
}

func TestEngine_Restore(t *testing.T) {
	indexName := "opencloud-test-engine-restore"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("mark document as not deleted", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		document.Deleted = true
		tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, document)))
		tc.Require.IndicesCount([]string{indexName}, nil, 1)

		body := opensearchtest.JSONMustMarshal(t, map[string]any{
			"query": map[string]any{
				"term": map[string]any{
					"Deleted": map[string]any{
						"value": true,
					},
				},
			},
		})

		tc.Require.IndicesCount([]string{indexName}, strings.NewReader(body), 1)

		require.NoError(t, backend.Restore(document.ID))
		tc.Require.IndicesCount([]string{indexName}, strings.NewReader(body), 0)
	})
}

func TestEngine_Purge(t *testing.T) {
	indexName := "opencloud-test-engine-purge"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("purge with full document", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, document)))
		tc.Require.IndicesCount([]string{indexName}, nil, 1)

		require.NoError(t, backend.Purge(document.ID, false))

		tc.Require.IndicesCount([]string{indexName}, nil, 0)
	})

	t.Run("purge resource trees", func(t *testing.T) {
		resourceFolder := opensearchtest.Testdata.Resources.Folder
		tc.Require.DocumentCreate(indexName, resourceFolder.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, resourceFolder)))

		resourceFile := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, resourceFile.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, resourceFile)))

		tc.Require.IndicesCount([]string{indexName}, nil, 2)

		require.NoError(t, backend.Purge(resourceFolder.ID, false))

		tc.Require.IndicesCount([]string{indexName}, nil, 0)
	})

	t.Run("purge resource trees and ignores undeleted resources", func(t *testing.T) {
		resourceFolder := opensearchtest.Testdata.Resources.Folder
		tc.Require.DocumentCreate(indexName, resourceFolder.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, resourceFolder)))

		resourceFile := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, resourceFile.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, resourceFile)))

		tc.Require.IndicesCount([]string{indexName}, nil, 2)

		require.NoError(t, backend.Delete(resourceFile.ID))
		tc.Require.IndicesRefresh([]string{indexName}, nil)
		require.NoError(t, backend.Purge(resourceFolder.ID, true))

		tc.Require.IndicesCount([]string{indexName}, nil, 1)
	})
}

func TestEngine_DocCount(t *testing.T) {
	indexName := "opencloud-test-engine-doc-count"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client())
	require.NoError(t, err)

	t.Run("ignore deleted documents", func(t *testing.T) {
		document := opensearchtest.Testdata.Resources.File
		tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, document)))
		tc.Require.IndicesCount([]string{indexName}, nil, 1)

		count, err := backend.DocCount()
		require.NoError(t, err)
		require.Equal(t, uint64(1), count)

		tc.Require.Update(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, map[string]any{
			"doc": map[string]any{
				"Deleted": true,
			},
		})))

		tc.Require.IndicesCount([]string{indexName}, nil, 1)

		count, err = backend.DocCount()
		require.NoError(t, err)
		require.Equal(t, uint64(0), count)
	})
}

// axisVector returns a one-hot vector of the schema dimensionality: distinct
// axes are orthogonal (cosine 0), same axes identical (cosine 1).
func axisVector(axis int) []float32 {
	v := make([]float32, content.ImageVectorDims)
	v[axis] = 1
	return v
}

// fakeVectorizer maps query texts to fixed vectors.
type fakeVectorizer struct{ byText map[string][]float32 }

func (f fakeVectorizer) VectorizeText(_ context.Context, text string) ([]float32, error) {
	v, ok := f.byText[strings.ToLower(text)]
	if !ok {
		return nil, fmt.Errorf("no vector for %q", text)
	}
	return v, nil
}

func TestEngine_SemanticSearch(t *testing.T) {
	indexName := "opencloud-test-engine-semantic"
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	defer tc.Require.IndicesDelete([]string{indexName})

	backend, err := opensearch.NewBackend(indexName, tc.Client(), opensearch.WithTextVectorizer(fakeVectorizer{
		byText: map[string][]float32{
			"cat": axisVector(0),
			"dog": axisVector(1),
		},
	}))
	require.NoError(t, err)

	upsert := func(id, path, name string, vector []float32) {
		r := opensearchtest.Testdata.Resources.File
		r.ID = id
		r.Path = path
		r.Name = name
		r.ImageVector = vector
		require.NoError(t, backend.Upsert(id, r))
	}
	upsert("1$1!10", "./cat.jpg", "cat.jpg", axisVector(0))
	upsert("1$1!11", "./dog.jpg", "dog.jpg", axisVector(1))
	upsert("1$1!12", "./note.txt", "note.txt", nil)
	tc.Require.IndicesRefresh([]string{indexName}, nil)

	t.Run("purely semantic query ranks the nearest image first", func(t *testing.T) {
		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: `semantic:"cat"`,
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Matches)
		require.Equal(t, "cat.jpg", resp.Matches[0].Entity.Name)
		// the only honest total is the number of semantic hits
		require.Equal(t, int32(2), resp.TotalMatches)
	})

	t.Run("semantic clause combines with a filter part", func(t *testing.T) {
		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: `Name:dog* AND semantic:"cat"`,
		})
		require.NoError(t, err)
		require.Len(t, resp.Matches, 1)
		require.Equal(t, "dog.jpg", resp.Matches[0].Entity.Name)
		require.Equal(t, int32(1), resp.TotalMatches)
	})

	t.Run("semantic inside a quoted value stays a literal", func(t *testing.T) {
		resp, err := backend.Search(t.Context(), &searchService.SearchIndexRequest{
			Query: `Name:"*semantic:cat*"`,
		})
		require.NoError(t, err)
		require.Empty(t, resp.Matches)
	})

	t.Run("rejects semantic queries without a vectorizer", func(t *testing.T) {
		bare, err := opensearch.NewBackend(indexName, tc.Client())
		require.NoError(t, err)
		_, err = bare.Search(t.Context(), &searchService.SearchIndexRequest{Query: `semantic:"cat"`})
		require.ErrorContains(t, err, "not configured")
	})
}
