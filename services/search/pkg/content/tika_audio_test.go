package content

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

var _ = Describe("getAudio", func() {
	It("maps the audio metadata to the audio facet", func() {
		meta := map[string][]string{
			"xmpDM:genre":       {"Some Genre"},
			"xmpDM:album":       {"Some Album"},
			"xmpDM:trackNumber": {"7"},
			"xmpDM:discNumber":  {"4"},
			"xmpDM:releaseDate": {"2004"},
			"xmpDM:artist":      {"Some Artist"},
			"xmpDM:albumArtist": {"Some AlbumArtist"},
			"dc:title":          {"Some Title"},
			"xmpDM:duration":    {"225.5"},
		}

		audio := Tika{}.getAudio(meta)
		Expect(audio).ToNot(BeNil())

		Expect(audio.Album).To(Equal(libregraph.PtrString("Some Album")))
		Expect(audio.AlbumArtist).To(Equal(libregraph.PtrString("Some AlbumArtist")))
		Expect(audio.Artist).To(Equal(libregraph.PtrString("Some Artist")))
		Expect(audio.Disc).To(Equal(libregraph.PtrInt32(4)))
		Expect(audio.Duration).To(Equal(libregraph.PtrInt64(225500)))
		Expect(audio.Genre).To(Equal(libregraph.PtrString("Some Genre")))
		Expect(audio.Title).To(Equal(libregraph.PtrString("Some Title")))
		Expect(audio.Track).To(Equal(libregraph.PtrInt32(7)))
		Expect(audio.Year).To(Equal(libregraph.PtrInt32(2004)))
	})

	It("returns nil when no audio metadata is present", func() {
		Expect(Tika{}.getAudio(map[string][]string{})).To(BeNil())
	})
})
