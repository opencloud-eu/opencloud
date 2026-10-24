package bleve_test

import (
	bleveSearch "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// geoFixture builds an in-memory bleve index with a single resource carrying
// the given lon/lat/alt, indexed through the full bleve pipeline.
func geoFixture(lon, lat, alt float64) bleveSearch.Index {
	idxMapping, err := bleve.NewMapping()
	Expect(err).ToNot(HaveOccurred())
	idx, err := bleveSearch.NewMemOnly(idxMapping)
	Expect(err).ToNot(HaveOccurred())

	r := search.Resource{
		ID: "x",
		Document: content.Document{
			Name: "team.jpg",
			Location: &libregraph.GeoCoordinates{
				Longitude: &lon,
				Latitude:  &lat,
				Altitude:  &alt,
			},
		},
	}
	doc, err := mapping.PrepareForIndex(r, r.SearchFieldOverrides())
	Expect(err).ToNot(HaveOccurred())
	Expect(idx.Index(r.ID, doc)).To(Succeed())
	return idx
}

var _ = Describe("Location geo queries", func() {
	// Every Location subfield (including altitude) must end up in hit.Fields
	// when a Resource is indexed through the full bleve pipeline. This is the
	// invariant the Move / Delete / Restore round-trip depends on.
	It("round-trips every Location subfield into hit.Fields", func() {
		idx := geoFixture(11.103870357204285, 49.48675890884328, 1047.7)

		req := bleveSearch.NewSearchRequest(bleveSearch.NewMatchAllQuery())
		req.Fields = []string{"*"}
		res, err := idx.Search(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Hits).ToNot(BeEmpty())

		for _, k := range []string{"location.longitude", "location.latitude", "location.altitude"} {
			Expect(res.Hits[0].Fields).To(HaveKey(k))
		}
	})

	It("matches a latitude numeric range and misses outside it", func() {
		idx := geoFixture(11.1, 49.48, 1000)

		min, max := 49.0, 50.0
		incl := true
		q := query.NewNumericRangeInclusiveQuery(&min, &max, &incl, &incl)
		q.SetField("location.latitude")
		res, err := idx.Search(bleveSearch.NewSearchRequest(q))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Hits).To(HaveLen(1))

		lowMin, lowMax := 0.0, 10.0
		q2 := query.NewNumericRangeInclusiveQuery(&lowMin, &lowMax, &incl, &incl)
		q2.SetField("location.latitude")
		res, err = idx.Search(bleveSearch.NewSearchRequest(q2))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Hits).To(BeEmpty())
	})

	It("matches a longitude numeric range", func() {
		idx := geoFixture(11.1, 49.48, 1000)

		min, max := 11.0, 12.0
		incl := true
		q := query.NewNumericRangeInclusiveQuery(&min, &max, &incl, &incl)
		q.SetField("location.longitude")
		res, err := idx.Search(bleveSearch.NewSearchRequest(q))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Hits).To(HaveLen(1))
	})

	It("matches an altitude lower-bound range and misses above it", func() {
		idx := geoFixture(11.1, 49.48, 1047.7)

		min := 1000.0
		incl := true
		q := query.NewNumericRangeInclusiveQuery(&min, nil, &incl, nil)
		q.SetField("location.altitude")
		res, err := idx.Search(bleveSearch.NewSearchRequest(q))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Hits).To(HaveLen(1))

		highMin := 2000.0
		q2 := query.NewNumericRangeInclusiveQuery(&highMin, nil, &incl, nil)
		q2.SetField("location.altitude")
		res, err = idx.Search(bleveSearch.NewSearchRequest(q2))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Hits).To(BeEmpty())
	})

	It("matches a geo-distance query near the point and misses far away", func() {
		// Nuremberg-ish coordinates.
		idx := geoFixture(11.103870357204285, 49.48675890884328, 1047.7)

		// 10 km radius around the indexed point should match.
		near := query.NewGeoDistanceQuery(11.103870357204285, 49.48675890884328, "10km")
		near.SetField("location" + mapping.GeopointSuffix)
		res, err := idx.Search(bleveSearch.NewSearchRequest(near))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Hits).To(HaveLen(1))

		// Far away (Berlin, ~400 km) with a 10 km radius should miss.
		far := query.NewGeoDistanceQuery(13.404954, 52.520008, "10km")
		far.SetField("location" + mapping.GeopointSuffix)
		res, err = idx.Search(bleveSearch.NewSearchRequest(far))
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Hits).To(BeEmpty())
	})
})
