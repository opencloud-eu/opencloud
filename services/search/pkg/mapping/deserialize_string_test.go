package mapping

import (
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type stringFacet struct {
	Artist   *string    `json:"artist,omitempty"`
	Year     *int32     `json:"year,omitempty"`
	Duration *int64     `json:"duration,omitempty"`
	Rating   *float64   `json:"rating,omitempty"`
	Explicit *bool      `json:"explicit,omitempty"`
	Taken    *time.Time `json:"takenDateTime,omitempty"`
}

var _ = Describe("DeserializeStringsAt", func() {
	It("errors on an unsupported target kind", func() {
		v := reflect.New(reflect.TypeFor[[]int]()).Elem() // settable slice
		Expect(setValueFromString(v, "x")).To(HaveOccurred(), "expected error for unsupported target kind (slice)")
	})

	It("parses basic types", func() {
		r := DeserializeStringsAt[stringFacet](map[string]string{
			"libre.graph.audio.artist":        "Queen",
			"libre.graph.audio.year":          "1975",
			"libre.graph.audio.duration":      "354000",
			"libre.graph.audio.rating":        "4.9",
			"libre.graph.audio.explicit":      "true",
			"libre.graph.audio.takenDateTime": "2024-01-02T03:04:05Z",
		}, "libre.graph.audio.")
		Expect(r).ToNot(BeNil())
		Expect(r.Artist).ToNot(BeNil())
		Expect(*r.Artist).To(Equal("Queen"))
		Expect(r.Year).ToNot(BeNil())
		Expect(*r.Year).To(Equal(int32(1975)))
		Expect(r.Duration).ToNot(BeNil())
		Expect(*r.Duration).To(Equal(int64(354000)))
		Expect(r.Rating).ToNot(BeNil())
		Expect(*r.Rating).To(Equal(4.9))
		Expect(r.Explicit).ToNot(BeNil())
		Expect(*r.Explicit).To(BeTrue())
		Expect(r.Taken).ToNot(BeNil())
		Expect(r.Taken.Equal(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC))).To(BeTrue(), "Taken: %#v", r.Taken)
	})

	It("returns nil when nothing matches the prefix", func() {
		r := DeserializeStringsAt[stringFacet](map[string]string{
			"libre.graph.image.width": "1200",
		}, "libre.graph.audio.")
		Expect(r).To(BeNil())
	})

	It("parses into a timestamppb.Timestamp", func() {
		type photoFacet struct {
			Taken *timestamppb.Timestamp `json:"takenDateTime,omitempty"`
		}
		r := DeserializeStringsAt[photoFacet](map[string]string{
			"libre.graph.photo.takenDateTime": "2024-05-06T07:08:09Z",
		}, "libre.graph.photo.")
		Expect(r).ToNot(BeNil())
		Expect(r.Taken).ToNot(BeNil())
		want := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
		Expect(r.Taken.AsTime().Equal(want)).To(BeTrue(), "Taken: got %v, want %v", r.Taken.AsTime(), want)
	})

	It("is fail-soft on malformed fields", func() {
		// A single malformed field (year is unparseable as int) must not drop
		// the whole facet. The bad field stays at zero value, the rest of the
		// facet still populates. Mirrors the bleve-hit Deserialize behavior.
		r := DeserializeStringsAt[stringFacet](map[string]string{
			"libre.graph.audio.artist":   "Iron Maiden",
			"libre.graph.audio.year":     "not-a-number",
			"libre.graph.audio.duration": "354000",
			"libre.graph.audio.explicit": "not-a-bool",
			"libre.graph.audio.rating":   "4.9",
		}, "libre.graph.audio.")
		Expect(r).ToNot(BeNil())
		Expect(r.Artist).ToNot(BeNil())
		Expect(*r.Artist).To(Equal("Iron Maiden"), "Artist should still be populated")
		Expect(r.Duration).ToNot(BeNil())
		Expect(*r.Duration).To(Equal(int64(354000)), "Duration should still be populated")
		Expect(r.Rating).ToNot(BeNil())
		Expect(*r.Rating).To(Equal(4.9), "Rating should still be populated")
		Expect(r.Year).To(BeNil(), "Year should stay nil for bad int")
		Expect(r.Explicit).To(BeNil(), "Explicit should stay nil for bad bool")
	})

	It("panics for a non-struct T", func() {
		Expect(func() {
			DeserializeStringsAt[int](nil, "")
		}).To(Panic())
	})
})
