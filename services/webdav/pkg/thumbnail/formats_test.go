package thumbnail_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

var _ = Describe("Formats", func() {
	It("writes an icon as a png by default, since no generator writes icons", func() {
		f := thumbnail.NewFormats(thumbnail.DefaultFormats)
		Expect(f.ExtFor("image/vnd.microsoft.icon")).To(Equal("png"))
		Expect(f.ExtFor("image/x-icon")).To(Equal("png"))
	})

	It("leaves every other type to the caller", func() {
		f := thumbnail.NewFormats(thumbnail.DefaultFormats)
		Expect(f.ExtFor("image/png")).To(BeEmpty())
		Expect(f.ExtFor("image/jpeg")).To(BeEmpty())
	})

	It("takes the mapping from the configuration", func() {
		f := thumbnail.NewFormats([]string{" Image/TIFF : Image/PNG ", "image/bmp:image/jpeg"})
		Expect(f.ExtFor("image/tiff; charset=binary")).To(Equal("png"))
		Expect(f.ExtFor("image/bmp")).To(Equal("jpg"))
		Expect(f.ExtFor("image/vnd.microsoft.icon")).To(BeEmpty(), "nothing is mapped unless configured")
	})
})

var _ = Describe("Formats, types the generator cannot write", func() {
	It("ignores a target the generator has no format for", func() {
		f := thumbnail.NewFormats([]string{"image/x-icon:image/vnd.microsoft.icon"})
		Expect(f.ExtFor("image/x-icon")).To(BeEmpty(), "an icon is what we are mapping away from")
	})
})
