package mapping

import (
	"reflect"
	"testing"
	"time"
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

// bleve wildcard falls back to keyword-ish text (bleve has no wildcard type).
func TestBleveWildcardFallback(t *testing.T) {
	type doc struct {
		Mime string `json:"mime"`
	}
	dm, err := BleveBuildMapping(reflect.TypeFor[doc](), map[string]FieldOpts{"mime": {Type: TypeWildcard}})
	if err != nil {
		t.Fatalf("BleveBuildMapping: %v", err)
	}
	fms := dm.Properties["mime"].Fields
	if len(fms) != 1 || fms[0].Type != "text" {
		t.Fatalf("wildcard should map to text, got %+v", fms)
	}
}

func TestBleveBuildMappingInferredTypes(t *testing.T) {
	dm, err := BleveBuildMapping(reflect.TypeFor[bleveDoc](), nil)
	if err != nil {
		t.Fatalf("BleveBuildMapping: %v", err)
	}
	cases := map[string]string{
		"Name":      "text",
		"Content":   "text",
		"Tags":      "text",
		"Size":      "number",
		"Deleted":   "boolean",
		"CreatedAt": "datetime",
	}
	for field, wantType := range cases {
		prop := dm.Properties[field]
		if prop == nil {
			t.Errorf("missing property %q", field)
			continue
		}
		if len(prop.Fields) == 0 {
			t.Errorf("%q: no field mappings", field)
			continue
		}
		if got := prop.Fields[0].Type; got != wantType {
			t.Errorf("%q: got type %q, want %q", field, got, wantType)
		}
	}
}

func TestBleveBuildMappingNestedIsSubDocument(t *testing.T) {
	dm, err := BleveBuildMapping(reflect.TypeFor[bleveDoc](), nil)
	if err != nil {
		t.Fatalf("BleveBuildMapping: %v", err)
	}
	sub := dm.Properties["nested"]
	if sub == nil {
		t.Fatal("missing nested sub-document")
	}
	if sub.Properties["artist"] == nil || sub.Properties["year"] == nil {
		t.Fatalf("nested fields missing: %#v", sub.Properties)
	}
	if got := sub.Properties["artist"].Fields[0].Type; got != "text" {
		t.Errorf("nested.artist: type %q, want text", got)
	}
	if got := sub.Properties["year"].Fields[0].Type; got != "number" {
		t.Errorf("nested.year: type %q, want number", got)
	}
}

func TestBleveBuildMappingOverrides(t *testing.T) {
	includeInAllFalse := false
	dm, err := BleveBuildMapping(reflect.TypeFor[bleveDoc](), map[string]FieldOpts{
		"Name":    {Analyzer: "lowercaseKeyword"},
		"Content": {Type: TypeFulltext},
		"Tags":    {Analyzer: "lowercaseKeyword", IncludeInAll: &includeInAllFalse},
	})
	if err != nil {
		t.Fatalf("BleveBuildMapping: %v", err)
	}
	nameField := dm.Properties["Name"].Fields[0]
	if nameField.Analyzer != "lowercaseKeyword" {
		t.Errorf("Name analyzer: %q, want lowercaseKeyword", nameField.Analyzer)
	}
	if !nameField.IncludeInAll {
		t.Errorf("Name IncludeInAll should stay default-true when not overridden")
	}
	contentField := dm.Properties["Content"].Fields[0]
	if contentField.Analyzer != "fulltext" {
		t.Errorf("Content analyzer: %q, want fulltext", contentField.Analyzer)
	}
	if contentField.IncludeInAll {
		t.Errorf("Content IncludeInAll should default to false for fulltext type")
	}
	tagsField := dm.Properties["Tags"].Fields[0]
	if tagsField.IncludeInAll {
		t.Errorf("Tags IncludeInAll should honor the explicit false override")
	}
}

func TestBleveBuildMappingGeopoint(t *testing.T) {
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
	if err != nil {
		t.Fatalf("BleveBuildMapping: %v", err)
	}
	// Original facet stays as an object sub-document with numeric
	// sub-properties - for data retrieval via hit.Fields and ordinary
	// numeric queries.
	loc := dm.Properties["location"]
	if loc == nil {
		t.Fatalf("location sub-document missing: %#v", dm.Properties)
	}
	if len(loc.Fields) != 0 {
		t.Errorf("location should not carry field mappings directly, got %#v", loc.Fields)
	}
	for _, sub := range []string{"longitude", "latitude", "altitude"} {
		prop, ok := loc.Properties[sub]
		if !ok {
			t.Errorf("missing sub-field %q under location (properties: %v)", sub, loc.Properties)
			continue
		}
		if len(prop.Fields) == 0 || prop.Fields[0].Type != "number" {
			t.Errorf("location.%s Fields: %#v, want [number]", sub, prop.Fields)
		}
	}
	// Sibling geopoint at "<name>_geopoint" for geo-distance queries.
	sibling := dm.Properties["location"+GeopointSuffix]
	if sibling == nil {
		t.Fatalf("location%s missing: %#v", GeopointSuffix, dm.Properties)
	}
	if len(sibling.Fields) == 0 || sibling.Fields[0].Type != "geopoint" {
		t.Errorf("location%s Fields: %#v, want [geopoint]", GeopointSuffix, sibling.Fields)
	}
}
