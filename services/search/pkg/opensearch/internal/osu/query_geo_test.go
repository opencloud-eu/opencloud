package osu_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
)

var _ = Describe("geo query builders", func() {
	It("builds a geo_distance query", func() {
		q := osu.NewGeoDistanceQuery("location_geopoint").Distance("5000m").Point(48.2, 16.3)
		Expect(q.Map()).To(Equal(map[string]any{
			"geo_distance": map[string]any{
				"distance":          "5000m",
				"location_geopoint": map[string]any{"lat": 48.2, "lon": 16.3},
			},
		}))
	})

	It("builds a geo_bounding_box query with top-left and bottom-right corners", func() {
		q := osu.NewGeoBoundingBoxQuery("location_geopoint").Box(47.9, 16.1, 48.3, 16.5)
		Expect(q.Map()).To(Equal(map[string]any{
			"geo_bounding_box": map[string]any{
				"location_geopoint": map[string]any{
					"top_left":     map[string]any{"lat": 48.3, "lon": 16.1},
					"bottom_right": map[string]any{"lat": 47.9, "lon": 16.5},
				},
			},
		}))
	})

	It("builds a geo_shape polygon query with a closed GeoJSON ring", func() {
		q := osu.NewGeoPolygonQuery("location_geopoint").
			Point(48.3, 16.1).Point(48.3, 16.5).Point(47.9, 16.5)
		Expect(q.Map()).To(Equal(map[string]any{
			"geo_shape": map[string]any{
				"location_geopoint": map[string]any{
					"shape": map[string]any{
						"type": "polygon",
						// GeoJSON [lon, lat], ring closed back to the first point.
						"coordinates": [][][2]float64{{
							{16.1, 48.3}, {16.5, 48.3}, {16.5, 47.9}, {16.1, 48.3},
						}},
					},
					"relation": "intersects",
				},
			},
		}))
	})
})
