package aggregation_test

import (
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
)

func metricOpt(field string, kind searchsvc.MetricKind) *searchsvc.AggregationOption {
	return &searchsvc.AggregationOption{Field: field, MetricDefinition: &searchsvc.MetricDefinition{Kind: kind}}
}

// metric is what an engine answers for the given values: all accumulators.
func metric(field string, kind searchsvc.MetricKind, values ...float64) *searchsvc.AggregationResult {
	m := &searchsvc.Metric{Kind: kind, Count: int64(len(values))}
	if len(values) > 0 {
		m.Min, m.Max = slices.Min(values), slices.Max(values)
	}
	for _, v := range values {
		m.Sum += v
	}
	return &searchsvc.AggregationResult{Field: field, Metric: m}
}

func buckets(field string, bs ...*searchsvc.Bucket) *searchsvc.AggregationResult {
	return &searchsvc.AggregationResult{Field: field, Buckets: bs}
}

func bucket(key string, count int64, subs ...*searchsvc.AggregationResult) *searchsvc.Bucket {
	return &searchsvc.Bucket{Key: key, Count: count, SubAggregations: subs}
}

func rangesOf(sortBy searchsvc.BucketSortBy, rs ...*searchsvc.BucketRange) *searchsvc.BucketDefinition {
	return &searchsvc.BucketDefinition{SortBy: sortBy, Ranges: rs}
}

// render flattens results like the parity suite: one line per bucket or
// metric, nested ones below their bucket.
func render(results []*searchsvc.AggregationResult) []string {
	var walk func(prefix string, results []*searchsvc.AggregationResult) []string
	walk = func(prefix string, results []*searchsvc.AggregationResult) []string {
		out := []string{}
		for _, r := range results {
			if m := r.GetMetric(); m != nil {
				if m.Value == nil {
					out = append(out, prefix+fmt.Sprintf("%s %s none", r.GetField(), m.GetKind()))
				} else {
					out = append(out, prefix+fmt.Sprintf("%s %s=%v", r.GetField(), m.GetKind(), m.GetValue()))
				}
				continue
			}
			for _, b := range r.GetBuckets() {
				line := prefix + fmt.Sprintf("%s %s=%d", r.GetField(), b.GetKey(), b.GetCount())
				out = append(out, line)
				out = append(out, walk(line+" / ", b.GetSubAggregations())...)
			}
		}
		return out
	}
	return walk("", results)
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
	const (
		sum = searchsvc.MetricKind_METRIC_KIND_SUM
		min = searchsvc.MetricKind_METRIC_KIND_MIN
		max = searchsvc.MetricKind_METRIC_KIND_MAX
		avg = searchsvc.MetricKind_METRIC_KIND_AVG
	)

	It("answers one result per option without any space, every range at zero", func() {
		opts := []*searchsvc.AggregationOption{
			{Field: "audio.artist"},
			metricOpt("audio.year", sum),
			{Field: "audio.year", BucketDefinition: rangesOf(searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_NUMBER, &searchsvc.BucketRange{To: "1980"}, &searchsvc.BucketRange{From: "1980"}),
				SubAggregations: []*searchsvc.AggregationOption{{Field: "audio.artist"}, metricOpt("audio.year", max)}},
		}
		got := merged(opts)
		Expect(got).To(HaveLen(3))
		Expect(got[0].GetField()).To(Equal("audio.artist"))
		Expect(got[0].GetBuckets()).To(BeEmpty())
		Expect(render(got)).To(Equal([]string{
			"audio.year METRIC_KIND_SUM none",
			"audio.year ..1980=0", "audio.year ..1980=0 / audio.year METRIC_KIND_MAX none",
			"audio.year 1980..=0", "audio.year 1980..=0 / audio.year METRIC_KIND_MAX none",
		}))
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
		opts := []*searchsvc.AggregationOption{{Field: "audio.artist"}, metricOpt("audio.year", sum)}
		space := []*searchsvc.AggregationResult{buckets("audio.artist", bucket("Saxon", 2)), metric("audio.year", sum, 1980)}
		merged(opts, space, space)
		Expect(space[0].GetBuckets()[0].GetCount()).To(Equal(int64(2)))
		Expect(space[1].GetMetric().GetSum()).To(Equal(1980.0))
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

	It("keeps range and metric aggregations on one field apart", func() {
		opts := []*searchsvc.AggregationOption{
			{Field: "audio.year", BucketDefinition: rangesOf(searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_NUMBER, &searchsvc.BucketRange{To: "1980"}, &searchsvc.BucketRange{From: "1980"})},
			metricOpt("audio.year", min),
			{Field: "audio.year", BucketDefinition: rangesOf(searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_NUMBER, &searchsvc.BucketRange{To: "2000"}, &searchsvc.BucketRange{From: "2000"})},
			metricOpt("audio.year", max),
		}
		space := func(below1980, from1980, below2000, from2000 int64, years ...float64) []*searchsvc.AggregationResult {
			return []*searchsvc.AggregationResult{
				buckets("audio.year", bucket("..1980", below1980), bucket("1980..", from1980)),
				metric("audio.year", min, years...),
				buckets("audio.year", bucket("..2000", below2000), bucket("2000..", from2000)),
				metric("audio.year", max, years...),
			}
		}
		got := merged(opts, space(1, 1, 2, 0, 1971, 1982), space(0, 2, 1, 1, 1999, 2001))
		Expect(render(got)).To(Equal([]string{
			"audio.year ..1980=1", "audio.year 1980..=3",
			"audio.year METRIC_KIND_MIN=1971",
			"audio.year ..2000=3", "audio.year 2000..=1",
			"audio.year METRIC_KIND_MAX=2001",
		}))
	})

	It("keeps sibling metrics on one field apart below a bucket that spans spaces", func() {
		opts := []*searchsvc.AggregationOption{{Field: "audio.artist", SubAggregations: []*searchsvc.AggregationOption{
			metricOpt("audio.year", sum),
			metricOpt("audio.year", avg),
		}}}
		space := func(years ...float64) []*searchsvc.AggregationResult {
			return []*searchsvc.AggregationResult{buckets("audio.artist",
				bucket("Saxon", int64(len(years)), metric("audio.year", sum, years...), metric("audio.year", avg, years...)),
			)}
		}
		got := merged(opts, space(1971, 1975), space(1980))
		Expect(render(got)).To(Equal([]string{
			"audio.artist Saxon=3",
			"audio.artist Saxon=3 / audio.year METRIC_KIND_SUM=5926",
			"audio.artist Saxon=3 / audio.year METRIC_KIND_AVG=1975.3333333333333",
		}))
	})

	It("answers nested results in request order", func() {
		opts := []*searchsvc.AggregationOption{{Field: "audio.artist", SubAggregations: []*searchsvc.AggregationOption{
			{Field: "audio.genre"}, {Field: "audio.album"}, metricOpt("audio.year", max), {Field: "audio.composers"},
		}}}
		space := []*searchsvc.AggregationResult{buckets("audio.artist", bucket("Saxon", 1,
			buckets("audio.genre", bucket("Metal", 1)),
			buckets("audio.album", bucket("Wheels of Steel", 1)),
			metric("audio.year", max, 1980),
			buckets("audio.composers", bucket("Byford", 1)),
		))}
		for range 20 {
			got := merged(opts, space, space)
			fields := []string{}
			for _, sub := range got[0].GetBuckets()[0].GetSubAggregations() {
				fields = append(fields, sub.GetField())
			}
			Expect(fields).To(Equal([]string{"audio.genre", "audio.album", "audio.year", "audio.composers"}))
		}
	})

	DescribeTable("reduces a metric over the values of all spaces",
		func(kind searchsvc.MetricKind, want string) {
			opts := []*searchsvc.AggregationOption{metricOpt("audio.year", kind)}
			got := merged(opts,
				[]*searchsvc.AggregationResult{metric("audio.year", kind, 1970, 1980, 1990)},
				[]*searchsvc.AggregationResult{metric("audio.year", kind)},
				[]*searchsvc.AggregationResult{metric("audio.year", kind, 2010)},
			)
			Expect(render(got)).To(Equal([]string{"audio.year " + want}))
		},
		Entry("sum", sum, "METRIC_KIND_SUM=7950"),
		Entry("min, a space without values is no zero", min, "METRIC_KIND_MIN=1970"),
		Entry("max", max, "METRIC_KIND_MAX=2010"),
		Entry("avg of the values, not of the averages of the spaces", avg, "METRIC_KIND_AVG=1987.5"),
	)

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

		It("sorts range buckets by their lower bound with keyAsNumber, an open one first", func() {
			opts := []*searchsvc.AggregationOption{{Field: "audio.year", BucketDefinition: rangesOf(searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_NUMBER,
				&searchsvc.BucketRange{From: "2000"}, &searchsvc.BucketRange{From: "990", To: "2000"}, &searchsvc.BucketRange{To: "990"},
			)}}
			got := merged(opts, []*searchsvc.AggregationResult{buckets("audio.year", bucket("2000..", 1), bucket("990..2000", 1), bucket("..990", 1))})
			Expect(render(got)).To(Equal([]string{"audio.year ..990=1", "audio.year 990..2000=1", "audio.year 2000..=1"}))
		})

		It("sorts date range buckets by their lower bound with keyAsNumber", func() {
			opts := []*searchsvc.AggregationOption{{Field: "photo.takenDateTime", BucketDefinition: rangesOf(searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_NUMBER,
				&searchsvc.BucketRange{From: "2019-01-01T00:00:00Z"}, &searchsvc.BucketRange{To: "2019-01-01T00:00:00Z"},
			)}}
			got := merged(opts, []*searchsvc.AggregationResult{buckets("photo.takenDateTime", bucket("2019-01-01T00:00:00Z..", 1), bucket("..2019-01-01T00:00:00Z", 3))})
			Expect(render(got)).To(Equal([]string{"photo.takenDateTime ..2019-01-01T00:00:00Z=3", "photo.takenDateTime 2019-01-01T00:00:00Z..=1"}))
		})
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

		It("returns every range of a range aggregation, whatever the size", func() {
			opts := []*searchsvc.AggregationOption{{Field: "audio.year", Size: 1, BucketDefinition: rangesOf(searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_NUMBER,
				&searchsvc.BucketRange{To: "1980"}, &searchsvc.BucketRange{From: "1980"},
			)}}
			got := merged(opts, []*searchsvc.AggregationResult{buckets("audio.year", bucket("..1980", 0), bucket("1980..", 3))})
			Expect(render(got)).To(Equal([]string{"audio.year ..1980=0", "audio.year 1980..=3"}))
		})

		It("drops buckets below the minimum count, counted across spaces", func() {
			opts := []*searchsvc.AggregationOption{{Field: "audio.artist", BucketDefinition: &searchsvc.BucketDefinition{MinimumCount: 2, IsDescending: true}}}
			got := merged(opts,
				[]*searchsvc.AggregationResult{buckets("audio.artist", bucket("Saxon", 1), bucket("Motörhead", 1))},
				[]*searchsvc.AggregationResult{buckets("audio.artist", bucket("Saxon", 1))},
			)
			Expect(render(got)).To(Equal([]string{"audio.artist Saxon=2"}))
		})

		It("shapes the buckets of a sub-aggregation by its own definition", func() {
			opts := []*searchsvc.AggregationOption{{Field: "audio.artist", SubAggregations: []*searchsvc.AggregationOption{{
				Field: "audio.album", Size: 2,
				BucketDefinition: &searchsvc.BucketDefinition{SortBy: searchsvc.BucketSortBy_BUCKET_SORT_BY_KEY_AS_STRING, MinimumCount: 2},
			}}}}
			space := []*searchsvc.AggregationResult{buckets("audio.artist", bucket("Saxon", 4, buckets("audio.album",
				bucket("Wheels of Steel", 1), bucket("Strong Arm of the Law", 1), bucket("Denim and Leather", 1), bucket("Crusader", 1),
			)))}
			Expect(render(merged(opts, space, space))).To(Equal([]string{
				"audio.artist Saxon=8",
				"audio.artist Saxon=8 / audio.album Crusader=2",
				"audio.artist Saxon=8 / audio.album Denim and Leather=2",
			}))
			Expect(render(merged(opts, space))).To(Equal([]string{"audio.artist Saxon=4"}), "one space alone stays below the minimum count")
		})
	})
})
