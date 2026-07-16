package bleve_test

import (
	"path/filepath"
	"testing"
	"time"

	bleveSearch "github.com/blevesearch/bleve/v2"
	bleveMapping "github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// oldMainMapping reconstructs the pre-refactor bleve mapping: only Name, Tags,
// Favorites and Content explicit, everything else dynamic. Reuses NewMapping's
// registered analyzers so indexing works.
func oldMainMapping(t *testing.T) bleveMapping.IndexMapping {
	m, err := bleve.NewMapping()
	require.NoError(t, err)
	impl := m.(*bleveMapping.IndexMappingImpl)

	doc := bleveSearch.NewDocumentMapping()
	name := bleveSearch.NewTextFieldMapping()
	name.Analyzer = "lowercaseKeyword"
	lc := bleveSearch.NewTextFieldMapping()
	lc.Analyzer = "lowercaseKeyword"
	lc.IncludeInAll = false
	ft := bleveSearch.NewTextFieldMapping()
	ft.Analyzer = "fulltext"
	ft.IncludeInAll = false
	doc.AddFieldMappingsAt("Name", name)
	doc.AddFieldMappingsAt("Tags", lc)
	doc.AddFieldMappingsAt("Favorites", lc)
	doc.AddFieldMappingsAt("Content", ft)
	impl.DefaultMapping = doc
	return impl
}

func fullDoc(id, name string, deleted bool) search.Resource {
	lon, lat, alt := 11.103870357204285, 49.48675890884328, 300.0
	return search.Resource{
		ID:       id,
		RootID:   "1$1!1",
		ParentID: "1$1!1",
		Path:     "./" + name,
		Type:     2,
		Deleted:  deleted,
		Document: content.Document{
			Name:     name,
			Title:    "title",
			Content:  "hello world",
			Size:     42,
			Mtime:    conversions.ToPointer(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
			MimeType: "image/jpeg",
			Tags:     []string{"tag1"},
			Location: &libregraph.GeoCoordinates{Longitude: &lon, Latitude: &lat, Altitude: &alt},
		},
	}
}

func TestOpenOrMigrate(t *testing.T) {
	buildOld := func(t *testing.T) string {
		root := t.TempDir()
		old, err := bleveSearch.New(filepath.Join(root, "bleve"), oldMainMapping(t))
		require.NoError(t, err)
		require.NoError(t, old.Index("1$1!1", fullDoc("1$1!1", "a.jpg", false)))
		require.NoError(t, old.Close())
		return root
	}

	t.Run("migrates an outdated revision when auto-migrate is on", func(t *testing.T) {
		// the old index carries no revision (0), the code revision is higher
		idx, classification, migrated, err := bleve.OpenOrMigrate(buildOld(t), true, log.NopLogger())
		require.NoError(t, err)
		require.NotNil(t, idx)
		defer func() { _ = idx.Close() }()
		require.Equal(t, searchmapping.VerdictEqual, classification.Verdict)
		require.Equal(t, 1, migrated)
	})

	t.Run("refuses to start when auto-migrate is off", func(t *testing.T) {
		idx, _, migrated, err := bleve.OpenOrMigrate(buildOld(t), false, log.NopLogger())
		require.ErrorIs(t, err, searchmapping.ErrManualActionRequired)
		require.Nil(t, idx)
		require.Equal(t, 0, migrated)
	})

	t.Run("does not migrate a fresh index", func(t *testing.T) {
		root := t.TempDir()
		// NewIndex creates a fresh index and stamps the current revision
		idx, _, err := bleve.NewIndex(root)
		require.NoError(t, err)
		require.NoError(t, idx.Close())

		// a fresh index is already at the current revision, so no migration runs
		idx2, classification, migrated, err := bleve.OpenOrMigrate(root, true, log.NopLogger())
		require.NoError(t, err)
		defer func() { _ = idx2.Close() }()
		require.Equal(t, searchmapping.VerdictEqual, classification.Verdict)
		require.Equal(t, 0, migrated)
	})
}

func TestMigrateIndex(t *testing.T) {
	root := t.TempDir()

	// 1) an index left behind by the old release: main-style mapping, no revision
	old, err := bleveSearch.New(filepath.Join(root, "bleve"), oldMainMapping(t))
	require.NoError(t, err)
	docs := []search.Resource{
		fullDoc("1$1!1", "photo.jpg", false),
		fullDoc("1$1!2", "song.mp3", false),
		fullDoc("1$1!3", "trashed.txt", true), // trash must survive the migration
	}
	for _, r := range docs {
		require.NoError(t, old.Index(r.ID, r))
	}
	require.NoError(t, old.Close())

	// 2) sanity: the old index is incompatible, the service would refuse to start
	_, _, err = bleve.NewIndex(root)
	require.ErrorIs(t, err, searchmapping.ErrManualActionRequired)

	// 3) migrate
	n, err := bleve.MigrateIndex(root, log.NopLogger())
	require.NoError(t, err)
	require.Equal(t, 3, n)

	// 4) the migrated index classifies as equal and holds every document
	migrated, classification, err := bleve.NewIndex(root)
	require.NoError(t, err)
	require.Equal(t, searchmapping.VerdictEqual, classification.Verdict)

	count, err := migrated.DocCount()
	require.NoError(t, err)
	require.Equal(t, uint64(3), count)

	// 4a) the trashed document survived (a rescan would have dropped it)
	trashed, err := migrated.Document("1$1!3")
	require.NoError(t, err)
	require.NotNil(t, trashed)

	// 5) location_geopoint was synthesized during migration (absent in the old index)
	near := query.NewGeoDistanceQuery(11.103870357204285, 49.48675890884328, "1km")
	near.SetField("location" + searchmapping.GeopointSuffix)
	res, err := migrated.Search(bleveSearch.NewSearchRequest(near))
	require.NoError(t, err)
	require.Equal(t, uint64(3), res.Total, "all three docs match a geo-distance query on the new sibling field")

	require.NoError(t, migrated.Close()) // release the lock before the second run

	// 6) idempotent: a second run is a no-op because the revision is now current
	n, err = bleve.MigrateIndex(root, log.NopLogger())
	require.NoError(t, err)
	require.Equal(t, 0, n)
}
