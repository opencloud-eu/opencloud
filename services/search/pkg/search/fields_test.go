package search_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ = Describe("IsNumericField", func() {
	DescribeTable("reports whether a field maps to a numeric type",
		func(field string, numeric bool) {
			Expect(search.IsNumericField(field)).To(Equal(numeric))
		},
		// top-level numeric fields on Resource / Document
		Entry("Size (uint64 via embedded Document)", "Size", true),
		Entry("Type (uint64 on Resource)", "Type", true),
		// top-level string fields
		Entry("Name", "Name", false),
		Entry("Path", "Path", false),
		Entry("MimeType", "MimeType", false),
		// nested audio
		Entry("audio.artist", "audio.artist", false),
		Entry("audio.album", "audio.album", false),
		Entry("audio.year", "audio.year", true),
		Entry("audio.bitrate", "audio.bitrate", true),
		Entry("audio.track", "audio.track", true),
		Entry("audio.hasDrm (bool, not numeric)", "audio.hasDrm", false),
		// nested image
		Entry("image.width", "image.width", true),
		Entry("image.height", "image.height", true),
		// nested photo
		Entry("photo.cameraMake", "photo.cameraMake", false),
		Entry("photo.iso", "photo.iso", true),
		Entry("photo.focalLength (float32)", "photo.focalLength", true),
		Entry("photo.exposureDenominator (float32)", "photo.exposureDenominator", true),
		Entry("photo.takenDateTime (time.Time, treated as numeric)", "photo.takenDateTime", true),
		// nested location
		Entry("location.altitude", "location.altitude", true),
		Entry("location.latitude", "location.latitude", true),
		Entry("location.longitude", "location.longitude", true),
		// unknown fields the caller may still aggregate on
		Entry("nonexistent", "nonexistent", false),
		Entry("audio.nonexistent", "audio.nonexistent", false),
	)
})
