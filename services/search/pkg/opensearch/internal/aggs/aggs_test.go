package aggs_test

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/aggs"
)

var _ = Describe("Build", func() {
	build := func(opts []*searchsvc.AggregationOption) map[string]any {
		res, err := aggs.Build(opts)
		Expect(err).ToNot(HaveOccurred())
		return res
	}

	It("builds a terms aggregation", func() {
		res := build([]*searchsvc.AggregationOption{
			{Field: "audio.artist", Size: 10},
		})
		Expect(res).ToNot(BeNil())
		entry, ok := res["a_0"].(map[string]any)
		Expect(ok).To(BeTrue())
		terms, ok := entry["terms"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(terms["field"]).To(Equal("audio.artist"))
		Expect(terms["size"]).To(Equal(10))
	})

	It("builds a date_range aggregation for date bounds", func() {
		res := build([]*searchsvc.AggregationOption{{
			Field: "photo.takenDateTime",
			BucketDefinition: &searchsvc.BucketDefinition{
				Ranges: []*searchsvc.BucketRange{
					{From: "2018-08-01", To: "2018-09-01"},
					{From: "2018-08-11T00:00:00Z"},
				},
			},
		}})
		r := res["a_0"].(map[string]any)["date_range"].(map[string]any)
		Expect(r["field"]).To(Equal("photo.takenDateTime"))
		ranges := r["ranges"].([]map[string]any)
		Expect(ranges).To(HaveLen(2))
		Expect(ranges[0]).To(SatisfyAll(
			HaveKeyWithValue("key", "2018-08-01-2018-09-01"),
			HaveKeyWithValue("from", "2018-08-01"),
			HaveKeyWithValue("to", "2018-09-01"),
		))
		Expect(ranges[1]).To(HaveKeyWithValue("from", "2018-08-11T00:00:00Z"))
		Expect(ranges[1]).ToNot(HaveKey("to"))
	})

	It("rejects a bound that is neither number nor date", func() {
		_, err := aggs.Build([]*searchsvc.AggregationOption{{
			Field: "photo.takenDateTime",
			BucketDefinition: &searchsvc.BucketDefinition{
				Ranges: []*searchsvc.BucketRange{
					{From: "2018-08-11T00:00:00Z", To: "not-a-date"},
				},
			},
		}})
		Expect(err).To(HaveOccurred())
	})

	It("builds a range aggregation with open-ended bounds", func() {
		res := build([]*searchsvc.AggregationOption{{
			Field: "audio.year",
			BucketDefinition: &searchsvc.BucketDefinition{
				Ranges: []*searchsvc.BucketRange{
					{From: "1970", To: "1980"},
					{To: "1970"},
					{From: "2020"},
				},
			},
		}})
		r := res["a_0"].(map[string]any)["range"].(map[string]any)
		Expect(r["field"]).To(Equal("audio.year"))
		ranges := r["ranges"].([]map[string]any)
		Expect(ranges).To(HaveLen(3))
		Expect(ranges[0]).To(SatisfyAll(
			HaveKeyWithValue("key", "1970-1980"),
			HaveKeyWithValue("from", 1970.0),
			HaveKeyWithValue("to", 1980.0),
		))
		Expect(ranges[1]).ToNot(HaveKey("from")) // open lower bound
		Expect(ranges[2]).ToNot(HaveKey("to"))   // open upper bound
	})

	DescribeTable("builds single-value metric aggregations",
		func(kind searchsvc.MetricKind, esKind string) {
			res := build([]*searchsvc.AggregationOption{
				{Field: "audio.duration", MetricKind: kind},
			})
			body, ok := res["a_0"].(map[string]any)[esKind].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(body["field"]).To(Equal("audio.duration"))
		},
		Entry("sum", searchsvc.MetricKind_METRIC_KIND_SUM, "sum"),
		Entry("min", searchsvc.MetricKind_METRIC_KIND_MIN, "min"),
		Entry("max", searchsvc.MetricKind_METRIC_KIND_MAX, "max"),
	)

	It("uses a stats aggregation for AVG", func() {
		res := build([]*searchsvc.AggregationOption{
			{Field: "audio.duration", MetricKind: searchsvc.MetricKind_METRIC_KIND_AVG},
		})
		stats, ok := res["a_0"].(map[string]any)["stats"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(stats["field"]).To(Equal("audio.duration"))
	})

	It("nests sub-aggregations under their parent bucket", func() {
		res := build([]*searchsvc.AggregationOption{{
			Field: "audio.artist", Size: 5,
			SubAggregations: []*searchsvc.AggregationOption{{
				Field: "audio.album", Size: 7,
				SubAggregations: []*searchsvc.AggregationOption{
					{Field: "audio.duration", MetricKind: searchsvc.MetricKind_METRIC_KIND_SUM},
				},
			}},
		}})
		album := res["a_0"].(map[string]any)["aggs"].(map[string]any)["a_0_0"].(map[string]any)
		albumTerms := album["terms"].(map[string]any)
		Expect(albumTerms["field"]).To(Equal("audio.album"))
		Expect(albumTerms["size"]).To(Equal(7))
		metric := album["aggs"].(map[string]any)["a_0_0_0"].(map[string]any)
		Expect(metric["sum"].(map[string]any)["field"]).To(Equal("audio.duration"))
	})
})

var _ = Describe("Parse", func() {
	It("parses flat term and range buckets, stringifying numeric keys", func() {
		raw := json.RawMessage(`{
			"a_0": {"buckets": [
				{"key": "Pink Floyd", "doc_count": 42},
				{"key": "Motörhead", "doc_count": 35}
			]},
			"a_1": {"buckets": [
				{"key": "1970-1980", "from": 1970.0, "to": 1980.0, "doc_count": 12}
			]},
			"a_2": {"buckets": [
				{"key": 9, "doc_count": 3}
			]}
		}`)
		out, err := aggs.Parse(raw, []*searchsvc.AggregationOption{
			{Field: "audio.artist"},
			{Field: "audio.year"},
			{Field: "audio.track"},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(HaveLen(3))

		Expect(out[0].Field).To(Equal("audio.artist"))
		Expect(out[0].Buckets).To(HaveLen(2))
		Expect(out[0].Buckets[0].Key).To(Equal("Pink Floyd"))
		Expect(out[0].Buckets[0].Count).To(Equal(int64(42)))

		Expect(out[1].Buckets[0].Key).To(Equal("1970-1980"))
		Expect(out[1].Buckets[0].Count).To(Equal(int64(12)))

		// numeric term key stringified without trailing zeros
		Expect(out[2].Buckets[0].Key).To(Equal("9"))
	})

	It("parses nested buckets carrying a metric", func() {
		raw := json.RawMessage(`{
			"a_0": {"buckets": [{
				"key": "Iron Maiden", "doc_count": 300,
				"a_0_0": {"buckets": [
					{"key": "The Number of the Beast", "doc_count": 8, "a_0_0_0": {"value": 2756000.0}},
					{"key": "Powerslave", "doc_count": 8, "a_0_0_0": {"value": 3061000.0}}
				]}
			}]}
		}`)
		out, err := aggs.Parse(raw, []*searchsvc.AggregationOption{{
			Field: "audio.artist",
			SubAggregations: []*searchsvc.AggregationOption{{
				Field: "audio.album",
				SubAggregations: []*searchsvc.AggregationOption{
					{Field: "audio.duration", MetricKind: searchsvc.MetricKind_METRIC_KIND_SUM},
				},
			}},
		}})
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(HaveLen(1))
		Expect(out[0].Field).To(Equal("audio.artist"))
		Expect(out[0].Buckets).To(HaveLen(1))

		artistBucket := out[0].Buckets[0]
		Expect(artistBucket.Key).To(Equal("Iron Maiden"))
		Expect(artistBucket.Count).To(Equal(int64(300)))
		Expect(artistBucket.SubAggregations).To(HaveLen(1))

		albumAgg := artistBucket.SubAggregations[0]
		Expect(albumAgg.Field).To(Equal("audio.album"))
		Expect(albumAgg.Buckets).To(HaveLen(2))

		nob := albumAgg.Buckets[0]
		Expect(nob.Key).To(Equal("The Number of the Beast"))
		Expect(nob.Count).To(Equal(int64(8)))
		Expect(nob.SubAggregations).To(HaveLen(1))

		metric := nob.SubAggregations[0]
		Expect(metric.Field).To(Equal("audio.duration"))
		Expect(metric.MetricKind).To(Equal(searchsvc.MetricKind_METRIC_KIND_SUM))
		Expect(metric.Value).To(Equal(2756000.0))
	})

	DescribeTable("parses single-value metrics",
		func(kind searchsvc.MetricKind, value float64) {
			raw := json.RawMessage(fmt.Sprintf(`{"a_0": {"value": %g}}`, value))
			out, err := aggs.Parse(raw, []*searchsvc.AggregationOption{
				{Field: "audio.duration", MetricKind: kind},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(HaveLen(1))
			Expect(out[0].MetricKind).To(Equal(kind))
			Expect(out[0].Value).To(Equal(value))
		},
		Entry("sum", searchsvc.MetricKind_METRIC_KIND_SUM, 1234.5),
		Entry("min", searchsvc.MetricKind_METRIC_KIND_MIN, 10.0),
		Entry("max", searchsvc.MetricKind_METRIC_KIND_MAX, 99.0),
	)

	It("decodes a null metric value to zero", func() {
		// OpenSearch returns value: null when a metric has no matching docs.
		out, err := aggs.Parse(json.RawMessage(`{"a_0": {"value": null}}`),
			[]*searchsvc.AggregationOption{{Field: "audio.duration", MetricKind: searchsvc.MetricKind_METRIC_KIND_SUM}})
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(HaveLen(1))
		Expect(out[0].Value).To(BeZero())
		Expect(out[0].MetricKind).To(Equal(searchsvc.MetricKind_METRIC_KIND_SUM))
	})

	It("carries avg transport (sum + count) from a stats response", func() {
		raw := json.RawMessage(`{
			"a_0": {"count": 100, "min": 30000.0, "max": 500000.0, "avg": 245000.0, "sum": 24500000.0}
		}`)
		out, err := aggs.Parse(raw, []*searchsvc.AggregationOption{
			{Field: "audio.duration", MetricKind: searchsvc.MetricKind_METRIC_KIND_AVG},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(HaveLen(1))
		Expect(out[0].MetricKind).To(Equal(searchsvc.MetricKind_METRIC_KIND_AVG))
		Expect(out[0].Sum).To(Equal(24500000.0))
		Expect(out[0].Count).To(Equal(int64(100)))
	})

	It("returns nil for empty raw or empty options", func() {
		got, err := aggs.Parse(nil, []*searchsvc.AggregationOption{{Field: "x"}})
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(BeNil())

		got, err = aggs.Parse(json.RawMessage(`{}`), nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(BeNil())
	})

	It("errors on malformed json and returns no result", func() {
		got, err := aggs.Parse(json.RawMessage(`not-json`), []*searchsvc.AggregationOption{{Field: "x"}})
		Expect(err).To(HaveOccurred())
		Expect(got).To(BeNil())
	})
})
