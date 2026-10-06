// Package aggregation holds what the engines and the service layer share
// about aggregations: the kind of an option, range bounds, metric
// accumulators and the cross-space merge.
package aggregation

import (
	"fmt"
	"math"
	"strconv"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
)

// Kind is what an aggregation option asks for.
type Kind int

const (
	KindTerms Kind = iota
	KindRange
	KindMetric
)

// KindOf tells the kind from the definitions an option carries.
func KindOf(opt *searchsvc.AggregationOption) Kind {
	switch {
	case opt.GetMetricDefinition() != nil:
		return KindMetric
	case len(opt.GetBucketDefinition().GetRanges()) > 0:
		return KindRange
	}
	return KindTerms
}

// MaxBuckets is how many buckets the aggregations of one request may have in
// one space: the search.max_buckets default of OpenSearch, held on every
// engine.
const MaxBuckets = math.MaxUint16

// ErrTooManyBuckets is what an engine answers beyond MaxBuckets; the service
// layer turns it into a bad request.
var ErrTooManyBuckets = fmt.Errorf("the aggregations have more than %d buckets, narrow the query or the aggregations", MaxBuckets)

// CheckBuckets holds the results of one space to MaxBuckets.
func CheckBuckets(results []*searchsvc.AggregationResult) error {
	if countBuckets(results) > MaxBuckets {
		return ErrTooManyBuckets
	}
	return nil
}

func countBuckets(results []*searchsvc.AggregationResult) int {
	n := 0
	for _, r := range results {
		n += len(r.GetBuckets())
	}
	return n
}

// NumberKey and BoolKey spell the bucket key of a numeric and a bool value,
// the same on every engine.
func NumberKey(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func BoolKey(v bool) string {
	return strconv.FormatBool(v)
}
