package aggs_test

import (
	"encoding/json"

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
})
