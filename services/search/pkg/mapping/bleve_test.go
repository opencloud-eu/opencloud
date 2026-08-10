package mapping

import (
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type bleveDoc struct {
	Name      string    `json:"Name"`
	Content   string    `json:"Content"`
	Tags      []string  `json:"Tags"`
	Size      uint64    `json:"Size"`
	Deleted   bool      `json:"Deleted"`
	CreatedAt time.Time `json:"CreatedAt"`
	Nested    *nested   `json:"nested,omitempty"`
}

type nested struct {
	Artist string `json:"artist"`
	Year   int    `json:"year"`
}

var _ = Describe("BleveBuildMapping", func() {
	It("falls back to text for wildcard fields", func() {
		// bleve wildcard falls back to keyword-ish text (bleve has no wildcard type).
		type doc struct {
			Mime string `json:"mime"`
		}
		dm, err := BleveBuildMapping(reflect.TypeFor[doc](), map[string]FieldOpts{"mime": {Type: TypeWildcard}})
		Expect(err).ToNot(HaveOccurred())
		fms := dm.Properties["mime"].Fields
		Expect(fms).To(HaveLen(1))
		Expect(fms[0].Type).To(Equal("text"), "wildcard should map to text")
	})

	DescribeTable("infers field types",
		func(field, wantType string) {
			dm, err := BleveBuildMapping(reflect.TypeFor[bleveDoc](), nil)
			Expect(err).ToNot(HaveOccurred())
			prop := dm.Properties[field]
			Expect(prop).ToNot(BeNil(), "missing property %q", field)
			Expect(prop.Fields).ToNot(BeEmpty(), "%q: no field mappings", field)
			Expect(prop.Fields[0].Type).To(Equal(wantType), "%q type", field)
		},
		Entry("Name", "Name", "text"),
		Entry("Content", "Content", "text"),
		Entry("Tags", "Tags", "text"),
		Entry("Size", "Size", "number"),
		Entry("Deleted", "Deleted", "boolean"),
		Entry("CreatedAt", "CreatedAt", "datetime"),
	)

	It("maps a nested struct as a sub-document", func() {
		dm, err := BleveBuildMapping(reflect.TypeFor[bleveDoc](), nil)
		Expect(err).ToNot(HaveOccurred())
		sub := dm.Properties["nested"]
		Expect(sub).ToNot(BeNil(), "missing nested sub-document")
		Expect(sub.Properties["artist"]).ToNot(BeNil())
		Expect(sub.Properties["year"]).ToNot(BeNil())
		Expect(sub.Properties["artist"].Fields[0].Type).To(Equal("text"), "nested.artist")
		Expect(sub.Properties["year"].Fields[0].Type).To(Equal("number"), "nested.year")
	})

	It("applies field overrides", func() {
		True, False := true, false
		dm, err := BleveBuildMapping(reflect.TypeFor[bleveDoc](), map[string]FieldOpts{
			"Name":    {CaseInsensitive: &True},
			"Content": {Type: TypeFulltext},
			"Tags":    {CaseInsensitive: &True, IncludeInAll: &False},
		})
		Expect(err).ToNot(HaveOccurred())
		// Name: case-preserved base keyword + lowercased sibling.
		Expect(dm.Properties["Name"]).ToNot(BeNil(), "Name base field")
		Expect(dm.Properties["Name"].Fields[0].Analyzer).To(Equal("keyword"), "Name base is a keyword")
		Expect(dm.Properties["Name"].Fields[0].Store).To(BeTrue(), "Name base is stored (returned)")
		Expect(dm.Properties["Name_lowercase"]).ToNot(BeNil(), "Name_lowercase sibling")
		// The sibling is a search-only shadow: indexed but never stored, kept out
		// of _all, no doc values (the base is what we return).
		sibling := dm.Properties["Name_lowercase"].Fields[0]
		Expect(sibling.Index).To(BeTrue(), "Name_lowercase is indexed")
		Expect(sibling.Store).To(BeFalse(), "Name_lowercase is not stored")
		Expect(sibling.IncludeInAll).To(BeFalse(), "Name_lowercase is out of _all")
		Expect(sibling.DocValues).To(BeFalse(), "Name_lowercase has no doc values")
		contentField := dm.Properties["Content"].Fields[0]
		Expect(contentField.Analyzer).To(Equal("fulltext"), "Content analyzer")
		Expect(contentField.IncludeInAll).To(BeFalse(), "Content IncludeInAll should default to false for fulltext type")
		// Tags: base + lowercased sibling, both honoring the IncludeInAll override.
		Expect(dm.Properties["Tags"].Fields[0].IncludeInAll).To(BeFalse(), "Tags base IncludeInAll honored")
		Expect(dm.Properties["Tags_lowercase"].Fields[0].IncludeInAll).To(BeFalse(), "Tags sibling IncludeInAll honored")
	})

	It("builds an object sub-document plus a geopoint sibling", func() {
		type geoDoc struct {
			Location *struct {
				Lon *float64 `json:"longitude,omitempty"`
				Lat *float64 `json:"latitude,omitempty"`
				Alt *float64 `json:"altitude,omitempty"`
			} `json:"location,omitempty"`
		}
		dm, err := BleveBuildMapping(reflect.TypeFor[geoDoc](), map[string]FieldOpts{
			"location": {Type: TypeGeopoint},
		})
		Expect(err).ToNot(HaveOccurred())
		// Original facet stays as an object sub-document with numeric
		// sub-properties - for data retrieval via hit.Fields and ordinary
		// numeric queries.
		loc := dm.Properties["location"]
		Expect(loc).ToNot(BeNil(), "location sub-document missing")
		Expect(loc.Fields).To(BeEmpty(), "location should not carry field mappings directly")
		for _, sub := range []string{"longitude", "latitude", "altitude"} {
			prop, ok := loc.Properties[sub]
			Expect(ok).To(BeTrue(), "missing sub-field %q under location", sub)
			Expect(prop.Fields).ToNot(BeEmpty(), "location.%s Fields", sub)
			Expect(prop.Fields[0].Type).To(Equal("number"), "location.%s type", sub)
		}
		// Sibling geopoint at "<name>_geopoint" for geo-distance queries.
		sibling := dm.Properties["location"+GeopointSuffix]
		Expect(sibling).ToNot(BeNil(), "location%s missing", GeopointSuffix)
		Expect(sibling.Fields).ToNot(BeEmpty(), "location%s Fields", GeopointSuffix)
		Expect(sibling.Fields[0].Type).To(Equal("geopoint"), "location%s type", GeopointSuffix)
	})
})
