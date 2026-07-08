package opensearch_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	opensearchtest "github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func TestOpenSearchBackend(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpenSearch Backend Suite")
}

func deleteIndexOnCleanup(tc *opensearchtest.TestClient, indexName string) {
	DeferCleanup(func() {
		Expect(tc.IndicesDelete(context.Background(), []string{indexName})).To(Succeed())
	})
}

func resourceByID(tc *opensearchtest.TestClient, index, id string) search.Resource {
	GinkgoHelper()

	body := opensearchtest.JSONMustMarshal(GinkgoTB(), map[string]any{
		"query": map[string]any{
			"ids": map[string]any{
				"values": []string{id},
			},
		},
	})

	resources := opensearchtest.SearchHitsMustBeConverted[search.Resource](GinkgoTB(), tc.Require.Search(index, strings.NewReader(body)).Hits)
	Expect(resources).To(HaveLen(1))
	return resources[0]
}

// otherRoot returns a copy of the given resource that lives in a different root (space)
// while keeping the same path, so it can be used to assert that cross-root updates do
// not affect identically-named resources in other roots.
func otherRoot(r search.Resource) search.Resource {
	r.ID = "2$2!3"
	r.RootID = "2$2!1"
	r.ParentID = "2$2!2"
	return r
}

func newBackend(indexName string, resources ...search.Resource) (*opensearch.Backend, *opensearchtest.TestClient) {
	GinkgoHelper()

	tc := opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
	tc.Require.IndicesReset([]string{indexName})
	tc.Require.IndicesCount([]string{indexName}, nil, 0)

	backend, err := opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
	Expect(err).ToNot(HaveOccurred())

	for _, r := range resources {
		tc.Require.DocumentCreate(indexName, r.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), r)))
	}
	tc.Require.IndicesCount([]string{indexName}, nil, len(resources))

	return backend, tc
}

var _ = Describe("Backend", func() {
	Describe("NewBackend", func() {
		It("fails to create if the cluster is not healthy", func() {
			client, err := opensearchgoAPI.NewClient(opensearchgoAPI.Config{
				Client: opensearchgo.Config{
					Addresses: []string{"http://localhost:1025"},
				},
			})
			Expect(err).ToNot(HaveOccurred(), "failed to create OpenSearch client")

			backend, err := opensearch.NewBackend(context.Background(), "test-engine-new-engine", client, log.NopLogger())
			Expect(backend).To(BeNil())
			Expect(err).To(MatchError(opensearch.ErrUnhealthyCluster))
		})
	})

	Describe("Search", func() {
		const indexName = "opencloud-test-engine-search"

		var (
			tc       *opensearchtest.TestClient
			backend  *opensearch.Backend
			document search.Resource
		)

		BeforeEach(func() {
			tc = opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{indexName})
			tc.Require.IndicesCount([]string{indexName}, nil, 0)
			deleteIndexOnCleanup(tc, indexName)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())

			document = opensearchtest.Testdata.Resources.File
			// Upsert (not DocumentCreate) so PrepareForIndex writes the _lowercase
			// siblings the case-insensitive Name and path scope searches rely on.
			Expect(backend.Upsert(document.ID, document)).To(Succeed())
			tc.Require.IndicesRefresh([]string{indexName}, nil)
			tc.Require.IndicesCount([]string{indexName}, nil, 1)
		})

		It("performs the most simple search", func() {
			resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{
				Query: fmt.Sprintf(`"%s"`, document.Name),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Matches).To(HaveLen(1))
			Expect(resp.TotalMatches).To(Equal(int32(1)))
			Expect(fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId)).To(Equal(document.ID))
		})

		It("ignores files that are marked as deleted", func() {
			deletedDocument := opensearchtest.Testdata.Resources.File
			deletedDocument.ID = "1$2!4"
			deletedDocument.Deleted = true

			Expect(backend.Upsert(deletedDocument.ID, deletedDocument)).To(Succeed())
			tc.Require.IndicesRefresh([]string{indexName}, nil)
			tc.Require.IndicesCount([]string{indexName}, nil, 2)

			resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{
				Query: fmt.Sprintf(`"%s"`, document.Name),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Matches).To(HaveLen(1))
			Expect(resp.TotalMatches).To(Equal(int32(1)))
			Expect(fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId)).To(Equal(document.ID))
		})

		It("restricts hits and totals to the path scope", func() {
			outside := opensearchtest.Testdata.Resources.File
			outside.ID = "1$1!5"
			outside.Path = "./other folder/else.jpg"
			Expect(backend.Upsert(outside.ID, outside)).To(Succeed())
			tc.Require.IndicesRefresh([]string{indexName}, nil)

			scoped := &searchMessage.Reference{
				ResourceId: &searchMessage.ResourceID{StorageId: "1", SpaceId: "1", OpaqueId: "1"},
				Path:       "./parent d!r",
			}
			resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{
				Query: fmt.Sprintf(`"%s"`, document.Name),
				Ref:   scoped,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Matches).To(HaveLen(1))
			Expect(resp.TotalMatches).To(Equal(int32(1)))
			Expect(resp.Matches[0].Entity.Ref.Path).To(Equal("./parent d!r/child.jpg"))

			// the scope is a reference and matches case-sensitively
			wrongCase := &searchMessage.Reference{
				ResourceId: &searchMessage.ResourceID{StorageId: "1", SpaceId: "1", OpaqueId: "1"},
				Path:       "./PARENT D!R",
			}
			respWrongCase, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{
				Query: fmt.Sprintf(`"%s"`, document.Name),
				Ref:   wrongCase,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(respWrongCase.Matches).To(HaveLen(0))
			Expect(respWrongCase.TotalMatches).To(Equal(int32(0)))
		})
	})

	Describe("FullTextSearch", func() {
		const indexName = "opencloud-test-engine-fulltext"

		var (
			tc      *opensearchtest.TestClient
			backend *opensearch.Backend
		)

		BeforeEach(func() {
			tc = opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{indexName})
			tc.Require.IndicesCount([]string{indexName}, nil, 0)
			deleteIndexOnCleanup(tc, indexName)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())

			document := opensearchtest.Testdata.Resources.File
			document.Content = "Running Foxes"
			Expect(backend.Upsert(document.ID, document)).To(Succeed())
			tc.Require.IndicesRefresh([]string{indexName}, nil)
		})

		It("searches content case-insensitively and stemmed, like bleve", func() {
			// case-folded and porter-stemmed by the fulltext analyzer; the match
			// query analyzes the query value the same way. "content:run*" is an
			// unanalyzed wildcard over the stemmed term "run", so it must still
			// route to a wildcard query (not degrade to a phrase match).
			for _, q := range []string{"content:running", "content:RUNNING", "content:run", "content:run*"} {
				resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: q})
				Expect(err).ToNot(HaveOccurred(), q)
				Expect(resp.Matches).To(HaveLen(1), q)
			}

			resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: "content:cat"})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Matches).To(HaveLen(0))
		})
	})

	Describe("CaseInsensitiveSearch", func() {
		const indexName = "opencloud-test-engine-ci"

		var (
			tc      *opensearchtest.TestClient
			backend *opensearch.Backend
		)

		BeforeEach(func() {
			tc = opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{indexName})
			tc.Require.IndicesCount([]string{indexName}, nil, 0)
			deleteIndexOnCleanup(tc, indexName)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())

			folder := opensearchtest.Testdata.Resources.Folder
			folder.ID = "1$2!cifolder"
			folder.Path = "./My Dir"
			folder.Tags = []string{"Work", "Urgent"}
			Expect(backend.Upsert(folder.ID, folder)).To(Succeed())

			child := opensearchtest.Testdata.Resources.File
			child.ID = "1$2!cichild"
			child.ParentID = folder.ID
			child.Path = "./My Dir/report.pdf"
			child.Tags = nil
			Expect(backend.Upsert(child.ID, child)).To(Succeed())

			// a doc outside the folder, so the path assertions below discriminate:
			// a phrase-matched path query would analyze into the "." prefix and
			// match this one too
			outside := opensearchtest.Testdata.Resources.File
			outside.ID = "1$2!cioutside"
			outside.Path = "./other.pdf"
			outside.Tags = nil
			Expect(backend.Upsert(outside.ID, outside)).To(Succeed())
			tc.Require.IndicesRefresh([]string{indexName}, nil)
		})

		It("matches tags case-insensitively (array sibling)", func() {
			for _, q := range []string{"tag:work", "tag:WORK", "Tags:Urgent"} {
				resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: q})
				Expect(err).ToNot(HaveOccurred(), q)
				Expect(resp.Matches).To(HaveLen(1), q)
			}
		})

		It("matches a spaced path on the folder and its descendants case-sensitively", func() {
			resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: `path:"./My Dir"`})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Matches).To(HaveLen(2)) // folder itself + the descendant, not the outside doc

			// paths act as references, a wrong-cased path must not match
			respWrongCase, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: `path:"./MY DIR"`})
			Expect(err).ToNot(HaveOccurred())
			Expect(respWrongCase.Matches).To(HaveLen(0))
		})

		It("matches a spaced descendant path only itself", func() {
			resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: `path:"./My Dir/report.pdf"`})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Matches).To(HaveLen(1))
		})
	})

	Describe("MediaTypeSearch", func() {
		const indexName = "opencloud-test-engine-mediatype"

		var (
			tc      *opensearchtest.TestClient
			backend *opensearch.Backend
		)

		BeforeEach(func() {
			tc = opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{indexName})
			tc.Require.IndicesCount([]string{indexName}, nil, 0)
			deleteIndexOnCleanup(tc, indexName)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())

			svg := opensearchtest.Testdata.Resources.File
			svg.ID = "1$2!svg"
			svg.MimeType = "image/svg+xml"
			Expect(backend.Upsert(svg.ID, svg)).To(Succeed())

			png := opensearchtest.Testdata.Resources.File
			png.ID = "1$2!png"
			png.MimeType = "image/png"
			Expect(backend.Upsert(png.ID, png)).To(Succeed())

			folder := opensearchtest.Testdata.Resources.Folder
			folder.ID = "1$2!dir"
			folder.MimeType = "httpd/unix-directory"
			Expect(backend.Upsert(folder.ID, folder)).To(Succeed())
			tc.Require.IndicesRefresh([]string{indexName}, nil)
		})

		DescribeTable("resolves the media type query",
			func(query string, want int) {
				resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: query})
				Expect(err).ToNot(HaveOccurred(), query)
				Expect(resp.Matches).To(HaveLen(want), query)
			},
			Entry("image/* wildcard matches both files", "mediatype:image", 2),
			Entry("categories are case-insensitive", "mediatype:IMAGE", 2),
			Entry("literal MIME (+ and /) via mediatype", "mediatype:image/svg+xml", 1),
			Entry("same literal via the raw field name", "MimeType:image/svg+xml", 1),
			Entry("literal png MIME", "mediatype:image/png", 1),
			Entry("no pdf documents", "mediatype:pdf", 0),
			Entry("folder category matches the directory only", "mediatype:folder", 1),
			Entry("file category matches both files, not the directory", "mediatype:file", 2),
			Entry("file category combined with a term", "mediatype:file AND MimeType:image/png", 1),
		)
	})

	Describe("Upsert", func() {
		const indexName = "opencloud-test-engine-upsert"

		var (
			tc      *opensearchtest.TestClient
			backend *opensearch.Backend
		)

		BeforeEach(func() {
			tc = opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{indexName})
			tc.Require.IndicesCount([]string{indexName}, nil, 0)
			deleteIndexOnCleanup(tc, indexName)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
		})

		It("upserts a full document", func() {
			document := opensearchtest.Testdata.Resources.File
			Expect(backend.Upsert(document.ID, document)).To(Succeed())

			tc.Require.IndicesCount([]string{indexName}, nil, 1)
		})

		It("upserts a document without an mtime", func() {
			// content.Extract leaves Mtime nil when the resource info carries none
			document := opensearchtest.Testdata.Resources.File
			document.ID = "1$1!4"
			document.Mtime = nil
			Expect(backend.Upsert(document.ID, document)).To(Succeed())

			tc.Require.IndicesCount([]string{indexName}, nil, 1)
		})
	})

	Describe("Move", func() {
		const indexName = "opencloud-test-engine-move"

		var (
			tc      *opensearchtest.TestClient
			backend *opensearch.Backend
		)

		BeforeEach(func() {
			tc = opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{indexName})
			tc.Require.IndicesCount([]string{indexName}, nil, 0)
			deleteIndexOnCleanup(tc, indexName)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
		})

		It("moves the document to a new path", func() {
			document := opensearchtest.Testdata.Resources.File
			tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), document)))
			tc.Require.IndicesCount([]string{indexName}, nil, 1)

			body := opensearchtest.JSONMustMarshal(GinkgoTB(), map[string]any{
				"query": map[string]any{
					"ids": map[string]any{
						"values": []string{document.ID},
					},
				},
			})

			resources := opensearchtest.SearchHitsMustBeConverted[search.Resource](GinkgoTB(), tc.Require.Search(indexName, strings.NewReader(body)).Hits)
			Expect(resources).To(HaveLen(1))
			Expect(resources[0].Path).To(Equal(document.Path))

			document.Path = "./new/path/to/resource"
			Expect(backend.Move(document.ID, document.ParentID, document.Path)).To(Succeed())

			resources = opensearchtest.SearchHitsMustBeConverted[search.Resource](GinkgoTB(), tc.Require.Search(indexName, strings.NewReader(body)).Hits)
			Expect(resources).To(HaveLen(1))
			Expect(resources[0].Path).To(Equal(document.Path))
		})

		It("keeps case-sensitive path search working after a move", func() {
			// Spaced paths so the queries only stay exact as term queries; a phrase
			// match would analyze into the "." prefix and match regardless.
			document := opensearchtest.Testdata.Resources.File
			document.ID = "1$2!cimove"
			document.Path = "./Foo Dir/Bar"
			Expect(backend.Upsert(document.ID, document)).To(Succeed())
			tc.Require.IndicesRefresh([]string{indexName}, nil)

			document.Path = "./Moved Dir/Bar"
			Expect(backend.Move(document.ID, document.ParentID, document.Path)).To(Succeed())
			tc.Require.IndicesRefresh([]string{indexName}, nil)

			// Path is case-sensitive by design: the exact new path matches, a
			// wrong-cased query does not, and the old path no longer matches.
			respNew, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: `path:"./Moved Dir/Bar"`})
			Expect(err).ToNot(HaveOccurred())
			Expect(respNew.Matches).To(HaveLen(1))

			respWrongCase, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: `path:"./MOVED DIR/BAR"`})
			Expect(err).ToNot(HaveOccurred())
			Expect(respWrongCase.Matches).To(HaveLen(0))

			respOld, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: `path:"./Foo Dir/Bar"`})
			Expect(err).ToNot(HaveOccurred())
			Expect(respOld.Matches).To(HaveLen(0))
		})
	})

	Describe("Delete", func() {
		const indexName = "opencloud-test-engine-delete"

		var (
			tc      *opensearchtest.TestClient
			backend *opensearch.Backend
		)

		BeforeEach(func() {
			tc = opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{indexName})
			tc.Require.IndicesCount([]string{indexName}, nil, 0)
			deleteIndexOnCleanup(tc, indexName)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
		})

		It("marks the document as deleted", func() {
			document := opensearchtest.Testdata.Resources.File
			tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), document)))
			tc.Require.IndicesCount([]string{indexName}, nil, 1)

			body := opensearchtest.JSONMustMarshal(GinkgoTB(), map[string]any{
				"query": map[string]any{
					"term": map[string]any{
						"Deleted": map[string]any{
							"value": true,
						},
					},
				},
			})

			tc.Require.IndicesCount([]string{indexName}, strings.NewReader(body), 0)

			Expect(backend.Delete(document.ID)).To(Succeed())
			tc.Require.IndicesCount([]string{indexName}, strings.NewReader(body), 1)
		})
	})

	Describe("Restore", func() {
		const indexName = "opencloud-test-engine-restore"

		var (
			tc      *opensearchtest.TestClient
			backend *opensearch.Backend
		)

		BeforeEach(func() {
			tc = opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{indexName})
			tc.Require.IndicesCount([]string{indexName}, nil, 0)
			deleteIndexOnCleanup(tc, indexName)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
		})

		It("marks the document as not deleted", func() {
			document := opensearchtest.Testdata.Resources.File
			document.Deleted = true
			tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), document)))
			tc.Require.IndicesCount([]string{indexName}, nil, 1)

			body := opensearchtest.JSONMustMarshal(GinkgoTB(), map[string]any{
				"query": map[string]any{
					"term": map[string]any{
						"Deleted": map[string]any{
							"value": true,
						},
					},
				},
			})

			tc.Require.IndicesCount([]string{indexName}, strings.NewReader(body), 1)

			Expect(backend.Restore(document.ID)).To(Succeed())
			tc.Require.IndicesCount([]string{indexName}, strings.NewReader(body), 0)
		})
	})

	Describe("Purge", func() {
		const indexName = "opencloud-test-engine-purge"

		var (
			tc      *opensearchtest.TestClient
			backend *opensearch.Backend
		)

		BeforeEach(func() {
			tc = opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{indexName})
			tc.Require.IndicesCount([]string{indexName}, nil, 0)
			deleteIndexOnCleanup(tc, indexName)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
		})

		It("purges a full document", func() {
			document := opensearchtest.Testdata.Resources.File
			tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), document)))
			tc.Require.IndicesCount([]string{indexName}, nil, 1)

			Expect(backend.Purge(document.ID, false)).To(Succeed())

			tc.Require.IndicesCount([]string{indexName}, nil, 0)
		})

		It("purges resource trees", func() {
			resourceFolder := opensearchtest.Testdata.Resources.Folder
			tc.Require.DocumentCreate(indexName, resourceFolder.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), resourceFolder)))

			resourceFile := opensearchtest.Testdata.Resources.File
			tc.Require.DocumentCreate(indexName, resourceFile.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), resourceFile)))

			tc.Require.IndicesCount([]string{indexName}, nil, 2)

			Expect(backend.Purge(resourceFolder.ID, false)).To(Succeed())

			tc.Require.IndicesCount([]string{indexName}, nil, 0)
		})

		It("purges resource trees and ignores undeleted resources", func() {
			resourceFolder := opensearchtest.Testdata.Resources.Folder
			tc.Require.DocumentCreate(indexName, resourceFolder.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), resourceFolder)))

			resourceFile := opensearchtest.Testdata.Resources.File
			tc.Require.DocumentCreate(indexName, resourceFile.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), resourceFile)))

			tc.Require.IndicesCount([]string{indexName}, nil, 2)

			Expect(backend.Delete(resourceFile.ID)).To(Succeed())
			tc.Require.IndicesRefresh([]string{indexName}, nil)
			Expect(backend.Purge(resourceFolder.ID, true)).To(Succeed())

			tc.Require.IndicesCount([]string{indexName}, nil, 1)
		})
	})

	Describe("DocCount", func() {
		const indexName = "opencloud-test-engine-doc-count"

		var (
			tc      *opensearchtest.TestClient
			backend *opensearch.Backend
		)

		BeforeEach(func() {
			tc = opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{indexName})
			tc.Require.IndicesCount([]string{indexName}, nil, 0)
			deleteIndexOnCleanup(tc, indexName)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
		})

		It("ignores deleted documents", func() {
			document := opensearchtest.Testdata.Resources.File
			tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), document)))
			tc.Require.IndicesCount([]string{indexName}, nil, 1)

			count, err := backend.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(uint64(1)))

			tc.Require.Update(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), map[string]any{
				"doc": map[string]any{
					"Deleted": true,
				},
			})))

			tc.Require.IndicesCount([]string{indexName}, nil, 1)

			count, err = backend.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(uint64(0)))
		})
	})

	// The following specs ensure that updates which affect a resource and its descendants
	// (Delete, Restore, Move) are scoped to the root (space) of the target resource. Two
	// resources living in different roots may share the exact same path, so matching by
	// path alone would incorrectly update the wrong resource.
	Describe("updateSelfAndDescendants root scope", func() {
		It("deletes only the resource in the target root", func() {
			const indexName = "opencloud-test-engine-root-scope-delete"

			target := opensearchtest.Testdata.Resources.File
			other := otherRoot(target)

			backend, tc := newBackend(indexName, target, other)
			deleteIndexOnCleanup(tc, indexName)

			Expect(backend.Delete(target.ID)).To(Succeed())

			Expect(resourceByID(tc, indexName, target.ID).Deleted).To(BeTrue(), "target resource should be marked as deleted")
			Expect(resourceByID(tc, indexName, other.ID).Deleted).To(BeFalse(), "resource in a different root must not be affected")
		})

		It("restores only the resource in the target root", func() {
			const indexName = "opencloud-test-engine-root-scope-restore"

			target := opensearchtest.Testdata.Resources.File
			target.Deleted = true
			other := otherRoot(target)

			backend, tc := newBackend(indexName, target, other)
			deleteIndexOnCleanup(tc, indexName)

			Expect(backend.Restore(target.ID)).To(Succeed())

			Expect(resourceByID(tc, indexName, target.ID).Deleted).To(BeFalse(), "target resource should be restored")
			Expect(resourceByID(tc, indexName, other.ID).Deleted).To(BeTrue(), "resource in a different root must not be affected")
		})

		It("moves only the resource in the target root", func() {
			const indexName = "opencloud-test-engine-root-scope-move"

			target := opensearchtest.Testdata.Resources.File
			other := otherRoot(target)

			backend, tc := newBackend(indexName, target, other)
			deleteIndexOnCleanup(tc, indexName)

			Expect(backend.Move(target.ID, target.ParentID, "./new/path/to/resource")).To(Succeed())

			Expect(resourceByID(tc, indexName, target.ID).Path).To(Equal("./new/path/to/resource"), "target resource should be moved")
			Expect(resourceByID(tc, indexName, other.ID).Path).To(Equal(other.Path), "resource in a different root must not be moved")
		})
	})
})
