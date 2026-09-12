package bleve

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/numeric"
	bleveSearch "github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/collector"
	index "github.com/blevesearch/bleve_index_api"

	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
)

// Bleve facets count one field, so metrics are folded from doc values by
// aggCollector, hooked into the collector walk through bleve's
// document-match-handler context key. Loading hits for them instead costs a
// stored-document decode per match.

// defaultFacetSize is used when no size is requested; the service layer trims
// after cross-space merge.
const defaultFacetSize = 1000

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

type aggLevel struct {
	opt *searchService.AggregationOption
}

func newAggLevel(opt *searchService.AggregationOption) (*aggLevel, error) {
	if opt.GetMetricKind() == searchService.MetricKind_METRIC_KIND_UNSPECIFIED {
		return nil, fmt.Errorf("sub-aggregations are not supported by bleve yet")
	}
	return &aggLevel{opt: opt}, nil
}

// fieldValues is per-document scratch, reused across documents.
type fieldValues struct {
	numbers []int64
}

type bucketAcc struct {
	value float64 // SUM/MIN/MAX
	sum   float64 // AVG numerator
	count int64   // AVG denominator
	seen  bool
}

func newBucketAcc(*aggLevel) *bucketAcc {
	return &bucketAcc{}
}

// aggCollector serves one search; bleve's collector is single-threaded.
type aggCollector struct {
	fields     map[string]*fieldValues
	fieldNames []string
	roots      map[int]*aggRoot // by position in the request's aggregations
}

type aggRoot struct {
	level *aggLevel
	acc   *bucketAcc
}

func newAggCollector(aggs []*searchService.AggregationOption) (*aggCollector, error) {
	c := &aggCollector{fields: map[string]*fieldValues{}, roots: map[int]*aggRoot{}}
	for i, agg := range aggs {
		if !collected(agg) {
			continue
		}
		l, err := newAggLevel(agg)
		if err != nil {
			return nil, err
		}
		c.register(l)
		c.roots[i] = &aggRoot{level: l, acc: newBucketAcc(l)}
	}
	if len(c.roots) == 0 {
		return nil, nil
	}
	return c, nil
}

func (c *aggCollector) register(l *aggLevel) {
	if _, ok := c.fields[l.opt.GetField()]; !ok {
		c.fields[l.opt.GetField()] = &fieldValues{}
		c.fieldNames = append(c.fieldNames, l.opt.GetField())
	}
}

// The handler runs for every match, before the top-n cut.
func (c *aggCollector) withContext(ctx context.Context) context.Context {
	maker := bleveSearch.MakeDocumentMatchHandler(func(sc *bleveSearch.SearchContext) (bleveSearch.DocumentMatchHandler, bool, error) {
		inner, loadID, err := collector.MakeTopNDocumentMatchHandler(sc)
		if err != nil {
			return nil, false, err
		}
		if inner == nil {
			return nil, false, errors.New("aggregations need the top-n collector")
		}
		dvr, err := sc.IndexReader.DocValueReader(c.fieldNames)
		if err != nil {
			return nil, false, err
		}
		return func(d *bleveSearch.DocumentMatch) error {
			if d != nil {
				if err := c.collect(sc.IndexReader, dvr, d); err != nil {
					return err
				}
			}
			return inner(d)
		}, loadID, nil
	})
	return context.WithValue(ctx, bleveSearch.MakeDocumentMatchHandlerKey, maker)
}

func (c *aggCollector) collect(reader index.IndexReader, dvr index.DocValueReader, d *bleveSearch.DocumentMatch) error {
	if d.IndexInternalID == nil {
		id, err := reader.InternalID(d.ID)
		if err != nil {
			return err
		}
		d.IndexInternalID = id
	}
	for _, fv := range c.fields {
		fv.numbers = fv.numbers[:0]
	}
	if err := dvr.VisitDocValues(d.IndexInternalID, c.visit); err != nil {
		return err
	}
	for _, root := range c.roots {
		c.fold(root.acc, root.level)
	}
	return nil
}

// Numeric and date doc values are prefix-coded at several precisions; only
// shift 0 carries the exact value.
func (c *aggCollector) visit(field string, term []byte) {
	fv, ok := c.fields[field]
	if !ok {
		return
	}
	pc := numeric.PrefixCoded(term)
	if shift, err := pc.Shift(); err == nil && shift == 0 {
		if v, err := pc.Int64(); err == nil {
			fv.numbers = append(fv.numbers, v)
		}
	}
}

func (c *aggCollector) fold(a *bucketAcc, l *aggLevel) {
	for _, raw := range c.fields[l.opt.GetField()].numbers {
		a.addMetric(l.opt.GetMetricKind(), numeric.Int64ToFloat64(raw))
	}
}

func (a *bucketAcc) addMetric(kind searchService.MetricKind, v float64) {
	switch kind {
	case searchService.MetricKind_METRIC_KIND_SUM:
		a.value += v
	case searchService.MetricKind_METRIC_KIND_MIN:
		if !a.seen || v < a.value {
			a.value = v
		}
	case searchService.MetricKind_METRIC_KIND_MAX:
		if !a.seen || v > a.value {
			a.value = v
		}
	case searchService.MetricKind_METRIC_KIND_AVG:
		a.sum += v
		a.count++
	}
	a.seen = true
}

func (a *bucketAcc) result(l *aggLevel) *searchService.AggregationResult {
	if !a.seen {
		return nil
	}
	r := &searchService.AggregationResult{Field: l.opt.GetField(), MetricKind: l.opt.GetMetricKind()}
	if l.opt.GetMetricKind() == searchService.MetricKind_METRIC_KIND_AVG {
		r.Sum = a.sum
		r.Count = a.count
	} else {
		r.Value = a.value
	}
	return r
}

func extractBleveAggregations(res *bleve.SearchResult, aggs []*searchService.AggregationOption, c *aggCollector) []*searchService.AggregationResult {
	if len(aggs) == 0 {
		return nil
	}
	out := make([]*searchService.AggregationResult, 0, len(aggs))
	for i, agg := range aggs {
		if collected(agg) {
			if c == nil {
				continue
			}
			if root, ok := c.roots[i]; ok {
				if r := root.acc.result(root.level); r != nil {
					out = append(out, r)
				}
			}
			continue
		}
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
