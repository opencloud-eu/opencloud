package opensearch_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	opensearchtest "github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// newMigrateTest returns a client and the current-revision target for a fresh
// migration test, deleting base, its versioned siblings and the target both
// before the test and via t.Cleanup.
func newMigrateTest(t *testing.T, base string) (*opensearchtest.TestClient, string) {
	tc := opensearchtest.NewDefaultTestClient(t, defaultConfig.Engine.OpenSearch.Client)
	indices := []string{base, base + "-v0", opensearch.TargetIndex(base)}
	reset := func() {
		for _, i := range indices {
			_, _ = tc.Client().Indices.Delete(context.Background(), opensearchgoAPI.IndicesDeleteReq{Indices: []string{i}})
		}
	}
	reset()
	t.Cleanup(reset)
	return tc, opensearch.TargetIndex(base)
}

func migrateDoc(id, name string, deleted bool) search.Resource {
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
			Content:  "hello world",
			Size:     42,
			Mtime:    conversions.ToPointer(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
			MimeType: "image/jpeg",
			Tags:     []string{"tag1"},
			Location: &libregraph.GeoCoordinates{Longitude: &lon, Latitude: &lat, Altitude: &alt},
		},
	}
}

func contentDoc(id, text string) search.Resource {
	return search.Resource{
		ID:       id,
		RootID:   "1$1!1",
		ParentID: "1$1!1",
		Path:     "./" + id,
		Type:     2,
		Document: content.Document{
			Name:    id,
			Content: text,
			Mtime:   conversions.ToPointer(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		},
	}
}

// bulkIndex writes total migrateDoc documents into base, in chunks.
func bulkIndex(t *testing.T, tc *opensearchtest.TestClient, base string, total int) {
	const chunk = 5000
	for i := 0; i < total; i += chunk {
		var bulk strings.Builder
		for j := i; j < i+chunk && j < total; j++ {
			id := fmt.Sprintf("doc%05d", j)
			fmt.Fprintf(&bulk, "{\"create\":{\"_index\":%q,\"_id\":%q}}\n", base, id)
			bulk.WriteString(opensearchtest.JSONMustMarshal(t, migrateDoc(id, id+".txt", false)))
			bulk.WriteString("\n")
		}
		_, err := tc.Client().Bulk(context.Background(), opensearchgoAPI.BulkReq{Body: strings.NewReader(bulk.String())})
		require.NoError(t, err)
	}
}

func TestMigrateIndex(t *testing.T) {
	ctx := context.Background()
	base := "opencloud-migrate-basic"
	tc, target := newMigrateTest(t, base)

	// 1) a legacy unversioned index from a pre-versioning release: dynamic mapping,
	//    documents including a trashed one, location as a plain object (no geopoint)
	tc.Require.IndicesCreate(base, strings.NewReader(`{}`))
	for _, d := range []search.Resource{
		migrateDoc("doc1", "photo.jpg", false),
		migrateDoc("doc2", "song.mp3", false),
		migrateDoc("doc3", "trashed.txt", true),
	} {
		tc.Require.DocumentCreate(base, d.ID, strings.NewReader(opensearchtest.JSONMustMarshal(t, d)))
	}
	tc.Require.IndicesCount([]string{base}, nil, 3)

	// 2) migrate builds the versioned target from the legacy index
	n, err := opensearch.MigrateIndex(ctx, base, tc.Client(), log.NopLogger())
	require.NoError(t, err)
	require.Equal(t, 3, n)

	// 3) the versioned target exists and holds every document
	tc.Require.IndicesCount([]string{target}, nil, 3)

	// 3a) the trashed document survived (a rescan would have dropped it)
	trash := `{"query":{"term":{"Deleted":true}}}`
	tc.Require.IndicesCount([]string{target}, strings.NewReader(trash), 1)

	// 4) location_geopoint was synthesized during migration (absent in the legacy index)
	geo := fmt.Sprintf(`{"query":{"geo_distance":{"distance":"1km","location%s":{"lat":%f,"lon":%f}}}}`,
		"_geopoint", 49.48675890884328, 11.103870357204285)
	tc.Require.IndicesCount([]string{target}, strings.NewReader(geo), 3)

	// 5) the service starts against the target and Apply classifies it equal
	require.NoError(t, opensearch.IndexManagerLatest.Apply(ctx, target, tc.Client(), log.NopLogger()))

	// 6) idempotent: the target now exists, so a second run creates nothing
	n, err = opensearch.MigrateIndex(ctx, base, tc.Client(), log.NopLogger())
	require.NoError(t, err)
	require.Equal(t, 0, n)
}

// TestMigrateLargeIndex covers reindexing past a single scroll page and past
// OpenSearch's default track_total_hits cap (10000) — the completeness check
// must use the exact total.
func TestMigrateLargeIndex(t *testing.T) {
	for _, row := range []struct {
		name  string
		total int
	}{
		{"pages beyond one scroll page", 2500},
		{"exceeds the 10k track_total_hits cap", 10500},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx := context.Background()
			base := fmt.Sprintf("opencloud-migrate-large-%d", row.total)
			tc, target := newMigrateTest(t, base)

			tc.Require.IndicesCreate(base, strings.NewReader(`{}`))
			bulkIndex(t, tc, base, row.total)
			tc.Require.IndicesRefresh([]string{base}, nil)
			tc.Require.IndicesCount([]string{base}, nil, row.total)

			n, err := opensearch.MigrateIndex(ctx, base, tc.Client(), log.NopLogger())
			require.NoError(t, err)
			require.Equal(t, row.total, n, "every document across all pages must be migrated")
			tc.Require.IndicesCount([]string{target}, nil, row.total)
		})
	}
}

// TestMigrateBackfillsWithoutOverwriting simulates the post-rollout / late run:
// the target already exists (an instance created it and wrote newer documents),
// and migrate must backfill the missing ones with create-only ops, never
// overwriting the newer ones.
func TestMigrateBackfillsWithoutOverwriting(t *testing.T) {
	ctx := context.Background()
	base := "opencloud-migrate-backfill"
	tc, target := newMigrateTest(t, base)

	// legacy index with three documents (old content)
	tc.Require.IndicesCreate(base, strings.NewReader(`{}`))
	for _, id := range []string{"a", "b", "c"} {
		tc.Require.DocumentCreate(base, id, strings.NewReader(opensearchtest.JSONMustMarshal(t, contentDoc(id, "OLD"))))
	}
	tc.Require.IndicesCount([]string{base}, nil, 3)

	// a new instance already created the target and wrote a newer "a" plus a new "d"
	tc.Require.IndicesCreate(target, strings.NewReader(opensearch.IndexManagerLatest.String()))
	tc.Require.DocumentCreate(target, "a", strings.NewReader(opensearchtest.JSONMustMarshal(t, contentDoc("a", "NEW"))))
	tc.Require.DocumentCreate(target, "d", strings.NewReader(opensearchtest.JSONMustMarshal(t, contentDoc("d", "NEW"))))
	tc.Require.IndicesRefresh([]string{target}, nil)

	// migrate backfills b and c, must not overwrite the newer a
	n, err := opensearch.MigrateIndex(ctx, base, tc.Client(), log.NopLogger())
	require.NoError(t, err)
	require.Equal(t, 2, n, "only b and c are newly created (a is skipped, d untouched)")

	tc.Require.IndicesRefresh([]string{target}, nil)
	tc.Require.IndicesCount([]string{target}, nil, 4) // a, b, c, d

	hits := tc.Require.Search(target, strings.NewReader(`{"query":{"ids":{"values":["a"]}}}`))
	got := opensearchtest.SearchHitsMustBeConverted[search.Resource](t, hits.Hits)
	require.Len(t, got, 1)
	require.Equal(t, "NEW", got[0].Content, "the newer document must not be overwritten by the migration")
}

func TestPruneOldIndices(t *testing.T) {
	ctx := context.Background()
	base := "opencloud-prune"
	tc, target := newMigrateTest(t, base)
	old := base + "-v0"

	// refuses while the current-revision index does not exist yet
	_, err := opensearch.PruneOldIndices(ctx, tc.Client(), base, log.NopLogger())
	require.Error(t, err, "prune must refuse when the current index is missing")

	// legacy, an older versioned index, and the current index all exist
	tc.Require.IndicesCreate(base, strings.NewReader(`{}`))
	tc.Require.IndicesCreate(old, strings.NewReader(`{}`))
	tc.Require.IndicesCreate(target, strings.NewReader(opensearch.IndexManagerLatest.String()))

	n, err := opensearch.PruneOldIndices(ctx, tc.Client(), base, log.NopLogger())
	require.NoError(t, err)
	require.Equal(t, 2, n, "legacy base and base-v0 are pruned, base-v1 stays")

	for _, gone := range []string{base, old} {
		resp, _ := tc.Client().Indices.Exists(ctx, opensearchgoAPI.IndicesExistsReq{Indices: []string{gone}})
		require.Equal(t, 404, resp.StatusCode, "%s must be pruned", gone)
	}
	resp, err := tc.Client().Indices.Exists(ctx, opensearchgoAPI.IndicesExistsReq{Indices: []string{target}})
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode, "the current index must survive prune")
}

// TestMigrateStripsEmptyMtime covers the one legacy value that would otherwise
// break the reindex: an empty Mtime string, which is not a valid date. It must
// be stripped so the document survives instead of being rejected by the new
// date mapping.
func TestMigrateStripsEmptyMtime(t *testing.T) {
	ctx := context.Background()
	base := "opencloud-migrate-empty-mtime"
	tc, target := newMigrateTest(t, base)

	// an old document with an empty Mtime, exactly as the pre-fix code wrote it
	tc.Require.IndicesCreate(base, strings.NewReader(`{}`))
	tc.Require.DocumentCreate(base, "x", strings.NewReader(`{"ID":"x","Name":"f.txt","Path":"./f.txt","Mtime":""}`))
	tc.Require.IndicesCount([]string{base}, nil, 1)

	n, err := opensearch.MigrateIndex(ctx, base, tc.Client(), log.NopLogger())
	require.NoError(t, err)
	require.Equal(t, 1, n, "the empty-mtime document must survive, not be dropped")

	tc.Require.IndicesCount([]string{target}, nil, 1)
	hits := tc.Require.Search(target, strings.NewReader(`{"query":{"match_all":{}}}`))
	resources := opensearchtest.SearchHitsMustBeConverted[search.Resource](t, hits.Hits)
	require.Len(t, resources, 1)
	require.Nil(t, resources[0].Mtime, "the empty mtime should be dropped, not carried over")
}
