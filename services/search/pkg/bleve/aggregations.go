package bleve

import (
	"fmt"
	"strconv"
	"time"

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

func newBleveFacetRequest(agg *searchService.AggregationOption) (*bleve.FacetRequest, error) {
	size := int(agg.GetSize())
	if size <= 0 {
		size = defaultFacetSize
	}
	fr := bleve.NewFacetRequest(agg.GetField(), size)
	ranges := aggregationRanges(agg)
	if rangesAreDates(ranges) {
		// bleve facets cannot mix numeric and date ranges, so one date-looking
		// bound switches the whole aggregation to date mode.
		for _, r := range ranges {
			start, end, err := parseDateRange(agg.GetField(), r)
			if err != nil {
				return nil, err
			}
			fr.AddDateTimeRange(rangeBucketKey(r), start, end)
		}
		return fr, nil
	}
	for _, r := range ranges {
		minP := parseFloatPtr(r.GetFrom())
		maxP := parseFloatPtr(r.GetTo())
		fr.AddNumericRange(rangeBucketKey(r), minP, maxP)
	}
	return fr, nil
}

var rangeTimeLayouts = []string{time.RFC3339, "2006-01-02"}

func rangesAreDates(ranges []*searchService.BucketRange) bool {
	for _, r := range ranges {
		for _, s := range []string{r.GetFrom(), r.GetTo()} {
			if s == "" {
				continue
			}
			if _, err := strconv.ParseFloat(s, 64); err == nil {
				continue
			}
			if _, err := parseRangeTime(s); err == nil {
				return true
			}
		}
	}
	return false
}

// The zero time marks an open bound.
func parseRangeTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	for _, layout := range rangeTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", s)
}

func parseDateRange(field string, r *searchService.BucketRange) (time.Time, time.Time, error) {
	start, err := parseRangeTime(r.GetFrom())
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date range bound %q on field %q", r.GetFrom(), field)
	}
	end, err := parseRangeTime(r.GetTo())
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date range bound %q on field %q", r.GetTo(), field)
	}
	return start, end, nil
}

func aggregationRanges(agg *searchService.AggregationOption) []*searchService.BucketRange {
	bd := agg.GetBucketDefinition()
	if bd == nil {
		return nil
	}
	return bd.GetRanges()
}

// rangeBucketKey formats a range as "from-to" for stable merge keys; open sides
// render as "-N" or "N-".
func rangeBucketKey(r *searchService.BucketRange) string {
	return r.GetFrom() + "-" + r.GetTo()
}

func parseFloatPtr(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

func facetBuckets(fr *bleveSearch.FacetResult, agg *searchService.AggregationOption) []*searchService.Bucket {
	buckets := make([]*searchService.Bucket, 0)
	if len(aggregationRanges(agg)) > 0 {
		for _, nr := range fr.NumericRanges {
			buckets = append(buckets, &searchService.Bucket{Key: nr.Name, Count: int64(nr.Count)})
		}
		for _, dr := range fr.DateRanges {
			buckets = append(buckets, &searchService.Bucket{Key: dr.Name, Count: int64(dr.Count)})
		}
		return buckets
	}
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
			Buckets: facetBuckets(fr, agg),
		})
	}
	return out
}
