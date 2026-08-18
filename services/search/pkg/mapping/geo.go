package mapping

import (
	"strconv"
	"strings"
)

// GeopointSuffix is appended to a field's name to produce the sibling key
// that carries the geo_point / bleve-geopoint representation of the
// original facet. For example, a libregraph "location" object with
// longitude / latitude / altitude is preserved as-is under "location" (for
// data retrieval and numeric queries) while "location_geopoint" carries
// the {lat, lon} form the geo indices understand.
const GeopointSuffix = "_geopoint"

// addGeopointSiblings walks the overrides; for each TypeGeopoint entry at
// a dotted path (e.g. "location" or "journey.start") it writes a sibling
// under the suffixed key with the {lat, lon} form both bleve's
// ExtractGeoPoint and OpenSearch's geo_point parser accept. The original
// facet object stays untouched so downstream code still sees the full
// libregraph shape (including altitude).
func addGeopointSiblings(m map[string]any, overrides map[string]FieldOpts) {
	for key, opts := range overrides {
		if opts.Type == TypeGeopoint {
			addGeopointSibling(m, key)
		}
	}
}

// addGeopointSibling resolves dottedPath within m and, if the target is a
// libregraph-shaped geo object (with numeric "longitude" and "latitude"),
// writes the `{lat, lon}` sibling at the same level under the suffixed key.
func addGeopointSibling(m map[string]any, dottedPath string) {
	parts := strings.Split(dottedPath, ".")
	parent := m
	for _, p := range parts[:len(parts)-1] {
		next, ok := parent[p].(map[string]any)
		if !ok {
			return
		}
		parent = next
	}
	leaf := parts[len(parts)-1]
	obj, ok := parent[leaf].(map[string]any)
	if !ok {
		return
	}
	lon, hasLon := obj["longitude"].(float64)
	lat, hasLat := obj["latitude"].(float64)
	if !hasLon || !hasLat {
		return
	}
	parent[leaf+GeopointSuffix] = map[string]any{"lat": lat, "lon": lon}
}

// GeohashSuffix + a precision produce the sibling keyword field carrying the
// geohash prefix of a geopoint at that precision (e.g. "location_geohash_6").
// bleve has no native geohash-grid aggregation, so a terms aggregation on the
// field of the requested precision reproduces it. OpenSearch runs geohash_grid
// on the _geopoint field directly and ignores these.
const GeohashSuffix = "_geohash_"

// MaxGeohashPrecision is the finest geohash length indexed as a sibling field.
const MaxGeohashPrecision = 12

// GeohashField returns the sibling field name carrying base's geohash at the
// given precision, e.g. GeohashField("location", 6) == "location_geohash_6".
func GeohashField(base string, precision int) string {
	return base + GeohashSuffix + strconv.Itoa(precision)
}

const geohashBase32 = "0123456789bcdefghjkmnpqrstuvwxyz"

// encodeGeohash returns the standard geohash of (lat, lon) at the given length,
// matching Lucene/OpenSearch so both backends bucket points into the same cells.
func encodeGeohash(lat, lon float64, precision int) string {
	latMin, latMax := -90.0, 90.0
	lonMin, lonMax := -180.0, 180.0
	var b strings.Builder
	even := true
	bit, ch := 0, 0
	for b.Len() < precision {
		if even {
			mid := (lonMin + lonMax) / 2
			if lon >= mid {
				ch |= 1 << (4 - bit)
				lonMin = mid
			} else {
				lonMax = mid
			}
		} else {
			mid := (latMin + latMax) / 2
			if lat >= mid {
				ch |= 1 << (4 - bit)
				latMin = mid
			} else {
				latMax = mid
			}
		}
		even = !even
		if bit < 4 {
			bit++
		} else {
			b.WriteByte(geohashBase32[ch])
			bit, ch = 0, 0
		}
	}
	return b.String()
}

// addGeohashSiblings writes, for each geopoint override, the geohash prefix of
// the point at every precision 1..MaxGeohashPrecision under the suffixed keys.
func addGeohashSiblings(m map[string]any, overrides map[string]FieldOpts) {
	for key, opts := range overrides {
		if opts.Type == TypeGeopoint {
			addGeohashSibling(m, key)
		}
	}
}

func addGeohashSibling(m map[string]any, dottedPath string) {
	parts := strings.Split(dottedPath, ".")
	parent := m
	for _, p := range parts[:len(parts)-1] {
		next, ok := parent[p].(map[string]any)
		if !ok {
			return
		}
		parent = next
	}
	leaf := parts[len(parts)-1]
	obj, ok := parent[leaf].(map[string]any)
	if !ok {
		return
	}
	lon, hasLon := obj["longitude"].(float64)
	lat, hasLat := obj["latitude"].(float64)
	if !hasLon || !hasLat {
		return
	}
	for p := 1; p <= MaxGeohashPrecision; p++ {
		parent[GeohashField(leaf, p)] = encodeGeohash(lat, lon, p)
	}
}
