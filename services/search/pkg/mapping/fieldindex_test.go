package mapping_test

import (
	"reflect"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func resourceFieldIndex() map[string]string {
	return mapping.FieldNameIndex(
		reflect.TypeFor[search.Resource](),
		search.Resource{}.SearchFieldOverrides(),
	)
}

// resolve looks up the lowercased key in the index, falling back to the key.
func resolve(idx map[string]string, key string) string {
	if v, ok := idx[strings.ToLower(key)]; ok {
		return v
	}
	return key
}

// NestInner is embedded with a json tag below, so it must nest, not flatten.
type NestInner struct {
	A string `json:"A"`
}

type taggedOuter struct {
	NestInner `json:"inner"`
	Top       string `json:"top"`
}

var _ = Describe("FieldNameIndex", func() {
	It("resolves top-level fields case-insensitively", func() {
		idx := resourceFieldIndex()
		for in, want := range map[string]string{
			"rootid": "RootID", "ROOTID": "RootID", "RootID": "RootID",
			"name": "Name", "NAME": "Name",
			"mimetype": "MimeType", "MimeType": "MimeType",
			"tags": "Tags", "favorites": "Favorites",
			"mtime": "Mtime", "parentid": "ParentID", "id": "ID",
		} {
			Expect(resolve(idx, in)).To(Equal(want), "resolve(%q)", in)
		}
	})

	// Facet sub-fields are lowerCamelCase in the index (from libregraph json
	// tags); the derived index resolves them case-insensitively.
	It("resolves facet sub-fields case-insensitively", func() {
		idx := resourceFieldIndex()
		for in, want := range map[string]string{
			// case-insensitive: same field, different casings
			"photo.cameramake": "photo.cameraMake",
			"photo.CAMERAMAKE": "photo.cameraMake",
			// a representative sub-field across each facet
			"photo.takendatetime": "photo.takenDateTime",
			"audio.artist":        "audio.artist",
			"audio.albumartist":   "audio.albumArtist",
			"image.width":         "image.width",
			"location.latitude":   "location.latitude",
		} {
			Expect(resolve(idx, in)).To(Equal(want), "resolve(%q)", in)
		}
	})

	It("passes unknown keys through unchanged", func() {
		idx := resourceFieldIndex()
		Expect(resolve(idx, "nope.field")).To(Equal("nope.field"))
		Expect(resolve(idx, "custom")).To(Equal("custom"))
	})

	// All top-level fields are covered from one derived source, so both
	// backends resolve them the same way.
	It("covers all top-level fields", func() {
		idx := resourceFieldIndex()
		for in, want := range map[string]string{
			"rootid": "RootID", "path": "Path", "id": "ID", "name": "Name",
			"size": "Size", "mtime": "Mtime", "type": "Type",
			"content": "Content", "hidden": "Hidden", "tags": "Tags",
			"favorites": "Favorites",
		} {
			Expect(resolve(idx, in)).To(Equal(want), "derived should cover %q", in)
		}
	})

	// A json-tagged embedded struct nests under its tag in the derived index
	// too (walkFields must match encoding/json), so its fields are "inner.A",
	// not "A".
	It("nests json-tagged embedded structs", func() {
		idx := mapping.FieldNameIndex(reflect.TypeFor[taggedOuter](), nil)
		Expect(resolve(idx, "inner.a")).To(Equal("inner.A")) // nested under the tag
		Expect(resolve(idx, "top")).To(Equal("top"))
		Expect(resolve(idx, "a")).To(Equal("a")) // not flattened: bare "a" is not a key
	})
})
