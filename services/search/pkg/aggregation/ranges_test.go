package aggregation_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
)

var _ = Describe("ParseRanges", func() {
	It("parses numeric ranges, an open side has no bound", func() {
		got, err := aggregation.ParseRanges("audio.year", []*searchsvc.BucketRange{{To: "1980"}, {From: "1980", To: "1990.5"}, {From: "-3"}})
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Dates).To(BeEmpty())
		Expect(got.Numeric).To(HaveLen(3))
		Expect(got.Numeric[0].Key).To(Equal("..1980"))
		Expect(got.Numeric[0].From).To(BeNil())
		Expect(*got.Numeric[0].To).To(Equal(1980.0))
		Expect(got.Numeric[1].Key).To(Equal("1980..1990.5"))
		Expect(*got.Numeric[1].To).To(Equal(1990.5))
		Expect(got.Numeric[2].Key).To(Equal("-3.."))
		Expect(*got.Numeric[2].From).To(Equal(-3.0))
		Expect(got.Numeric[2].To).To(BeNil())
	})

	It("parses RFC3339 date ranges", func() {
		got, err := aggregation.ParseRanges("photo.takenDateTime", []*searchsvc.BucketRange{{From: "2018-08-11T00:00:00Z", To: "2018-08-12T00:00:00+02:00"}, {From: "2019-01-01T00:00:00Z"}})
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Numeric).To(BeEmpty())
		Expect(got.Dates).To(HaveLen(2))
		Expect(got.Dates[0].From).To(BeTemporally("==", time.Date(2018, 8, 11, 0, 0, 0, 0, time.UTC)))
		Expect(got.Dates[0].To).To(BeTemporally("==", time.Date(2018, 8, 11, 22, 0, 0, 0, time.UTC)))
		Expect(got.Dates[1].To.IsZero()).To(BeTrue())
	})

	DescribeTable("rejects",
		func(ranges ...*searchsvc.BucketRange) {
			_, err := aggregation.ParseRanges("field", ranges)
			Expect(err).To(HaveOccurred())
		},
		Entry("a range without a bound", &searchsvc.BucketRange{}),
		Entry("a bound that is no number", &searchsvc.BucketRange{From: "1970", To: "198o"}),
		Entry("a bound that is no date", &searchsvc.BucketRange{From: "2018-08-11T00:00:00Z", To: "not-a-date"}),
		Entry("a date without a time", &searchsvc.BucketRange{From: "2018-08-11"}),
		Entry("a number and a date in one range", &searchsvc.BucketRange{From: "1970", To: "2018-08-11T00:00:00Z"}),
		Entry("numeric and date ranges mixed", &searchsvc.BucketRange{From: "1970"}, &searchsvc.BucketRange{From: "2018-08-11T00:00:00Z"}),
		Entry("the same range twice", &searchsvc.BucketRange{From: "1970"}, &searchsvc.BucketRange{From: "1970"}),
		Entry("NaN", &searchsvc.BucketRange{From: "NaN"}),
		Entry("infinity", &searchsvc.BucketRange{To: "Inf"}),
		Entry("query syntax", &searchsvc.BucketRange{From: "1980 OR Name:*", To: "1990"}),
	)
})
