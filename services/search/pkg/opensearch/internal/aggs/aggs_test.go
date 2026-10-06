package aggs_test

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/aggs"
)

var _ = Describe("Aggregations", func() {
	build := func(opts ...*searchsvc.AggregationOption) map[string]any {
		body, err := aggs.Build(opts)
		Expect(err).ToNot(HaveOccurred())
		return body
	}

	parse := func(response string, opts ...*searchsvc.AggregationOption) []*searchsvc.AggregationResult {
		out, err := aggs.Parse(opts, json.RawMessage(response))
		Expect(err).ToNot(HaveOccurred())
		return out
	}

	keys := func(res *searchsvc.AggregationResult) []string {
		out := []string{}
		for _, b := range res.GetBuckets() {
			out = append(out, fmt.Sprintf("%s=%d", b.GetKey(), b.GetCount()))
		}
		return out
	}

	It("builds nothing without aggregations", func() {
		Expect(build()).To(BeNil())
	})

	It("asks for every bucket of a terms aggregation up to the limit, whatever its size", func() {
		terms := build(&searchsvc.AggregationOption{Field: "audio.artist", Size: 10})["a_0"].(map[string]any)["terms"].(map[string]any)
		Expect(terms["field"]).To(Equal("audio.artist"))
		Expect(terms["size"]).To(Equal(aggregation.MaxBuckets), "the size cuts after the cross-space merge")
	})

	It("reads the buckets of a terms aggregation", func() {
		out := parse(`{"a_0": {"sum_other_doc_count": 0, "buckets": [
			{"key": "Motörhead", "doc_count": 3},
			{"key": "Accept", "doc_count": 2}
		]}}`, &searchsvc.AggregationOption{Field: "audio.artist"})
		Expect(out).To(HaveLen(1))
		Expect(out[0].GetField()).To(Equal("audio.artist"))
		Expect(keys(out[0])).To(Equal([]string{"Motörhead=3", "Accept=2"}))
	})

	It("spells numeric and bool keys like the bleve backend", func() {
		out := parse(`{
			"a_0": {"buckets": [{"key": 9, "doc_count": 3}, {"key": 4.5, "doc_count": 1}]},
			"a_1": {"buckets": [{"key": 1, "key_as_string": "true", "doc_count": 2}, {"key": 0, "key_as_string": "false", "doc_count": 1}]}
		}`, &searchsvc.AggregationOption{Field: "audio.track"}, &searchsvc.AggregationOption{Field: "audio.hasDrm"})
		Expect(keys(out[0])).To(Equal([]string{"9=3", "4.5=1"}))
		Expect(keys(out[1])).To(Equal([]string{"true=2", "false=1"}))
	})

	It("leaves out the bucket of an empty value", func() {
		out := parse(`{"a_0": {"buckets": [
			{"key": "", "doc_count": 40},
			{"key": "Wheels of Steel", "doc_count": 2}
		]}}`, &searchsvc.AggregationOption{Field: "Title"})
		Expect(keys(out[0])).To(Equal([]string{"Wheels of Steel=2"}))
	})

	It("refuses a terms aggregation with more buckets than the limit instead of cutting it short", func() {
		_, err := aggs.Parse([]*searchsvc.AggregationOption{{Field: "audio.artist"}}, json.RawMessage(`{"a_0": {
			"sum_other_doc_count": 4465,
			"buckets": [{"key": "Accept", "doc_count": 2}]
		}}`))
		Expect(err).To(MatchError(aggregation.ErrTooManyBuckets))
	})

	It("answers an aggregation without buckets for an empty response block", func() {
		for _, response := range []string{``, `{}`} {
			out := parse(response, &searchsvc.AggregationOption{Field: "audio.artist"})
			Expect(out).To(HaveLen(1))
			Expect(out[0].GetBuckets()).To(BeEmpty())
		}
	})

	DescribeTable("builds a range or date_range aggregation by the kind of the bounds, open ones without",
		func(field, kind string, ranges []*searchsvc.BucketRange, want []map[string]any) {
			r := build(&searchsvc.AggregationOption{Field: field, BucketDefinition: &searchsvc.BucketDefinition{Ranges: ranges}})["a_0"].(map[string]any)[kind].(map[string]any)
			Expect(r["field"]).To(Equal(field))
			Expect(r["ranges"]).To(Equal(want))
		},
		Entry("numbers", "audio.year", "range",
			[]*searchsvc.BucketRange{{From: "1970", To: "1980"}, {To: "1970"}, {From: "2020"}},
			[]map[string]any{{"key": "1970..1980", "from": 1970.0, "to": 1980.0}, {"key": "..1970", "to": 1970.0}, {"key": "2020..", "from": 2020.0}}),
		Entry("dates", "photo.takenDateTime", "date_range",
			[]*searchsvc.BucketRange{{From: "2018-08-01T00:00:00Z", To: "2018-09-01T00:00:00Z"}, {From: "2018-08-11T00:00:00Z"}},
			[]map[string]any{{"key": "2018-08-01T00:00:00Z..2018-09-01T00:00:00Z", "from": "2018-08-01T00:00:00Z", "to": "2018-09-01T00:00:00Z"}, {"key": "2018-08-11T00:00:00Z..", "from": "2018-08-11T00:00:00Z"}}),
	)

	malformed := &searchsvc.AggregationOption{
		Field:            "photo.takenDateTime",
		BucketDefinition: &searchsvc.BucketDefinition{Ranges: []*searchsvc.BucketRange{{From: "2018-08-11T00:00:00Z", To: "not-a-date"}}},
	}

	It("rejects a malformed range", func() {
		_, err := aggs.Build([]*searchsvc.AggregationOption{malformed})
		Expect(err).To(HaveOccurred())
	})

	It("lists every requested range in request order, empty ones included", func() {
		out := parse(`{"a_0": {"buckets": [
			{"key": "1970..1980", "doc_count": 4},
			{"key": "1980..1990", "doc_count": 0}
		]}}`, &searchsvc.AggregationOption{
			Field: "audio.year",
			BucketDefinition: &searchsvc.BucketDefinition{
				Ranges: []*searchsvc.BucketRange{{From: "1980", To: "1990"}, {From: "1970", To: "1980"}, {From: "1990"}},
			},
		})
		Expect(keys(out[0])).To(Equal([]string{"1980..1990=0", "1970..1980=4", "1990..=0"}))
	})

	DescribeTable("asks for all accumulators of a metric, whatever its kind",
		func(kind searchsvc.MetricKind) {
			opt := &searchsvc.AggregationOption{Field: "audio.duration", MetricDefinition: &searchsvc.MetricDefinition{Kind: kind}}
			stats := build(opt)["a_0"].(map[string]any)["stats"].(map[string]any)
			Expect(stats["field"]).To(Equal("audio.duration"))

			m := parse(`{"a_0": {"count": 100, "min": 30000.0, "max": 500000.0, "avg": 245000.0, "sum": 24500000.0}}`, opt)[0].GetMetric()
			Expect(m.GetKind()).To(Equal(kind))
			Expect(m.GetCount()).To(Equal(int64(100)))
			Expect(m.GetSum()).To(Equal(24500000.0))
			Expect(m.GetMin()).To(Equal(30000.0))
			Expect(m.GetMax()).To(Equal(500000.0))
			Expect(m.Value).To(BeNil(), "the service layer reduces to the value after the merge")
		},
		Entry("sum", searchsvc.MetricKind_METRIC_KIND_SUM),
		Entry("min", searchsvc.MetricKind_METRIC_KIND_MIN),
		Entry("max", searchsvc.MetricKind_METRIC_KIND_MAX),
		Entry("avg", searchsvc.MetricKind_METRIC_KIND_AVG),
	)

	It("leaves a metric without a single value empty instead of reading null as zero", func() {
		m := parse(`{"a_0": {"count": 0, "min": null, "max": null, "avg": null, "sum": 0.0}}`,
			&searchsvc.AggregationOption{Field: "audio.duration", MetricDefinition: &searchsvc.MetricDefinition{Kind: searchsvc.MetricKind_METRIC_KIND_MIN}})[0].GetMetric()
		Expect(m.GetKind()).To(Equal(searchsvc.MetricKind_METRIC_KIND_MIN))
		Expect(m.GetCount()).To(BeZero())
	})

	It("answers one result per aggregation, in request order", func() {
		out := parse(`{}`,
			&searchsvc.AggregationOption{Field: "audio.year", MetricDefinition: &searchsvc.MetricDefinition{Kind: searchsvc.MetricKind_METRIC_KIND_SUM}},
			&searchsvc.AggregationOption{Field: "audio.artist"},
			&searchsvc.AggregationOption{Field: "audio.year", BucketDefinition: &searchsvc.BucketDefinition{Ranges: []*searchsvc.BucketRange{{To: "1980"}}}},
			&searchsvc.AggregationOption{Field: "audio.year", MetricDefinition: &searchsvc.MetricDefinition{Kind: searchsvc.MetricKind_METRIC_KIND_MAX}},
		)
		Expect(out).To(HaveLen(4))
		Expect(out[0].GetMetric().GetKind()).To(Equal(searchsvc.MetricKind_METRIC_KIND_SUM))
		Expect(out[1].GetMetric()).To(BeNil())
		Expect(out[1].GetBuckets()).To(BeEmpty())
		Expect(keys(out[2])).To(Equal([]string{"..1980=0"}))
		Expect(out[3].GetMetric().GetKind()).To(Equal(searchsvc.MetricKind_METRIC_KIND_MAX))
	})

	DescribeTable("reports a result it cannot decode instead of dropping it",
		func(opt *searchsvc.AggregationOption, response string) {
			_, err := aggs.Parse([]*searchsvc.AggregationOption{opt}, json.RawMessage(response))
			Expect(err).To(HaveOccurred())
		},
		Entry("malformed json", &searchsvc.AggregationOption{Field: "audio.artist"}, `{`),
		Entry("a terms aggregation", &searchsvc.AggregationOption{Field: "audio.artist"}, `{"a_0": {"buckets": "none"}}`),
		Entry("a bucket", &searchsvc.AggregationOption{Field: "audio.artist"}, `{"a_0": {"buckets": [{"key": "Saxon", "doc_count": "many"}]}}`),
		Entry("a metric",
			&searchsvc.AggregationOption{Field: "audio.year", MetricDefinition: &searchsvc.MetricDefinition{Kind: searchsvc.MetricKind_METRIC_KIND_SUM}},
			`{"a_0": {"count": "many"}}`),
	)
})
