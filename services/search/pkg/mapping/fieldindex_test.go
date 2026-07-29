package mapping_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func resourceFieldIndex(testing.TB) map[string]string {
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

func TestFieldNameIndex_TopLevelCaseInsensitive(t *testing.T) {
	idx := resourceFieldIndex(t)
	for in, want := range map[string]string{
		"rootid": "RootID", "ROOTID": "RootID", "RootID": "RootID",
		"name": "Name", "NAME": "Name",
		"mimetype": "MimeType", "MimeType": "MimeType",
		"tags": "Tags", "favorites": "Favorites",
		"mtime": "Mtime", "parentid": "ParentID", "id": "ID",
	} {
		require.Equalf(t, want, resolve(idx, in), "resolve(%q)", in)
	}
}

// Facet sub-fields are lowerCamelCase in the index (from libregraph json tags);
// the derived index resolves them case-insensitively.
func TestFieldNameIndex_FacetsCaseInsensitive(t *testing.T) {
	idx := resourceFieldIndex(t)
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
		require.Equalf(t, want, resolve(idx, in), "resolve(%q)", in)
	}
}

func TestFieldNameIndex_UnknownPassesThrough(t *testing.T) {
	idx := resourceFieldIndex(t)
	require.Equal(t, "nope.field", resolve(idx, "nope.field"))
	require.Equal(t, "custom", resolve(idx, "custom"))
}

// All top-level fields are covered from one derived source, so both backends
// resolve them the same way.
func TestFieldNameIndex_CoversAllTopLevelFields(t *testing.T) {
	idx := resourceFieldIndex(t)
	for in, want := range map[string]string{
		"rootid": "RootID", "path": "Path", "id": "ID", "name": "Name",
		"size": "Size", "mtime": "Mtime", "type": "Type",
		"content": "Content", "hidden": "Hidden", "tags": "Tags",
		"favorites": "Favorites",
	} {
		require.Equalf(t, want, resolve(idx, in), "derived should cover %q", in)
	}
}
