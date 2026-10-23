package thumbnail_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

var _ = Describe("SourceTypes", func() {
	defaults := thumbnail.NewSourceTypes(nil)
	withHeic := thumbnail.NewSourceTypes([]string{"image/heic"})

	DescribeTable("GeneratorReads",
		func(types thumbnail.SourceTypes, mimeType string, want bool) {
			Expect(types.GeneratorReads(mimeType)).To(Equal(want))
		},
		Entry("an image both backends read", defaults, "image/jpeg", true),
		Entry("with parameters", defaults, "image/png; charset=binary", true),
		Entry("not a camera raw, which is an image nobody renders", defaults, "image/x-nikon-nef", false),
		Entry("not a document, whatever the list says", defaults, "application/pdf", false),
		Entry("not audio, which is announced but extracted", defaults, "audio/mpeg", false),
		Entry("an image type the deployment added", withHeic, "image/heic", true),
		Entry("and the built-in ones are still in there", withHeic, "image/tiff", true),
		Entry("an unparsable type", defaults, "not a mime", false),
	)

	It("falls back to the built-in image types when nothing was configured", func() {
		Expect(thumbnail.SourceTypes{}.GeneratorReads("image/jpeg")).To(BeTrue())
		Expect(thumbnail.SourceTypes{}.GeneratorReads("image/heic")).To(BeFalse())
	})
})
