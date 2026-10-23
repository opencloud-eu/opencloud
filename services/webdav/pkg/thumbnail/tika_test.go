package thumbnail_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

var _ = Describe("Tika", func() {
	none := thumbnail.NewTika("", thumbnail.DefaultTikaTypes, nil)
	all := thumbnail.NewTika("http://tika:9998", []string{"*"}, nil)
	nothing := thumbnail.NewTika("http://tika:9998", nil, nil)
	some := thumbnail.NewTika("http://tika:9998", []string{" Application/PDF ", "audio/mpeg", "image/x-nikon-nef:image/x-raw-nikon"}, nil)
	built := thumbnail.NewTika("http://tika:9998", thumbnail.DefaultTikaTypes, nil)
	extended := thumbnail.NewTika("http://tika:9998", thumbnail.DefaultTikaTypes, []string{"application/zip", "image/heic"})
	extendedAll := thumbnail.NewTika("http://tika:9998", thumbnail.DefaultTikaTypes, []string{"*"})

	DescribeTable("Supports",
		func(tika thumbnail.Tika, mimeType string, want bool) {
			Expect(tika.Supports(mimeType)).To(Equal(want))
		},
		Entry("nothing without a server", none, "application/pdf", false),
		Entry("nothing with an empty list", nothing, "application/pdf", false),
		Entry("everything with the default list", all, "application/x-anything; charset=binary", true),
		Entry("but no directories", all, "httpd/unix-directory", false),
		Entry("nor an unparsable type", all, "not a mime", false),
		Entry("a listed type", some, "application/pdf", true),
		Entry("a listed type with parameters", some, "audio/mpeg; charset=binary", true),
		Entry("a mapped type", some, "image/x-nikon-nef", true),
		Entry("an unlisted type", some, "application/zip", false),
		Entry("a document from the built-in list", built, "application/pdf", true),
		Entry("a camera raw from the built-in list, which is why it is named there", built, "image/x-nikon-nef", true),
		Entry("but not a type the built-in list leaves alone", built, "application/zip", false),
		Entry("a type the additional list adds", extended, "application/zip", true),
		Entry("while the built-in list is still in effect", extended, "application/pdf", true),
		Entry("and a catch-all in the additional list takes the rest", extendedAll, "application/x-anything", true),
	)

	DescribeTable("ConfiguredFor",
		func(tika thumbnail.Tika, mimeType string, want bool) {
			Expect(tika.ConfiguredFor(mimeType)).To(Equal(want))
		},
		Entry("a named type", built, "image/x-nikon-nef", true),
		Entry("not a type the catch-all takes", all, "application/pdf", false),
		Entry("not an unnamed type", built, "application/zip", false),
	)

	DescribeTable("ContentType",
		func(tika thumbnail.Tika, mimeType, want string) {
			Expect(tika.ContentType(mimeType)).To(Equal(want))
		},
		Entry("pins the leaf type, so the detector does not stop at tiff and lose the embedded preview", some, "image/x-nikon-nef", "image/x-raw-nikon"),
		Entry("passes an unmapped type through, without parameters", some, "application/pdf; charset=binary", "application/pdf"),
		Entry("passes everything through with the default list", all, "audio/mpeg", "audio/mpeg"),
	)
})
