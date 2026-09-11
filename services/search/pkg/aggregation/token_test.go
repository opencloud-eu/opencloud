package aggregation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
)

var _ = Describe("Token", func() {
	Describe("EncodeTermsToken", func() {
		It("encodes the key as quoted ǂǂ-prefixed lowercase hex", func() {
			Expect(aggregation.EncodeTermsToken("And the Bands Played On")).To(Equal(`"ǂǂ416e64207468652042616e647320506c61796564204f6e"`))
		})
	})

	Describe("EncodeRangeToken", func() {
		DescribeTable("bounds",
			func(from, to, want string) {
				Expect(aggregation.EncodeRangeToken(from, to)).To(Equal(want))
			},
			Entry("closed", "0", "100", "range(0,100)"),
			Entry("open lower", "", "100", "range(min,100)"),
			Entry("open upper", "0", "", "range(0,max)"),
		)
	})

	Describe("DecodeAggregationFilter", func() {
		DescribeTable("valid tokens",
			func(filter, want string) {
				got, err := aggregation.DecodeAggregationFilter(filter)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(want))
			},
			Entry("terms with a space", `audio.artist:"ǂǂ5361786f6e"`, `audio.artist:"Saxon"`),
			Entry("closed range", "Size:range(0,100)", "(Size>=0 AND Size<=100)"),
			Entry("open lower range", "Size:range(min,100)", "(Size<=100)"),
			Entry("open upper range", "Size:range(0,max)", "(Size>=0)"),
			Entry("or of two terms",
				`audio.artist:or("ǂǂ5361786f6e","ǂǂ49726f6e204d616964656e")`,
				`(audio.artist:"Saxon" OR audio.artist:"Iron Maiden")`),
		)

		DescribeTable("rejected tokens",
			func(filter string) {
				_, err := aggregation.DecodeAggregationFilter(filter)
				Expect(err).To(HaveOccurred())
			},
			Entry("no colon", `audio.artist"ǂǂ00"`),
			Entry("empty field", `:"ǂǂ00"`),
			Entry("missing ǂǂ prefix", `audio.artist:"deadbeef"`),
			Entry("odd hex", `audio.artist:"ǂǂabc"`),
			Entry("range without bounds", "Size:range(min,max)"),
		)

		It("round-trips a terms key through encode+decode", func() {
			got, err := aggregation.DecodeAggregationFilter("audio.artist:" + aggregation.EncodeTermsToken("AC/DC"))
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(`audio.artist:"AC/DC"`))
		})

		It("rejects a decoded value containing a double quote", func() {
			// 22 is a double quote; it cannot be expressed in a KQL string.
			_, err := aggregation.DecodeAggregationFilter(`audio.artist:"ǂǂ22"`)
			Expect(err).To(HaveOccurred())
		})
	})
})
