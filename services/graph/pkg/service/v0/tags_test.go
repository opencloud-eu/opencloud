package svc_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
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

	// matchesWithTags returns one search match per argument, with the given tags
	matchesWithTags := func(tagsPerMatch ...[]string) []*searchmsg.Match {
		matches := make([]*searchmsg.Match, 0, len(tagsPerMatch))
		for _, tags := range tagsPerMatch {
			matches = append(matches, &searchmsg.Match{Entity: &searchmsg.Entity{Tags: tags}})
		}
		return matches
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

	It("asks the search for all tagged resources", func() {
		searchService.EXPECT().Search(mock.Anything, mock.MatchedBy(func(req *searchsvc.SearchRequest) bool {
			return req.GetQuery() == "Tags:*" && req.GetPageSize() == -1
		})).Return(&searchsvc.SearchResponse{}, nil)

		Expect(getTags()).To(BeEmpty())
	})

	It("renders an empty list when no tags are in use", func() {
		searchService.EXPECT().Search(mock.Anything, mock.Anything).Return(&searchsvc.SearchResponse{}, nil)

		svc.GetTags(tape, httptest.NewRequest(http.MethodGet, "/graph/v1.0/extensions/org.libregraph/tags", nil))
		Expect(tape.Code).To(Equal(http.StatusOK))
		Expect(tape.Body.String()).To(MatchJSON(`{"value": []}`))
	})

	It("lists every tag once", func() {
		searchService.EXPECT().Search(mock.Anything, mock.Anything).Return(&searchsvc.SearchResponse{
			Matches: matchesWithTags([]string{"work", "draft"}, []string{"draft", ""}, nil, []string{"work"}),
		}, nil)

		Expect(getTags()).To(Equal([]string{"draft", "work"}))
	})

	It("sorts the tags case-insensitively, with a fixed order for tags differing only in case", func() {
		searchService.EXPECT().Search(mock.Anything, mock.Anything).Return(&searchsvc.SearchResponse{
			Matches: matchesWithTags([]string{"cherry", "apple"}, []string{"Banana", "Apple"}),
		}, nil)

		Expect(getTags()).To(Equal([]string{"Apple", "apple", "Banana", "cherry"}))
	})

	It("returns the same list for the same tags, whatever the order of the search results", func() {
		first := matchesWithTags([]string{"c", "a"}, []string{"b"})
		reversed := matchesWithTags([]string{"b"}, []string{"a", "c"})

		searchService.EXPECT().Search(mock.Anything, mock.Anything).Return(&searchsvc.SearchResponse{Matches: first}, nil).Once()
		firstResult := getTags()

		tape = httptest.NewRecorder()
		searchService.EXPECT().Search(mock.Anything, mock.Anything).Return(&searchsvc.SearchResponse{Matches: reversed}, nil).Once()
		Expect(getTags()).To(Equal(firstResult))
	})

	It("returns only the first tags in sort order when more tags are in use than it lists", func() {
		limit := service.MaxListedTagsForTest
		// one more tag than listed, spread over several matches, in descending order
		all := make([]string, 0, limit+1)
		for i := limit; i >= 0; i-- {
			all = append(all, fmt.Sprintf("tag%03d", i))
		}
		searchService.EXPECT().Search(mock.Anything, mock.Anything).Return(&searchsvc.SearchResponse{
			Matches: matchesWithTags(all[:limit/2], all[limit/2:]),
		}, nil)

		listed := getTags()
		Expect(listed).To(HaveLen(limit))
		Expect(listed[0]).To(Equal("tag000"))
		Expect(listed[limit-1]).To(Equal(fmt.Sprintf("tag%03d", limit-1)))
		Expect(listed).ToNot(ContainElement(fmt.Sprintf("tag%03d", limit)))
	})

	It("fails when the search fails", func() {
		searchService.EXPECT().Search(mock.Anything, mock.Anything).Return(nil, errors.New("search is down"))

		svc.GetTags(tape, httptest.NewRequest(http.MethodGet, "/graph/v1.0/extensions/org.libregraph/tags", nil))
		Expect(tape.Code).To(Equal(http.StatusInternalServerError))
	})
})
