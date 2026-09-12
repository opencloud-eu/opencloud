package bleve

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

var _ = Describe("geohash", func() {
	// reference vector from the geohash spec, Lucene and OpenSearch agree
	const refLat, refLon = 57.64911, 10.40744

	It("matches the canonical vector", func() {
		Expect(encodeGeohash(refLat, refLon, 11)).To(Equal("u4pruydqqvj"))
	})

	It("is a prefix code", func() {
		full := encodeGeohash(refLat, refLon, 12)
		for p := 1; p <= 12; p++ {
			Expect(encodeGeohash(refLat, refLon, p)).To(Equal(full[:p]))
		}
	})

	It("writes the geohash next to the geopoint sibling of every geopoint override", func() {
		// journey.start is the dotted-path example of addGeopointSibling, broken
		// has no usable lat/lon and gets no geohash, like it gets no sibling
		overrides := map[string]searchmapping.FieldOpts{
			"location":      {Type: searchmapping.TypeGeopoint},
			"journey.start": {Type: searchmapping.TypeGeopoint},
			"broken":        {Type: searchmapping.TypeGeopoint},
		}
		doc := map[string]any{
			"location_geopoint": map[string]any{"lat": refLat, "lon": refLon},
			"journey": map[string]any{
				"start_geopoint": map[string]any{"lat": 0.0, "lon": 0.0},
			},
			"broken_geopoint": map[string]any{"lat": "x"},
		}
		addGeohashValues(doc, overrides)
		Expect(doc["location_geohash"]).To(Equal(encodeGeohash(refLat, refLon, 12)))
		Expect(doc["journey"].(map[string]any)["start_geohash"]).To(Equal("s00000000000"))
		Expect(doc).ToNot(HaveKey("broken_geohash"))
	})
})
