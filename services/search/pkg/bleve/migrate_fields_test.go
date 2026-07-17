package bleve_test

import (
	"path/filepath"
	"sort"
	"testing"
	"time"

	bleveSearch "github.com/blevesearch/bleve/v2"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// fullyPopulated exercises every field, including the facets and slices the
// count-only test skipped.
func fullyPopulated() search.Resource {
	return search.Resource{
		ID:       "1$2!3",
		RootID:   "1$2!2",
		ParentID: "1$2!2",
		Path:     "./a/b/song.mp3",
		Type:     2,
		Deleted:  true,
		Hidden:   true,
		Document: content.Document{
			Title:     "the title",
			Name:      "song.mp3",
			Content:   "some extracted content",
			Size:      123456,
			Mtime:     conversions.ToPointer(time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)),
			MimeType:  "audio/mpeg",
			Tags:      []string{"alpha", "beta", "gamma"},
			Favorites: []string{"user-a", "user-b"},
			Audio: &libregraph.Audio{
				Album:  libregraph.PtrString("the album"),
				Artist: libregraph.PtrString("the artist"),
				Track:  libregraph.PtrInt32(7),
				Year:   libregraph.PtrInt32(1998),
				HasDrm: libregraph.PtrBool(false),
			},
			Image: &libregraph.Image{Width: libregraph.PtrInt32(1920), Height: libregraph.PtrInt32(1080)},
			Photo: &libregraph.Photo{CameraMake: libregraph.PtrString("Canon"), Iso: libregraph.PtrInt32(400), FNumber: libregraph.PtrFloat64(2.8)},
			Location: &libregraph.GeoCoordinates{
				Longitude: libregraph.PtrFloat64(11.103870357204285),
				Latitude:  libregraph.PtrFloat64(49.48675890884328),
				Altitude:  libregraph.PtrFloat64(300.0),
			},
		},
	}
}

func TestMigratePreservesAllFields(t *testing.T) {
	root := t.TempDir()
	want := fullyPopulated()

	old, err := bleveSearch.New(filepath.Join(root, "bleve"), oldMainMapping(t))
	require.NoError(t, err)
	require.NoError(t, old.Index(want.ID, want))
	require.NoError(t, old.Close())

	_, err = bleve.MigrateIndex(root, log.NopLogger())
	require.NoError(t, err)

	idx, _, err := bleve.NewIndex(root)
	require.NoError(t, err)
	defer func() { _ = idx.Close() }()

	// read the migrated document back through the production reconstruction path
	req := bleveSearch.NewSearchRequest(bleveSearch.NewMatchAllQuery())
	req.Fields = []string{"*"}
	res, err := idx.Search(req)
	require.NoError(t, err)
	require.Len(t, res.Hits, 1)
	got := searchmapping.Deserialize[search.Resource](res.Hits[0].Fields)

	require.NotNil(t, got.Mtime, "Mtime")
	require.True(t, want.Mtime.Equal(*got.Mtime), "Mtime: want %v got %v", want.Mtime, got.Mtime)
	got.Mtime = want.Mtime            // normalize time.Time location/monotonic
	sort.Strings(got.Tags)            // bleve may return multi-value fields reordered
	sort.Strings(got.Favorites)       //
	require.Equal(t, want, *got)
}
