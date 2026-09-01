package content

import (
	"fmt"
	"math"
	"strconv"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

// graph geoCoordinates.altitude is in feet, exif GPS altitude (geo:alt) in metres.
const metresToFeet = 3.280839895

func (t Tika) getLocation(meta map[string][]string) *libregraph.GeoCoordinates {
	// the facet needs a sane coordinate pair, an altitude alone is useless
	lat, latErr := parseCoordinate(meta, "geo:lat", 90)
	long, longErr := parseCoordinate(meta, "geo:long", 180)
	if latErr != nil || longErr != nil {
		return nil
	}

	location := libregraph.NewGeoCoordinates()
	location.SetLatitude(lat)
	location.SetLongitude(long)

	// tika emits metres (negative below sea level), graph wants feet
	if v, err := getFirstValue(meta, "geo:alt"); err == nil {
		if metres, err := strconv.ParseFloat(v, 64); err == nil {
			location.SetAltitude(metres * metresToFeet)
		}
	}

	return location
}

func parseCoordinate(meta map[string][]string, key string, limit float64) (float64, error) {
	v, err := getFirstValue(meta, key)
	if err != nil {
		return 0, err
	}

	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, err
	}
	// ParseFloat accepts "NaN", which json cannot marshal
	if math.IsNaN(f) || math.Abs(f) > limit {
		return 0, fmt.Errorf("%s out of range: %v", key, f)
	}

	return f, nil
}
