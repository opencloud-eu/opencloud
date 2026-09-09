package bleve

import (
	"fmt"
	"runtime"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func newTestIndex(t testing.TB) bleve.Index {
	t.Helper()
	idx, _, err := NewIndex(t.TempDir(), log.NopLogger())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = idx.Close() })
	return idx
}

func indexResources(t testing.TB, idx bleve.Index, resources ...search.Resource) {
	t.Helper()
	batch := idx.NewBatch()
	for _, r := range resources {
		if err := batch.Index(r.ID, r); err != nil {
			t.Fatal(err)
		}
		if batch.Size() >= 1000 {
			if err := idx.Batch(batch); err != nil {
				t.Fatal(err)
			}
			batch.Reset()
		}
	}
	if err := idx.Batch(batch); err != nil {
		t.Fatal(err)
	}
}

func TestSearchResourcesByPath(t *testing.T) {
	idx := newTestIndex(t)

	const rootA, rootB = "s$a!root", "s$b!root"
	var docs []search.Resource
	add := func(root, id, path string) {
		docs = append(docs, search.Resource{ID: id, RootID: root, Path: path, Type: 1})
	}
	// 1001 descendants: crosses the 500-term batch boundary twice, once
	// mid-batch and once with a single-element tail
	var wantIDs []string
	for i := 0; i < 1001; i++ {
		id := fmt.Sprintf("s$a!f%04d", i)
		add(rootA, id, fmt.Sprintf("./big/f%04d.txt", i))
		wantIDs = append(wantIDs, id)
	}
	add(rootA, "s$a!big", "./big")           // the folder itself: not a descendant
	add(rootA, "s$a!big2", "./big2/x.txt")   // sibling with prefix name: excluded
	add(rootB, "s$b!clone", "./big/f0000.txt") // same path, other space: excluded
	// special characters the old query-string escaping had to handle
	add(rootA, "s$a!odd", `./odd name*[1]/file:with spaces?.txt`)
	indexResources(t, idx, docs...)

	got, err := searchResourcesByPath(rootA, "./big", idx)
	if err != nil {
		t.Fatal(err)
	}
	gotIDs := make([]string, 0, len(got))
	for _, r := range got {
		gotIDs = append(gotIDs, r.ID)
	}
	sort.Strings(gotIDs)
	sort.Strings(wantIDs)
	if len(gotIDs) != len(wantIDs) {
		t.Fatalf("expected %d descendants, got %d", len(wantIDs), len(gotIDs))
	}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("descendant sets differ at %d: want %s, got %s", i, wantIDs[i], gotIDs[i])
		}
	}

	odd, err := searchResourcesByPath(rootA, "./odd name*[1]", idx)
	if err != nil {
		t.Fatal(err)
	}
	if len(odd) != 1 || odd[0].ID != "s$a!odd" {
		t.Fatalf("special-character path: expected [s$a!odd], got %v", odd)
	}
}

// TestSearchResourcesByPathMemoryBounded guards against the descendant lookup
// regressing to an implementation whose live memory scales with the number of
// descendants (e.g. the former Path:<folder>/* wildcard, which materialised
// one term searcher per descendant and OOM-killed servers on folder deletes;
// it holds ~200MB here). The batched term-query implementation stays under
// ~20MB regardless of folder size.
func TestSearchResourcesByPathMemoryBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("indexes 20k documents")
	}
	idx := newTestIndex(t)

	const n, rootID = 20_000, "s$mem!root"
	docs := make([]search.Resource, 0, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("s$mem!f%05d", i)
		docs = append(docs, search.Resource{
			ID: id, RootID: rootID, Type: 1,
			Path: fmt.Sprintf("./big/dir%02d/file-%05d-%032x.txt", i%50, i, uint64(i)*2654435761),
		})
	}
	indexResources(t, idx, docs...)

	runtime.GC()
	var base runtime.MemStats
	runtime.ReadMemStats(&base)

	var peak atomic.Uint64
	stop := make(chan struct{})
	go func() {
		var m runtime.MemStats
		for {
			select {
			case <-stop:
				return
			default:
				runtime.ReadMemStats(&m)
				if m.HeapInuse > peak.Load() {
					peak.Store(m.HeapInuse)
				}
				time.Sleep(200 * time.Microsecond)
			}
		}
	}()

	res, err := searchResourcesByPath(rootID, "./big", idx)
	close(stop)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != n {
		t.Fatalf("expected %d descendants, got %d", n, len(res))
	}

	const limit = 64 << 20 // generous 3x headroom over the fix, far below the wildcard's cost
	if delta := peak.Load() - base.HeapInuse; delta > limit {
		t.Fatalf("descendant lookup held %dMB live heap for %d docs (limit %dMB): "+
			"searcher memory must stay bounded, not scale with folder size", delta>>20, n, limit>>20)
	}
}
