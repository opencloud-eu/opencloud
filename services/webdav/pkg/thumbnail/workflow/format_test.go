package workflow

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("thumbnail format", func() {
	DescribeTable("extOf reads the format back out of cached bytes",
		func(data []byte, ext string) {
			Expect(extOf(data, "fallback")).To(Equal(ext))
		},
		Entry("png", []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, "png"),
		Entry("jpeg", []byte{0xff, 0xd8, 0xff, 0xe0}, "jpg"),
		Entry("gif", []byte("GIF89a"), "gif"),
		Entry("anything else keeps the fallback", []byte("whatever"), "fallback"),
	)
})
