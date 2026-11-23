package parity

import (
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func photoGroup() queryGroup {
	return queryGroup{
		name: "photo",
		fixtures: []search.Resource{
			fixtureDoc("bright.jpg", withMime("image/jpeg"), withPhoto(&libregraph.Photo{CameraMake: libregraph.PtrString("Acme"), Iso: libregraph.PtrInt32(100), FNumber: libregraph.PtrFloat64(1.8), FocalLength: libregraph.PtrFloat64(26), Orientation: libregraph.PtrInt32(1)})),
			fixtureDoc("dark.jpg", withMime("image/jpeg"), withPhoto(&libregraph.Photo{Iso: libregraph.PtrInt32(3200), FNumber: libregraph.PtrFloat64(4), FocalLength: libregraph.PtrFloat64(85), Orientation: libregraph.PtrInt32(6)})),
			fixtureDoc("plain.txt"),
		},
		cases: []queryCase{
			{id: 1, query: `photo.iso>=1000`, want: []string{"dark.jpg"}},
			{id: 2, query: `Photo.FNumber<2`, want: []string{"bright.jpg"}},
			{id: 3, query: `photo.focalLength>30 AND photo.iso<=3200`, want: []string{"dark.jpg"}},
			{id: 4, query: `photo.orientation>1`, want: []string{"dark.jpg"}},
			{id: 5, query: `photo.iso<=100`, want: []string{"bright.jpg"}},
			{id: 6, query: `photo.cameraMake>1`}, // a text field takes no range
		},
	}
}
