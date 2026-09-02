package content

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

var _ = Describe("getMotionPhoto", func() {
	It("maps the current MotionPhoto XMP scheme (container item length)", func() {
		mp := Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhotoVersion":                 {"1"},
			"Camera:MotionPhotoPresentationTimestampUs": {"1500000"},
			"Container:Directory/Item[2]/Item:Semantic": {"MotionPhoto"},
			"Container:Directory/Item[2]/Item:Length":   {"1048576"},
		})
		Expect(mp).ToNot(BeNil())
		Expect(mp.Version).To(Equal(libregraph.PtrInt32(1)))
		Expect(mp.PresentationTimestampUs).To(Equal(libregraph.PtrInt64(1500000)))
		Expect(mp.VideoSize).To(Equal(libregraph.PtrInt64(1048576)))
	})

	It("maps the legacy MicroVideo XMP scheme (offset is the length)", func() {
		mp := Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MicroVideoVersion":                 {"1"},
			"Camera:MicroVideoPresentationTimestampUs": {"1500000"},
			"Camera:MicroVideoOffset":                  {"2097152"},
		})
		Expect(mp).ToNot(BeNil())
		Expect(mp.Version).To(Equal(libregraph.PtrInt32(1)))
		Expect(mp.PresentationTimestampUs).To(Equal(libregraph.PtrInt64(1500000)))
		Expect(mp.VideoSize).To(Equal(libregraph.PtrInt64(2097152)))
	})

	It("drops the facet without a video size", func() {
		Expect(Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhotoVersion": {"1"},
		})).To(BeNil())
	})

	It("returns nil when no motion photo metadata is present", func() {
		Expect(Tika{}.getMotionPhoto(map[string][]string{})).To(BeNil())
	})

	It("treats a zero MotionPhoto marker as a still image", func() {
		Expect(Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhoto":                        {"0"},
			"Camera:MotionPhotoVersion":                 {"1"},
			"Container:Directory/Item[2]/Item:Semantic": {"MotionPhoto"},
			"Container:Directory/Item[2]/Item:Length":   {"1048576"},
		})).To(BeNil())
		Expect(Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MicroVideo":       {"0"},
			"Camera:MicroVideoOffset": {"2097152"},
		})).To(BeNil())
	})

	It("treats undefined marker values as a still image", func() {
		Expect(Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhoto":                        {"2"},
			"Container:Directory/Item[2]/Item:Semantic": {"MotionPhoto"},
			"Container:Directory/Item[2]/Item:Length":   {"1048576"},
		})).To(BeNil())
	})

	It("prefers the current scheme when both are present", func() {
		mp := Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhotoVersion":                 {"2"},
			"Camera:MicroVideoVersion":                  {"1"},
			"Camera:MicroVideoOffset":                   {"2097152"},
			"Container:Directory/Item[2]/Item:Semantic": {"MotionPhoto"},
			"Container:Directory/Item[2]/Item:Length":   {"1048576"},
		})
		Expect(mp).ToNot(BeNil())
		Expect(mp.Version).To(Equal(libregraph.PtrInt32(2)))
		Expect(mp.VideoSize).To(Equal(libregraph.PtrInt64(1048576)))
	})
})

var _ = Describe("isVideo", func() {
	DescribeTable("recognizes the video tika extracted from a motion photo",
		func(meta map[string][]string, expected bool) {
			Expect(isVideo(meta)).To(Equal(expected))
		},
		Entry("mp4", map[string][]string{"Content-Type": {"video/mp4"}}, true),
		Entry("quicktime", map[string][]string{"Content-Type": {"video/quicktime"}}, true),
		Entry("with parameters", map[string][]string{"Content-Type": {"video/mp4; codecs=avc1"}}, true),
		Entry("the image itself", map[string][]string{"Content-Type": {"image/jpeg"}}, false),
		Entry("another attachment", map[string][]string{"Content-Type": {"application/pdf"}}, false),
		Entry("no content type", map[string][]string{"Camera:MotionPhoto": {"1"}}, false),
	)
})
