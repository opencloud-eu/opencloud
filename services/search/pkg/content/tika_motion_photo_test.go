package content

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

var _ = Describe("getMotionPhoto", func() {
	It("maps the current MotionPhoto XMP scheme", func() {
		mp := Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhoto":                        {"1"},
			"Camera:MotionPhotoVersion":                 {"1"},
			"Camera:MotionPhotoPresentationTimestampUs": {"1500000"},
		}, map[string][]string{"Content-Length": {"1048576"}, "Content-Type": {"video/mp4"}})
		Expect(mp).ToNot(BeNil())
		Expect(mp.Version).To(Equal(libregraph.PtrInt32(1)))
		Expect(mp.PresentationTimestampUs).To(Equal(libregraph.PtrInt64(1500000)))
		Expect(mp.VideoSize).To(Equal(libregraph.PtrInt64(1048576)))
	})

	It("maps the legacy MicroVideo XMP scheme", func() {
		mp := Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MicroVideo":                        {"1"},
			"Camera:MicroVideoVersion":                 {"1"},
			"Camera:MicroVideoPresentationTimestampUs": {"1500000"},
		}, map[string][]string{"Content-Length": {"1048576"}, "Content-Type": {"video/mp4"}})
		Expect(mp).ToNot(BeNil())
		Expect(mp.Version).To(Equal(libregraph.PtrInt32(1)))
		Expect(mp.PresentationTimestampUs).To(Equal(libregraph.PtrInt64(1500000)))
		Expect(mp.VideoSize).To(Equal(libregraph.PtrInt64(1048576)))
	})

	It("drops the facet when the video reports no length", func() {
		Expect(Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhoto":        {"1"},
			"Camera:MotionPhotoVersion": {"1"},
		}, map[string][]string{"Content-Type": {"video/mp4"}})).To(BeNil())
	})

	It("returns nil without the marker, a picture may just carry a video", func() {
		Expect(Tika{}.getMotionPhoto(map[string][]string{}, map[string][]string{"Content-Length": {"1048576"}, "Content-Type": {"video/mp4"}})).To(BeNil())
	})

	It("treats a zero MotionPhoto marker as a still image", func() {
		Expect(Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhoto":        {"0"},
			"Camera:MotionPhotoVersion": {"1"},
		}, map[string][]string{"Content-Length": {"1048576"}, "Content-Type": {"video/mp4"}})).To(BeNil())
		Expect(Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MicroVideo": {"0"},
		}, map[string][]string{"Content-Length": {"1048576"}, "Content-Type": {"video/mp4"}})).To(BeNil())
	})

	It("treats undefined marker values as a still image", func() {
		Expect(Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhoto": {"2"},
		}, map[string][]string{"Content-Length": {"1048576"}, "Content-Type": {"video/mp4"}})).To(BeNil())
	})

	It("prefers the current scheme when both are present", func() {
		mp := Tika{}.getMotionPhoto(map[string][]string{
			"Camera:MotionPhoto":        {"1"},
			"Camera:MotionPhotoVersion": {"2"},
			"Camera:MicroVideoVersion":  {"1"},
		}, map[string][]string{"Content-Length": {"1048576"}, "Content-Type": {"video/mp4"}})
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
