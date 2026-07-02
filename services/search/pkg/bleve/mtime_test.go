package bleve_test

import (
	"testing"

	bleveSearch "github.com/blevesearch/bleve/v2"
	bquery "github.com/blevesearch/bleve/v2/search/query"

	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// Mtime is typed as a date, so range queries are chronological, not a
// lexicographic keyword compare.
func TestMtimeDateRange(t *testing.T) {
	m, err := bleve.NewMapping()
	if err != nil {
		t.Fatal(err)
	}
	idx, err := bleveSearch.NewMemOnly(m)
	if err != nil {
		t.Fatal(err)
	}
	r := search.Resource{ID: "x", Document: content.Document{Name: "f", Mtime: "2026-03-15T12:00:00.123456789Z"}}
	doc, err := mapping.PrepareForIndex(r, r.SearchFieldOverrides())
	if err != nil {
		t.Fatal(err)
	}
	if err := idx.Index(r.ID, doc); err != nil {
		t.Fatal(err)
	}

	hits := func(qs string) uint64 {
		res, err := idx.Search(bleveSearch.NewSearchRequest(bquery.NewQueryStringQuery(qs)))
		if err != nil {
			t.Fatalf("%s: %v", qs, err)
		}
		return res.Total
	}
	if got := hits(`Mtime:>"2026-01-01T00:00:00Z"`); got != 1 {
		t.Errorf("in-range: got %d hits, want 1", got)
	}
	if got := hits(`Mtime:>"2026-06-01T00:00:00Z"`); got != 0 {
		t.Errorf("out-of-range: got %d hits, want 0", got)
	}
}
