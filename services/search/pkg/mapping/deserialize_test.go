package mapping

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("Deserialize", func() {
	It("panics for a non-struct type at a prefix", func() {
		Expect(func() {
			_ = DeserializeAt[int](map[string]any{}, "")
		}).To(Panic())
	})

	It("panics for a non-struct T", func() {
		Expect(func() {
			Deserialize[int](nil)
		}).To(Panic())
	})

	It("deserializes leaf fields", func() {
		r := Deserialize[Leaf](map[string]any{
			"Name":    "n",
			"Size":    float64(42),
			"Deleted": true,
		})
		Expect(r.Name).To(Equal("n"))
		Expect(r.Size).To(Equal(uint64(42)))
		Expect(r.Deleted).To(BeTrue())
	})

	It("coerces a scalar into a slice field", func() {
		r := Deserialize[Leaf](map[string]any{
			"Tags":      "single",
			"Favorites": []any{"a", "b"},
		})
		Expect(r.Tags).To(Equal([]string{"single"}))
		Expect(r.Favorites).To(Equal([]string{"a", "b"}))
	})

	It("parses timestamps", func() {
		r := Deserialize[embedded](map[string]any{
			"photo.takenDateTime": "2024-01-02T03:04:05Z",
			"photo.mtime":         "2024-05-06T07:08:09Z",
		})
		Expect(r.Photo).ToNot(BeNil())
		Expect(r.Photo.Taken).ToNot(BeNil())
		expected := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
		Expect(r.Photo.Taken.AsTime().Equal(expected)).To(BeTrue(), "Taken: got %v, want %v", r.Photo.Taken.AsTime(), expected)
		Expect(r.Photo.Mtime).ToNot(BeNil())
		Expect(r.Photo.Mtime.Equal(time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC))).To(BeTrue(), "Mtime: %v", r.Photo.Mtime)
	})

	It("is fail-soft on malformed values", func() {
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
		Expect(r).ToNot(BeNil())
		Expect(r.Name).To(Equal("n"))
		Expect(r.Size).To(Equal(uint64(0)), "Size should stay zero on mismatch")
		Expect(r.Deleted).To(BeTrue())
		Expect(r.Photo).ToNot(BeNil(), "Photo should be populated because Mtime parsed ok")
		Expect(r.Photo.Taken).To(BeNil(), "Taken should stay nil for unparseable time")
		Expect(r.Photo.Mtime).ToNot(BeNil(), "Mtime should be parsed")
	})

	It("returns nil when nothing matches the prefix", func() {
		r := DeserializeAt[audio](map[string]any{"Name": "n"}, "audio")
		Expect(r).To(BeNil())
	})

	It("returns a value when the prefix matches", func() {
		r := DeserializeAt[audio](map[string]any{
			"audio.artist": "A",
			"audio.year":   float64(2024), // setValue: pointer + numeric convert
		}, "audio")
		Expect(r).ToNot(BeNil())
		Expect(r.Artist).ToNot(BeNil())
		Expect(*r.Artist).To(Equal("A"))
		Expect(r.Year).ToNot(BeNil())
		Expect(*r.Year).To(Equal(int32(2024)))
	})
})
