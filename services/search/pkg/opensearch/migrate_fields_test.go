package opensearch_test

import (
	"context"
	"strings"
	"testing"
	"time"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	opensearchtest "github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

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
	ctx := context.Background()
	base := "opencloud-migrate-fields"
	tc, target := newMigrateTest(t, base)

	want := fullyPopulated()
	tc.Require.IndicesCreate(base, strings.NewReader(`{}`))
	tc.Require.DocumentCreate(base, want.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, want)))
	tc.Require.IndicesCount([]string{base}, nil, 1)

	_, err := opensearch.MigrateIndex(ctx, base, tc.Client(), log.NopLogger())
	require.NoError(t, err)

	// read the migrated document back from the versioned target; every field must round-trip
	hits := tc.Require.Search(target, strings.NewReader(`{"size":10,"query":{"match_all":{}}}`))
	resources := opensearchtest.SearchHitsMustBeConverted[search.Resource](t, hits.Hits)
	require.Len(t, resources, 1)
	got := resources[0]

	require.NotNil(t, got.Mtime, "Mtime")
	require.True(t, want.Mtime.Equal(*got.Mtime), "Mtime: want %v got %v", want.Mtime, got.Mtime)
	got.Mtime = want.Mtime // normalize time.Time location/monotonic before the deep-equal
	require.Equal(t, want, got)
}
