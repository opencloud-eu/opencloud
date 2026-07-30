package kql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/ast"
)

// radiusUnits maps a distance suffix to its length in meters. Ordered longest
// first so parseRadius matches "km"/"mi" before the bare "m".
var radiusUnits = []struct {
	suffix string
	meters float64
}{
	{"km", 1000},
	{"mi", 1609.344},
	{"m", 1},
}

// parseRadius parses a distance like "5km", "500m" or "3mi" into meters.
func parseRadius(s string) (float64, error) {
	s = strings.TrimSpace(s)
	for _, u := range radiusUnits {
		if num, ok := strings.CutSuffix(s, u.suffix); ok {
			v, err := strconv.ParseFloat(strings.TrimSpace(num), 64)
			if err != nil {
				return 0, fmt.Errorf("invalid radius %q: %w", s, err)
			}
			return v * u.meters, nil
		}
	}
	return 0, fmt.Errorf("radius %q needs a unit (km, m, mi)", s)
}

func parseCoord(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

// parseGeoDistanceArgs parses "lat, lon, radius".
func parseGeoDistanceArgs(s string) (lat, lon, radius float64, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("geo.distance expects lat, lon, radius, got %q", s)
	}
	if lat, err = parseCoord(parts[0]); err != nil {
		return
	}
	if lon, err = parseCoord(parts[1]); err != nil {
		return
	}
	radius, err = parseRadius(parts[2])
	return
}

// parseGeoBoundingBoxArgs parses "minLat, minLon, maxLat, maxLon".
func parseGeoBoundingBoxArgs(s string) (minLat, minLon, maxLat, maxLon float64, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("geo.bbox expects minLat, minLon, maxLat, maxLon, got %q", s)
	}
	coords := make([]float64, 4)
	for i, p := range parts {
		if coords[i], err = parseCoord(p); err != nil {
			return
		}
	}
	return coords[0], coords[1], coords[2], coords[3], nil
}

// parseGeoPolygonArgs parses "lat lon, lat lon, ..." into at least three vertices.
func parseGeoPolygonArgs(s string) ([]ast.GeoPoint, error) {
	parts := strings.Split(s, ",")
	if len(parts) < 3 {
		return nil, fmt.Errorf("geo.polygon needs at least 3 points, got %q", s)
	}
	points := make([]ast.GeoPoint, 0, len(parts))
	for _, p := range parts {
		fields := strings.Fields(p)
		if len(fields) != 2 {
			return nil, fmt.Errorf("geo.polygon point %q must be 'lat lon'", strings.TrimSpace(p))
		}
		lat, err := parseCoord(fields[0])
		if err != nil {
			return nil, err
		}
		lon, err := parseCoord(fields[1])
		if err != nil {
			return nil, err
		}
		points = append(points, ast.GeoPoint{Lat: lat, Lon: lon})
	}
	return points, nil
}
