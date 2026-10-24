package opensearch_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/internal/opensearchtest"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
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
