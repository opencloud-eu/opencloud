// Package aggs translates proto aggregation options into the OpenSearch
// aggregation DSL and parses the response. Internal subpackage so its unit
// tests skip the parent package's Docker OpenSearch container.
package aggs

import (
	"encoding/json"
	"fmt"
	"time"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
)

// Build renders the aggregation tree as native OpenSearch aggregations, each
// named by its position among its siblings. A terms aggregation asks for
// every bucket up to the limit: the service layer cuts to the requested size
// after the cross-space merge, a cut here would make that merge inexact.
func Build(opts []*searchsvc.AggregationOption) (map[string]any, error) {
	if len(opts) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(opts))
	for i, opt := range opts {
		entry, err := build(opt)
		if err != nil {
			return nil, err
		}
		out[aggName(i)] = entry
	}
	return out, nil
}

func aggName(position int) string {
	return fmt.Sprintf("a_%d", position)
}

func build(opt *searchsvc.AggregationOption) (map[string]any, error) {
	var entry map[string]any
	switch aggregation.KindOf(opt) {
	case aggregation.KindMetric:
		// all accumulators, whatever the kind: the service layer merges them
		// across spaces and reduces to the kind's value
		return map[string]any{"stats": map[string]any{"field": opt.GetField()}}, nil
	case aggregation.KindRange:
		ranges, err := buildRanges(opt.GetField(), opt.GetBucketDefinition().GetRanges())
		if err != nil {
			return nil, err
		}
		entry = ranges
	default:
		entry = map[string]any{"terms": map[string]any{"field": opt.GetField(), "size": aggregation.MaxBuckets}}
	}
	children, err := Build(opt.GetSubAggregations())
	if err != nil {
		return nil, err
	}
	if children != nil {
		entry["aggs"] = children
	}
	return entry, nil
}

// buildRanges renders the numeric "range" or the "date_range" aggregation.
func buildRanges(field string, ranges []*searchsvc.BucketRange) (map[string]any, error) {
	parsed, err := aggregation.ParseRanges(field, ranges)
	if err != nil {
		return nil, err
	}
	kind := "range"
	out := make([]map[string]any, 0, len(ranges))
	for _, r := range parsed.Numeric {
		entry := map[string]any{"key": r.Key}
		if r.From != nil {
			entry["from"] = *r.From
		}
		if r.To != nil {
			entry["to"] = *r.To
		}
		out = append(out, entry)
	}
	for _, r := range parsed.Dates {
		kind = "date_range"
		entry := map[string]any{"key": r.Key}
		if !r.From.IsZero() {
			entry["from"] = r.From.Format(time.RFC3339Nano)
		}
		if !r.To.IsZero() {
			entry["to"] = r.To.Format(time.RFC3339Nano)
		}
		out = append(out, entry)
	}
	return map[string]any{kind: map[string]any{"field": field, "ranges": out}}, nil
}

// Parse reads the aggregation block of a response into one result per option,
// in request order.
func Parse(opts []*searchsvc.AggregationOption, raw json.RawMessage) ([]*searchsvc.AggregationResult, error) {
	node := aggNode{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &node); err != nil {
			return nil, fmt.Errorf("decode opensearch aggregations: %w", err)
		}
	}
	return parse(opts, node)
}

// aggNode is one level of the aggregation response, by aggregation name.
type aggNode map[string]json.RawMessage

func parse(opts []*searchsvc.AggregationOption, node aggNode) ([]*searchsvc.AggregationResult, error) {
	if len(opts) == 0 {
		return nil, nil
	}
	out := make([]*searchsvc.AggregationResult, 0, len(opts))
	for i, opt := range opts {
		res, err := result(opt, node[aggName(i)])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func result(opt *searchsvc.AggregationOption, raw json.RawMessage) (*searchsvc.AggregationResult, error) {
	if aggregation.KindOf(opt) == aggregation.KindMetric {
		return metricResult(opt, raw)
	}
	var body struct {
		Buckets          []json.RawMessage `json:"buckets"`
		SumOtherDocCount int64             `json:"sum_other_doc_count"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, fmt.Errorf("decode aggregation on %s: %w", opt.GetField(), err)
		}
	}
	// matches OpenSearch left out of the buckets: there are more than asked for
	if body.SumOtherDocCount > 0 {
		return nil, aggregation.ErrTooManyBuckets
	}

	res := &searchsvc.AggregationResult{Field: opt.GetField(), Buckets: []*searchsvc.Bucket{}}
	for _, rawBucket := range body.Buckets {
		bucket, err := parseBucket(opt, rawBucket)
		if err != nil {
			return nil, err
		}
		// a field without a value has no bucket, an empty one neither
		if bucket.GetKey() != "" {
			res.Buckets = append(res.Buckets, bucket)
		}
	}
	if aggregation.KindOf(opt) == aggregation.KindRange {
		byKey := make(map[string]*searchsvc.Bucket, len(res.Buckets))
		for _, b := range res.Buckets {
			byKey[b.GetKey()] = b
		}
		res.Buckets = aggregation.RangeBuckets(opt, byKey)
	}
	return res, nil
}

func parseBucket(opt *searchsvc.AggregationOption, raw json.RawMessage) (*searchsvc.Bucket, error) {
	var head struct {
		Key         any     `json:"key"`
		KeyAsString *string `json:"key_as_string"`
		DocCount    int64   `json:"doc_count"`
	}
	children := aggNode{}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("decode bucket of %s: %w", opt.GetField(), err)
	}
	if err := json.Unmarshal(raw, &children); err != nil {
		return nil, fmt.Errorf("decode bucket of %s: %w", opt.GetField(), err)
	}
	subs, err := parse(opt.GetSubAggregations(), children)
	if err != nil {
		return nil, err
	}
	return &searchsvc.Bucket{Key: bucketKey(head.Key, head.KeyAsString), Count: head.DocCount, SubAggregations: subs}, nil
}

// bucketKey spells a response key like the bleve backend does: a number
// without trailing zeros, a bool as true or false. OpenSearch keys a bool as
// 1 or 0 and spells it out next to it.
func bucketKey(key any, asString *string) string {
	if asString != nil {
		return *asString
	}
	switch k := key.(type) {
	case string:
		return k
	case float64:
		return aggregation.NumberKey(k)
	}
	return ""
}

// metricResult reads a stats aggregation; min and max are null without a
// single value.
func metricResult(opt *searchsvc.AggregationOption, raw json.RawMessage) (*searchsvc.AggregationResult, error) {
	res := &searchsvc.AggregationResult{
		Field:  opt.GetField(),
		Metric: &searchsvc.Metric{Kind: opt.GetMetricDefinition().GetKind()},
	}
	if len(raw) == 0 {
		return res, nil
	}
	var body struct {
		Count int64    `json:"count"`
		Sum   float64  `json:"sum"`
		Min   *float64 `json:"min"`
		Max   *float64 `json:"max"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("decode metric aggregation on %s: %w", opt.GetField(), err)
	}
	if body.Count == 0 || body.Min == nil || body.Max == nil {
		return res, nil
	}
	res.Metric.Count, res.Metric.Sum, res.Metric.Min, res.Metric.Max = body.Count, body.Sum, *body.Min, *body.Max
	return res, nil
}
