package mapping

import (
	"reflect"
	"testing"
	"time"
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

func TestOpenSearchNumericTypes(t *testing.T) {
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
	if err != nil {
		t.Fatalf("OpenSearchBuildMapping: %v", err)
	}
	want := map[string]string{"a": "short", "b": "short", "c": "integer", "d": "long", "e": "short", "f": "long", "g": "float", "h": "double"}
	for k, wt := range want {
		if got := props[k].(map[string]any)["type"]; got != wt {
			t.Errorf("%s: type = %v, want %v", k, got, wt)
		}
	}
}

func TestOpenSearchBuildMappingInferred(t *testing.T) {
	props, err := OpenSearchBuildMapping(reflect.TypeFor[osDoc](), nil)
	if err != nil {
		t.Fatalf("OpenSearchBuildMapping: %v", err)
	}
	want := map[string]string{
		"ID":        "keyword",
		"Size":      "long",
		"Deleted":   "boolean",
		"CreatedAt": "date",
		"Rating":    "double",
	}
	for k, v := range want {
		m, ok := props[k].(map[string]any)
		if !ok {
			t.Errorf("%s: missing or not a map: %#v", k, props[k])
			continue
		}
		if got := m["type"]; got != v {
			t.Errorf("%s: type %v, want %v", k, got, v)
		}
	}
}

func TestOpenSearchBuildMappingNested(t *testing.T) {
	props, err := OpenSearchBuildMapping(reflect.TypeFor[osDoc](), nil)
	if err != nil {
		t.Fatalf("OpenSearchBuildMapping: %v", err)
	}
	nested, ok := props["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested: not a map: %#v", props["nested"])
	}
	sub, ok := nested["properties"].(map[string]any)
	if !ok {
		t.Fatalf("nested.properties: missing: %#v", nested)
	}
	artist, ok := sub["artist"].(map[string]any)
	if !ok {
		t.Fatalf("nested.artist: %#v", sub)
	}
	if artist["type"] != "keyword" {
		t.Errorf("nested.artist.type: %v", artist["type"])
	}
	year, ok := sub["year"].(map[string]any)
	if !ok {
		t.Fatalf("nested.year: %#v", sub)
	}
	if year["type"] != "integer" {
		t.Errorf("nested.year.type: %v (int32 → integer expected)", year["type"])
	}
}

func TestOpenSearchBuildMappingOverrides(t *testing.T) {
	type doc struct {
		Name     string `json:"Name"`
		Content  string `json:"Content"`
		Path     string `json:"Path"`
		MimeType string `json:"MimeType"`
	}
	props, err := OpenSearchBuildMapping(reflect.TypeFor[doc](), map[string]FieldOpts{
		"Name":     {Analyzer: "lowercaseKeyword"},
		"Content":  {Type: TypeFulltext},
		"Path":     {Type: TypePath},
		"MimeType": {Type: TypeWildcard},
	})
	if err != nil {
		t.Fatalf("OpenSearchBuildMapping: %v", err)
	}
	name := props["Name"].(map[string]any)
	if name["type"] != "text" || name["analyzer"] != "lowercaseKeyword" {
		t.Errorf("Name: %#v", name)
	}
	content := props["Content"].(map[string]any)
	if content["type"] != "text" || content["term_vector"] != "with_positions_offsets" {
		t.Errorf("Content: %#v", content)
	}
	if _, ok := content["analyzer"]; ok {
		t.Errorf("Content should leave analyzer unset (use OpenSearch default), got %#v", content["analyzer"])
	}
	path := props["Path"].(map[string]any)
	if path["type"] != "text" || path["analyzer"] != "path_hierarchy" {
		t.Errorf("Path: %#v", path)
	}
	mime := props["MimeType"].(map[string]any)
	if mime["type"] != "wildcard" {
		t.Errorf("MimeType: %#v", mime)
	}
}

func TestOpenSearchBuildMappingGeopoint(t *testing.T) {
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
	if err != nil {
		t.Fatalf("OpenSearchBuildMapping: %v", err)
	}
	// Object for libregraph-shape data retrieval.
	loc, ok := props["location"].(map[string]any)
	if !ok {
		t.Fatalf("location: %#v", props["location"])
	}
	sub, ok := loc["properties"].(map[string]any)
	if !ok {
		t.Fatalf("location should have numeric sub-properties, got %#v", loc)
	}
	for _, k := range []string{"longitude", "latitude", "altitude"} {
		prop, ok := sub[k].(map[string]any)
		if !ok || prop["type"] != "double" {
			t.Errorf("location.%s: %#v", k, sub[k])
		}
	}
	// Sibling geo_point for spatial queries.
	gp, ok := props["location"+GeopointSuffix].(map[string]any)
	if !ok {
		t.Fatalf("location%s: %#v", GeopointSuffix, props["location"+GeopointSuffix])
	}
	if gp["type"] != "geo_point" {
		t.Errorf("location%s.type: %v", GeopointSuffix, gp["type"])
	}
}
