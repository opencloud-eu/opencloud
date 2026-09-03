package parity

import (
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func at(lat, lon float64) fixtureOption {
	return withLocation(&libregraph.GeoCoordinates{
		Latitude:  libregraph.PtrFloat64(lat),
		Longitude: libregraph.PtrFloat64(lon),
	})
}

func geoGroup() queryGroup {
	return queryGroup{
		name: "geo",
		fixtures: []search.Resource{
			// vienna, and a second point ~35km east of it
			fixtureDoc("vienna.txt", at(48.2082, 16.3738)),
			fixtureDoc("bruck.txt", at(48.2082, 16.8500)),
			fixtureDoc("berlin.txt", at(52.5200, 13.4050)),
			fixtureDoc("nowhere.txt"),
		},
		cases: []queryCase{
			{id: 1, query: `location:geo.distance(48.2082, 16.3738, 5km)`, want: []string{"vienna.txt"}},
			{id: 2, query: `location:geo.distance(48.2082, 16.3738, 50km)`, want: []string{"vienna.txt", "bruck.txt"}},
			{id: 3, query: `location:geo.distance(48.2082, 16.3738, 1m)`, want: []string{"vienna.txt"}},
			{id: 4, query: `location:geo.bbox(47.9, 16.1, 48.3, 16.5)`, want: []string{"vienna.txt"}},
			// the corners may come in either order, both name the same box
			{id: 5, query: `location:geo.bbox(48.3, 16.1, 47.9, 16.5)`, want: []string{"vienna.txt"}},
			{id: 6, query: `location:geo.polygon(48.3 16.1, 48.3 16.5, 47.9 16.5, 47.9 16.1)`, want: []string{"vienna.txt"}},
			// a document without a location is never a geo hit
			{id: 7, query: `location:geo.distance(0, 0, 20000km)`, want: []string{"vienna.txt", "bruck.txt", "berlin.txt"}},
			// geo predicates only apply to geopoint fields, both engines reject the rest
			{id: 8, query: `name:geo.distance(48.2082, 16.3738, 5km)`, wantBadRequest: true},
		},
	}
}
