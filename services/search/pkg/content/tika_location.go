package content

import (
	"strconv"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

// graph geoCoordinates.altitude is in feet, exif GPS altitude (geo:alt) in metres.
const metresToFeet = 3.280839895

func (t Tika) getLocation(meta map[string][]string) *libregraph.GeoCoordinates {
	var location *libregraph.GeoCoordinates
	initLocation := func() {
		if location == nil {
			location = libregraph.NewGeoCoordinates()
		}
	}

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

	// tika emits metres (negative below sea level), graph wants feet
	if v, err := getFirstValue(meta, "geo:alt"); err == nil {
		if metres, err := strconv.ParseFloat(v, 64); err == nil {
			initLocation()
			location.SetAltitude(metres * metresToFeet)
		}
	}

	return location
}
