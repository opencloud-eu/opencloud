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

})

var _ = Describe("Parse", func() {
	It("parses term buckets, stringifying numeric keys", func() {
		raw := json.RawMessage(`{
			"a_0": {"buckets": [
				{"key": "Pink Floyd", "doc_count": 42},
				{"key": "Motörhead", "doc_count": 35}
			]},
			"a_1": {"buckets": [
				{"key": 9, "doc_count": 3}
			]}
		}`)
		out, err := aggs.Parse(raw, []*searchsvc.AggregationOption{
			{Field: "audio.artist"},
			{Field: "audio.track"},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(HaveLen(2))

		Expect(out[0].Field).To(Equal("audio.artist"))
		Expect(out[0].Buckets).To(HaveLen(2))
		Expect(out[0].Buckets[0].Key).To(Equal("Pink Floyd"))
		Expect(out[0].Buckets[0].Count).To(Equal(int64(42)))

		// numeric term key stringified without trailing zeros
		Expect(out[1].Buckets[0].Key).To(Equal("9"))
	})
})
