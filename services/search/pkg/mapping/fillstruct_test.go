package mapping

import (
	"errors"
	"reflect"
	"testing"
)

// The walker is shared by both deserializers, so its structural behavior
// (flattening embedded structs, recursing into nested pointers, joining the
// prefix, keeping a pointer only when a field was set, fail-soft skipping) is
// tested here once against a trivial string setter instead of twice through
// Deserialize and DeserializeStringsAt.

// FsEmbVal/FsEmbPtr are exported so their embedded field is exported; an
// unexported embedded type would be skipped by resolveField.
type FsEmbVal struct {
	EV string `json:"ev"`
}

type FsEmbPtr struct {
	EP string `json:"ep"`
}

type fsNested struct {
	N string `json:"n"`
}

type fsRoot struct {
	FsEmbVal            // embedded value struct: fields promoted
	*FsEmbPtr           // embedded pointer struct: allocated on demand
	Leaf      string    `json:"leaf"`
	Nested    *fsNested `json:"nested"` // nested pointer: recursed under "nested."
}

var errBadLeaf = errors.New("bad leaf")

// fsSet writes raw into a string field; the "BAD" sentinel simulates a parse
// failure so the fail-soft path can be exercised.
func fsSet(v reflect.Value, raw string) error {
	if raw == "BAD" {
		return errBadLeaf
	}
	v.SetString(raw)
	return nil
}

func TestFillStruct(t *testing.T) {
	fill := func(fields map[string]string, prefix string) (fsRoot, bool) {
		var root fsRoot
		touched := fillStruct(reflect.ValueOf(&root).Elem(), fields, prefix, fsSet)
		return root, touched
	}

	t.Run("flattens embedded, recurses nested", func(t *testing.T) {
		root, touched := fill(map[string]string{
			"leaf":     "L",
			"ev":       "EV",
			"ep":       "EP",
			"nested.n": "N",
		}, "")
		if !touched {
			t.Fatal("expected touched")
		}
		if root.Leaf != "L" {
			t.Errorf("Leaf: %q", root.Leaf)
		}
		if root.EV != "EV" {
			t.Errorf("embedded value not promoted: %q", root.EV)
		}
		if root.FsEmbPtr == nil || root.EP != "EP" {
			t.Errorf("embedded pointer not allocated: %+v", root.FsEmbPtr)
		}
		if root.Nested == nil || root.Nested.N != "N" {
			t.Errorf("nested pointer not populated: %+v", root.Nested)
		}
	})

	t.Run("nothing matches: touched false, pointers stay nil", func(t *testing.T) {
		root, touched := fill(map[string]string{"other": "x"}, "")
		if touched {
			t.Fatal("expected untouched")
		}
		if root.Nested != nil {
			t.Errorf("Nested should stay nil: %+v", root.Nested)
		}
		if root.FsEmbPtr != nil {
			t.Errorf("embedded pointer should stay nil: %+v", root.FsEmbPtr)
		}
	})

	t.Run("prefix arg is joined with the field name", func(t *testing.T) {
		root, touched := fill(map[string]string{
			"pre.leaf":     "L",
			"pre.nested.n": "N",
		}, "pre")
		if !touched || root.Leaf != "L" || root.Nested == nil || root.Nested.N != "N" {
			t.Fatalf("prefix not joined: leaf=%q nested=%+v", root.Leaf, root.Nested)
		}
	})

	t.Run("fail-soft: errored leaf stays zero, walk continues", func(t *testing.T) {
		root, touched := fill(map[string]string{
			"leaf": "BAD",
			"ev":   "EV",
		}, "")
		if !touched {
			t.Fatal("expected touched because ev was set")
		}
		if root.Leaf != "" {
			t.Errorf("errored leaf should stay zero, got %q", root.Leaf)
		}
		if root.EV != "EV" {
			t.Errorf("walk should continue past the error: %q", root.EV)
		}
	})

	t.Run("embedded pointer dropped when its only field errors", func(t *testing.T) {
		root, touched := fill(map[string]string{"ep": "BAD"}, "")
		if touched {
			t.Fatal("expected untouched")
		}
		if root.FsEmbPtr != nil {
			t.Errorf("embedded pointer should stay nil on error, got %+v", root.FsEmbPtr)
		}
	})
}
