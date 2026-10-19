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
		p := getPreview("audio/mpeg", []map[string][]string{audio, cover})
		Expect(p).ToNot(BeNil())
		Expect(*p).To(Equal(Preview{Width: 500, Height: 400, MimeType: "image/jpeg"}))
	})

	It("stores the media type without the parameters Tika sends", func() {
		p := getPreview("audio/mpeg", []map[string][]string{audio, coverWithParams})
		Expect(p).ToNot(BeNil())
		Expect(p.MimeType).To(Equal("image/jpeg"))
	})

	It("returns nil when the audio has no cover", func() {
		Expect(getPreview("audio/mpeg", []map[string][]string{audio})).To(BeNil())
	})

	It("returns nil when the marked cover lacks dimensions", func() {
		Expect(getPreview("audio/mpeg", []map[string][]string{audio, coverNoDims})).To(BeNil())
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
		p := getPreview("audio/mpeg", []map[string][]string{audio, inline, marked})
		Expect(p).ToNot(BeNil())
		Expect(*p).To(Equal(Preview{Width: 64, Height: 40, MimeType: "image/png"}), "the type decides what a thumbnail of the cover is written in")
	})

	It("ignores an embedded image Tika does not mark", func() {
		unmarked := map[string][]string{
			"Content-Type":    {"image/jpeg"},
			"tiff:ImageWidth": {"90"}, "tiff:ImageLength": {"90"},
		}
		Expect(getPreview("audio/mpeg", []map[string][]string{audio, unmarked})).To(BeNil())
	})

	It("is gated to embedded-preview types", func() {
		// an image is unconditional; its preview is not driven by oc.preview.
		Expect(getPreview("image/png", []map[string][]string{cover})).To(BeNil())
	})
})
