package aggregation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
)

var _ = Describe("ParseFilter", func() {
	parse := func(f *searchsvc.AggregationFilter) (aggregation.Buckets, error) {
		return aggregation.ParseFilter(f, fieldTypes(f.GetField()))
	}

	It("reads the key of a keyword bucket as it is", func() {
		got, err := parse(&searchsvc.AggregationFilter{Field: "audio.artist", Terms: []string{`Wh*t? "Live"`, "true", "1980"}})
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Terms).To(Equal([]aggregation.Term{{Key: `Wh*t? "Live"`}, {Key: "true"}, {Key: "1980"}}))
	})

	It("reads the key of a numeric bucket as a number", func() {
		got, err := parse(&searchsvc.AggregationFilter{Field: "audio.year", Terms: []string{"1980", "4.5"}})
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Terms).To(HaveLen(2))
		Expect(got.Terms[0].Number).To(HaveValue(Equal(1980.0)))
		Expect(got.Terms[1].Number).To(HaveValue(Equal(4.5)))
	})

	It("reads the key of a bool bucket as a bool", func() {
		got, err := parse(&searchsvc.AggregationFilter{Field: "audio.hasDrm", Terms: []string{"true", "false"}})
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Terms).To(HaveLen(2))
		Expect(got.Terms[0].Bool).To(HaveValue(BeTrue()))
		Expect(got.Terms[1].Bool).To(HaveValue(BeFalse()))
	})

	It("reads the ranges of a filter", func() {
		got, err := parse(&searchsvc.AggregationFilter{Field: "audio.year", Ranges: []*searchsvc.BucketRange{{To: "1980"}, {From: "2010"}}})
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Terms).To(BeEmpty())
		Expect(got.Ranges.Numeric).To(HaveLen(2))
	})

	DescribeTable("rejects what no aggregation on the field could have answered",
		func(f *searchsvc.AggregationFilter) {
			_, err := parse(f)
			Expect(err).To(HaveOccurred())
		},
		Entry("an unknown field", &searchsvc.AggregationFilter{Field: "audio.nonexistent", Terms: []string{"Saxon"}}),
		Entry("no bucket at all", &searchsvc.AggregationFilter{Field: "audio.artist"}),
		Entry("a key that is no number on a numeric field", &searchsvc.AggregationFilter{Field: "audio.year", Terms: []string{"nineteen-eighty"}}),
		Entry("an empty key on a numeric field", &searchsvc.AggregationFilter{Field: "audio.year", Terms: []string{""}}),
		Entry("a number spelled otherwise than its bucket key", &searchsvc.AggregationFilter{Field: "audio.year", Terms: []string{"1980.0"}}),
		Entry("a number in scientific notation", &searchsvc.AggregationFilter{Field: "audio.year", Terms: []string{"1e3"}}),
		Entry("a number with a plus sign", &searchsvc.AggregationFilter{Field: "audio.year", Terms: []string{"+1980"}}),
		Entry("the bleve spelling of a bool", &searchsvc.AggregationFilter{Field: "audio.hasDrm", Terms: []string{"T"}}),
		Entry("another spelling of a bool", &searchsvc.AggregationFilter{Field: "audio.hasDrm", Terms: []string{"1"}}),
		Entry("a key on a date field", &searchsvc.AggregationFilter{Field: "photo.takenDateTime", Terms: []string{"2018-08-11T00:00:00Z"}}),
		Entry("a key on full text", &searchsvc.AggregationFilter{Field: "Content", Terms: []string{"fox"}}),
		Entry("a numeric range on a keyword", &searchsvc.AggregationFilter{Field: "audio.artist", Ranges: []*searchsvc.BucketRange{{From: "1980"}}}),
		Entry("a date range on a number", &searchsvc.AggregationFilter{Field: "audio.year", Ranges: []*searchsvc.BucketRange{{From: "2018-08-11T00:00:00Z"}}}),
	)
})
