package thumbnail_test

import (
	"image"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

func sizes(entries ...string) *thumbnail.Sizes {
	s, err := thumbnail.ParseSizes(entries)
	Expect(err).ToNot(HaveOccurred())
	return s
}

func box(w, h int) image.Rectangle { return image.Rect(0, 0, w, h) }

var _ = Describe("Sizes", func() {
	Describe("ParseSizes", func() {
		It("reads plain edge lengths", func() {
			Expect(sizes("64", "16", "32").Max()).To(Equal(64))
		})

		It("takes both sides of a legacy resolution", func() {
			s := sizes("500x280")
			Expect(s.Snap(box(300, 300))).To(Equal(box(500, 500)))
			Expect(s.Snap(box(100, 100))).To(Equal(box(280, 280)))
		})

		It("caps the sides of a legacy resolution at what the default generator accepts", func() {
			s := sizes("16x16", "1920x1080", "3840x2160", "7680x4320")
			Expect(s.Max()).To(Equal(2048))
			Expect(s.Snap(box(0, 1000))).To(Equal(box(2048, 1080)))
		})

		It("takes a plain length above that as it is", func() {
			Expect(sizes("16", "4096").Max()).To(Equal(4096))
		})

		It("rejects an entry that is not a number", func() {
			_, err := thumbnail.ParseSizes([]string{"large"})
			Expect(err).To(HaveOccurred())
		})

		It("ignores empty entries and sides that are not positive", func() {
			Expect(sizes("", " 64 ", "0", "-5").Max()).To(Equal(64))
		})
	})

	Describe("Snap", func() {
		ladder := []string{"16", "32", "64", "128", "320", "512", "1024", "1280"}

		DescribeTable("rounds every axis up on its own",
			func(req, want image.Rectangle) {
				Expect(sizes(ladder...).Snap(req)).To(Equal(want))
			},
			Entry("an exact length stays", box(320, 320), box(320, 320)),
			Entry("a square request never picks an oriented box", box(448, 448), box(512, 512)),
			Entry("the axes are independent", box(448, 500), box(512, 512)),
			Entry("a wide request keeps its shape", box(1024, 320), box(1024, 320)),
			Entry("a tall request keeps its shape", box(320, 1024), box(320, 1024)),
			Entry("above the ladder it caps at the largest", box(4000, 4000), box(1280, 1280)),
		)

		It("gives an unconstrained axis the largest length, so the result stays bounded", func() {
			Expect(sizes(ladder...).Snap(box(0, 512))).To(Equal(box(1280, 512)))
			Expect(sizes(ladder...).Snap(box(512, 0))).To(Equal(box(512, 1280)))
		})

		It("passes the request through when nothing is configured", func() {
			Expect(sizes().Snap(box(448, 500))).To(Equal(box(448, 500)))
		})
	})
})
