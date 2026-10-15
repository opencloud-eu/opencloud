package thumbnail_test

import (
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail"
)

func resourceInfo(mime string, meta map[string]string) *provider.ResourceInfo {
	ri := &provider.ResourceInfo{MimeType: mime}
	if meta != nil {
		ri.ArbitraryMetadata = &provider.ArbitraryMetadata{Metadata: meta}
	}
	return ri
}

var _ = Describe("HasPreview", func() {
	DescribeTable("preview availability",
		func(md *provider.ResourceInfo, want bool) {
			Expect(thumbnail.HasPreview(md)).To(Equal(want))
		},
		Entry("nil", nil, false),
		Entry("unconditional image", resourceInfo("image/png", nil), true),
		Entry("unconditional text", resourceInfo("text/plain", nil), true),
		Entry("unsupported type", resourceInfo("application/pdf", nil), false),
		Entry("audio without preview dims", resourceInfo("audio/mpeg", nil), false),
		Entry("audio with zero dims", resourceInfo("audio/mpeg", map[string]string{
			thumbnail.PreviewWidthKey: "0", thumbnail.PreviewHeightKey: "0",
		}), false),
		Entry("audio with preview dims", resourceInfo("audio/mpeg", map[string]string{
			thumbnail.PreviewWidthKey: "500", thumbnail.PreviewHeightKey: "500",
		}), true),
	)

	It("SupportedMimeTypes is the union of both preview sets", func() {
		for k := range thumbnail.UnconditionalPreviewMimeTypes {
			Expect(thumbnail.SupportedMimeTypes).To(HaveKey(k))
		}
		for k := range thumbnail.EmbeddedPreviewMimeTypes {
			Expect(thumbnail.SupportedMimeTypes).To(HaveKey(k))
		}
	})
})
