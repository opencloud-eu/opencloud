package svc_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	searchmocks "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	"github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("GetTags", func() {
	var (
		svc           service.Service
		searchService *searchmocks.SearchProviderService
		tape          *httptest.ResponseRecorder
	)

	// tagBuckets answers like the search does: one bucket per tag
	tagBuckets := func(tags ...string) *searchsvc.SearchResponse {
		buckets := make([]*searchsvc.Bucket, 0, len(tags))
		for _, tag := range tags {
			buckets = append(buckets, &searchsvc.Bucket{Key: tag, Count: 1})
		}
		return &searchsvc.SearchResponse{Aggregations: []*searchsvc.AggregationResult{{Field: "Tags", Buckets: buckets}}}
	}

	getTags := func() []string {
		svc.GetTags(tape, httptest.NewRequest(http.MethodGet, "/graph/v1.0/extensions/org.libregraph/tags", nil))
		Expect(tape.Code).To(Equal(http.StatusOK))

		var collection libregraph.CollectionOfTags
		Expect(json.Unmarshal(tape.Body.Bytes(), &collection)).To(Succeed())
		return collection.GetValue()
	}

	BeforeEach(func() {
		logger := log.NewLogger()
		metrics, _ := metrics.New(prometheus.NewRegistry(), &logger, func([]string) (string, string) { return "", "" })
		searchService = searchmocks.NewSearchProviderService(GinkgoT())
		tape = httptest.NewRecorder()

		cfg := defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = "" // skip the startup checks, we don't use LDAP at all in this tests
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		var err error
		svc, err = service.NewService(
			service.Config(cfg),
			service.Metrics(metrics),
			service.WithIdentityBackend(&identitymocks.Backend{}),
			service.WithSearchService(searchService),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	It("asks the search for the first tags in byte order, without matches", func() {
		searchService.EXPECT().Search(mock.Anything, mock.MatchedBy(func(req *searchsvc.SearchRequest) bool {
			aggs := req.GetAggregations()
			return req.GetQuery() == "Tags:*" && req.PageSize != nil && req.GetPageSize() == 0 &&
				len(aggs) == 1 && aggs[0].GetField() == "Tags" &&
				aggs[0].GetSize() == int32(service.MaxListedTagsForTest) &&
				aggs[0].GetBucketDefinition().GetSortBy() == searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_STRING &&
				!aggs[0].GetBucketDefinition().GetIsDescending()
		})).Return(&searchsvc.SearchResponse{}, nil)

		Expect(getTags()).To(BeEmpty())
	})

	It("renders an empty list when no tags are in use", func() {
		searchService.EXPECT().Search(mock.Anything, mock.Anything).Return(tagBuckets(), nil)

		svc.GetTags(tape, httptest.NewRequest(http.MethodGet, "/graph/v1.0/extensions/org.libregraph/tags", nil))
		Expect(tape.Code).To(Equal(http.StatusOK))
		Expect(tape.Body.String()).To(MatchJSON(`{"value": []}`))
	})

	It("lists the tags as the search orders them", func() {
		searchService.EXPECT().Search(mock.Anything, mock.Anything).Return(tagBuckets("Apple", "Banana", "apple", "cherry"), nil)

		Expect(getTags()).To(Equal([]string{"Apple", "Banana", "apple", "cherry"}))
	})

	It("fails when the search fails", func() {
		searchService.EXPECT().Search(mock.Anything, mock.Anything).Return(nil, errors.New("search is down"))

		svc.GetTags(tape, httptest.NewRequest(http.MethodGet, "/graph/v1.0/extensions/org.libregraph/tags", nil))
		Expect(tape.Code).To(Equal(http.StatusInternalServerError))
	})
})
