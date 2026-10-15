package content

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("getPreview", func() {
	audio := map[string][]string{"Content-Type": {"audio/mpeg"}}
	cover := map[string][]string{
		"Content-Type":     {"image/jpeg"},
		"tiff:ImageWidth":  {"500"},
		"tiff:ImageLength": {"400"},
	}
	coverNoDims := map[string][]string{"Content-Type": {"image/jpeg"}}

	It("returns the embedded cover dimensions for audio", func() {
		p := getPreview("audio/mpeg", []map[string][]string{audio, cover})
		Expect(p).ToNot(BeNil())
		Expect(*p).To(Equal(Preview{Width: 500, Height: 400}))
	})

	It("returns nil when the audio has no cover", func() {
		Expect(getPreview("audio/mpeg", []map[string][]string{audio})).To(BeNil())
	})

	It("returns nil when the cover lacks dimensions", func() {
		Expect(getPreview("audio/mpeg", []map[string][]string{audio, coverNoDims})).To(BeNil())
	})

	It("prefers the front cover over an earlier back cover", func() {
		back := map[string][]string{
			"Content-Type": {"image/jpeg"}, "dc:description": {"Cover (back)"},
			"tiff:ImageWidth": {"30"}, "tiff:ImageLength": {"30"},
		}
		front := map[string][]string{
			"Content-Type": {"image/jpeg"}, "dc:description": {"Cover (front)"},
			"tiff:ImageWidth": {"64"}, "tiff:ImageLength": {"40"},
		}
		p := getPreview("audio/mpeg", []map[string][]string{audio, back, front})
		Expect(p).ToNot(BeNil())
		Expect(*p).To(Equal(Preview{Width: 64, Height: 40}))
	})

	It("is gated to embedded-preview types", func() {
		// an image is unconditional; its preview is not driven by oc.preview.
		Expect(getPreview("image/png", []map[string][]string{cover})).To(BeNil())
	})
})
