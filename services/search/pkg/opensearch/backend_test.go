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
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/internal/opensearchtest"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
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

	Describe("Semantic search", func() {
		const indexName = "opencloud-test-engine-semantic"

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
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger(), opensearch.WithTextVectorizer(fakeVectorizer{
				byText: map[string][]float32{
					"cat": axisVector(0),
					"dog": axisVector(1),
				},
			}))
			Expect(err).ToNot(HaveOccurred())

			upsert := func(id, path, name string, vector []float32) {
				r := opensearchtest.Testdata.Resources.File
				r.ID = id
				r.Path = path
				r.Name = name
				r.ImageVector = vector
				Expect(backend.Upsert(id, r)).To(Succeed())
			}
			upsert("1$1!10", "./cat.jpg", "cat.jpg", axisVector(0))
			upsert("1$1!11", "./dog.jpg", "dog.jpg", axisVector(1))
			upsert("1$1!12", "./note.txt", "note.txt", nil)
			tc.Require.IndicesRefresh([]string{physical}, nil)
		})

		It("ranks the nearest image first for a purely semantic query", func() {
			resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: `semantic:"cat"`})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Matches).ToNot(BeEmpty())
			Expect(resp.Matches[0].Entity.Name).To(Equal("cat.jpg"))
			// the only honest total is the number of semantic hits
			Expect(resp.TotalMatches).To(Equal(int32(2)))
		})

		It("combines a semantic clause with a filter part", func() {
			resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: `Name:dog* AND semantic:"cat"`})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Matches).To(HaveLen(1))
			Expect(resp.Matches[0].Entity.Name).To(Equal("dog.jpg"))
			Expect(resp.TotalMatches).To(Equal(int32(1)))
		})

		It("keeps semantic inside a quoted value literal", func() {
			resp, err := backend.Search(context.Background(), &searchService.SearchIndexRequest{Query: `Name:"*semantic:cat*"`})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Matches).To(BeEmpty())
		})

		It("rejects semantic queries without a vectorizer", func() {
			bare, err := opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
			_, err = bare.Search(context.Background(), &searchService.SearchIndexRequest{Query: `semantic:"cat"`})
			Expect(err).To(MatchError(ContainSubstring("not configured")))
		})
	})
})

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
