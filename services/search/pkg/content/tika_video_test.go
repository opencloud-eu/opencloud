package content

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

var _ = Describe("getVideo", func() {
	It("maps the video metadata to the video facet", func() {
		video := Tika{}.getVideo(map[string][]string{
			"Content-Type":           {"video/mp4"},
			"video:fourcc":           {"avc1"},
			"tiff:ImageWidth":        {"1920"},
			"tiff:ImageLength":       {"1080"},
			"xmpDM:duration":         {"12.5"},
			"video:bitrate":          {"47280"},
			"video:frame-rate":       {"25.0"},
			"xmpDM:audioSampleRate":  {"48000"},
			"audio:bits-per-sample":  {"16"},
			"audio:channels":         {"6"},
			"xmpDM:audioChannelType": {"Stereo"},
			"audio:fourcc":           {"mp4a"},
		})
		Expect(video).ToNot(BeNil())

		Expect(video.Width).To(Equal(libregraph.PtrInt32(1920)))
		Expect(video.Height).To(Equal(libregraph.PtrInt32(1080)))
		Expect(video.Duration).To(Equal(libregraph.PtrInt64(12500)))
		Expect(video.FourCC).To(Equal(libregraph.PtrString("avc1")))
		Expect(video.Bitrate).To(Equal(libregraph.PtrInt32(47280)))
		Expect(video.FrameRate).To(Equal(libregraph.PtrFloat64(25.0)))
		Expect(video.AudioSamplesPerSecond).To(Equal(libregraph.PtrInt32(48000)))
		Expect(video.AudioBitsPerSample).To(Equal(libregraph.PtrInt32(16)))
		Expect(video.AudioChannels).To(Equal(libregraph.PtrInt32(6)), "the numeric count wins over the enum")
		Expect(video.AudioFormat).To(Equal(libregraph.PtrString("mp4a")))
	})

	It("ignores the compressor name, it is no FourCC", func() {
		video := Tika{}.getVideo(map[string][]string{
			"Content-Type":          {"video/mp4"},
			"xmpDM:videoCompressor": {"AVC Coding"},
			"xmpDM:duration":        {"1.0"},
		})
		Expect(video).ToNot(BeNil())
		Expect(video.FourCC).To(BeNil())
	})

	It("falls back to the channel enum without a numeric count", func() {
		video := Tika{}.getVideo(map[string][]string{
			"Content-Type":           {"video/mp4"},
			"xmpDM:audioChannelType": {"5.1"},
		})
		Expect(video).ToNot(BeNil())
		Expect(video.AudioChannels).To(Equal(libregraph.PtrInt32(6)))
	})

	It("returns nil for a non-video file (image content type)", func() {
		Expect(Tika{}.getVideo(map[string][]string{
			"Content-Type":     {"image/jpeg"},
			"tiff:ImageWidth":  {"1920"},
			"tiff:ImageLength": {"1080"},
		})).To(BeNil())
	})

	It("returns nil when no video metadata is present", func() {
		Expect(Tika{}.getVideo(map[string][]string{})).To(BeNil())
	})
})
