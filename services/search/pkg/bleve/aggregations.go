package bleve

import (
	"context"
	"errors"
	"time"

	"github.com/blevesearch/bleve/v2/numeric"
	bleveSearch "github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/collector"
	index "github.com/blevesearch/bleve_index_api"

	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// Bleve facets count the indexed terms of one field (numbers among them as
// prefix-coded terms), so every aggregation is folded from doc values by
// aggCollector, hooked into the collector walk through bleve's
// document-match-handler context key. Loading hits for them instead costs a
// stored-document decode per match.

// termKey spells an indexed term as a bucket key: bleve indexes a bool as T
// or F. A field without a value has no bucket, an empty one neither.
func termKey(fieldType, term string) (string, bool) {
	if term == "" {
		return "", false
	}
	if fieldType == mapping.TypeBool {
		return aggregation.BoolKey(term == "T"), true
	}
	return term, true
}

type aggLevel struct {
	opt       *searchService.AggregationOption
	kind      aggregation.Kind
	fieldType string
	ranges    aggregation.Ranges
}

// numeric values are prefix-coded in the index, their terms come from the
// decoded doc values
func (l *aggLevel) numericTerms() bool {
	return l.kind == aggregation.KindTerms && l.fieldType == mapping.TypeNumeric
}

func newAggLevel(opt *searchService.AggregationOption) (*aggLevel, error) {
	l := &aggLevel{opt: opt, kind: aggregation.KindOf(opt), fieldType: search.FieldType(opt.GetField())}
	if l.kind == aggregation.KindRange {
		ranges, err := aggregation.ParseRanges(opt.GetField(), opt.GetBucketDefinition().GetRanges())
		if err != nil {
			return nil, err
		}
		l.ranges = ranges
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
	metric *searchService.Metric
}

func newBucketAcc(l *aggLevel) *bucketAcc {
	if l.kind == aggregation.KindMetric {
		return &bucketAcc{metric: &searchService.Metric{Kind: l.opt.GetMetricDefinition().GetKind()}}
	}
	return &bucketAcc{counts: map[string]int64{}}
}

// aggCollector serves one search; bleve's collector is single-threaded.
type aggCollector struct {
	fields     map[string]*fieldValues
	fieldNames []string
	roots      []*aggRoot
}

type aggRoot struct {
	level *aggLevel
	acc   *bucketAcc
}

func newAggCollector(aggs []*searchService.AggregationOption) (*aggCollector, error) {
	if len(aggs) == 0 {
		return nil, nil
	}
	c := &aggCollector{fields: map[string]*fieldValues{}}
	for _, agg := range aggs {
		l, err := newAggLevel(agg)
		if err != nil {
			return nil, err
		}
		c.register(l)
		c.roots = append(c.roots, &aggRoot{level: l, acc: newBucketAcc(l)})
	}
	return c, nil
}

func (c *aggCollector) register(l *aggLevel) {
	fv, ok := c.fields[l.opt.GetField()]
	if !ok {
		fv = &fieldValues{}
		c.fields[l.opt.GetField()] = fv
		c.fieldNames = append(c.fieldNames, l.opt.GetField())
	}
	if l.kind == aggregation.KindTerms && !l.numericTerms() {
		fv.asTerms = true
	} else {
		fv.asNumbers = true
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
				if err := c.collect(dvr, d); err != nil {
					return err
				}
			}
			return inner(d)
		}, loadID, nil
	})
	return context.WithValue(ctx, bleveSearch.MakeDocumentMatchHandlerKey, maker)
}

func (c *aggCollector) collect(dvr index.DocValueReader, d *bleveSearch.DocumentMatch) error {
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
	fv := c.fields[l.opt.GetField()]
	switch l.kind {
	case aggregation.KindMetric:
		for _, raw := range fv.numbers {
			aggregation.Observe(a.metric, numeric.Int64ToFloat64(raw))
		}
	case aggregation.KindTerms:
		if l.numericTerms() {
			for _, raw := range fv.numbers {
				c.foldBucket(a, l, aggregation.NumberKey(numeric.Int64ToFloat64(raw)))
			}
			return
		}
		for _, term := range fv.terms {
			if key, ok := termKey(l.fieldType, term); ok {
				c.foldBucket(a, l, key)
			}
		}
	case aggregation.KindRange:
		for _, raw := range fv.numbers {
			for _, r := range l.ranges.Numeric {
				if r.Contains(numeric.Int64ToFloat64(raw)) {
					c.foldBucket(a, l, r.Key)
				}
			}
			for _, r := range l.ranges.Dates {
				if r.Contains(time.Unix(0, raw)) {
					c.foldBucket(a, l, r.Key)
				}
			}
		}
	}
}

func (c *aggCollector) foldBucket(a *bucketAcc, _ *aggLevel, key string) {
	a.counts[key]++
}

// results returns one result per aggregation, in request order.
func (c *aggCollector) results() []*searchService.AggregationResult {
	out := make([]*searchService.AggregationResult, 0, len(c.roots))
	for _, root := range c.roots {
		out = append(out, root.acc.result(root.level))
	}
	return out
}

func (a *bucketAcc) result(l *aggLevel) *searchService.AggregationResult {
	r := &searchService.AggregationResult{Field: l.opt.GetField()}
	if l.kind == aggregation.KindMetric {
		r.Metric = a.metric
		return r
	}

	counted := make(map[string]*searchService.Bucket, len(a.counts))
	for key, count := range a.counts {
		counted[key] = &searchService.Bucket{Key: key, Count: count}
	}
	if l.kind == aggregation.KindRange {
		r.Buckets = aggregation.RangeBuckets(l.opt, counted)
		return r
	}
	r.Buckets = make([]*searchService.Bucket, 0, len(counted))
	for _, b := range counted {
		r.Buckets = append(r.Buckets, b)
	}
	return r
}
