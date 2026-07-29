package content

import (
	"strconv"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

func (t Tika) getLocation(meta map[string][]string) *libregraph.GeoCoordinates {
	var location *libregraph.GeoCoordinates
	initLocation := func() {
		if location == nil {
			location = libregraph.NewGeoCoordinates()
		}
	}

	// TODO: location.Altitute: transform the following data to … feet above sea level.
	// "GPS:GPS Altitude":                          []string{"227.4 metres"},
	// "GPS:GPS Altitude Ref":                      []string{"Sea level"},

	if v, err := getFirstValue(meta, "geo:lat"); err == nil {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			initLocation()
			location.SetLatitude(i)
		}
	}

	if v, err := getFirstValue(meta, "geo:long"); err == nil {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			initLocation()
			location.SetLongitude(i)
		}
	}

	return location
}
