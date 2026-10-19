package thumbnail_test

import (
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

func resource(mimeType string, meta map[string]string) *provider.ResourceInfo {
	return &provider.ResourceInfo{
		MimeType:          mimeType,
		ArbitraryMetadata: &provider.ArbitraryMetadata{Metadata: meta},
	}
}

var _ = Describe("HasPreview", func() {
	types := thumbnail.NewSourceTypes(nil)

	DescribeTable("answers from the type, or from what the index saw",
		func(md *provider.ResourceInfo, want bool) {
			Expect(types.HasPreview(md)).To(Equal(want))
		},
		Entry("an image always has one", resource("image/png", nil), true),
		Entry("a type nobody makes a preview of has none", resource("application/zip", nil), false),
		Entry("audio without indexed dimensions has none", resource("audio/mpeg", nil), false),
		Entry("audio the index saw, with its type, has one",
			resource("audio/mpeg", map[string]string{
				thumbnail.PreviewWidthKey:       "600",
				thumbnail.PreviewHeightKey:      "600",
				thumbnail.PreviewContentTypeKey: "image/png",
			}), true),
		Entry("audio the index measured but could not type has none",
			resource("audio/mpeg", map[string]string{
				thumbnail.PreviewWidthKey:  "600",
				thumbnail.PreviewHeightKey: "600",
			}), false),
		Entry("audio with zero dimensions has none",
			resource("audio/mpeg", map[string]string{
				thumbnail.PreviewWidthKey:  "0",
				thumbnail.PreviewHeightKey: "0",
			}), false),
		Entry("no resource at all", nil, false),
	)

	It("answers the same for a caller that only has the mimetype", func() {
		Expect(types.HasPreviewForMimeType("audio/mpeg", true)).To(BeTrue())
		Expect(types.HasPreviewForMimeType("audio/mpeg", false)).To(BeFalse())
		Expect(types.HasPreviewForMimeType("image/png", false)).To(BeTrue())
	})
})
