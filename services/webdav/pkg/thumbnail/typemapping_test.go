package thumbnail_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/generator"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

var _ = Describe("TypeMapping", func() {
	m := thumbnail.NewTypeMapping(nil)

	It("answers for every type whose output does not depend on the file", func() {
		for _, mimeType := range thumbnail.NewSourceTypes(nil).List() {
			if thumbnail.IsEmbeddedSourceType(mimeType) {
				continue
			}
			Expect(m.TargetFor(mimeType)).ToNot(BeEmpty(), "%s has no output type", mimeType)
		}
	})

	It("says nothing about a type that carries its preview inside", func() {
		// the embedded image's own type decides, and the index knows it
		for _, mimeType := range thumbnail.EmbeddedSourceTypes {
			Expect(m.TargetFor(mimeType)).To(BeEmpty(), "%s should follow the embedded type", mimeType)
		}
	})

	It("keeps a format that carries alpha or animation", func() {
		Expect(m.TargetFor("image/png")).To(Equal("image/png"))
		Expect(m.TargetFor("image/gif")).To(Equal("image/gif"))
	})

	It("writes an icon as a png, since no generator writes icons", func() {
		Expect(m.TargetFor("image/vnd.microsoft.icon")).To(Equal("image/png"))
		Expect(m.TargetFor("image/x-icon")).To(Equal("image/png"))
	})

	It("reads a type with parameters", func() {
		Expect(m.TargetFor("image/png; charset=binary")).To(Equal("image/png"))
	})

	It("leaves a type nobody mapped to the caller", func() {
		Expect(m.TargetFor("application/pdf")).To(BeEmpty())
		Expect(m.TargetFor("not a mime type")).To(BeEmpty())
	})
})

var _ = Describe("TypeMapping from the configuration", func() {
	It("overrides a default and keeps the rest", func() {
		m := thumbnail.NewTypeMapping([]string{" Image/TIFF : Image/PNG "})

		Expect(m.TargetFor("image/tiff")).To(Equal("image/png"))
		Expect(m.TargetFor("image/png")).To(Equal("image/png"), "a default nobody touched still answers")
		Expect(m.TargetFor("image/bmp")).To(Equal("image/jpeg"))
	})

	It("hands on a target the generator cannot write, for the caller to reject", func() {
		m := thumbnail.NewTypeMapping([]string{"image/png:image/vnd.microsoft.icon"})

		Expect(m.TargetFor("image/png")).To(Equal("image/vnd.microsoft.icon"))
		Expect(generator.NewFormats(nil).For("image/vnd.microsoft.icon")).To(BeEmpty())
	})

	It("takes an empty target as back to the default", func() {
		m := thumbnail.NewTypeMapping([]string{"image/png:"})

		Expect(m.TargetFor("image/png")).To(BeEmpty())
	})
})
