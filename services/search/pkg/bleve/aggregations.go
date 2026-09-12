package bleve

import (
	"github.com/blevesearch/bleve/v2"
	bleveSearch "github.com/blevesearch/bleve/v2/search"

	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
)

// defaultFacetSize is used when no size is requested; the service layer trims
// after cross-space merge.
const defaultFacetSize = 1000

// collected reports an aggregation bleve facets cannot answer: a metric or one
// with sub-aggregations. Facets count one field and cannot nest.
func collected(agg *searchService.AggregationOption) bool {
	return agg.GetMetricKind() != searchService.MetricKind_METRIC_KIND_UNSPECIFIED || len(agg.GetSubAggregations()) > 0
}

func newBleveFacetRequest(agg *searchService.AggregationOption) *bleve.FacetRequest {
	size := int(agg.GetSize())
	if size <= 0 {
		size = defaultFacetSize
	}
	return bleve.NewFacetRequest(agg.GetField(), size)
}

func facetBuckets(fr *bleveSearch.FacetResult) []*searchService.Bucket {
	buckets := make([]*searchService.Bucket, 0)
	for _, t := range fr.Terms.Terms() {
		buckets = append(buckets, &searchService.Bucket{Key: t.Term, Count: int64(t.Count)})
	}
	return buckets
}

func extractBleveAggregations(res *bleve.SearchResult, aggs []*searchService.AggregationOption) []*searchService.AggregationResult {
	if len(aggs) == 0 {
		return nil
	}
	out := make([]*searchService.AggregationResult, 0, len(aggs))
	for _, agg := range aggs {
		fr, ok := res.Facets[agg.GetField()]
		if !ok {
			continue
		}
		out = append(out, &searchService.AggregationResult{
			Field:   agg.GetField(),
			Buckets: facetBuckets(fr),
		})
	}
	return out
}
