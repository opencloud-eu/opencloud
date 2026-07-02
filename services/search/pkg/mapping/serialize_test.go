package mapping

import (
	"reflect"
	"testing"
)

func TestPrepareForIndexError(t *testing.T) {
	// a func field can't be json-marshalled -> conversions.To errors
	type bad struct {
		F func() `json:"f"`
	}
	if _, err := PrepareForIndex(bad{}, nil); err == nil {
		t.Error("expected error for non-marshallable value")
	}
}

func TestPrepareForIndexNil(t *testing.T) {
	// a typed nil pointer marshals to null -> nil map, no error, no panic
	out, err := PrepareForIndex((*struct{})(nil), nil)
	if err != nil || out != nil {
		t.Errorf("got (%v, %v), want (nil, nil)", out, err)
	}
}

func TestPrepareForIndexFlattensEmbedded(t *testing.T) {
	type inner struct {
		Name string `json:"Name"`
		Size uint64 `json:"Size"`
	}
	type outer struct {
		inner
		ID string `json:"ID"`
	}
	m, err := PrepareForIndex(outer{inner: inner{Name: "a", Size: 7}, ID: "x"}, nil)
	if err != nil {
		t.Fatalf("PrepareForIndex: %v", err)
	}
	want := map[string]any{"Name": "a", "Size": float64(7), "ID": "x"}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("got %#v, want %#v", m, want)
	}
}

func TestPrepareForIndexOmitsNilWithOmitempty(t *testing.T) {
	type facet struct {
		Artist string `json:"artist"`
	}
	type doc struct {
		Name  string `json:"Name"`
		Audio *facet `json:"audio,omitempty"`
	}
	m, err := PrepareForIndex(doc{Name: "n"}, nil)
	if err != nil {
		t.Fatalf("PrepareForIndex: %v", err)
	}
	if _, ok := m["audio"]; ok {
		t.Errorf("audio should be omitted when nil: %#v", m)
	}
	if m["Name"] != "n" {
		t.Errorf("Name: %v", m["Name"])
	}
}

func TestPrepareForIndexIncludesNestedWhenSet(t *testing.T) {
	type facet struct {
		Artist string `json:"artist"`
	}
	type doc struct {
		Audio *facet `json:"audio,omitempty"`
	}
	m, err := PrepareForIndex(doc{Audio: &facet{Artist: "A"}}, nil)
	if err != nil {
		t.Fatalf("PrepareForIndex: %v", err)
	}
	nested, ok := m["audio"].(map[string]any)
	if !ok {
		t.Fatalf("audio should be a nested map: %#v", m["audio"])
	}
	if nested["artist"] != "A" {
		t.Errorf("audio.artist: %v", nested["artist"])
	}
}
