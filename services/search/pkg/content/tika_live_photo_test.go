package content

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

var _ = Describe("getLivePhoto", func() {
	It("maps the video half of a live photo", func() {
		livePhoto := Tika{}.getLivePhoto(map[string][]string{
			"Content-Type":                                            {"video/quicktime"},
			"com.apple.quicktime.content.identifier":                  {"6F1A2B3C-1234-4E5F-9A8B-0011223344CC"},
			"quicktime:still-image-time":                              {"1500000"},
			"com.apple.quicktime.live-photo.auto":                     {"1"},
			"com.apple.quicktime.live-photo.vitality-score":           {"0.75"},
			"com.apple.quicktime.live-photo.vitality-scoring-version": {"4"},
		})
		Expect(livePhoto).ToNot(BeNil())

		Expect(livePhoto.ContentId).To(Equal("6F1A2B3C-1234-4E5F-9A8B-0011223344CC"))
		Expect(livePhoto.StillImageTimeUs).To(Equal(libregraph.PtrInt64(1500000)))
		Expect(livePhoto.Auto).To(Equal(libregraph.PtrBool(true)))
		Expect(livePhoto.VitalityScore).To(Equal(libregraph.PtrFloat64(0.75)))
		Expect(livePhoto.VitalityScoringVersion).To(Equal(libregraph.PtrInt64(4)))
	})

	It("maps the still half via the Apple maker-note content identifier", func() {
		// the still carries the pairing id in the Apple maker note, which tika
		// surfaces under "Content Identifier" (HEIC maker note parsed since
		// metadata-extractor 2.21.0, drewnoakes/metadata-extractor#739).
		livePhoto := Tika{}.getLivePhoto(map[string][]string{
			"Content-Type":       {"image/heic"},
			"Content Identifier": {"6F1A2B3C-1234-4E5F-9A8B-0011223344CC"},
		})
		Expect(livePhoto).ToNot(BeNil())
		Expect(livePhoto.ContentId).To(Equal("6F1A2B3C-1234-4E5F-9A8B-0011223344CC"))
	})

	It("returns nil without a content identifier", func() {
		Expect(Tika{}.getLivePhoto(map[string][]string{
			"Content-Type": {"image/jpeg"},
		})).To(BeNil())
	})
})
