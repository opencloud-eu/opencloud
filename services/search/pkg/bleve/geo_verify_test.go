package bleve_test

import (
	"sort"
	"testing"

	bleveSearch "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// geoFixture builds an in-memory bleve index with a single resource that
// carries the given lon/lat/alt. Used by the search tests below.
func geoFixture(t *testing.T, lon, lat, alt float64) bleveSearch.Index {
	t.Helper()
	idxMapping, err := bleve.NewMapping()
	if err != nil {
		t.Fatalf("NewMapping: %v", err)
	}
	idx, err := bleveSearch.NewMemOnly(idxMapping)
	if err != nil {
		t.Fatalf("NewMemOnly: %v", err)
	}
	r := search.Resource{
		ID: "x",
		Document: content.Document{
			Name: "team.jpg",
			Location: &libregraph.GeoCoordinates{
				Longitude: &lon,
				Latitude:  &lat,
				Altitude:  &alt,
			},
		},
	}
	doc, err := mapping.PrepareForIndex(r, r.SearchFieldOverrides())
	if err != nil {
		t.Fatalf("PrepareForIndex: %v", err)
	}
	if err := idx.Index(r.ID, doc); err != nil {
		t.Fatalf("Index: %v", err)
	}
	return idx
}

// TestLocationAltitudeRoundTrip proves that every subfield of Location
// (including altitude) ends up in hit.Fields when a Resource is indexed
// through the full bleve pipeline. This is the invariant the Move /
// Delete / Restore round-trip depends on.
func TestLocationAltitudeRoundTrip(t *testing.T) {
	idx := geoFixture(t, 11.103870357204285, 49.48675890884328, 1047.7)

	req := bleveSearch.NewSearchRequest(bleveSearch.NewMatchAllQuery())
	req.Fields = []string{"*"}
	res, err := idx.Search(req)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) == 0 {
		t.Fatal("no hits")
	}
	keys := make([]string, 0, len(res.Hits[0].Fields))
	for k := range res.Hits[0].Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("hit.Fields keys: %v", keys)

	for _, k := range []string{"location.longitude", "location.latitude", "location.altitude"} {
		if _, ok := res.Hits[0].Fields[k]; !ok {
			t.Errorf("missing %q in hit.Fields (got %v)", k, keys)
		}
	}
}

func TestLocationLatitudeRangeQueryMatches(t *testing.T) {
	idx := geoFixture(t, 11.1, 49.48, 1000)

	// numeric range on the sub-field
	min, max := 49.0, 50.0
	incl := true
	q := query.NewNumericRangeInclusiveQuery(&min, &max, &incl, &incl)
	q.SetField("location.latitude")
	res, err := idx.Search(bleveSearch.NewSearchRequest(q))
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("latitude range: got %d hits, want 1", len(res.Hits))
	}

	// same range excluding the indexed latitude => no match
	lowMin, lowMax := 0.0, 10.0
	q2 := query.NewNumericRangeInclusiveQuery(&lowMin, &lowMax, &incl, &incl)
	q2.SetField("location.latitude")
	res, err = idx.Search(bleveSearch.NewSearchRequest(q2))
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 0 {
		t.Errorf("latitude range outside value: got %d hits, want 0", len(res.Hits))
	}
}

func TestLocationLongitudeRangeQueryMatches(t *testing.T) {
	idx := geoFixture(t, 11.1, 49.48, 1000)

	min, max := 11.0, 12.0
	incl := true
	q := query.NewNumericRangeInclusiveQuery(&min, &max, &incl, &incl)
	q.SetField("location.longitude")
	res, err := idx.Search(bleveSearch.NewSearchRequest(q))
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("longitude range: got %d hits, want 1", len(res.Hits))
	}
}

func TestLocationAltitudeRangeQueryMatches(t *testing.T) {
	idx := geoFixture(t, 11.1, 49.48, 1047.7)

	min := 1000.0
	incl := true
	q := query.NewNumericRangeInclusiveQuery(&min, nil, &incl, nil)
	q.SetField("location.altitude")
	res, err := idx.Search(bleveSearch.NewSearchRequest(q))
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("altitude >= 1000: got %d hits, want 1", len(res.Hits))
	}

	// altitude floor above the indexed value => no hits
	highMin := 2000.0
	q2 := query.NewNumericRangeInclusiveQuery(&highMin, nil, &incl, nil)
	q2.SetField("location.altitude")
	res, err = idx.Search(bleveSearch.NewSearchRequest(q2))
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 0 {
		t.Errorf("altitude >= 2000 against 1047.7: got %d hits, want 0", len(res.Hits))
	}
}

func TestLocationGeoDistanceQueryMatches(t *testing.T) {
	// Nuremberg-ish coordinates.
	idx := geoFixture(t, 11.103870357204285, 49.48675890884328, 1047.7)

	// 10 km radius around the indexed point should match.
	near := query.NewGeoDistanceQuery(11.103870357204285, 49.48675890884328, "10km")
	near.SetField("location" + mapping.GeopointSuffix)
	res, err := idx.Search(bleveSearch.NewSearchRequest(near))
	if err != nil {
		t.Fatalf("Search (near): %v", err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("geo distance near: got %d hits, want 1", len(res.Hits))
	}

	// Far away (Berlin, ~400 km) with a 10 km radius should miss.
	far := query.NewGeoDistanceQuery(13.404954, 52.520008, "10km")
	far.SetField("location" + mapping.GeopointSuffix)
	res, err = idx.Search(bleveSearch.NewSearchRequest(far))
	if err != nil {
		t.Fatalf("Search (far): %v", err)
	}
	if len(res.Hits) != 0 {
		t.Errorf("geo distance far: got %d hits, want 0", len(res.Hits))
	}
}
