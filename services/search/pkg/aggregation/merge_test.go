package aggregation_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
)

func buckets(field string, bs ...*searchsvc.Bucket) *searchsvc.AggregationResult {
	return &searchsvc.AggregationResult{Field: field, Buckets: bs}
}

func bucket(key string, count int64) *searchsvc.Bucket {
	return &searchsvc.Bucket{Key: key, Count: count}
}

// render flattens results like the parity suite: one line per bucket.
func render(results []*searchsvc.AggregationResult) []string {
	out := []string{}
	for _, r := range results {
		for _, b := range r.GetBuckets() {
			out = append(out, fmt.Sprintf("%s %s=%d", r.GetField(), b.GetKey(), b.GetCount()))
		}
	}
	return out
}

// merged runs the service layer's fold over the answers of several spaces.
func merged(opts []*searchsvc.AggregationOption, spaces ...[]*searchsvc.AggregationResult) []*searchsvc.AggregationResult {
	acc := aggregation.Empty(opts)
	for _, results := range spaces {
		acc = aggregation.Merge(opts, acc, results)
	}
	aggregation.Finalize(opts, acc)
	return acc
}

var _ = Describe("Merge and Finalize", func() {
	It("answers one result per option without any space", func() {
		opts := []*searchsvc.AggregationOption{{Field: "audio.artist"}, {Field: "audio.album"}}
		got := merged(opts)
		Expect(got).To(HaveLen(2))
		Expect(got[0].GetField()).To(Equal("audio.artist"))
		Expect(got[0].GetBuckets()).To(BeEmpty())
		Expect(got[1].GetField()).To(Equal("audio.album"))
	})

	It("adds up the counts of a bucket across spaces", func() {
		opts := []*searchsvc.AggregationOption{{Field: "audio.artist"}}
		got := merged(opts,
			[]*searchsvc.AggregationResult{buckets("audio.artist", bucket("Saxon", 2), bucket("Motörhead", 1))},
			[]*searchsvc.AggregationResult{buckets("audio.artist", bucket("Saxon", 3), bucket("Led Zeppelin", 1))},
		)
		Expect(render(got)).To(Equal([]string{"audio.artist Saxon=5", "audio.artist Led Zeppelin=1", "audio.artist Motörhead=1"}))
	})

	It("leaves the answers of the spaces untouched", func() {
		opts := []*searchsvc.AggregationOption{{Field: "audio.artist"}}
		space := []*searchsvc.AggregationResult{buckets("audio.artist", bucket("Saxon", 2))}
		merged(opts, space, space)
		Expect(space[0].GetBuckets()[0].GetCount()).To(Equal(int64(2)))
	})

	It("keeps two terms aggregations on one field apart", func() {
		opts := []*searchsvc.AggregationOption{{Field: "audio.artist", Size: 1}, {Field: "audio.artist", Size: 2}}
		space := []*searchsvc.AggregationResult{
			buckets("audio.artist", bucket("Saxon", 2), bucket("Motörhead", 1)),
			buckets("audio.artist", bucket("Saxon", 2), bucket("Motörhead", 1)),
		}
		Expect(render(merged(opts, space, space))).To(Equal([]string{
			"audio.artist Saxon=4",
			"audio.artist Saxon=4", "audio.artist Motörhead=2",
		}))
	})

	Describe("bucket order", func() {
		space := []*searchsvc.AggregationResult{buckets("audio.track",
			bucket("10", 2), bucket("9", 2), bucket("2", 5), bucket("b-side", 1),
		)}
		keys := func(sortBy searchsvc.BucketSortBy, descending bool) []string {
			opts := []*searchsvc.AggregationOption{{Field: "audio.track", BucketDefinition: &searchsvc.BucketDefinition{SortBy: sortBy, IsDescending: descending}}}
			out := []string{}
			for _, b := range merged(opts, space)[0].GetBuckets() {
				out = append(out, b.GetKey())
			}
			return out
		}

		It("defaults to count descending, ties by key", func() {
			opts := []*searchsvc.AggregationOption{{Field: "audio.track"}}
			Expect(render(merged(opts, space))).To(Equal([]string{"audio.track 2=5", "audio.track 10=2", "audio.track 9=2", "audio.track b-side=1"}))
		})

		DescribeTable("follows sortBy and isDescending, ties by key",
			func(sortBy searchsvc.BucketSortBy, descending bool, want ...string) {
				Expect(keys(sortBy, descending)).To(Equal(want))
			},
			Entry("count ascending", searchsvc.BucketSortBy_BUCKET_SORT_BY_COUNT, false, "b-side", "10", "9", "2"),
			Entry("count descending", searchsvc.BucketSortBy_BUCKET_SORT_BY_COUNT, true, "2", "10", "9", "b-side"),
			Entry("unspecified is the facet order", searchsvc.BucketSortBy_BUCKET_SORT_BY_UNSPECIFIED, true, "2", "10", "9", "b-side"),
			Entry("unspecified ignores isDescending", searchsvc.BucketSortBy_BUCKET_SORT_BY_UNSPECIFIED, false, "2", "10", "9", "b-side"),
			Entry("keyAsString ascending", searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_STRING, false, "10", "2", "9", "b-side"),
			Entry("keyAsString descending", searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_STRING, true, "b-side", "9", "2", "10"),
			Entry("keyAsNumber ascending, no number last", searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_NUMBER, false, "2", "9", "10", "b-side"),
			Entry("keyAsNumber descending, no number still last", searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_NUMBER, true, "10", "9", "2", "b-side"),
		)
	})

	Describe("bucket selection", func() {
		It("trims a terms aggregation to its size after the merge", func() {
			opts := []*searchsvc.AggregationOption{{Field: "audio.artist", Size: 1}}
			got := merged(opts,
				[]*searchsvc.AggregationResult{buckets("audio.artist", bucket("Motörhead", 3), bucket("Saxon", 2))},
				[]*searchsvc.AggregationResult{buckets("audio.artist", bucket("Saxon", 2))},
			)
			Expect(render(got)).To(Equal([]string{"audio.artist Saxon=4"}))
		})

		It("drops buckets below the minimum count, counted across spaces", func() {
			opts := []*searchsvc.AggregationOption{{Field: "audio.artist", BucketDefinition: &searchsvc.BucketDefinition{MinimumCount: 2, IsDescending: true}}}
			got := merged(opts,
				[]*searchsvc.AggregationResult{buckets("audio.artist", bucket("Saxon", 1), bucket("Motörhead", 1))},
				[]*searchsvc.AggregationResult{buckets("audio.artist", bucket("Saxon", 1))},
			)
			Expect(render(got)).To(Equal([]string{"audio.artist Saxon=2"}))
		})
	})
})
