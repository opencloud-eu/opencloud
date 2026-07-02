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
