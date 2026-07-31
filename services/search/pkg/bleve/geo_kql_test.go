package bleve_test

import (
	bleveSearch "github.com/blevesearch/bleve/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qbleve "github.com/opencloud-eu/opencloud/services/search/pkg/query/bleve"
)

var _ = Describe("Geo KQL predicates", func() {
	// Nuremberg-ish point indexed by geoFixture (see geo_verify_test.go).
	const lat, lon = 49.48675890884328, 11.103870357204285

	// hits compiles the KQL query through the full bleve pipeline and searches a
	// fresh fixture holding the single point above.
	hits := func(kqlQuery string) int {
		q, err := qbleve.DefaultCreator.Create(kqlQuery)
		Expect(err).ToNot(HaveOccurred())
		res, err := geoFixture(lon, lat, 1047.7).Search(bleveSearch.NewSearchRequest(q))
		Expect(err).ToNot(HaveOccurred())
		return len(res.Hits)
	}

	DescribeTable("geo.distance matches within the radius and misses outside",
		func(kqlQuery string, want int) {
			Expect(hits(kqlQuery)).To(Equal(want))
		},
		Entry("near", "location:geo.distance(49.48675890884328, 11.103870357204285, 10km)", 1),
		Entry("far (Berlin, ~400km)", "location:geo.distance(52.520008, 13.404954, 10km)", 0),
	)

	It("geo.bbox matches a box around the point and misses a distant box", func() {
		Expect(hits("location:geo.bbox(49.0, 11.0, 50.0, 12.0)")).To(Equal(1))
		Expect(hits("location:geo.bbox(52.0, 13.0, 53.0, 14.0)")).To(Equal(0))
	})

	It("geo.polygon matches a polygon around the point and misses a distant one", func() {
		Expect(hits("location:geo.polygon(49.0 11.0, 50.0 11.0, 50.0 12.0, 49.0 12.0)")).To(Equal(1))
		Expect(hits("location:geo.polygon(52.0 13.0, 53.0 13.0, 53.0 14.0, 52.0 14.0)")).To(Equal(0))
	})

	It("composes with other predicates via AND", func() {
		near := "location:geo.distance(49.48675890884328, 11.103870357204285, 10km)"
		Expect(hits(near + " AND location:geo.bbox(49.0, 11.0, 50.0, 12.0)")).To(Equal(1))
		Expect(hits(near + " AND location:geo.bbox(52.0, 13.0, 53.0, 14.0)")).To(Equal(0))
	})

	It("matches a bbox given with inverted latitude order", func() {
		// Latitude args inverted (max, ..., min): normalised centrally so bleve
		// gets a well-formed box instead of erroring. Same box as the passing case.
		Expect(hits("location:geo.bbox(50.0, 11.0, 49.0, 12.0)")).To(Equal(1))
	})

	It("rejects a geo predicate on a non-geopoint field", func() {
		_, err := qbleve.DefaultCreator.Create("name:geo.distance(48.2, 16.3, 5km)")
		Expect(err).To(HaveOccurred())
	})
})
