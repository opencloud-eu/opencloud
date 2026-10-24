package parity

import (
	"fmt"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func metadataGroup() responseGroup {
	song := fixtureDoc("some_song.mp3",
		withID("1$1!5"),
		withMime("audio/mpeg"),
		withAudio(&libregraph.Audio{
			Album:             libregraph.PtrString("Some Album"),
			AlbumArtist:       libregraph.PtrString("Some AlbumArtist"),
			Artist:            libregraph.PtrString("Some Artist"),
			Bitrate:           libregraph.PtrInt64(192),
			Composers:         libregraph.PtrString("Some Composers"),
			Copyright:         libregraph.PtrString(""),
			Disc:              libregraph.PtrInt32(2),
			DiscCount:         libregraph.PtrInt32(5),
			Duration:          libregraph.PtrInt64(225000),
			Genre:             libregraph.PtrString("Some Genre"),
			HasDrm:            libregraph.PtrBool(false),
			IsVariableBitrate: libregraph.PtrBool(true),
			Title:             libregraph.PtrString("Some Title"),
			Track:             libregraph.PtrInt32(34),
			TrackCount:        libregraph.PtrInt32(99),
			Year:              libregraph.PtrInt32(2004),
		}),
	)

	team := fixtureDoc("team.jpg",
		withID("1$1!6"),
		withMime("image/jpeg"),
		withLocation(&libregraph.GeoCoordinates{
			Altitude:  libregraph.PtrFloat64(1047.7),
			Latitude:  libregraph.PtrFloat64(49.48675890884328),
			Longitude: libregraph.PtrFloat64(11.103870357204285),
		}),
	)

	indexed := []string{
		"Album=Some Album",
		"AlbumArtist=Some AlbumArtist",
		"Artist=Some Artist",
		"Bitrate=192",
		"Composers=Some Composers",
		"Copyright=",
		"Disc=2",
		"DiscCount=5",
		"Duration=225000",
		"Genre=Some Genre",
		"HasDrm=false",
		"IsVariableBitrate=true",
		"Title=Some Title",
		"Track=34",
		"TrackCount=99",
		"Year=2004",
	}

	located := []string{
		"Altitude=1047.7",
		"Latitude=49.48675890884328",
		"Longitude=11.103870357204285",
	}

	return responseGroup{
		name:     "metadata",
		fixtures: []search.Resource{song, team},
		cases: []responseCase{
			{
				id: 1, query: `*song*`, reads: "Audio", want: unchanged(indexed),
				read: readsMany(func(m *searchMessage.Match) []string {
					audio := m.GetEntity().GetAudio()

					return changes(indexed, []string{
						fmt.Sprintf("Album=%s", audio.GetAlbum()),
						fmt.Sprintf("AlbumArtist=%s", audio.GetAlbumArtist()),
						fmt.Sprintf("Artist=%s", audio.GetArtist()),
						fmt.Sprintf("Bitrate=%d", audio.GetBitrate()),
						fmt.Sprintf("Composers=%s", audio.GetComposers()),
						fmt.Sprintf("Copyright=%s", audio.GetCopyright()),
						fmt.Sprintf("Disc=%d", audio.GetDisc()),
						fmt.Sprintf("DiscCount=%d", audio.GetDiscCount()),
						fmt.Sprintf("Duration=%d", audio.GetDuration()),
						fmt.Sprintf("Genre=%s", audio.GetGenre()),
						fmt.Sprintf("HasDrm=%t", audio.GetHasDrm()),
						fmt.Sprintf("IsVariableBitrate=%t", audio.GetIsVariableBitrate()),
						fmt.Sprintf("Title=%s", audio.GetTitle()),
						fmt.Sprintf("Track=%d", audio.GetTrack()),
						fmt.Sprintf("TrackCount=%d", audio.GetTrackCount()),
						fmt.Sprintf("Year=%d", audio.GetYear()),
					})
				}),
			},
			{
				id: 2, query: `*team*`, reads: "Location", want: unchanged(located),
				read: readsMany(func(m *searchMessage.Match) []string {
					location := m.GetEntity().GetLocation()

					return changes(located, []string{
						fmt.Sprintf("Altitude=%v", location.GetAltitude()),
						fmt.Sprintf("Latitude=%v", location.GetLatitude()),
						fmt.Sprintf("Longitude=%v", location.GetLongitude()),
					})
				}),
			},
			{
				id: 3, query: `*team*`, reads: "Audio", want: []string{"none"},
				read: reads(func(m *searchMessage.Match) string {
					if m.GetEntity().GetAudio() == nil {
						return "none"
					}

					return "set"
				}),
			},
		},
	}
}

func unchanged(indexed []string) []string {
	return []string{fmt.Sprintf("all %d fields unchanged", len(indexed))}
}

func changes(indexed, got []string) []string {
	var off []string
	for i, field := range indexed {
		if got[i] != field {
			off = append(off, fmt.Sprintf("%s instead of %s", got[i], field))
		}
	}

	if len(off) == 0 {
		return unchanged(indexed)
	}

	return off
}
