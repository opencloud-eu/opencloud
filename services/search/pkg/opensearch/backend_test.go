package opensearch_test

import (
	"context"
	"testing"
	"time"

	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/internal/opensearchtest"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
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
			tc      *opensearchtest.TestClient
			backend *opensearch.Backend
		)

		BeforeEach(func() {
			physical := opensearch.VersionedIndexName(indexName)

			tc = opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{physical})
			deleteIndexOnCleanup(tc, physical)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
		})

		It("answers the indexer's up-to-date check", func() {
			document := opensearchtest.Testdata.Resources.File
			Expect(backend.Upsert(document.ID, document)).To(Succeed())

			res, err := backend.Search(context.Background(), &search.IndexQuery{Query: search.UpToDateQuery(document.ID, *document.Mtime)})
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Matches).To(HaveLen(1))

			res, err = backend.Search(context.Background(), &search.IndexQuery{Query: search.UpToDateQuery(document.ID, document.Mtime.Add(time.Second))})
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Matches).To(BeEmpty())
		})

		It("applies the default query template", func() {
			keep := func(id, path, name string) search.Resource {
				return search.Resource{ID: id, RootID: "1$2!2", Path: path, Document: content.Document{Name: name, Tags: []string{"keep"}}}
			}
			for _, r := range []search.Resource{
				keep("1$2!2", ".", "root"),
				keep("1$2!5", "./.space/readme.md", "readme.md"),
				keep("1$2!6", "./child.pdf", "child.pdf"),
			} {
				Expect(backend.Upsert(r.ID, r)).To(Succeed())
			}
			template, err := search.ParseQueryTemplate(defaults.DefaultConfig().QueryTemplate)
			Expect(err).ToNot(HaveOccurred())
			query, err := search.ParseQuery(`tag:keep`)
			Expect(err).ToNot(HaveOccurred())
			space := &sprovider.StorageSpace{SpaceType: "personal", Root: &sprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"}}

			res, err := backend.Search(context.Background(), &search.IndexQuery{Query: template.Expand(query, space)})
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Matches).To(HaveLen(1))
			Expect(res.Matches[0].Entity.Name).To(Equal("child.pdf"))
		})
	})

	Describe("Upsert", func() {
		const indexName = "opencloud-test-engine-upsert"

		var (
			tc      *opensearchtest.TestClient
			backend *opensearch.Backend
		)

		BeforeEach(func() {
			// the backend versions the physical index by schema generation
			physical := opensearch.VersionedIndexName(indexName)

			tc = opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{physical})
			deleteIndexOnCleanup(tc, physical)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
		})

		It("upserts a full document", func() {
			document := opensearchtest.Testdata.Resources.File
			Expect(backend.Upsert(document.ID, document)).To(Succeed())

			tc.Require.IndicesCount([]string{opensearch.VersionedIndexName(indexName)}, nil, 1)
		})

		It("upserts a document without an mtime", func() {
			// content.Extract leaves Mtime nil when the resource info carries none
			document := opensearchtest.Testdata.Resources.File
			document.ID = "1$1!4"
			document.Mtime = nil
			Expect(backend.Upsert(document.ID, document)).To(Succeed())

			tc.Require.IndicesCount([]string{opensearch.VersionedIndexName(indexName)}, nil, 1)
		})
	})

})
