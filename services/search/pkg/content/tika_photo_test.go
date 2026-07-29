package content

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

var _ = Describe("getPhoto", func() {
	It("maps the exif metadata to the photo facet", func() {
		photo := Tika{}.getPhoto(map[string][]string{
			"tiff:Make":             {"Canon"},
			"tiff:Model":            {"Canon EOS 5D"},
			"exif:ExposureTime":     {"0.001"},
			"exif:FNumber":          {"1.8"},
			"exif:FocalLength":      {"50"},
			"Base ISO":              {"100"},
			"tiff:Orientation":      {"1"},
			"exif:DateTimeOriginal": {"2018-01-01T12:34:56"},
		})
		Expect(photo).ToNot(BeNil())
		Expect(photo.CameraMake).To(Equal(libregraph.PtrString("Canon")))
		Expect(photo.CameraModel).To(Equal(libregraph.PtrString("Canon EOS 5D")))
		Expect(photo.ExposureNumerator).To(Equal(libregraph.PtrFloat64(1)))
		Expect(photo.ExposureDenominator).To(Equal(libregraph.PtrFloat64(1000)))
		Expect(photo.FNumber).To(Equal(libregraph.PtrFloat64(1.8)))
		Expect(photo.FocalLength).To(Equal(libregraph.PtrFloat64(50)))
		Expect(photo.Iso).To(Equal(libregraph.PtrInt32(100)))
		Expect(photo.Orientation).To(Equal(libregraph.PtrInt32(1)))
		Expect(photo.TakenDateTime).To(Equal(libregraph.PtrTime(time.Date(2018, 1, 1, 12, 34, 56, 0, time.UTC))))
	})

	It("returns nil when no photo metadata is present", func() {
		Expect(Tika{}.getPhoto(map[string][]string{})).To(BeNil())
	})
})
