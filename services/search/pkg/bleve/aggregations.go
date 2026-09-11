package bleve

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/numeric"
	bleveSearch "github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/collector"
	index "github.com/blevesearch/bleve_index_api"

	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	searchQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

// Bleve facets count one field and cannot nest, so metrics and
// sub-aggregations are folded from doc values by aggCollector, hooked into
// the collector walk through bleve's document-match-handler context key.
// Loading hits for them instead costs a stored-document decode per match.

// defaultFacetSize is used when no size is requested; the service layer trims
// after cross-space merge.
const defaultFacetSize = 1000

func collected(agg *searchService.AggregationOption) bool {
	return agg.GetMetricKind() != searchService.MetricKind_METRIC_KIND_UNSPECIFIED || len(agg.GetSubAggregations()) > 0
}

// geohashLevel resolves a geohash aggregation to the geohash sibling field of
// its geopoint and the term prefix of the requested precision: the sibling
// holds one depth-tagged term per precision (see geohash.go), so the terms
// with the prefix "<precision>/" are the cells of that precision.
func geohashLevel(agg *searchService.AggregationOption) (field, prefix string, err error) {
	p := int(agg.GetGeohashPrecision())
	if p < 1 || p > geohashPrecision {
		return "", "", fmt.Errorf("geohash precision %d out of range 1-%d", p, geohashPrecision)
	}
	base, ok := searchQuery.ResolveGeopointField(agg.GetField())
	if !ok {
		return "", "", fmt.Errorf("geohash aggregation on non-geo field %q", agg.GetField())
	}
	return base + geohashSuffix, strconv.Itoa(p) + "/", nil
}

func newBleveFacetRequest(agg *searchService.AggregationOption) (*bleve.FacetRequest, error) {
	size := int(agg.GetSize())
	if size <= 0 {
		size = defaultFacetSize
	}
	if agg.GetGeohashPrecision() != 0 {
		field, prefix, err := geohashLevel(agg)
		if err != nil {
			return nil, err
		}
		fr := bleve.NewFacetRequest(field, size)
		fr.TermPrefix = prefix
		return fr, nil
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
	// a geohash facet carries the depth tag in every term, the cell is the rest
	var prefix string
	if agg.GetGeohashPrecision() != 0 {
		prefix = strconv.Itoa(int(agg.GetGeohashPrecision())) + "/"
	}
	for _, t := range fr.Terms.Terms() {
		buckets = append(buckets, &searchService.Bucket{Key: strings.TrimPrefix(t.Term, prefix), Count: int64(t.Count)})
	}
	return buckets
}

type levelKind int

const (
	levelTerms levelKind = iota
	levelGeohash
	levelNumericRange
	levelDateRange
	levelMetric
)

type numericRange struct {
	name     string
	min, max *float64
}

type dateRange struct {
	name       string
	start, end time.Time
}

type aggLevel struct {
	opt      *searchService.AggregationOption
	field    string // the indexed field the doc values are read from
	kind     levelKind
	prefix   string // geohash: the depth tag of the requested precision
	numeric  []numericRange
	dates    []dateRange
	children []*aggLevel
}

func newAggLevel(opt *searchService.AggregationOption) (*aggLevel, error) {
	l := &aggLevel{opt: opt, field: opt.GetField()}
	switch {
	case opt.GetMetricKind() != searchService.MetricKind_METRIC_KIND_UNSPECIFIED:
		l.kind = levelMetric
	case opt.GetGeohashPrecision() != 0:
		field, prefix, err := geohashLevel(opt)
		if err != nil {
			return nil, err
		}
		l.kind, l.field, l.prefix = levelGeohash, field, prefix
	case len(aggregationRanges(opt)) > 0:
		ranges := aggregationRanges(opt)
		if rangesAreDates(ranges) {
			l.kind = levelDateRange
			for _, r := range ranges {
				start, end, err := parseDateRange(opt.GetField(), r)
				if err != nil {
					return nil, err
				}
				l.dates = append(l.dates, dateRange{name: rangeBucketKey(r), start: start, end: end})
			}
		} else {
			l.kind = levelNumericRange
			for _, r := range ranges {
				l.numeric = append(l.numeric, numericRange{name: rangeBucketKey(r), min: parseFloatPtr(r.GetFrom()), max: parseFloatPtr(r.GetTo())})
			}
		}
	default:
		l.kind = levelTerms
	}
	for _, sub := range opt.GetSubAggregations() {
		child, err := newAggLevel(sub)
		if err != nil {
			return nil, err
		}
		l.children = append(l.children, child)
	}
	return l, nil
}

// fieldValues is per-document scratch, reused across documents.
type fieldValues struct {
	asTerms   bool
	asNumbers bool
	terms     []string
	numbers   []int64
}

type bucketAcc struct {
	counts map[string]int64
	subs   map[string][]*bucketAcc

	value float64 // SUM/MIN/MAX
	sum   float64 // AVG numerator
	count int64   // AVG denominator
	seen  bool
}

func newBucketAcc(l *aggLevel) *bucketAcc {
	a := &bucketAcc{}
	if l.kind != levelMetric {
		a.counts = map[string]int64{}
		if len(l.children) > 0 {
			a.subs = map[string][]*bucketAcc{}
		}
	}
	return a
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
	fv, ok := c.fields[l.field]
	if !ok {
		fv = &fieldValues{}
		c.fields[l.field] = fv
		c.fieldNames = append(c.fieldNames, l.field)
	}
	if l.kind == levelTerms || l.kind == levelGeohash {
		fv.asTerms = true
	} else {
		fv.asNumbers = true
	}
	for _, child := range l.children {
		c.register(child)
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
		fv.terms = fv.terms[:0]
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
	if fv.asTerms {
		fv.terms = append(fv.terms, string(term))
	}
	if fv.asNumbers {
		pc := numeric.PrefixCoded(term)
		if shift, err := pc.Shift(); err == nil && shift == 0 {
			if v, err := pc.Int64(); err == nil {
				fv.numbers = append(fv.numbers, v)
			}
		}
	}
}

func (c *aggCollector) fold(a *bucketAcc, l *aggLevel) {
	fv := c.fields[l.field]
	switch l.kind {
	case levelMetric:
		for _, raw := range fv.numbers {
			a.addMetric(l.opt.GetMetricKind(), numeric.Int64ToFloat64(raw))
		}
	case levelTerms:
		for _, term := range fv.terms {
			if term != "" {
				c.foldBucket(a, l, term)
			}
		}
	case levelGeohash:
		for _, term := range fv.terms {
			if cell, ok := strings.CutPrefix(term, l.prefix); ok {
				c.foldBucket(a, l, cell)
			}
		}
	case levelNumericRange:
		for _, raw := range fv.numbers {
			v := numeric.Int64ToFloat64(raw)
			for _, r := range l.numeric {
				if (r.min == nil || v >= *r.min) && (r.max == nil || v < *r.max) {
					c.foldBucket(a, l, r.name)
				}
			}
		}
	case levelDateRange:
		for _, raw := range fv.numbers {
			t := time.Unix(0, raw)
			for _, r := range l.dates {
				if (r.start.IsZero() || !t.Before(r.start)) && (r.end.IsZero() || t.Before(r.end)) {
					c.foldBucket(a, l, r.name)
				}
			}
		}
	}
}

func (c *aggCollector) foldBucket(a *bucketAcc, l *aggLevel, key string) {
	a.counts[key]++
	a.seen = true
	if len(l.children) == 0 {
		return
	}
	subs, ok := a.subs[key]
	if !ok {
		subs = make([]*bucketAcc, len(l.children))
		for i, child := range l.children {
			subs[i] = newBucketAcc(child)
		}
		a.subs[key] = subs
	}
	for i, child := range l.children {
		c.fold(subs[i], child)
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
	if l.kind == levelMetric {
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

	buckets := make([]*searchService.Bucket, 0, len(a.counts))
	for key, count := range a.counts {
		b := &searchService.Bucket{Key: key, Count: count}
		for i, child := range l.children {
			if sub := a.subs[key][i].result(child); sub != nil {
				b.SubAggregations = append(b.SubAggregations, sub)
			}
		}
		buckets = append(buckets, b)
	}
	// same order as a bleve terms facet
	sort.Slice(buckets, func(i, j int) bool {
		if buckets[i].Count == buckets[j].Count {
			return buckets[i].Key < buckets[j].Key
		}
		return buckets[i].Count > buckets[j].Count
	})
	if size := int(l.opt.GetSize()); size > 0 && len(buckets) > size {
		buckets = buckets[:size]
	}
	return &searchService.AggregationResult{Field: l.opt.GetField(), Buckets: buckets}
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
