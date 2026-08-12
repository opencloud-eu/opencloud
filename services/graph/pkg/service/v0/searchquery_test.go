package svc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	// ginkgo qualified: the svc package declares Context (option.go), which
	// would collide with a dot-import.
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go-micro.dev/v4/client"

	"github.com/opencloud-eu/opencloud/pkg/log"
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

func postSearchQuery(g Graph, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/search/query", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	g.SearchQuery(rr, req)
	return rr
}

func int32Ptr(v int32) *int32 { return &v }

var _ = ginkgo.Describe("SearchQuery", func() {
	ginkgo.It("forwards aggregations to the search service and groups results by request", func() {
		var captured *searchsvc.SearchRequest
		g := graphWithSearch(stubSearchService{
			search: func(req *searchsvc.SearchRequest) (*searchsvc.SearchResponse, error) {
				captured = req
				return &searchsvc.SearchResponse{
					TotalMatches: 10,
					Aggregations: []*searchsvc.AggregationResult{{
						Field: "audio.artist",
						Buckets: []*searchsvc.Bucket{
							{Key: "Pink Floyd", Count: 7},
							{Key: "Motörhead", Count: 3},
						},
					}},
				}, nil
			},
		})

		rr := postSearchQuery(g, `{
			"requests": [{
				"entityTypes": ["driveItem"],
				"query": {"queryString": "mediatype:audio"},
				"size": 0,
				"aggregations": [{
					"field": "audio.artist",
					"size": 5,
					"bucketDefinition": {"sortBy": "count", "isDescending": true}
				}]
			}]
		}`)

		Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())
		Expect(captured).ToNot(BeNil())
		Expect(captured.Aggregations).To(HaveLen(1))
		Expect(captured.Aggregations[0].Field).To(Equal("audio.artist"))

		var decoded struct {
			Value []struct {
				HitsContainers []struct {
					Aggregations []struct {
						Field   *string `json:"field"`
						Buckets []struct {
							Key   *string `json:"key"`
							Count *int64  `json:"count"`
						} `json:"buckets"`
					} `json:"aggregations"`
				} `json:"hitsContainers"`
			} `json:"value"`
		}
		Expect(json.Unmarshal(rr.Body.Bytes(), &decoded)).To(Succeed())
		Expect(decoded.Value).To(HaveLen(1))
		Expect(decoded.Value[0].HitsContainers).To(HaveLen(1))

		aggs := decoded.Value[0].HitsContainers[0].Aggregations
		Expect(aggs).To(HaveLen(1))
		Expect(aggs[0].Field).To(HaveValue(Equal("audio.artist")))
		Expect(aggs[0].Buckets).To(HaveLen(2))
	})

	ginkgo.DescribeTable("clampPagination keeps from/size within valid bounds",
		func(from, size *int32, wantFrom, wantSize int32) {
			gotFrom, gotSize := clampPagination(from, size)
			Expect(gotFrom).To(Equal(wantFrom))
			Expect(gotSize).To(Equal(wantSize))
		},
		ginkgo.Entry("defaults", nil, nil, int32(0), int32(25)),
		ginkgo.Entry("zero size", int32Ptr(5), int32Ptr(0), int32(5), int32(0)),
		ginkgo.Entry("negative from clamps to zero", int32Ptr(-10), int32Ptr(5), int32(0), int32(5)),
		ginkgo.Entry("negative size clamps to zero", int32Ptr(10), int32Ptr(-1), int32(10), int32(0)),
		ginkgo.Entry("oversized size clamps to max", int32Ptr(0), int32Ptr(1000), int32(0), int32(500)),
		ginkgo.Entry("from+size overflow collapses", int32Ptr(1<<31-1), int32Ptr(500), int32(1<<31-1-500), int32(500)),
	)

	ginkgo.It("forwards sortProperties to the search service as order_by", func() {
		var captured *searchsvc.SearchRequest
		g := graphWithSearch(stubSearchService{
			search: func(req *searchsvc.SearchRequest) (*searchsvc.SearchResponse, error) {
				captured = req
				return &searchsvc.SearchResponse{}, nil
			},
		})
		rr := postSearchQuery(g, `{
			"requests": [{
				"entityTypes": ["driveItem"],
				"query": {"queryString": "mediatype:image"},
				"sortProperties": [
					{"name": "photo.takenDateTime", "isDescending": true},
					{"name": "name"}
				]
			}]
		}`)
		Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())
		Expect(captured).ToNot(BeNil())
		Expect(captured.OrderBy).To(HaveLen(2))
		Expect(captured.OrderBy[0].Name).To(Equal("photo.takenDateTime"))
		Expect(captured.OrderBy[0].IsDescending).To(BeTrue())
		Expect(captured.OrderBy[1].Name).To(Equal("name"))
		Expect(captured.OrderBy[1].IsDescending).To(BeFalse())
	})

	ginkgo.It("rejects sorting by an unsupported field with 400", func() {
		g := graphWithSearch(stubSearchService{
			search: func(*searchsvc.SearchRequest) (*searchsvc.SearchResponse, error) {
				ginkgo.Fail("search service must not be called when validation fails")
				return nil, nil
			},
		})
		rr := postSearchQuery(g, `{
			"requests": [{
				"entityTypes": ["driveItem"],
				"query": {"queryString": "mediatype:image"},
				"sortProperties": [{"name": "photo.iso"}]
			}]
		}`)
		Expect(rr.Code).To(Equal(http.StatusBadRequest), rr.Body.String())
		Expect(rr.Body.String()).To(ContainSubstring("photo.iso"))
	})

	ginkgo.It("rejects a terms aggregation on a numeric field with 400", func() {
		g := graphWithSearch(stubSearchService{
			search: func(*searchsvc.SearchRequest) (*searchsvc.SearchResponse, error) {
				ginkgo.Fail("search service must not be called when validation fails")
				return nil, nil
			},
		})
		rr := postSearchQuery(g, `{
			"requests": [{
				"entityTypes": ["driveItem"],
				"query": {"queryString": "mediatype:audio"},
				"size": 0,
				"aggregations": [{"field": "audio.year"}]
			}]
		}`)
		Expect(rr.Code).To(Equal(http.StatusBadRequest), rr.Body.String())
	})

	ginkgo.It("allows a range aggregation on a numeric field", func() {
		called := false
		g := graphWithSearch(stubSearchService{
			search: func(*searchsvc.SearchRequest) (*searchsvc.SearchResponse, error) {
				called = true
				return &searchsvc.SearchResponse{}, nil
			},
		})
		rr := postSearchQuery(g, `{
			"requests": [{
				"entityTypes": ["driveItem"],
				"query": {"queryString": "mediatype:audio"},
				"size": 0,
				"aggregations": [{
					"field": "audio.year",
					"bucketDefinition": {
						"sortBy": "keyAsString",
						"ranges": [{"from": "1970", "to": "1980"}]
					}
				}]
			}]
		}`)
		Expect(rr.Code).To(Equal(http.StatusOK), rr.Body.String())
		Expect(called).To(BeTrue())
	})
})
