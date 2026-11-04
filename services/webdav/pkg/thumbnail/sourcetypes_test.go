package thumbnail_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

var _ = Describe("SourceTypes", func() {
	defaults := thumbnail.NewSourceTypes(nil)
	withHeic := thumbnail.NewSourceTypes([]string{"image/heic"})

	DescribeTable("Supports",
		func(types thumbnail.SourceTypes, mimeType string, want bool) {
			Expect(types.Supports(mimeType)).To(Equal(want))
		},
		Entry("an image both backends read", defaults, "image/jpeg", true),
		Entry("with parameters", defaults, "image/png; charset=binary", true),
		Entry("built-in audio", defaults, "audio/mpeg", true),
		Entry("not a camera raw by default", defaults, "image/x-nikon-nef", false),
		Entry("an image type the deployment added", withHeic, "image/heic", true),
		Entry("and the built-in ones are still in there", withHeic, "image/tiff", true),
		Entry("an unparsable type", defaults, "not a mime", false),
	)
})
