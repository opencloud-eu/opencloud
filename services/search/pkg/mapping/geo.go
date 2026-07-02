package mapping

import "strings"

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
