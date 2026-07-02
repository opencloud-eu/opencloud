package mapping

import (
	"reflect"
	"strings"
	"testing"
)

type inner struct {
	Artist string `json:"artist"`
}

type sample struct {
	Name     string    `json:"Name"`
	Audio    *inner    `json:"audio,omitempty"`
	Location *struct { //nolint:unused
		Lon float64 `json:"longitude"`
		Lat float64 `json:"latitude"`
	} `json:"location,omitempty"`
}

func TestValidateAccepts(t *testing.T) {
	err := Validate(reflect.TypeFor[sample](), map[string]FieldOpts{
		"Name":         {Analyzer: "lowercaseKeyword"},
		"audio":        {Type: TypeObject},
		"audio.artist": {Analyzer: "lowercaseKeyword"},
		"location":     {Type: TypeGeopoint},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsUnknown(t *testing.T) {
	err := Validate(reflect.TypeFor[sample](), map[string]FieldOpts{
		"nope":      {},
		"audio.zzz": {},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "nope") || !strings.Contains(err.Error(), "audio.zzz") {
		t.Fatalf("error missing keys: %v", err)
	}
}

func TestValidateEmpty(t *testing.T) {
	if err := Validate(reflect.TypeFor[sample](), nil); err != nil {
		t.Fatalf("empty overrides should pass: %v", err)
	}
}
