package content

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

var _ = Describe("getImage", func() {
	It("maps the image dimensions to the image facet", func() {
		image := Tika{}.getImage(map[string][]string{
			"Content-Type":     {"image/jpeg"},
			"tiff:ImageWidth":  {"100"},
			"tiff:ImageLength": {"200"},
		})
		Expect(image).ToNot(BeNil())
		Expect(image.Width).To(Equal(libregraph.PtrInt32(100)))
		Expect(image.Height).To(Equal(libregraph.PtrInt32(200)))
	})

	It("returns nil for a video even though it carries the same tiff dimensions", func() {
		Expect(Tika{}.getImage(map[string][]string{
			"Content-Type":     {"video/mp4"},
			"tiff:ImageWidth":  {"1920"},
			"tiff:ImageLength": {"1080"},
		})).To(BeNil())
	})

	It("returns nil when no image metadata is present", func() {
		Expect(Tika{}.getImage(map[string][]string{})).To(BeNil())
	})
})
