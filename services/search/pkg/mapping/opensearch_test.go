package mapping

import (
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type osDoc struct {
	ID        string    `json:"ID"`
	Size      uint64    `json:"Size"`
	Deleted   bool      `json:"Deleted"`
	CreatedAt time.Time `json:"CreatedAt"`
	Rating    float64   `json:"Rating"`
	Nested    *struct {
		Artist string `json:"artist"`
		Year   int32  `json:"year"`
	} `json:"nested,omitempty"`
}

var _ = Describe("OpenSearchBuildMapping", func() {
	DescribeTable("maps numeric Go types to OpenSearch types",
		func(field, wantType string) {
			type doc struct {
				A int8    `json:"a"`
				B int16   `json:"b"`
				C int32   `json:"c"`
				D int64   `json:"d"`
				E uint8   `json:"e"`
				F uint64  `json:"f"`
				G float32 `json:"g"`
				H float64 `json:"h"`
			}
			props, err := OpenSearchBuildMapping(reflect.TypeFor[doc](), nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(props[field].(map[string]any)["type"]).To(Equal(wantType), "%s type", field)
		},
		Entry("int8 -> short", "a", "short"),
		Entry("int16 -> short", "b", "short"),
		Entry("int32 -> integer", "c", "integer"),
		Entry("int64 -> long", "d", "long"),
		Entry("uint8 -> short", "e", "short"),
		Entry("uint64 -> long", "f", "long"),
		Entry("float32 -> float", "g", "float"),
		Entry("float64 -> double", "h", "double"),
	)

	DescribeTable("infers field types",
		func(field, wantType string) {
			props, err := OpenSearchBuildMapping(reflect.TypeFor[osDoc](), nil)
			Expect(err).ToNot(HaveOccurred())
			m, ok := props[field].(map[string]any)
			Expect(ok).To(BeTrue(), "%s: missing or not a map: %#v", field, props[field])
			Expect(m["type"]).To(Equal(wantType), "%s type", field)
		},
		Entry("ID -> keyword", "ID", "keyword"),
		Entry("Size -> long", "Size", "long"),
		Entry("Deleted -> boolean", "Deleted", "boolean"),
		Entry("CreatedAt -> date", "CreatedAt", "date"),
		Entry("Rating -> double", "Rating", "double"),
	)

	It("maps nested structs with their sub-properties", func() {
		props, err := OpenSearchBuildMapping(reflect.TypeFor[osDoc](), nil)
		Expect(err).ToNot(HaveOccurred())
		nested, ok := props["nested"].(map[string]any)
		Expect(ok).To(BeTrue(), "nested: not a map: %#v", props["nested"])
		sub, ok := nested["properties"].(map[string]any)
		Expect(ok).To(BeTrue(), "nested.properties: missing: %#v", nested)
		artist, ok := sub["artist"].(map[string]any)
		Expect(ok).To(BeTrue(), "nested.artist: %#v", sub)
		Expect(artist["type"]).To(Equal("keyword"), "nested.artist.type")
		year, ok := sub["year"].(map[string]any)
		Expect(ok).To(BeTrue(), "nested.year: %#v", sub)
		Expect(year["type"]).To(Equal("integer"), "nested.year.type (int32 -> integer expected)")
	})

	It("applies field overrides", func() {
		True := true
		type doc struct {
			Name     string `json:"Name"`
			Content  string `json:"Content"`
			Path     string `json:"Path"`
			MimeType string `json:"MimeType"`
		}
		props, err := OpenSearchBuildMapping(reflect.TypeFor[doc](), map[string]FieldOpts{
			"Name":     {CaseInsensitive: &True},
			"Content":  {Type: TypeFulltext},
			"Path":     {Type: TypePath, CaseInsensitive: &True},
			"MimeType": {Type: TypeWildcard},
		})
		Expect(err).ToNot(HaveOccurred())
		// Name: case-preserved keyword base + lowercased search-only sibling.
		Expect(props["Name"]).To(Equal(map[string]any{"type": "keyword"}))
		Expect(props["Name_lowercase"]).To(Equal(map[string]any{"type": "keyword", "doc_values": false}))
		content := props["Content"].(map[string]any)
		Expect(content["type"]).To(Equal("text"), "Content: %#v", content)
		Expect(content["term_vector"]).To(Equal("with_positions_offsets"), "Content: %#v", content)
		Expect(content["analyzer"]).To(Equal(WordsAnalyzer), "Content uses the words analyzer, like bleve")
		// Path: path_hierarchy base + lowercased sibling, both case-preserving.
		Expect(props["Path"]).To(Equal(map[string]any{"type": "text", "analyzer": "path_hierarchy"}))
		Expect(props["Path_lowercase"]).To(Equal(map[string]any{"type": "text", "analyzer": "path_hierarchy"}))
		mime := props["MimeType"].(map[string]any)
		Expect(mime["type"]).To(Equal("wildcard"), "MimeType: %#v", mime)
	})

	It("gives a keyword its lowercase and words siblings by default", func() {
		type doc struct {
			Name string `json:"Name"`
		}
		props, err := OpenSearchBuildMapping(reflect.TypeFor[doc](), nil)
		Expect(err).ToNot(HaveOccurred())
		// the base stays a keyword, the words go to their own sibling
		Expect(props["Name"]).To(Equal(map[string]any{"type": "keyword"}))
		Expect(props["Name_lowercase"]).To(Equal(map[string]any{"type": "keyword", "doc_values": false}))
		Expect(props["Name_words"]).To(Equal(map[string]any{"type": "text", "analyzer": WordsAnalyzer}))
	})

	It("leaves a keyword one whole value with NoWordBreaker", func() {
		True := true
		type doc struct {
			Tag string `json:"Tag"`
		}
		props, err := OpenSearchBuildMapping(reflect.TypeFor[doc](), map[string]FieldOpts{
			"Tag": {NoWordBreaker: &True},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(props).To(HaveKey("Tag_lowercase"))
		Expect(props).ToNot(HaveKey("Tag_words"))
	})

	It("builds an object plus a geo_point sibling for geopoints", func() {
		type doc struct {
			Location *struct {
				Lon float64 `json:"longitude"`
				Lat float64 `json:"latitude"`
				Alt float64 `json:"altitude"`
			} `json:"location,omitempty"`
		}
		props, err := OpenSearchBuildMapping(reflect.TypeFor[doc](), map[string]FieldOpts{
			"location": {Type: TypeGeopoint},
		})
		Expect(err).ToNot(HaveOccurred())
		// Object for libregraph-shape data retrieval.
		loc, ok := props["location"].(map[string]any)
		Expect(ok).To(BeTrue(), "location: %#v", props["location"])
		sub, ok := loc["properties"].(map[string]any)
		Expect(ok).To(BeTrue(), "location should have numeric sub-properties, got %#v", loc)
		for _, k := range []string{"longitude", "latitude", "altitude"} {
			prop, ok := sub[k].(map[string]any)
			Expect(ok).To(BeTrue(), "location.%s: %#v", k, sub[k])
			Expect(prop["type"]).To(Equal("double"), "location.%s: %#v", k, sub[k])
		}
		// Sibling geo_point for spatial queries.
		gp, ok := props["location"+GeopointSuffix].(map[string]any)
		Expect(ok).To(BeTrue(), "location%s: %#v", GeopointSuffix, props["location"+GeopointSuffix])
		Expect(gp["type"]).To(Equal("geo_point"), "location%s.type", GeopointSuffix)
	})
})
