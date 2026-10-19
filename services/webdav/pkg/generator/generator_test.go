package generator_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/generator"
)

var _ = Describe("ParseMaxInputFileSize", func() {
	DescribeTable("reads a size the way the thumbnails service did",
		func(input string, want uint64, wantErr bool) {
			got, err := generator.ParseMaxInputFileSize(input)
			if wantErr {
				Expect(err).To(HaveOccurred())

				return
			}

			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(want))
		},
		Entry("empty means no limit", "", uint64(0), false),
		Entry("plain bytes", "1234", uint64(1234), false),
		Entry("KB is decimal", "50KB", uint64(50_000), false),
		Entry("KiB is binary", "50KiB", uint64(50*1024), false),
		Entry("MB is decimal", "5MB", uint64(5_000_000), false),
		Entry("MiB is binary", "5MiB", uint64(5*1024*1024), false),
		Entry("GB is decimal", "2GB", uint64(2_000_000_000), false),
		Entry("GiB is binary", "2GiB", uint64(2*1024*1024*1024), false),
		Entry("TB", "2TB", uint64(2_000_000_000_000), false),
		Entry("whitespace around the value", "  5MB  ", uint64(5_000_000), false),
		Entry("space before the suffix", "5 MB", uint64(5_000_000), false),
		Entry("a lowercase unit", "50kB", uint64(0), true),
		Entry("not a size", "abc", uint64(0), true),
		Entry("a negative size", "-5MB", uint64(0), true),
	)
})

var _ = Describe("Formats", func() {
	DescribeTable("names the format the generator's filter uses for a type",
		func(contentType, want string) {
			Expect(generator.NewFormats(nil).For(contentType)).To(Equal(want))
		},
		Entry("png", "image/png", "png"),
		Entry("gif", "image/gif", "gif"),
		Entry("jpeg", "image/jpeg", "jpg"),
		Entry("a type with parameters", "image/jpeg; q=1", "jpg"),
		Entry("webp, which the built-in generator cannot write", "image/webp", ""),
		Entry("an icon, which no generator writes", "image/vnd.microsoft.icon", ""),
		Entry("nonsense", "not a mime type", ""),
	)

	It("takes a format the built-in generator cannot write, for a generator that can", func() {
		formats := generator.NewFormats([]string{" Image/WEBP : WebP "})

		Expect(formats.For("image/webp")).To(Equal("webp"))
		Expect(formats.For("image/gif")).To(Equal("gif"), "a default nobody touched still answers")
	})

	It("takes an empty format as no format", func() {
		Expect(generator.NewFormats([]string{"image/png:"}).For("image/png")).To(BeEmpty())
	})
})

var _ = Describe("BuildURL", func() {
	DescribeTable("renders the generator's image endpoint",
		func(base string, width, height int32, operation, format string, noUpscale bool, want string) {
			Expect(generator.BuildURL(base, width, height, operation, format, noUpscale)).To(Equal(want))
		},
		Entry("fill asks for the interesting region of the box",
			"http://generator", int32(128), int32(128), generator.OpFill, "jpeg", false,
			"http://generator/unsafe/128x128/smart/filters:format(jpeg)/"),
		Entry("fill with no_upscale caps the resize at the source size",
			"http://generator", int32(128), int32(128), generator.OpFill, "jpeg", true,
			"http://generator/unsafe/128x128/smart/filters:no_upscale():format(jpeg)/"),
		Entry("fit-in preserves the aspect ratio within the box",
			"http://generator", int32(1024), int32(1024), generator.OpFitIn, "jpeg", false,
			"http://generator/unsafe/fit-in/1024x1024/filters:format(jpeg)/"),
		Entry("stretch resizes to the exact box",
			"http://generator/", int32(64), int32(32), generator.OpStretch, "png", false,
			"http://generator/unsafe/stretch/64x32/filters:format(png)/"),
	)
})
