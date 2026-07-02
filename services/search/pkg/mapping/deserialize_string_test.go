package mapping

import (
	"reflect"
	"testing"
	"time"

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

func TestSetValueFromStringUnsupportedKind(t *testing.T) {
	v := reflect.New(reflect.TypeFor[[]int]()).Elem() // settable slice
	if err := setValueFromString(v, "x"); err == nil {
		t.Error("expected error for unsupported target kind (slice)")
	}
}

func TestDeserializeStringsAtBasicTypes(t *testing.T) {
	r := DeserializeStringsAt[stringFacet](map[string]string{
		"libre.graph.audio.artist":        "Queen",
		"libre.graph.audio.year":          "1975",
		"libre.graph.audio.duration":      "354000",
		"libre.graph.audio.rating":        "4.9",
		"libre.graph.audio.explicit":      "true",
		"libre.graph.audio.takenDateTime": "2024-01-02T03:04:05Z",
	}, "libre.graph.audio.")
	if r == nil {
		t.Fatal("expected non-nil *stringFacet")
	}
	if r.Artist == nil || *r.Artist != "Queen" {
		t.Errorf("Artist: %#v", r.Artist)
	}
	if r.Year == nil || *r.Year != 1975 {
		t.Errorf("Year: %#v", r.Year)
	}
	if r.Duration == nil || *r.Duration != 354000 {
		t.Errorf("Duration: %#v", r.Duration)
	}
	if r.Rating == nil || *r.Rating != 4.9 {
		t.Errorf("Rating: %#v", r.Rating)
	}
	if r.Explicit == nil || !*r.Explicit {
		t.Errorf("Explicit: %#v", r.Explicit)
	}
	if r.Taken == nil || !r.Taken.Equal(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("Taken: %#v", r.Taken)
	}
}

func TestDeserializeStringsAtReturnsNilWhenEmpty(t *testing.T) {
	r := DeserializeStringsAt[stringFacet](map[string]string{
		"libre.graph.image.width": "1200",
	}, "libre.graph.audio.")
	if r != nil {
		t.Fatalf("expected nil, got %#v", r)
	}
}

func TestDeserializeStringsAtTimestamppb(t *testing.T) {
	type photoFacet struct {
		Taken *timestamppb.Timestamp `json:"takenDateTime,omitempty"`
	}
	r := DeserializeStringsAt[photoFacet](map[string]string{
		"libre.graph.photo.takenDateTime": "2024-05-06T07:08:09Z",
	}, "libre.graph.photo.")
	if r == nil || r.Taken == nil {
		t.Fatalf("Taken missing: %#v", r)
	}
	want := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
	if !r.Taken.AsTime().Equal(want) {
		t.Errorf("Taken: got %v, want %v", r.Taken.AsTime(), want)
	}
}

func TestDeserializeStringsAtIsFailSoft(t *testing.T) {
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
	if r == nil {
		t.Fatal("expected non-nil *stringFacet despite bad fields")
	}
	if r.Artist == nil || *r.Artist != "Iron Maiden" {
		t.Errorf("Artist should still be populated, got %#v", r.Artist)
	}
	if r.Duration == nil || *r.Duration != 354000 {
		t.Errorf("Duration should still be populated, got %#v", r.Duration)
	}
	if r.Rating == nil || *r.Rating != 4.9 {
		t.Errorf("Rating should still be populated, got %#v", r.Rating)
	}
	if r.Year != nil {
		t.Errorf("Year should stay nil for bad int, got %#v", r.Year)
	}
	if r.Explicit != nil {
		t.Errorf("Explicit should stay nil for bad bool, got %#v", r.Explicit)
	}
}

func TestDeserializeStringsAtPanicsOnNonStruct(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for non-struct T")
		}
	}()
	DeserializeStringsAt[int](nil, "")
}
