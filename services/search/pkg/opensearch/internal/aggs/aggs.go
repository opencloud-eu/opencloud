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
// (terms). Entries get an index-derived
// name so repeated aggs on one field don't collide. A range bound that is
// neither a number nor a date is an error.
func Build(opts []*searchsvc.AggregationOption) (map[string]any, error) {
	return buildLevel(opts, "a")
}

func buildLevel(opts []*searchsvc.AggregationOption, prefix string) (map[string]any, error) {
	if len(opts) == 0 {
		return nil, nil
	}
	aggs := map[string]any{}
	for i, opt := range opts {
		name := fmt.Sprintf("%s_%d", prefix, i)
		entry, err := buildOne(opt)
		if err != nil {
			return nil, err
		}
		if entry != nil {
			aggs[name] = entry
		}
	}
	if len(aggs) == 0 {
		return nil, nil
	}
	return aggs, nil
}

func buildOne(opt *searchsvc.AggregationOption) (map[string]any, error) {
	field := opt.GetField()
	size := int(opt.GetSize())
	if size <= 0 {
		size = DefaultFacetSize
	}
	return map[string]any{
		"terms": map[string]any{
			"field": field,
			"size":  size,
		},
	}, nil
}

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
		if res := parseOne(raw, opt); res != nil {
			out = append(out, res)
		}
	}
	return out
}

func parseOne(raw json.RawMessage, opt *searchsvc.AggregationOption) *searchsvc.AggregationResult {
	field := opt.GetField()
	var body struct {
		Buckets []json.RawMessage `json:"buckets"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil
	}
	buckets := make([]*searchsvc.Bucket, 0, len(body.Buckets))
	for _, b := range body.Buckets {
		if bucket := parseBucket(b); bucket != nil {
			buckets = append(buckets, bucket)
		}
	}
	return &searchsvc.AggregationResult{
		Field:   field,
		Buckets: buckets,
	}
}

func parseBucket(raw json.RawMessage) *searchsvc.Bucket {
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
	return b
}

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
