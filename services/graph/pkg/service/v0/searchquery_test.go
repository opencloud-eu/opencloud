package svc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	// ginkgo qualified: the svc package declares Context (option.go), which
	// would collide with a dot-import.
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"go-micro.dev/v4/client"
	merrors "go-micro.dev/v4/errors"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
)

type stubSearchService struct {
	search func(*searchsvc.SearchRequest) (*searchsvc.SearchResponse, error)
}

func (s stubSearchService) Search(_ context.Context, req *searchsvc.SearchRequest, _ ...client.CallOption) (*searchsvc.SearchResponse, error) {
	return s.search(req)
}

func (s stubSearchService) IndexSpace(_ context.Context, _ *searchsvc.IndexSpaceRequest, _ ...client.CallOption) (searchsvc.SearchProvider_IndexSpaceService, error) {
	return nil, nil
}

func graphWithSearch(stub stubSearchService) Graph {
	logger := log.NewLogger()
	return Graph{
		BaseGraphService: BaseGraphService{logger: &logger},
		searchService:    stub,
	}
}

// graphWithSearchAnswer answers every search with resp and hands out the last
// request it saw.
func graphWithSearchAnswer(resp *searchsvc.SearchResponse) (Graph, func() *searchsvc.SearchRequest) {
	var captured *searchsvc.SearchRequest
	g := graphWithSearch(stubSearchService{
		search: func(req *searchsvc.SearchRequest) (*searchsvc.SearchResponse, error) {
			captured = req
			return resp, nil
		},
	})
	return g, func() *searchsvc.SearchRequest { return captured }
}

func graphWithAggregations(aggregations ...*searchsvc.AggregationResult) (Graph, func() *searchsvc.SearchRequest) {
	return graphWithSearchAnswer(&searchsvc.SearchResponse{Aggregations: aggregations})
}

func graphWithoutSearch() Graph {
	return graphWithSearch(stubSearchService{
		search: func(*searchsvc.SearchRequest) (*searchsvc.SearchResponse, error) {
			ginkgo.Fail("search service must not be called when validation fails")
			return nil, nil
		},
	})
}

func postSearchQuery(g Graph, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/search/query", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	g.SearchQuery(rr, req)
	return rr
}

func searchQueryBody(fragment string) string {
	return `{"requests": [{"entityTypes": ["driveItem"], "query": {"queryString": "mediatype:audio"}, "size": 0, ` + fragment + `}]}`
}

func oneMatch(entity *searchmsg.Entity) *searchsvc.SearchResponse {
	return &searchsvc.SearchResponse{TotalMatches: 1, Matches: []*searchmsg.Match{{Entity: entity}}}
}

type searchAggregationJSON struct {
	Field  string `json:"field"`
	Metric *struct {
		Kind  string   `json:"kind"`
		Value *float64 `json:"value"`
	} `json:"@libre.graph.metric"`
	Buckets []struct {
		Key             string                  `json:"key"`
		Count           int64                   `json:"count"`
		Token           string                  `json:"aggregationFilterToken"`
		SubAggregations []searchAggregationJSON `json:"@libre.graph.subAggregations"`
	} `json:"buckets"`
}

type searchHitsContainerJSON struct {
	Hits []struct {
		Rank     int32 `json:"rank"`
		Resource struct {
			RemoteItem *struct {
				Id   string `json:"id"`
				Name string `json:"name"`
				Path string `json:"path"`
			} `json:"remoteItem"`
			ParentReference struct {
				Id   string  `json:"id"`
				Path string  `json:"path"`
				Name *string `json:"name"`
			} `json:"parentReference"`
		} `json:"resource"`
	} `json:"hits"`
	Total                int64                   `json:"total"`
	MoreResultsAvailable bool                    `json:"moreResultsAvailable"`
	Aggregations         []searchAggregationJSON `json:"aggregations"`
}

// hitsContainers decodes the one hits container of every request.
func hitsContainers(rr *httptest.ResponseRecorder) []searchHitsContainerJSON {
	var decoded struct {
		Value []struct {
			HitsContainers []searchHitsContainerJSON `json:"hitsContainers"`
		} `json:"value"`
	}
	Expect(json.Unmarshal(rr.Body.Bytes(), &decoded)).To(Succeed())
	out := make([]searchHitsContainerJSON, 0, len(decoded.Value))
	for _, v := range decoded.Value {
		Expect(v.HitsContainers).To(HaveLen(1))
		out = append(out, v.HitsContainers[0])
	}
	return out
}

func hitsContainer(rr *httptest.ResponseRecorder) searchHitsContainerJSON {
	containers := hitsContainers(rr)
	Expect(containers).To(HaveLen(1))
	return containers[0]
}

var _ = ginkgo.Describe("SearchQuery", func() {
	ginkgo.DescribeTable("rejects a malformed request with 400 before any search runs",
		func(body, message string) {
			rr := postSearchQuery(graphWithoutSearch(), body)
			Expect(rr.Code).To(Equal(http.StatusBadRequest), rr.Body.String())
			Expect(rr.Body.String()).To(ContainSubstring(message))
		},
		ginkgo.Entry("entityTypes absent, the generated model requires it", `{"requests": [{"query": {"queryString": "fox"}}]}`, "invalid body"),
		ginkgo.Entry("entityTypes empty", `{"requests": [{"entityTypes": [], "query": {"queryString": "fox"}}]}`, "entityTypes"),
		ginkgo.Entry("another entity", `{"requests": [{"entityTypes": ["message"], "query": {"queryString": "fox"}}]}`, "message"),
		ginkgo.Entry("a page beyond the first 10000 matches, whatever the engine",
			`{"requests": [{"entityTypes": ["driveItem"], "query": {"queryString": "notes"}, "from": 10000, "size": 25}]}`,
			`"message":"from and size reach beyond`),
		ginkgo.Entry("a bad second request, every request is validated before the first one runs", `{"requests": [
			{"entityTypes": ["driveItem"], "query": {"queryString": "notes"}},
			{"entityTypes": ["driveItem"], "query": {"queryString": "notes"}, "aggregations": [{"field": "audio.nonexistent"}]}
		]}`, "audio.nonexistent"),
	)

	ginkgo.It("answers one hits container per request, in request order", func() {
		g := graphWithSearch(stubSearchService{
			search: func(req *searchsvc.SearchRequest) (*searchsvc.SearchResponse, error) {
				return &searchsvc.SearchResponse{TotalMatches: int32(len(req.GetQuery()))}, nil
			},
		})
		rr := postSearchQuery(g, `{"requests": [
			{"entityTypes": ["driveItem"], "query": {"queryString": "a"}, "size": 0},
			{"entityTypes": ["driveItem"], "query": {"queryString": "abc"}, "size": 0}
		]}`)
		Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())

		containers := hitsContainers(rr)
		Expect(containers).To(HaveLen(2))
		Expect(containers[0].Total).To(Equal(int64(1)))
		Expect(containers[1].Total).To(Equal(int64(3)))
	})

	ginkgo.DescribeTable("describes a hit from a shared space as a remote item",
		func(entity *searchmsg.Entity, wantID, wantPath, wantName string) {
			g, _ := graphWithSearchAnswer(oneMatch(entity))
			rr := postSearchQuery(g, searchQueryBody(`"from": 0`))
			Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())

			remote := hitsContainer(rr).Hits[0].Resource.RemoteItem
			if wantID == "" {
				Expect(remote).To(BeNil())
				return
			}
			Expect(remote.Id).To(Equal(wantID))
			Expect(remote.Path).To(Equal(wantPath))
			Expect(remote.Name).To(Equal(wantName), "the mountpoint name the caller sees")
		},
		ginkgo.Entry("a hit from a shared space", &searchmsg.Entity{
			Id:            &searchmsg.ResourceID{StorageId: "1", SpaceId: "2", OpaqueId: "3"},
			Name:          "contract.pdf",
			ShareRootName: "/Project X",
			RemoteItemId:  &searchmsg.ResourceID{StorageId: "4", SpaceId: "5", OpaqueId: "6"},
		}, "4$5!6", "/Project X", "Project X"),
		ginkgo.Entry("a hit from the caller's own space has none", &searchmsg.Entity{
			Id:   &searchmsg.ResourceID{StorageId: "1", SpaceId: "2", OpaqueId: "3"},
			Name: "notes.txt",
		}, "", "", ""),
	)

	ginkgo.DescribeTable("describes the parent of a hit by its path from the space root",
		func(refPath, wantPath string, wantName *string) {
			g, _ := graphWithSearchAnswer(oneMatch(&searchmsg.Entity{
				Id:       &searchmsg.ResourceID{StorageId: "1", SpaceId: "2", OpaqueId: "3"},
				ParentId: &searchmsg.ResourceID{StorageId: "1", SpaceId: "2", OpaqueId: "4"},
				Ref:      &searchmsg.Reference{Path: refPath},
				Name:     "notes.txt",
			}))
			rr := postSearchQuery(g, searchQueryBody(`"from": 0`))
			Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())

			parent := hitsContainer(rr).Hits[0].Resource.ParentReference
			Expect(parent.Id).To(Equal("1$2!4"))
			Expect(parent.Path).To(Equal(wantPath))
			if wantName == nil {
				Expect(parent.Name).To(BeNil())
			} else {
				Expect(parent.Name).To(HaveValue(Equal(*wantName)))
			}
		},
		ginkgo.Entry("a hit in a folder", "./projects/2026/notes.txt", "/projects/2026", libregraph.PtrString("2026")),
		ginkgo.Entry("a hit in the space root, the root has no name", "./notes.txt", "/", nil),
	)

	ginkgo.DescribeTable("validatePagination enforces the spec's from/size bounds",
		func(from, size *int32, wantErr bool) {
			err := validatePagination(from, size)
			if wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
		ginkgo.Entry("absent values are valid", nil, nil, false),
		ginkgo.Entry("zero size is valid", libregraph.PtrInt32(5), libregraph.PtrInt32(0), false),
		ginkgo.Entry("maximum size is valid", libregraph.PtrInt32(0), libregraph.PtrInt32(500), false),
		ginkgo.Entry("negative from is rejected", libregraph.PtrInt32(-10), libregraph.PtrInt32(5), true),
		ginkgo.Entry("negative size is rejected", libregraph.PtrInt32(10), libregraph.PtrInt32(-1), true),
		ginkgo.Entry("oversized size is rejected", libregraph.PtrInt32(0), libregraph.PtrInt32(1000), true),
		ginkgo.Entry("the last page within the first 10000 matches is valid", libregraph.PtrInt32(9975), libregraph.PtrInt32(25), false),
		ginkgo.Entry("a page reaching beyond the first 10000 matches is rejected", libregraph.PtrInt32(9990), libregraph.PtrInt32(25), true),
		ginkgo.Entry("the default size counts", libregraph.PtrInt32(10000), nil, true),
		ginkgo.Entry("a from that overflows with the size is rejected", libregraph.PtrInt32(2147483647), libregraph.PtrInt32(500), true),
	)

	ginkgo.It("pushes from/size to the search service and offsets the ranks", func() {
		g, captured := graphWithSearchAnswer(&searchsvc.SearchResponse{
			TotalMatches: 42,
			Matches: []*searchmsg.Match{
				{Entity: &searchmsg.Entity{Id: &searchmsg.ResourceID{StorageId: "1", SpaceId: "2", OpaqueId: "a"}, Name: "a.txt"}},
				{Entity: &searchmsg.Entity{Id: &searchmsg.ResourceID{StorageId: "1", SpaceId: "2", OpaqueId: "b"}, Name: "b.txt"}},
			},
		})
		rr := postSearchQuery(g, `{"requests": [{"entityTypes": ["driveItem"], "query": {"queryString": "notes"}, "from": 10, "size": 2}]}`)
		Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())
		Expect(captured().GetFrom()).To(Equal(int32(10)))
		Expect(captured().GetPageSize()).To(Equal(int32(2)))

		hc := hitsContainer(rr)
		Expect(hc.Total).To(Equal(int64(42)))
		Expect(hc.MoreResultsAvailable).To(BeTrue())
		Expect(hc.Hits).To(HaveLen(2))
		Expect(hc.Hits[0].Rank).To(Equal(int32(11)))
		Expect(hc.Hits[1].Rank).To(Equal(int32(12)))
	})

	ginkgo.DescribeTable("promises more results only within reach of from and size",
		func(from, size, total int, want bool) {
			g, _ := graphWithSearchAnswer(&searchsvc.SearchResponse{TotalMatches: int32(total)})
			rr := postSearchQuery(g, fmt.Sprintf(`{"requests": [{"entityTypes": ["driveItem"], "query": {"queryString": "notes"}, "from": %d, "size": %d}]}`, from, size))
			Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())
			Expect(hitsContainer(rr).MoreResultsAvailable).To(Equal(want))
		},
		ginkgo.Entry("more matches than the page", 0, 25, 42, true),
		ginkgo.Entry("the last page", 25, 25, 42, false),
		ginkgo.Entry("a next page within the first 10000 matches", 9950, 25, 20000, true),
		ginkgo.Entry("a next page the search would refuse", 9975, 25, 20000, false),
	)

	ginkgo.It("sends an explicit zero page size for facet-only requests", func() {
		g, captured := graphWithSearchAnswer(&searchsvc.SearchResponse{TotalMatches: 7})
		rr := postSearchQuery(g, searchQueryBody(`"from": 0`))
		Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())
		Expect(captured().PageSize).To(HaveValue(Equal(int32(0))))

		hc := hitsContainer(rr)
		Expect(hc.Total).To(Equal(int64(7)))
		Expect(hc.Hits).To(BeEmpty())
	})

	// the rules of the index are pinned in the aggregation package; one entry
	// per kind proves the handler asks them before the search service
	ginkgo.DescribeTable("rejects an aggregation the spec or the index does not allow with 400",
		func(fragment string) {
			rr := postSearchQuery(graphWithoutSearch(), searchQueryBody(fragment))
			Expect(rr.Code).To(Equal(http.StatusBadRequest), rr.Body.String())
			Expect(rr.Body.String()).To(ContainSubstring("invalidRequest"))
		},
		ginkgo.Entry("an empty field", `"aggregations": [{"field": ""}]`),
		ginkgo.Entry("a bucket and a metric definition at once",
			`"aggregations": [{"field": "audio.year", "bucketDefinition": {"sortBy": "count", "ranges": [{"from": "1980"}]}, "@libre.graph.metricDefinition": {"kind": "sum"}}]`),
		ginkgo.Entry("an unknown metric kind", `"aggregations": [{"field": "audio.year", "@libre.graph.metricDefinition": {"kind": "median"}}]`),
		ginkgo.Entry("an unknown sortBy", `"aggregations": [{"field": "audio.artist", "bucketDefinition": {"sortBy": "relevance"}}]`),
		ginkgo.Entry("a size below one", `"aggregations": [{"field": "audio.artist", "size": 0}]`),
		ginkgo.Entry("a negative minimumCount", `"aggregations": [{"field": "audio.artist", "bucketDefinition": {"sortBy": "count", "minimumCount": -1}}]`),
		ginkgo.Entry("an unknown metric kind two levels down",
			`"aggregations": [{"field": "audio.artist", "@libre.graph.subAggregations": [{"field": "audio.album", "@libre.graph.subAggregations": [{"field": "audio.year", "@libre.graph.metricDefinition": {"kind": "median"}}]}]}]`),
		ginkgo.Entry("a field the index does not know", `"aggregations": [{"field": "audio.nonexistent"}]`),
		ginkgo.Entry("a range bound that is no number", `"aggregations": [{"field": "audio.year", "bucketDefinition": {"sortBy": "count", "ranges": [{"from": "1970", "to": "198o"}]}}]`),
		// the token grammar is pinned in the filtertoken package
		ginkgo.Entry("an aggregation filter that is no server-issued token", `"aggregationFilters": ["audio.artist:\"not a token\""]`),
		ginkgo.Entry("an aggregation filter on an internal field", `"aggregationFilters": ["favorites:\"ǂǂ5361786f6e\""]`),
	)

	ginkgo.It("translates the bucket definition for the search service", func() {
		g, captured := graphWithAggregations()
		rr := postSearchQuery(g, searchQueryBody(`"aggregations": [
			{"field": "audio.artist", "size": 5, "bucketDefinition": {"sortBy": "keyAsString", "isDescending": true, "minimumCount": 2}},
			{"field": "audio.year", "bucketDefinition": {"sortBy": "keyAsNumber", "ranges": [{"to": "1980"}, {"from": "1980", "to": "1990"}]}},
			{"field": "audio.genre", "bucketDefinition": {"sortBy": "count"}}
		]`))
		Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())

		Expect(captured().GetAggregations()).To(BeComparableTo([]*searchsvc.AggregationOption{
			{Field: "audio.artist", Size: 5, BucketDefinition: &searchsvc.BucketDefinition{SortBy: searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_STRING, IsDescending: true, MinimumCount: 2}},
			{Field: "audio.year", BucketDefinition: &searchsvc.BucketDefinition{SortBy: searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_NUMBER, Ranges: []*searchsvc.BucketRange{{To: "1980"}, {From: "1980", To: "1990"}}}},
			{Field: "audio.genre", BucketDefinition: &searchsvc.BucketDefinition{SortBy: searchsvc.BucketSortBy_BUCKET_SORT_BY_COUNT}},
		}, protocmp.Transform()))
	})

	ginkgo.DescribeTable("resolves the driveItem spelling of a field to the index field and answers in the spelling of the request",
		func(requested, indexed string) {
			g, captured := graphWithAggregations(&searchsvc.AggregationResult{
				Field:   indexed,
				Buckets: []*searchsvc.Bucket{{Key: "audio/mpeg", Count: 3}},
			})
			rr := postSearchQuery(g, searchQueryBody(fmt.Sprintf(`"aggregations": [{"field": %q}], "aggregationFilters": [%q]`, requested, requested+`:"ǂǂ617564696f2f6d706567"`)))
			Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())

			Expect(captured().GetAggregations()[0].GetField()).To(Equal(indexed))
			Expect(captured().GetAggregationFilters()[0].GetField()).To(Equal(indexed))
			Expect(captured().GetAggregationFilters()[0].GetTerms()).To(Equal([]string{"audio/mpeg"}))
			Expect(hitsContainer(rr).Aggregations[0].Field).To(Equal(requested))
		},
		ginkgo.Entry("mimeType", "mimeType", "MimeType"),
		ginkgo.Entry("name", "name", "Name"),
		ginkgo.Entry("tags", "@libre.graph.tags", "Tags"),
		ginkgo.Entry("a facet property", "audio.artist", "audio.artist"),
	)

	ginkgo.DescribeTable("rejects a field not spelled like the driveItem property with 400, naming it as the request did",
		func(field string) {
			rr := postSearchQuery(graphWithoutSearch(), searchQueryBody(fmt.Sprintf(`"aggregations": [{"field": %q}]`, field)))
			Expect(rr.Code).To(Equal(http.StatusBadRequest), rr.Body.String())
			Expect(rr.Body.String()).To(ContainSubstring(field))
			Expect(rr.Body.String()).ToNot(ContainSubstring("Mtime"), "no index names")
		},
		ginkgo.Entry("the index spelling of a property", "MimeType"),
		ginkgo.Entry("another case", "MIMETYPE"),
		ginkgo.Entry("the index spelling of the modification time", "mtime"),
		ginkgo.Entry("a KQL alias", "tag"),
		ginkgo.Entry("a facet in another case", "Audio.Artist"),
		ginkgo.Entry("the nested driveItem path of the mime type", "file.mimeType"),
		ginkgo.Entry("a property the index rules refuse for terms", "lastModifiedDateTime"),
	)

	ginkgo.It("resolves lastModifiedDateTime to the modification time of the index", func() {
		g, captured := graphWithAggregations(&searchsvc.AggregationResult{
			Field:   "Mtime",
			Buckets: []*searchsvc.Bucket{{Key: "2026-01-01T00:00:00Z..", Count: 3}},
		})
		rr := postSearchQuery(g, searchQueryBody(`"aggregations": [{"field": "lastModifiedDateTime", "bucketDefinition": {"sortBy": "count", "ranges": [{"from": "2026-01-01T00:00:00Z"}]}}]`))
		Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())

		Expect(captured().GetAggregations()[0].GetField()).To(Equal("Mtime"))
		Expect(hitsContainer(rr).Aggregations[0].Field).To(Equal("lastModifiedDateTime"))
	})

	ginkgo.It("maps a metric result, without a value when the metric has none", func() {
		value := 1986.5
		g, captured := graphWithAggregations(
			&searchsvc.AggregationResult{Field: "audio.year", Metric: &searchsvc.Metric{Kind: searchsvc.MetricKind_METRIC_KIND_AVG, Value: &value, Sum: 3973, Count: 2}},
			&searchsvc.AggregationResult{Field: "audio.year", Metric: &searchsvc.Metric{Kind: searchsvc.MetricKind_METRIC_KIND_MIN}},
		)
		rr := postSearchQuery(g, searchQueryBody(`"aggregations": [
			{"field": "audio.year", "@libre.graph.metricDefinition": {"kind": "avg"}},
			{"field": "audio.year", "@libre.graph.metricDefinition": {"kind": "min"}}
		]`))
		Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())
		Expect(captured().GetAggregations()[0].GetMetricDefinition().GetKind()).To(Equal(searchsvc.MetricKind_METRIC_KIND_AVG))

		aggs := hitsContainer(rr).Aggregations
		Expect(aggs).To(HaveLen(2))
		Expect(aggs[0].Metric.Kind).To(Equal("avg"))
		Expect(aggs[0].Metric.Value).To(HaveValue(Equal(1986.5)))
		Expect(aggs[1].Metric.Kind).To(Equal("min"))
		Expect(aggs[1].Metric.Value).To(BeNil(), "no value, not zero")
	})

	ginkgo.DescribeTable("answers a property it does not evaluate with 501 instead of ignoring it",
		func(fragment, property string) {
			rr := postSearchQuery(graphWithoutSearch(), searchQueryBody(fragment))
			Expect(rr.Code).To(Equal(http.StatusNotImplemented), rr.Body.String())
			Expect(rr.Body.String()).To(ContainSubstring("notSupported"))
			Expect(rr.Body.String()).To(ContainSubstring(property))
		},
		ginkgo.Entry("a geohash aggregation",
			`"aggregations": [{"field": "location", "@libre.graph.geohashDefinition": {"precision": 5}}]`, "geohashDefinition"),
		ginkgo.Entry("a nested geohash aggregation",
			`"aggregations": [{"field": "audio.artist", "@libre.graph.subAggregations": [{"field": "location", "@libre.graph.geohashDefinition": {"precision": 5}}]}]`, "geohashDefinition"),
	)

	ginkgo.It("forwards sortProperties to the search service as order_by", func() {
		g, captured := graphWithSearchAnswer(&searchsvc.SearchResponse{})
		rr := postSearchQuery(g, searchQueryBody(`"sortProperties": [{"name": "photo.takenDateTime", "isDescending": true}, {"name": "name"}]`))
		Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())

		Expect(captured().GetOrderBy()).To(HaveLen(2))
		Expect(captured().GetOrderBy()[0].GetName()).To(Equal("photo.takenDateTime"))
		Expect(captured().GetOrderBy()[0].GetIsDescending()).To(BeTrue())
		Expect(captured().GetOrderBy()[1].GetName()).To(Equal("name"))
		Expect(captured().GetOrderBy()[1].GetIsDescending()).To(BeFalse())
	})

	ginkgo.It("rejects sorting by an unsupported field with 400", func() {
		rr := postSearchQuery(graphWithoutSearch(), searchQueryBody(`"sortProperties": [{"name": "photo.iso"}]`))
		Expect(rr.Code).To(Equal(http.StatusBadRequest), rr.Body.String())
		Expect(rr.Body.String()).To(ContainSubstring("photo.iso"))
	})

	ginkgo.It("rejects an $expand it does not know with 400", func() {
		req := httptest.NewRequest(http.MethodPost, "/search/query?$expand=thumbnails,permissions", bytes.NewBufferString(searchQueryBody(`"from": 0`)))
		rr := httptest.NewRecorder()
		graphWithoutSearch().SearchQuery(rr, req)
		Expect(rr.Code).To(Equal(http.StatusBadRequest), rr.Body.String())
		Expect(rr.Body.String()).To(ContainSubstring("permissions"))
	})

	ginkgo.DescribeTable("answers with the status the search service failed with",
		func(serviceErr error, status int, code, message string) {
			g := graphWithSearch(stubSearchService{
				search: func(*searchsvc.SearchRequest) (*searchsvc.SearchResponse, error) { return nil, serviceErr },
			})
			rr := postSearchQuery(g, searchQueryBody(`"aggregations": [{"field": "audio.artist"}]`))
			Expect(rr.Code).To(Equal(status), rr.Body.String())

			var decoded struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			Expect(json.Unmarshal(rr.Body.Bytes(), &decoded)).To(Succeed())
			Expect(decoded.Error.Code).To(Equal(code))
			Expect(decoded.Error.Message).To(Equal(message), "the detail, not the error envelope of the service")
		},
		ginkgo.Entry("a request the service refuses",
			merrors.BadRequest("search", "empty query provided"),
			http.StatusBadRequest, "invalidRequest", "empty query provided"),
		ginkgo.Entry("a timeout", merrors.Timeout("search", "deadline exceeded"),
			http.StatusServiceUnavailable, "serviceNotAvailable", "deadline exceeded"),
		ginkgo.Entry("a failing engine", merrors.InternalServerError("search", "engine down"),
			http.StatusInternalServerError, "generalException", "engine down"),
		ginkgo.Entry("an error that is no service error", errors.New("connection refused"),
			http.StatusInternalServerError, "generalException", "connection refused"),
	)

	ginkgo.It("issues terms and range tokens by the definition at the position of the result", func() {
		// the range aggregation and the metric share a field: the result at
		// position 0 is the range aggregation, whatever follows on that field
		g, _ := graphWithAggregations(
			&searchsvc.AggregationResult{Field: "audio.year", Buckets: []*searchsvc.Bucket{
				{Key: "..1980", Count: 2},
				{Key: "1980..1990", Count: 1},
				{Key: "1990..", Count: 4},
			}},
			&searchsvc.AggregationResult{Field: "audio.artist", Buckets: []*searchsvc.Bucket{
				{Key: "Saxon", Count: 2, SubAggregations: []*searchsvc.AggregationResult{
					{Field: "audio.album", Buckets: []*searchsvc.Bucket{{Key: "Wheels of Steel", Count: 2}}},
					{Field: "audio.year", Buckets: []*searchsvc.Bucket{{Key: "1980..", Count: 2}}},
				}},
			}},
			&searchsvc.AggregationResult{Field: "audio.year", Metric: &searchsvc.Metric{Kind: searchsvc.MetricKind_METRIC_KIND_MAX}},
		)
		rr := postSearchQuery(g, searchQueryBody(`"aggregations": [
			{"field": "audio.year", "bucketDefinition": {"sortBy": "keyAsNumber", "ranges": [{"to": "1980"}, {"from": "1980", "to": "1990"}, {"from": "1990"}]}},
			{"field": "audio.artist", "@libre.graph.subAggregations": [
				{"field": "audio.album"},
				{"field": "audio.year", "bucketDefinition": {"sortBy": "keyAsNumber", "ranges": [{"from": "1980"}]}}
			]},
			{"field": "audio.year", "@libre.graph.metricDefinition": {"kind": "max"}}
		]`))
		Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())

		aggs := hitsContainer(rr).Aggregations
		Expect(aggs).To(HaveLen(3))

		years := aggs[0].Buckets
		Expect(years).To(HaveLen(3))
		Expect(years[0].Token).To(Equal("range(min, 1980)"))
		Expect(years[1].Token).To(Equal("range(1980, 1990)"))
		Expect(years[2].Token).To(Equal(`range(1990, max, to="le")`))

		saxon := aggs[1].Buckets[0]
		Expect(saxon.Token).To(Equal(`"ǂǂ5361786f6e"`))
		Expect(saxon.SubAggregations).To(HaveLen(2))
		Expect(saxon.SubAggregations[0].Buckets[0].Token).To(Equal(`"ǂǂ576865656c73206f6620537465656c"`))
		Expect(saxon.SubAggregations[1].Buckets[0].Token).To(Equal(`range(1980, max, to="le")`))

		Expect(aggs[2].Metric).ToNot(BeNil())
		Expect(aggs[2].Buckets).To(BeEmpty())
	})

	ginkgo.DescribeTable("hands the buckets of an aggregation filter to the search service",
		func(filter string, want *searchsvc.AggregationFilter) {
			g, captured := graphWithAggregations()
			rr := postSearchQuery(g, searchQueryBody(fmt.Sprintf(`"aggregationFilters": [%q]`, filter)))
			Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())

			Expect(captured().GetAggregationFilters()).To(BeComparableTo([]*searchsvc.AggregationFilter{want}, protocmp.Transform()))
		},
		ginkgo.Entry("several terms", `audio.artist:or("ǂǂ5361786f6e", "ǂǂ49726f6e204d616964656e")`,
			&searchsvc.AggregationFilter{Field: "audio.artist", Terms: []string{"Saxon", "Iron Maiden"}}),
		ginkgo.Entry("several ranges", `audio.year:or(range(min, 1980),range(2010, max, to="le"))`,
			&searchsvc.AggregationFilter{Field: "audio.year", Ranges: []*searchsvc.BucketRange{{To: "1980"}, {From: "2010"}}}),
		ginkgo.Entry("a date range", `photo.takenDateTime:range(2018-08-11T00:00:00Z, 2018-08-12T00:00:00Z)`,
			&searchsvc.AggregationFilter{Field: "photo.takenDateTime", Ranges: []*searchsvc.BucketRange{{From: "2018-08-11T00:00:00Z", To: "2018-08-12T00:00:00Z"}}}),
	)
})
