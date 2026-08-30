package thumbnail_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

var _ = Describe("Tika", func() {
	none := thumbnail.NewTika("", nil)
	all := thumbnail.NewTika("http://tika:9998", nil)
	some := thumbnail.NewTika("http://tika:9998", []string{" Application/PDF ", "audio/mpeg", "image/x-raw-samsung:image/x-samsung-srw"})

	DescribeTable("Supports",
		func(tika thumbnail.Tika, mimeType string, want bool) {
			Expect(tika.Supports(mimeType)).To(Equal(want))
		},
		Entry("nothing without a server", none, "application/pdf", false),
		Entry("everything by default", all, "application/x-anything; charset=binary", true),
		Entry("but no directories", all, "httpd/unix-directory", false),
		Entry("nor an unparsable type", all, "not a mime", false),
		Entry("a listed type", some, "application/pdf", true),
		Entry("a listed type with parameters", some, "audio/mpeg; charset=binary", true),
		Entry("a mapped type", some, "image/x-raw-samsung", true),
		Entry("an unlisted type", some, "application/zip", false),
	)

	DescribeTable("ContentType",
		func(tika thumbnail.Tika, mimeType, want string) {
			Expect(tika.ContentType(mimeType)).To(Equal(want))
		},
		Entry("maps a type to the one Tika knows", some, "image/x-raw-samsung", "image/x-samsung-srw"),
		Entry("passes an unmapped type through, without parameters", some, "application/pdf; charset=binary", "application/pdf"),
		Entry("passes everything through by default", all, "audio/mpeg", "audio/mpeg"),
	)
})
