package svc

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/generator"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

var _ = Describe("checkTypeMapping", func() {
	It("accepts the defaults", func() {
		Expect(checkTypeMapping(thumbnail.NewTypeMapping(nil), generator.NewFormats(nil))).To(Succeed())
	})

	It("accepts a pair that was configured together", func() {
		Expect(checkTypeMapping(
			thumbnail.NewTypeMapping([]string{"image/png:image/webp"}),
			generator.NewFormats([]string{"image/webp:webp"}),
		)).To(Succeed())
	})

	It("refuses a target the generator has no format for", func() {
		err := checkTypeMapping(
			thumbnail.NewTypeMapping([]string{"image/png:image/webp"}),
			generator.NewFormats(nil),
		)

		Expect(err).To(MatchError(ContainSubstring("no generator format for image/webp (mapped from image/png)")))
	})

	It("names every bad entry", func() {
		err := checkTypeMapping(
			thumbnail.NewTypeMapping([]string{"image/png:image/avif", "image/gif:image/webp"}),
			generator.NewFormats(nil),
		)

		Expect(err).To(MatchError(ContainSubstring("image/avif (mapped from image/png)")))
		Expect(err).To(MatchError(ContainSubstring("image/webp (mapped from image/gif)")))
	})

	It("takes an empty target as no override", func() {
		Expect(checkTypeMapping(
			thumbnail.NewTypeMapping([]string{"image/png:"}),
			generator.NewFormats(nil),
		)).To(Succeed())
	})
})
