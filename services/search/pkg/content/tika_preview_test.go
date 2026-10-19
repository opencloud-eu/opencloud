package content

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("getPreview", func() {
	audio := map[string][]string{"Content-Type": {"audio/mpeg"}}
	cover := map[string][]string{
		"Content-Type":              {"image/jpeg"},
		"tiff:ImageWidth":           {"500"},
		"tiff:ImageLength":          {"400"},
		"tk:embedded-resource-type": {"THUMBNAIL"},
	}
	coverNoDims := map[string][]string{
		"Content-Type":              {"image/jpeg"},
		"tk:embedded-resource-type": {"THUMBNAIL"},
	}
	coverWithParams := map[string][]string{
		"Content-Type":              {"image/jpeg; quality=90"},
		"tiff:ImageWidth":           {"80"},
		"tiff:ImageLength":          {"60"},
		"tk:embedded-resource-type": {"THUMBNAIL"},
	}

	It("returns the cover Tika marks as the thumbnail", func() {
		p := getPreview([]map[string][]string{audio, cover})
		Expect(p).ToNot(BeNil())
		Expect(*p).To(Equal(Preview{Width: 500, Height: 400, MimeType: "image/jpeg"}))
	})

	It("stores the media type without the parameters Tika sends", func() {
		p := getPreview([]map[string][]string{audio, coverWithParams})
		Expect(p).ToNot(BeNil())
		Expect(p.MimeType).To(Equal("image/jpeg"))
	})

	It("returns nil when the audio has no cover", func() {
		Expect(getPreview([]map[string][]string{audio})).To(BeNil())
	})

	It("returns nil when the marked cover lacks dimensions", func() {
		Expect(getPreview([]map[string][]string{audio, coverNoDims})).To(BeNil())
	})

	It("leaves the selection to Tika, whatever the order", func() {
		inline := map[string][]string{
			"Content-Type": {"image/jpeg"}, "dc:description": {"Other"},
			"tiff:ImageWidth": {"30"}, "tiff:ImageLength": {"30"},
			"tk:embedded-resource-type": {"INLINE"},
		}
		marked := map[string][]string{
			"Content-Type": {"image/png"}, "dc:description": {"Cover (front)"},
			"tiff:ImageWidth": {"64"}, "tiff:ImageLength": {"40"},
			"tk:embedded-resource-type": {"THUMBNAIL"},
		}
		p := getPreview([]map[string][]string{audio, inline, marked})
		Expect(p).ToNot(BeNil())
		Expect(*p).To(Equal(Preview{Width: 64, Height: 40, MimeType: "image/png"}), "the type decides what a thumbnail of the cover is written in")
	})

	It("ignores an embedded image Tika does not mark", func() {
		unmarked := map[string][]string{
			"Content-Type":    {"image/jpeg"},
			"tiff:ImageWidth": {"90"}, "tiff:ImageLength": {"90"},
		}
		Expect(getPreview([]map[string][]string{audio, unmarked})).To(BeNil())
	})

	It("takes a document's stored thumbnail the same way", func() {
		odt := map[string][]string{"Content-Type": {"application/vnd.oasis.opendocument.text"}}
		stored := map[string][]string{
			"Content-Type":    {"image/png"},
			"tiff:ImageWidth": {"362"}, "tiff:ImageLength": {"512"},
			"tk:embedded-resource-type": {"THUMBNAIL"},
		}
		p := getPreview([]map[string][]string{odt, stored})
		Expect(p).ToNot(BeNil())
		Expect(*p).To(Equal(Preview{Width: 362, Height: 512, MimeType: "image/png"}))
	})

	It("returns nil for a file Tika marks nothing in, an image for one", func() {
		Expect(getPreview([]map[string][]string{{"Content-Type": {"image/png"}, "tiff:ImageWidth": {"10"}, "tiff:ImageLength": {"10"}}})).To(BeNil())
	})
})
