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

// Build renders the aggregations as native OpenSearch aggregations, each
// named by its position. A terms aggregation asks for every bucket up to the
// limit: the service layer cuts to the requested size after the cross-space
// merge, a cut here would make that merge inexact.
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
	case aggregation.KindRange:
		ranges, err := buildRanges(opt.GetField(), opt.GetBucketDefinition().GetRanges())
		if err != nil {
			return nil, err
		}
		entry = ranges
	default:
		entry = map[string]any{"terms": map[string]any{"field": opt.GetField(), "size": aggregation.MaxBuckets}}
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
	byKey := make(map[string]*searchsvc.Bucket, len(body.Buckets))
	for _, rawBucket := range body.Buckets {
		bucket, err := parseBucket(opt, rawBucket)
		if err != nil {
			return nil, err
		}
		// a field without a value has no bucket, an empty one neither
		if bucket.GetKey() != "" {
			res.Buckets = append(res.Buckets, bucket)
			byKey[bucket.GetKey()] = bucket
		}
	}
	if aggregation.KindOf(opt) == aggregation.KindRange {
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
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("decode bucket of %s: %w", opt.GetField(), err)
	}
	return &searchsvc.Bucket{Key: bucketKey(head.Key, head.KeyAsString), Count: head.DocCount}, nil
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
