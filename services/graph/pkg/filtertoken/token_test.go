package filtertoken_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/filtertoken"
)

var _ = Describe("Token", func() {
	Describe("EncodeTermsToken", func() {
		It("encodes the key as quoted ǂǂ-prefixed lowercase hex", func() {
			Expect(filtertoken.EncodeTermsToken("And the Bands Played On")).To(Equal(`"ǂǂ416e64207468652042616e647320506c61796564204f6e"`))
		})
	})

	Describe("EncodeRangeToken", func() {
		DescribeTable("bounds, and the token decodes back to them",
			func(from, to, want string) {
				Expect(filtertoken.EncodeRangeToken(from, to)).To(Equal(want))
				got, err := filtertoken.DecodeFilter("Size:" + want)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(filtertoken.Filter{Field: "Size", Ranges: []filtertoken.Range{{From: from, To: to}}}))
			},
			Entry("closed", "0", "100", "range(0, 100)"),
			Entry("open lower", "", "100", "range(min, 100)"),
			Entry("open upper", "0", "", `range(0, max, to="le")`),
		)
	})

	Describe("DecodeFilter", func() {
		DescribeTable("valid tokens",
			func(filter string, want filtertoken.Filter) {
				got, err := filtertoken.DecodeFilter(filter)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(want))
			},
			Entry("terms", `audio.artist:"ǂǂ5361786f6e"`,
				filtertoken.Filter{Field: "audio.artist", Terms: []string{"Saxon"}}),
			Entry("closed range", "Size:range(0,100)",
				filtertoken.Filter{Field: "Size", Ranges: []filtertoken.Range{{From: "0", To: "100"}}}),
			Entry("open lower range", "Size:range(min,100)",
				filtertoken.Filter{Field: "Size", Ranges: []filtertoken.Range{{To: "100"}}}),
			Entry("open upper range", "Size:range(0,max)",
				filtertoken.Filter{Field: "Size", Ranges: []filtertoken.Range{{From: "0"}}}),
			Entry("open upper range with le marker", `Size:range(0, max, to="le")`,
				filtertoken.Filter{Field: "Size", Ranges: []filtertoken.Range{{From: "0"}}}),
			Entry("date range, the colons of the bounds are no field separator",
				"photo.takenDateTime:range(2018-08-11T00:00:00Z, 2018-08-12T00:00:00Z)",
				filtertoken.Filter{Field: "photo.takenDateTime", Ranges: []filtertoken.Range{{From: "2018-08-11T00:00:00Z", To: "2018-08-12T00:00:00Z"}}}),
			Entry("or of two terms with spaces",
				`audio.artist:or("ǂǂ5361786f6e", "ǂǂ49726f6e204d616964656e")`,
				filtertoken.Filter{Field: "audio.artist", Terms: []string{"Saxon", "Iron Maiden"}}),
			Entry("or of two ranges, the commas of a range do not split it",
				`audio.year:or(range(min, 1980),range(2010, max, to="le"))`,
				filtertoken.Filter{Field: "audio.year", Ranges: []filtertoken.Range{{To: "1980"}, {From: "2010"}}}),
		)

		DescribeTable("rejected tokens",
			func(filter string) {
				_, err := filtertoken.DecodeFilter(filter)
				Expect(err).To(HaveOccurred())
			},
			Entry("no colon", `audio.artist"ǂǂ00"`),
			Entry("empty field", `:"ǂǂ00"`),
			Entry("missing ǂǂ prefix", `audio.artist:"deadbeef"`),
			Entry("missing quotes", `audio.artist:ǂǂ5361786f6e`),
			Entry("odd hex", `audio.artist:"ǂǂabc"`),
			Entry("uppercase hex, the server issues lowercase", `audio.artist:"ǂǂ5361786F6E"`),
			Entry("empty key", `audio.artist:"ǂǂ"`),
			Entry("hex that is no UTF-8", `audio.artist:"ǂǂff"`),
			Entry("terms and ranges in one or", `audio.year:or("ǂǂ31393830", range(1990, max, to="le"))`),
			Entry("range without bounds", "Size:range(min,max)"),
			Entry("range with one argument", "Size:range(100)"),
			Entry("range with an empty bound", "Size:range(,100)"),
			Entry("le marker on a closed range", `Size:range(0, 100, to="le")`),
			Entry("empty or", "audio.artist:or()"),
			Entry("nested or", `audio.artist:or(or("ǂǂ5361786f6e"))`),
			Entry("plain value", "audio.artist:Saxon"),
		)

		DescribeTable("round-trips every terms key, a term is data and no query syntax",
			func(key string) {
				got, err := filtertoken.DecodeFilter("audio.artist:" + filtertoken.EncodeTermsToken(key))
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(filtertoken.Filter{Field: "audio.artist", Terms: []string{key}}))
			},
			Entry("slash", "AC/DC"),
			Entry("double quote", `The "Chirping" Crickets`),
			Entry("wildcards", "Wh*t?"),
			Entry("comma and parentheses", "Emerson, Lake (and Palmer)"),
			Entry("colon", "Sunn O))): live"),
			Entry("umlaut", "Motörhead"),
		)

		It("returns range bounds as written, they never become query syntax", func() {
			got, err := filtertoken.DecodeFilter("audio.year:range(1980 OR Name:*, 1990)")
			Expect(err).ToNot(HaveOccurred())
			Expect(got.Ranges).To(Equal([]filtertoken.Range{{From: "1980 OR Name:*", To: "1990"}}))
		})
	})
})
