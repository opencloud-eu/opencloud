package bleve

import (
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"

	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// A geohash sibling of every geopoint field, bleve only: OpenSearch can
// bucket a geohash_grid aggregation on the geo_point itself, bleve cannot.
// The geohash is indexed with the geohash analyzer, one depth-tagged term per
// precision ("1/u", "2/u4", ...), so the terms with the prefix "<precision>/"
// are the cells at that precision. Nothing queries it yet (#3272).
const (
	geohashAnalyzer  = "geohash"
	geohashSuffix    = "_geohash"
	geohashPrecision = 12
	geohashBase32    = "0123456789bcdefghjkmnpqrstuvwxyz"
)

// encodeGeohash matches Lucene/OpenSearch so both engines bucket into the
// same cells
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

// geopointFields yields the parent path and leaf name of every TypeGeopoint
// override, the same fields addGeopointSiblings gives a geopoint sibling
func geopointFields(overrides map[string]searchmapping.FieldOpts, fn func(parents []string, leaf string)) {
	for key, opts := range overrides {
		if opts.Type == searchmapping.TypeGeopoint {
			parts := strings.Split(key, ".")
			fn(parts[:len(parts)-1], parts[len(parts)-1])
		}
	}
}

// addGeohashFields maps a <name>_geohash field next to the <name>_geopoint
// field of every geopoint override
func addGeohashFields(dm *mapping.DocumentMapping, overrides map[string]searchmapping.FieldOpts) {
	geopointFields(overrides, func(parents []string, leaf string) {
		parent := dm
		for _, p := range parents {
			if parent = parent.Properties[p]; parent == nil {
				return
			}
		}
		fm := bleve.NewTextFieldMapping()
		fm.Analyzer = geohashAnalyzer
		fm.Store = false
		fm.IncludeInAll = false
		fm.IncludeTermVectors = false
		parent.AddFieldMappingsAt(leaf+geohashSuffix, fm)
	})
}

// addGeohashValues writes the geohash next to the {lat, lon} sibling of every
// geopoint override in a prepared document
func addGeohashValues(doc map[string]any, overrides map[string]searchmapping.FieldOpts) {
	geopointFields(overrides, func(parents []string, leaf string) {
		parent := doc
		for _, p := range parents {
			next, ok := parent[p].(map[string]any)
			if !ok {
				return
			}
			parent = next
		}
		obj, ok := parent[leaf+searchmapping.GeopointSuffix].(map[string]any)
		if !ok {
			return
		}
		lat, hasLat := obj["lat"].(float64)
		lon, hasLon := obj["lon"].(float64)
		if hasLat && hasLon {
			parent[leaf+geohashSuffix] = encodeGeohash(lat, lon, geohashPrecision)
		}
	})
}
