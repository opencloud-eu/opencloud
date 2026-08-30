package workflow

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("thumbnail format", func() {
	DescribeTable("typeOfThumbnail reads the type back out of cached bytes",
		func(data []byte, contentType string) {
			Expect(typeOfThumbnail(data, "fallback")).To(Equal(contentType))
		},
		Entry("png", []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, "image/png"),
		Entry("jpeg", []byte{0xff, 0xd8, 0xff, 0xe0}, "image/jpeg"),
		Entry("gif", []byte("GIF89a"), "image/gif"),
		Entry("anything else keeps the fallback", []byte("whatever"), "fallback"),
	)
})
