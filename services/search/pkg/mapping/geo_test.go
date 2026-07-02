package mapping

import (
	"reflect"
	"testing"
)

// build-mapping error paths: an override that doesn't fit the Go field.
func TestBuildMappingErrors(t *testing.T) {
	type doc struct {
		Name string `json:"name"`
	}
	// Geopoint on a non-struct field must error on both backends.
	if _, err := BleveBuildMapping(reflect.TypeFor[doc](), map[string]FieldOpts{"name": {Type: TypeGeopoint}}); err == nil {
		t.Error("bleve: expected error for geopoint on string field")
	}
	if _, err := OpenSearchBuildMapping(reflect.TypeFor[doc](), map[string]FieldOpts{"name": {Type: TypeGeopoint}}); err == nil {
		t.Error("opensearch: expected error for geopoint on string field")
	}
}

func TestAddGeopointSiblingMissingIntermediate(t *testing.T) {
	m := map[string]any{"journey": "not-a-map"}
	addGeopointSibling(m, "journey.start") // must bail, not panic
	if _, ok := m["journey.start"+GeopointSuffix]; ok {
		t.Error("no sibling should be written when the path can't be resolved")
	}
}

func TestPrepareForIndexAddsGeopointSibling(t *testing.T) {
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
	if err != nil {
		t.Fatalf("PrepareForIndex: %v", err)
	}

	// Original location object stays untouched (full libregraph shape).
	orig, ok := m["location"].(map[string]any)
	if !ok {
		t.Fatalf("expected location object preserved, got %T", m["location"])
	}
	if orig["longitude"] != lon || orig["latitude"] != lat || orig["altitude"] != alt {
		t.Errorf("location object: %#v", orig)
	}

	// Sibling location_geopoint has {lat, lon} for the geo indices.
	gp, ok := m["location"+GeopointSuffix].(map[string]any)
	if !ok {
		t.Fatalf("expected location_geopoint sibling, got %T", m["location"+GeopointSuffix])
	}
	if gp["lat"] != lat || gp["lon"] != lon {
		t.Errorf("sibling: %#v", gp)
	}
}

func TestPrepareForIndexSkipsIncompleteGeopoint(t *testing.T) {
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
	if err != nil {
		t.Fatalf("PrepareForIndex: %v", err)
	}
	// Original stays (altitude alone is still useful metadata).
	if _, ok := m["location"]; !ok {
		t.Error("location should still be present when only altitude is set")
	}
	// No sibling without both lon and lat.
	if _, ok := m["location"+GeopointSuffix]; ok {
		t.Errorf("no sibling expected, got %#v", m["location"+GeopointSuffix])
	}
}

func TestPrepareForIndexWithoutOverrideNoSibling(t *testing.T) {
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
	if err != nil {
		t.Fatalf("PrepareForIndex: %v", err)
	}
	if _, ok := m["location"+GeopointSuffix]; ok {
		t.Errorf("no sibling expected without override, got %#v", m["location"+GeopointSuffix])
	}
}

func TestPrepareForIndexHandlesNestedGeopoint(t *testing.T) {
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
	if err != nil {
		t.Fatalf("PrepareForIndex: %v", err)
	}
	j, ok := m["journey"].(map[string]any)
	if !ok {
		t.Fatalf("journey not an object: %T", m["journey"])
	}
	startGp, ok := j["start"+GeopointSuffix].(map[string]any)
	if !ok || startGp["lat"] != slat || startGp["lon"] != slon {
		t.Errorf("journey.start sibling: %#v", j["start"+GeopointSuffix])
	}
	endGp, ok := j["end"+GeopointSuffix].(map[string]any)
	if !ok || endGp["lat"] != elat || endGp["lon"] != elon {
		t.Errorf("journey.end sibling: %#v", j["end"+GeopointSuffix])
	}
}
