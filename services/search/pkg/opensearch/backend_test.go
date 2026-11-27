package opensearch_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	revactx "github.com/opencloud-eu/reva/v3/pkg/ctx"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/internal/opensearchtest"
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

	Describe("Get", func() {
		const indexName = "opencloud-test-engine-lookup"

		var backend *opensearch.Backend

		BeforeEach(func() {
			physical := opensearch.VersionedIndexName(indexName)

			tc := opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{physical})
			deleteIndexOnCleanup(tc, physical)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns the indexed resource with its exact mtime", func() {
			mtime := time.Unix(1700000000, 123456789).UTC()
			document := opensearchtest.Testdata.Resources.File
			document.Mtime = &mtime
			Expect(backend.Upsert(document.ID, document)).To(Succeed())

			r, err := backend.Get(context.Background(), document.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.ID).To(Equal(document.ID))
			Expect(r.Mtime).ToNot(BeNil())
			Expect(r.Mtime.Equal(mtime)).To(BeTrue(), "got %s", r.Mtime)
		})

		It("returns a trashed resource", func() {
			document := opensearchtest.Testdata.Resources.File
			document.Deleted = true
			Expect(backend.Upsert(document.ID, document)).To(Succeed())

			r, err := backend.Get(context.Background(), document.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Deleted).To(BeTrue())
		})

		It("fails with ErrResourceNotFound for an unknown id", func() {
			_, err := backend.Get(context.Background(), "1$2!unknown")
			Expect(err).To(MatchError(search.ErrResourceNotFound))
		})
	})

	Describe("Update", func() {
		const indexName = "opencloud-test-engine-update"

		var backend *opensearch.Backend

		BeforeEach(func() {
			physical := opensearch.VersionedIndexName(indexName)

			tc := opensearchtest.NewDefaultTestClient(GinkgoTB(), defaultConfig.Engine.OpenSearch.Client)
			tc.Require.IndicesReset([]string{physical})
			deleteIndexOnCleanup(tc, physical)

			var err error
			backend, err = opensearch.NewBackend(context.Background(), indexName, tc.Client(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())
		})

		tagHits := func(tag string) int {
			res, err := backend.Search(context.Background(), &search.IndexQuery{Query: `tag:("` + tag + `")`})
			Expect(err).ToNot(HaveOccurred())
			return len(res.Matches)
		}

		It("rewrites tags and favorites and keeps the extracted fields", func() {
			document := opensearchtest.Testdata.Resources.File
			document.Content = "quarterly report"
			document.Tags = []string{"alpha"}
			Expect(backend.Upsert(document.ID, document)).To(Succeed())

			Expect(backend.Update(context.Background(), document.ID, search.MetadataUpdate{
				Tags:      []string{"Beta"},
				Favorites: []string{"user-1"},
			})).To(Succeed())

			r, err := backend.Get(context.Background(), document.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Tags).To(Equal([]string{"Beta"}))
			Expect(r.Favorites).To(Equal([]string{"user-1"}))
			Expect(r.Content).To(Equal("quarterly report"))

			Expect(tagHits("beta")).To(Equal(1), "the lowercase sibling follows the new tag")
			Expect(tagHits("alpha")).To(BeZero())
		})

		It("clears the tags and their lowercase sibling", func() {
			document := opensearchtest.Testdata.Resources.File
			document.Tags = []string{"alpha"}
			Expect(backend.Upsert(document.ID, document)).To(Succeed())

			Expect(backend.Update(context.Background(), document.ID, search.MetadataUpdate{})).To(Succeed())

			r, err := backend.Get(context.Background(), document.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Tags).To(BeEmpty())
			Expect(tagHits("ALPHA")).To(BeZero())
		})

		It("fails with ErrResourceNotFound for an unknown id", func() {
			err := backend.Update(context.Background(), "1$2!unknown", search.MetadataUpdate{Tags: []string{"alpha"}})
			Expect(err).To(MatchError(search.ErrResourceNotFound))
		})
	})

	Describe("Search", func() {
		const indexName = "opencloud-test-engine-search-preference"

		It("sends every search of a user to the same shard copies", func() {
			physical := opensearch.VersionedIndexName(indexName)
			cfg := defaultConfig.Engine.OpenSearch.Client
			tc := opensearchtest.NewDefaultTestClient(GinkgoTB(), cfg)
			tc.Require.IndicesReset([]string{physical})
			deleteIndexOnCleanup(tc, physical)

			searches := &searchRecorder{}
			client, err := opensearchgoAPI.NewClient(opensearchgoAPI.Config{Client: opensearchgo.Config{
				Addresses: cfg.Addresses, Username: cfg.Username, Password: cfg.Password, Transport: searches,
			}})
			Expect(err).ToNot(HaveOccurred())
			backend, err := opensearch.NewBackend(context.Background(), indexName, client, log.NopLogger())
			Expect(err).ToNot(HaveOccurred())

			ctx := revactx.ContextSetUser(context.Background(), &userv1beta1.User{Id: &userv1beta1.UserId{OpaqueId: "einstein"}})
			_, err = backend.Search(ctx, &search.IndexQuery{Query: "name:a"})
			Expect(err).ToNot(HaveOccurred())
			_, err = backend.Search(context.Background(), &search.IndexQuery{Query: "name:a"})
			Expect(err).ToNot(HaveOccurred())

			Expect(searches.preferences).To(Equal([]string{"user-einstein", ""}))
		})
	})
})

type searchRecorder struct {
	preferences []string
}

func (r *searchRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasSuffix(req.URL.Path, "/_search") {
		r.preferences = append(r.preferences, req.URL.Query().Get("preference"))
	}
	return http.DefaultTransport.RoundTrip(req)
}
