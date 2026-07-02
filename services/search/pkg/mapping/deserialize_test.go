package mapping

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type Leaf struct {
	Name      string   `json:"Name"`
	Size      uint64   `json:"Size"`
	Deleted   bool     `json:"Deleted"`
	Tags      []string `json:"Tags"`
	Favorites []string `json:"Favorites"`
}

type audio struct {
	Artist *string `json:"artist,omitempty"`
	Year   *int32  `json:"year,omitempty"`
}

type photo struct {
	Taken *timestamppb.Timestamp `json:"takenDateTime,omitempty"`
	Mtime *time.Time             `json:"mtime,omitempty"`
}

type embedded struct {
	Leaf
	Audio *audio `json:"audio,omitempty"`
	Photo *photo `json:"photo,omitempty"`
}

func TestDeserializeAtNonStructPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic for non-struct type")
		}
	}()
	_ = DeserializeAt[int](map[string]any{}, "")
}

func TestDeserializeLeafFields(t *testing.T) {
	r := Deserialize[Leaf](map[string]any{
		"Name":    "n",
		"Size":    float64(42),
		"Deleted": true,
	})
	if r.Name != "n" || r.Size != 42 || !r.Deleted {
		t.Fatalf("got %#v", r)
	}
}

func TestDeserializeScalarToSlice(t *testing.T) {
	r := Deserialize[Leaf](map[string]any{
		"Tags":      "single",
		"Favorites": []any{"a", "b"},
	})
	if len(r.Tags) != 1 || r.Tags[0] != "single" {
		t.Errorf("Tags: %#v", r.Tags)
	}
	if len(r.Favorites) != 2 || r.Favorites[0] != "a" || r.Favorites[1] != "b" {
		t.Errorf("Favorites: %#v", r.Favorites)
	}
}

func TestDeserializeTimestamp(t *testing.T) {
	r := Deserialize[embedded](map[string]any{
		"photo.takenDateTime": "2024-01-02T03:04:05Z",
		"photo.mtime":         "2024-05-06T07:08:09Z",
	})
	if r.Photo == nil {
		t.Fatal("Photo is nil")
	}
	if r.Photo.Taken == nil {
		t.Fatal("Taken is nil")
	}
	expected := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	if !r.Photo.Taken.AsTime().Equal(expected) {
		t.Errorf("Taken: got %v, want %v", r.Photo.Taken.AsTime(), expected)
	}
	if r.Photo.Mtime == nil {
		t.Fatal("Mtime is nil")
	}
	if !r.Photo.Mtime.Equal(time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)) {
		t.Errorf("Mtime: %v", r.Photo.Mtime)
	}
}

func TestDeserializeIsFailSoft(t *testing.T) {
	// Malformed values (type mismatch, unparseable time) leave the
	// affected field at its zero value instead of dropping the whole
	// record. Matches the pre-refactor getFieldValue behavior so
	// matchToResource never returns nil on a corrupted hit.
	r := Deserialize[embedded](map[string]any{
		"Name":                "n",
		"Size":                "not-a-number", // wrong type
		"Deleted":             true,
		"photo.takenDateTime": "not-an-rfc3339-time",
		"photo.mtime":         "2024-05-06T07:08:09Z",
	})
	if r == nil {
		t.Fatal("expected non-nil *embedded even with partial corruption")
	}
	if r.Name != "n" {
		t.Errorf("Name: %q", r.Name)
	}
	if r.Size != 0 {
		t.Errorf("Size should stay zero on mismatch, got %d", r.Size)
	}
	if !r.Deleted {
		t.Errorf("Deleted should still be true")
	}
	if r.Photo == nil {
		t.Fatal("Photo should be populated because Mtime parsed ok")
	}
	if r.Photo.Taken != nil {
		t.Errorf("Taken should stay nil for unparseable time, got %v", r.Photo.Taken)
	}
	if r.Photo.Mtime == nil {
		t.Error("Mtime should be parsed")
	}
}

func TestDeserializePanicsOnNonStruct(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for non-struct T")
		}
	}()
	Deserialize[int](nil)
}

func TestDeserializeAtReturnsNilWhenNothingMatches(t *testing.T) {
	r := DeserializeAt[audio](map[string]any{"Name": "n"}, "audio")
	if r != nil {
		t.Fatalf("expected nil, got %#v", r)
	}
}

func TestDeserializeAtReturnsValueWhenPrefixMatches(t *testing.T) {
	r := DeserializeAt[audio](map[string]any{
		"audio.artist": "A",
		"audio.year":   float64(2024), // setValue: pointer + numeric convert
	}, "audio")
	if r == nil {
		t.Fatal("expected non-nil *audio")
	}
	if r.Artist == nil || *r.Artist != "A" {
		t.Errorf("Artist: %#v", r.Artist)
	}
	if r.Year == nil || *r.Year != 2024 {
		t.Errorf("Year: %#v", r.Year)
	}
}
