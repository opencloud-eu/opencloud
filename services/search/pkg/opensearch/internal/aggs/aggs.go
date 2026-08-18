// Package aggs translates proto aggregation options into the OpenSearch
// aggregation DSL and parses the response. Internal subpackage so its unit
// tests skip the parent package's Docker OpenSearch container.
package aggs

import (
	"encoding/json"
	"fmt"
	"strconv"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
)

// DefaultFacetSize matches the bleve backend: pull a generous bucket count per
// space, the service layer trims to top N after cross-space merge.
const DefaultFacetSize = 1000

// Build translates AggregationOptions into the OpenSearch aggregation DSL
// (terms, range, metric, nested). Entries get an index-derived name so repeated
// aggs on one field don't collide.
func Build(opts []*searchsvc.AggregationOption) map[string]any {
	return buildLevel(opts, "a")
}

func buildLevel(opts []*searchsvc.AggregationOption, prefix string) map[string]any {
	if len(opts) == 0 {
		return nil
	}
	aggs := map[string]any{}
	for i, opt := range opts {
		name := fmt.Sprintf("%s_%d", prefix, i)
		if entry := buildOne(opt, name); entry != nil {
			aggs[name] = entry
		}
	}
	if len(aggs) == 0 {
		return nil
	}
	return aggs
}

func buildOne(opt *searchsvc.AggregationOption, name string) map[string]any {
	field := opt.GetField()
	if mk := opt.GetMetricKind(); mk != searchsvc.MetricKind_METRIC_KIND_UNSPECIFIED {
		return buildMetric(field, mk)
	}
	var entry map[string]any
	if ranges := rangesOf(opt); len(ranges) > 0 {
		entry = map[string]any{
			"range": map[string]any{
				"field":  field,
				"ranges": buildRanges(ranges),
			},
		}
	} else {
		size := int(opt.GetSize())
		if size <= 0 {
			size = DefaultFacetSize
		}
		entry = map[string]any{
			"terms": map[string]any{
				"field": field,
				"size":  size,
			},
		}
	}
	if subs := opt.GetSubAggregations(); len(subs) > 0 {
		if nested := buildLevel(subs, name); nested != nil {
			entry["aggs"] = nested
		}
	}
	return entry
}

// buildMetric emits the sum/min/max metric. AVG uses a stats agg to transport
// (sum, count) for the cross-space merge; the service layer collapses to the average.
func buildMetric(field string, kind searchsvc.MetricKind) map[string]any {
	switch kind {
	case searchsvc.MetricKind_METRIC_KIND_SUM:
		return map[string]any{"sum": map[string]any{"field": field}}
	case searchsvc.MetricKind_METRIC_KIND_MIN:
		return map[string]any{"min": map[string]any{"field": field}}
	case searchsvc.MetricKind_METRIC_KIND_MAX:
		return map[string]any{"max": map[string]any{"field": field}}
	case searchsvc.MetricKind_METRIC_KIND_AVG:
		return map[string]any{"stats": map[string]any{"field": field}}
	}
	return nil
}

func rangesOf(opt *searchsvc.AggregationOption) []*searchsvc.BucketRange {
	bd := opt.GetBucketDefinition()
	if bd == nil {
		return nil
	}
	return bd.GetRanges()
}

func buildRanges(ranges []*searchsvc.BucketRange) []map[string]any {
	out := make([]map[string]any, 0, len(ranges))
	for _, r := range ranges {
		entry := map[string]any{
			"key": RangeKey(r),
		}
		if v, err := strconv.ParseFloat(r.GetFrom(), 64); err == nil {
			entry["from"] = v
		}
		if v, err := strconv.ParseFloat(r.GetTo(), 64); err == nil {
			entry["to"] = v
		}
		out = append(out, entry)
	}
	return out
}

// RangeKey mirrors the bleve backend so cross-space merging keys match.
func RangeKey(r *searchsvc.BucketRange) string {
	return r.GetFrom() + "-" + r.GetTo()
}

// Parse converts the response aggregations block into proto results, preserving
// request order and recursing into sub-aggregations. Empty input yields
// (nil, nil); invalid JSON yields an error.
func Parse(raw json.RawMessage, opts []*searchsvc.AggregationOption) ([]*searchsvc.AggregationResult, error) {
	if len(raw) == 0 || len(opts) == 0 {
		return nil, nil
	}
	node, err := parseNode(raw)
	if err != nil {
		return nil, err
	}
	return parseLevel(node, opts, "a"), nil
}

// aggNode is a lazily-decoded cursor over one level of the aggs response.
type aggNode map[string]json.RawMessage

func parseNode(raw json.RawMessage) (aggNode, error) {
	var m aggNode
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("decode opensearch aggregations: %w", err)
	}
	return m, nil
}

func parseLevel(node aggNode, opts []*searchsvc.AggregationOption, prefix string) []*searchsvc.AggregationResult {
	out := make([]*searchsvc.AggregationResult, 0, len(opts))
	for i, opt := range opts {
		name := fmt.Sprintf("%s_%d", prefix, i)
		raw, ok := node[name]
		if !ok {
			continue
		}
		if res := parseOne(raw, opt, name); res != nil {
			out = append(out, res)
		}
	}
	return out
}

func parseOne(raw json.RawMessage, opt *searchsvc.AggregationOption, name string) *searchsvc.AggregationResult {
	field := opt.GetField()
	if mk := opt.GetMetricKind(); mk != searchsvc.MetricKind_METRIC_KIND_UNSPECIFIED {
		return parseMetric(raw, field, mk)
	}
	var body struct {
		Buckets []json.RawMessage `json:"buckets"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil
	}
	buckets := make([]*searchsvc.Bucket, 0, len(body.Buckets))
	for _, b := range body.Buckets {
		if bucket := parseBucket(b, opt.GetSubAggregations(), name); bucket != nil {
			buckets = append(buckets, bucket)
		}
	}
	return &searchsvc.AggregationResult{
		Field:   field,
		Buckets: buckets,
	}
}

func parseBucket(raw json.RawMessage, subs []*searchsvc.AggregationOption, prefix string) *searchsvc.Bucket {
	var head struct {
		Key      any   `json:"key"`
		DocCount int64 `json:"doc_count"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil
	}
	b := &searchsvc.Bucket{
		Key:   bucketKeyToString(head.Key),
		Count: head.DocCount,
	}
	if len(subs) > 0 {
		node, err := parseNode(raw)
		if err == nil {
			b.SubAggregations = parseLevel(node, subs, prefix)
		}
	}
	return b
}

func parseMetric(raw json.RawMessage, field string, kind searchsvc.MetricKind) *searchsvc.AggregationResult {
	switch kind {
	case searchsvc.MetricKind_METRIC_KIND_SUM,
		searchsvc.MetricKind_METRIC_KIND_MIN,
		searchsvc.MetricKind_METRIC_KIND_MAX:
		var body struct {
			Value *float64 `json:"value"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil
		}
		res := &searchsvc.AggregationResult{
			Field:      field,
			MetricKind: kind,
		}
		if body.Value != nil {
			res.Value = *body.Value
		}
		return res
	case searchsvc.MetricKind_METRIC_KIND_AVG:
		var body struct {
			Sum   float64 `json:"sum"`
			Count int64   `json:"count"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil
		}
		return &searchsvc.AggregationResult{
			Field:      field,
			Sum:        body.Sum,
			Count:      body.Count,
			MetricKind: kind,
		}
	}
	return nil
}

// bucketKeyToString normalises a response key to a string (terms are strings,
// ranges use our "from-to" key, numeric terms come back as JSON numbers).
func bucketKeyToString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		// format without trailing zeros so keys match filter values
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	case nil:
		return ""
	default:
		return ""
	}
}
