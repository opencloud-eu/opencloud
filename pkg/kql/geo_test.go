package kql_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
)

// firstNode parses q and returns its single restriction node.
func firstNode(q string) (ast.Node, error) {
	a, err := kql.Builder{}.Build(q)
	if err != nil {
		return nil, err
	}
	Expect(a.Nodes).To(HaveLen(1))
	return a.Nodes[0], nil
}

var _ = Describe("geo KQL predicates", func() {
	It("parses geo.distance into a GeoDistanceNode", func() {
		n, err := firstNode("location:geo.distance(48.2082, 16.3738, 5km)")
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(BeAssignableToTypeOf(&ast.GeoDistanceNode{}))
		d := n.(*ast.GeoDistanceNode)
		Expect(d.Key).To(Equal("location"))
		Expect(d.Lat).To(Equal(48.2082))
		Expect(d.Lon).To(Equal(16.3738))
		Expect(d.Radius).To(Equal(5000.0))
	})

	DescribeTable("normalises the radius unit to meters",
		func(radius string, meters float64) {
			n, err := firstNode("location:geo.distance(1, 2, " + radius + ")")
			Expect(err).ToNot(HaveOccurred())
			Expect(n.(*ast.GeoDistanceNode).Radius).To(Equal(meters))
		},
		Entry("kilometers", "5km", 5000.0),
		Entry("meters", "500m", 500.0),
		Entry("miles", "3mi", 3*1609.344),
		Entry("fractional km", "1.5km", 1500.0),
	)

	It("parses geo.bbox into a GeoBoundingBoxNode", func() {
		n, err := firstNode("location:geo.bbox(47.9, 16.1, 48.3, 16.5)")
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(&ast.GeoBoundingBoxNode{
			Base:   n.(*ast.GeoBoundingBoxNode).Base,
			Key:    "location",
			MinLat: 47.9, MinLon: 16.1, MaxLat: 48.3, MaxLon: 16.5,
		}))
	})

	It("parses geo.polygon into a GeoPolygonNode with its vertices", func() {
		n, err := firstNode("location:geo.polygon(48.3 16.1, 48.3 16.5, 47.9 16.5)")
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(BeAssignableToTypeOf(&ast.GeoPolygonNode{}))
		p := n.(*ast.GeoPolygonNode)
		Expect(p.Key).To(Equal("location"))
		Expect(p.Points).To(Equal([]ast.GeoPoint{
			{Lat: 48.3, Lon: 16.1}, {Lat: 48.3, Lon: 16.5}, {Lat: 47.9, Lon: 16.5},
		}))
	})

	It("composes a geo predicate with a plain restriction", func() {
		a, err := kql.Builder{}.Build("mediatype:audio AND location:geo.bbox(47.9, 16.1, 48.3, 16.5)")
		Expect(err).ToNot(HaveOccurred())
		Expect(a.Nodes).To(HaveLen(3)) // string, operator, geo
		Expect(a.Nodes[2]).To(BeAssignableToTypeOf(&ast.GeoBoundingBoxNode{}))
	})

	DescribeTable("rejects malformed geo predicates",
		func(q string) {
			_, err := kql.Builder{}.Build(q)
			Expect(err).To(HaveOccurred())
		},
		Entry("distance without a unit", "location:geo.distance(1, 2, 5)"),
		Entry("distance with too few args", "location:geo.distance(1, 2)"),
		Entry("distance with a non-numeric coord", "location:geo.distance(x, 2, 5km)"),
		Entry("bbox with too few args", "location:geo.bbox(1, 2, 3)"),
		Entry("polygon with fewer than three points", "location:geo.polygon(1 2, 3 4)"),
		Entry("polygon point without a lon", "location:geo.polygon(1, 2, 3)"),
	)
})
