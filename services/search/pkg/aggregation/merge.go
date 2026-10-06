package aggregation

import (
	"cmp"
	"math"
	"slices"
	"strconv"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
)

// Empty returns the results of a search without matches: one per option, a
// range aggregation with every range at a count of zero.
func Empty(opts []*searchsvc.AggregationOption) []*searchsvc.AggregationResult {
	if len(opts) == 0 {
		return nil
	}
	out := make([]*searchsvc.AggregationResult, len(opts))
	for i, opt := range opts {
		out[i] = &searchsvc.AggregationResult{Field: opt.GetField()}
		if KindOf(opt) == KindRange {
			out[i].Buckets = RangeBuckets(opt, nil)
		}
	}
	return out
}

// RangeBuckets lists every range of a range aggregation in request order: the
// answered bucket where there is one, a count of zero otherwise.
func RangeBuckets(opt *searchsvc.AggregationOption, answered map[string]*searchsvc.Bucket) []*searchsvc.Bucket {
	ranges := opt.GetBucketDefinition().GetRanges()
	out := make([]*searchsvc.Bucket, 0, len(ranges))
	for _, r := range ranges {
		bucket, ok := answered[RangeKey(r)]
		if !ok {
			bucket = &searchsvc.Bucket{Key: RangeKey(r)}
		}
		out = append(out, bucket)
	}
	return out
}

// Merge folds the results of one space into acc and returns it. Results
// belong to the option at their position, so several aggregations on one
// field stay apart.
func Merge(opts []*searchsvc.AggregationOption, acc, results []*searchsvc.AggregationResult) []*searchsvc.AggregationResult {
	if acc == nil {
		acc = Empty(opts)
	}
	for i := range opts {
		if i >= len(results) || results[i] == nil {
			continue
		}
		byKey := make(map[string]*searchsvc.Bucket, len(acc[i].GetBuckets()))
		for _, b := range acc[i].GetBuckets() {
			byKey[b.GetKey()] = b
		}
		for _, b := range results[i].GetBuckets() {
			merged, ok := byKey[b.GetKey()]
			if !ok {
				merged = &searchsvc.Bucket{Key: b.GetKey()}
				byKey[b.GetKey()] = merged
				acc[i].Buckets = append(acc[i].Buckets, merged)
			}
			merged.Count += b.GetCount()
		}
	}
	return acc
}

// Finalize shapes merged results per their options: buckets get their minimum
// count, order and size.
func Finalize(opts []*searchsvc.AggregationOption, results []*searchsvc.AggregationResult) {
	for i, opt := range opts {
		results[i].Buckets = shapeBuckets(opt, results[i].GetBuckets())
	}
}

func shapeBuckets(opt *searchsvc.AggregationOption, buckets []*searchsvc.Bucket) []*searchsvc.Bucket {
	bd := opt.GetBucketDefinition()
	if minCount := int64(bd.GetMinimumCount()); minCount > 0 {
		buckets = slices.DeleteFunc(buckets, func(b *searchsvc.Bucket) bool { return b.GetCount() < minCount })
	}

	slices.SortFunc(buckets, bucketOrder(opt))

	// size shapes terms aggregations only, a range aggregation returns the
	// ranges it was asked for
	if size := int(opt.GetSize()); size > 0 && KindOf(opt) == KindTerms && len(buckets) > size {
		buckets = buckets[:size]
	}
	return buckets
}

// bucketOrder follows sortBy and isDescending, ties by key. Without a sortBy
// the buckets come like a facet, by count descending.
func bucketOrder(opt *searchsvc.AggregationOption) func(a, b *searchsvc.Bucket) int {
	bd := opt.GetBucketDefinition()
	direction := 1
	if bd.GetIsDescending() {
		direction = -1
	}
	byKey := func(a, b *searchsvc.Bucket) int { return cmp.Compare(a.GetKey(), b.GetKey()) }
	switch bd.GetSortBy() {
	case searchsvc.BucketSortBy_BUCKET_SORT_BY_COUNT:
		return func(a, b *searchsvc.Bucket) int {
			return cmp.Or(direction*cmp.Compare(a.GetCount(), b.GetCount()), byKey(a, b))
		}
	case searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_STRING:
		return func(a, b *searchsvc.Bucket) int { return direction * byKey(a, b) }
	case searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_NUMBER:
		// keys that are no number come last, whatever the direction
		number := keyNumbers(opt)
		return func(a, b *searchsvc.Bucket) int {
			av, aok := number(a.GetKey())
			bv, bok := number(b.GetKey())
			return cmp.Or(cmp.Compare(boolInt(!aok), boolInt(!bok)), direction*cmp.Compare(av, bv), byKey(a, b))
		}
	}
	return func(a, b *searchsvc.Bucket) int {
		return cmp.Or(cmp.Compare(b.GetCount(), a.GetCount()), byKey(a, b))
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// keyNumbers reads a terms key as a number and a range key as its lower
// bound, an open one as the smallest.
func keyNumbers(opt *searchsvc.AggregationOption) func(key string) (float64, bool) {
	if KindOf(opt) != KindRange {
		return func(key string) (float64, bool) {
			v, err := strconv.ParseFloat(key, 64)
			return v, err == nil
		}
	}
	lower := map[string]float64{}
	if ranges, err := ParseRanges(opt.GetField(), opt.GetBucketDefinition().GetRanges()); err == nil {
		for _, r := range ranges.Numeric {
			lower[r.Key] = math.Inf(-1)
			if r.From != nil {
				lower[r.Key] = *r.From
			}
		}
		for _, r := range ranges.Dates {
			lower[r.Key] = math.Inf(-1)
			if !r.From.IsZero() {
				lower[r.Key] = float64(r.From.UnixNano())
			}
		}
	}
	return func(key string) (float64, bool) {
		v, ok := lower[key]
		return v, ok
	}
}
