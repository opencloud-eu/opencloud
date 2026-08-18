package mapping

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BuildMapping geopoint errors", func() {
	// build-mapping error paths: an override that doesn't fit the Go field.
	It("errors for geopoint on a non-struct field on both backends", func() {
		type doc struct {
			Name string `json:"name"`
		}
		// Geopoint on a non-struct field must error on both backends.
		_, err := BleveBuildMapping(reflect.TypeFor[doc](), map[string]FieldOpts{"name": {Type: TypeGeopoint}})
		Expect(err).To(HaveOccurred(), "bleve: expected error for geopoint on string field")
		_, err = OpenSearchBuildMapping(reflect.TypeFor[doc](), map[string]FieldOpts{"name": {Type: TypeGeopoint}})
		Expect(err).To(HaveOccurred(), "opensearch: expected error for geopoint on string field")
	})
})

var _ = Describe("addGeopointSibling", func() {
	It("bails without panicking when the intermediate is missing", func() {
		m := map[string]any{"journey": "not-a-map"}
		addGeopointSibling(m, "journey.start") // must bail, not panic
		Expect(m).ToNot(HaveKey("journey.start" + GeopointSuffix))
	})
})

var _ = Describe("PrepareForIndex geopoint", func() {
	It("adds a geopoint sibling", func() {
		type geoDoc struct {
			Location *struct {
				Longitude *float64 `json:"longitude,omitempty"`
				Latitude  *float64 `json:"latitude,omitempty"`
				Altitude  *float64 `json:"altitude,omitempty"`
			} `json:"location,omitempty"`
		}
		lon, lat, alt := 11.1, 49.4, 1047.7
		doc := geoDoc{Location: &struct {
			Longitude *float64 `json:"longitude,omitempty"`
			Latitude  *float64 `json:"latitude,omitempty"`
			Altitude  *float64 `json:"altitude,omitempty"`
		}{Longitude: &lon, Latitude: &lat, Altitude: &alt}}

		m, err := PrepareForIndex(doc, map[string]FieldOpts{
			"location": {Type: TypeGeopoint},
		})
		Expect(err).ToNot(HaveOccurred())

		// Original location object stays untouched (full libregraph shape).
		orig, ok := m["location"].(map[string]any)
		Expect(ok).To(BeTrue(), "expected location object preserved, got %T", m["location"])
		Expect(orig["longitude"]).To(Equal(lon))
		Expect(orig["latitude"]).To(Equal(lat))
		Expect(orig["altitude"]).To(Equal(alt))

		// Sibling location_geopoint has {lat, lon} for the geo indices.
		gp, ok := m["location"+GeopointSuffix].(map[string]any)
		Expect(ok).To(BeTrue(), "expected location_geopoint sibling, got %T", m["location"+GeopointSuffix])
		Expect(gp["lat"]).To(Equal(lat))
		Expect(gp["lon"]).To(Equal(lon))
	})

	It("skips incomplete geopoints", func() {
		type geoDoc struct {
			Location *struct {
				Altitude *float64 `json:"altitude,omitempty"`
			} `json:"location,omitempty"`
		}
		alt := 100.0
		doc := geoDoc{Location: &struct {
			Altitude *float64 `json:"altitude,omitempty"`
		}{Altitude: &alt}}

		m, err := PrepareForIndex(doc, map[string]FieldOpts{
			"location": {Type: TypeGeopoint},
		})
		Expect(err).ToNot(HaveOccurred())
		// Original stays (altitude alone is still useful metadata).
		Expect(m).To(HaveKey("location"), "location should still be present when only altitude is set")
		// No sibling without both lon and lat.
		Expect(m).ToNot(HaveKey("location"+GeopointSuffix), "no sibling expected")
	})

	It("writes no sibling without the geopoint override", func() {
		type geoDoc struct {
			Location *struct {
				Longitude *float64 `json:"longitude,omitempty"`
				Latitude  *float64 `json:"latitude,omitempty"`
			} `json:"location,omitempty"`
		}
		lon, lat := 11.1, 49.4
		doc := geoDoc{Location: &struct {
			Longitude *float64 `json:"longitude,omitempty"`
			Latitude  *float64 `json:"latitude,omitempty"`
		}{Longitude: &lon, Latitude: &lat}}

		m, err := PrepareForIndex(doc, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(m).ToNot(HaveKey("location"+GeopointSuffix), "no sibling expected without override")
	})

	It("handles nested geopoints via the dotted-path walker", func() {
		// journey.start and journey.end - two geopoints in the same facet,
		// demonstrating the dotted-path walker.
		type geo struct {
			Longitude *float64 `json:"longitude,omitempty"`
			Latitude  *float64 `json:"latitude,omitempty"`
		}
		type journey struct {
			Start *geo `json:"start,omitempty"`
			End   *geo `json:"end,omitempty"`
		}
		type doc struct {
			Journey *journey `json:"journey,omitempty"`
		}
		slon, slat := 11.0, 49.0
		elon, elat := 13.4, 52.5
		d := doc{Journey: &journey{
			Start: &geo{Longitude: &slon, Latitude: &slat},
			End:   &geo{Longitude: &elon, Latitude: &elat},
		}}

		m, err := PrepareForIndex(d, map[string]FieldOpts{
			"journey.start": {Type: TypeGeopoint},
			"journey.end":   {Type: TypeGeopoint},
		})
		Expect(err).ToNot(HaveOccurred())
		j, ok := m["journey"].(map[string]any)
		Expect(ok).To(BeTrue(), "journey not an object: %T", m["journey"])
		startGp, ok := j["start"+GeopointSuffix].(map[string]any)
		Expect(ok).To(BeTrue(), "journey.start sibling: %#v", j["start"+GeopointSuffix])
		Expect(startGp["lat"]).To(Equal(slat))
		Expect(startGp["lon"]).To(Equal(slon))
		endGp, ok := j["end"+GeopointSuffix].(map[string]any)
		Expect(ok).To(BeTrue(), "journey.end sibling: %#v", j["end"+GeopointSuffix])
		Expect(endGp["lat"]).To(Equal(elat))
		Expect(endGp["lon"]).To(Equal(elon))
	})
})

var _ = Describe("geohash", func() {
	// Wikipedia's canonical reference point.
	const refLat, refLon = 57.64911, 10.40744

	It("encodes the canonical geohash vector (matches Lucene/OpenSearch)", func() {
		Expect(encodeGeohash(refLat, refLon, 11)).To(Equal("u4pruydqqvj"))
	})

	It("is prefix-consistent across precisions", func() {
		full := encodeGeohash(refLat, refLon, 11)
		for p := 1; p <= 11; p++ {
			Expect(encodeGeohash(refLat, refLon, p)).To(Equal(full[:p]),
				"precision %d must be the %d-char prefix", p, p)
		}
	})

	It("adds geohash prefix siblings for a geopoint at every precision", func() {
		type geoDoc struct {
			Location *struct {
				Longitude *float64 `json:"longitude,omitempty"`
				Latitude  *float64 `json:"latitude,omitempty"`
			} `json:"location,omitempty"`
		}
		lon, lat := refLon, refLat
		doc := geoDoc{Location: &struct {
			Longitude *float64 `json:"longitude,omitempty"`
			Latitude  *float64 `json:"latitude,omitempty"`
		}{Longitude: &lon, Latitude: &lat}}

		m, err := PrepareForIndex(doc, map[string]FieldOpts{"location": {Type: TypeGeopoint}})
		Expect(err).ToNot(HaveOccurred())
		for p := 1; p <= MaxGeohashPrecision; p++ {
			Expect(m[GeohashField("location", p)]).To(Equal(encodeGeohash(lat, lon, p)),
				"location_geohash_%d", p)
		}
		Expect(m[GeohashField("location", 5)]).To(Equal("u4pru"))
	})
})
