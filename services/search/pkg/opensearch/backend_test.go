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

	backend, err := opensearch.NewBackend(indexName, tc.Client())
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

			backend, err := opensearch.NewBackend("test-engine-new-engine", client)
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
			backend, err = opensearch.NewBackend(indexName, tc.Client())
			Expect(err).ToNot(HaveOccurred())

			document = opensearchtest.Testdata.Resources.File
			tc.Require.DocumentCreate(indexName, document.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), document)))
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

			tc.Require.DocumentCreate(indexName, deletedDocument.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), deletedDocument)))
			tc.Require.IndicesCount([]string{indexName}, nil, 2)

			resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{
				Query: fmt.Sprintf(`"%s"`, document.Name),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Matches).To(HaveLen(1))
			Expect(resp.TotalMatches).To(Equal(int32(1)))
			Expect(fmt.Sprintf("%s$%s!%s", resp.Matches[0].Entity.Id.StorageId, resp.Matches[0].Entity.Id.SpaceId, resp.Matches[0].Entity.Id.OpaqueId)).To(Equal(document.ID))
		})
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
			backend, err = opensearch.NewBackend(indexName, tc.Client())
			Expect(err).ToNot(HaveOccurred())
		})

		It("upserts a full document", func() {
			document := opensearchtest.Testdata.Resources.File
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
			backend, err = opensearch.NewBackend(indexName, tc.Client())
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
			backend, err = opensearch.NewBackend(indexName, tc.Client())
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
			backend, err = opensearch.NewBackend(indexName, tc.Client())
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
			backend, err = opensearch.NewBackend(indexName, tc.Client())
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

	Describe("PurgeSpace", func() {
		const indexName = "opencloud-test-engine-purge-space"

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
			backend, err = opensearch.NewBackend(indexName, tc.Client())
			Expect(err).ToNot(HaveOccurred())
		})

		It("takes every record of that space out of the index", func() {
			gone := opensearchtest.Testdata.Resources.File
			tc.Require.DocumentCreate(indexName, gone.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), gone)))

			stays := opensearchtest.Testdata.Resources.File
			stays.ID = "1$2!3"
			stays.RootID = "1$2!2"
			tc.Require.DocumentCreate(indexName, stays.ID, strings.NewReader(opensearchtest.JSONMustMarshal(GinkgoTB(), stays)))

			tc.Require.IndicesCount([]string{indexName}, nil, 2)

			Expect(backend.PurgeSpace(gone.RootID)).To(Succeed())

			tc.Require.IndicesRefresh([]string{indexName}, nil)
			left := opensearchtest.SearchHitsMustBeConverted[search.Resource](
				GinkgoTB(),
				tc.Require.Search(indexName, strings.NewReader(`{"query":{"match_all":{}}}`)).Hits,
			)
			Expect(left).To(HaveLen(1), "only the records of that space are gone")
			Expect(left[0].ID).To(Equal(stays.ID))
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
			backend, err = opensearch.NewBackend(indexName, tc.Client())
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
