package bleve

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// BenchmarkSearchResourcesByPath measures the descendant lookup that backs
// folder Delete/Move/Restore/Purge. Cumulative allocations are similar for any
// implementation that visits every descendant; what OOMs servers is the peak
// LIVE heap while the lookup runs, so that is reported as peak-MB.
func BenchmarkSearchResourcesByPath(b *testing.B) {
	for _, n := range []int{20_000, 100_000} {
		b.Run(fmt.Sprintf("docs=%d", n), func(b *testing.B) {
			idx, _, err := NewIndex(b.TempDir(), log.NopLogger())
			if err != nil {
				b.Fatal(err)
			}
			defer idx.Close()

			const rootID = "storage$space!root"
			batch := idx.NewBatch()
			for i := 0; i < n; i++ {
				id := fmt.Sprintf("storage$space!f%06d", i)
				err := batch.Index(id, search.Resource{
					ID:       id,
					RootID:   rootID,
					ParentID: "storage$space!big",
					Path:     fmt.Sprintf("./big/dir%02d/file-%06d-%032x.txt", i%50, i, uint64(i)*2654435761),
					Type:     1,
				})
				if err != nil {
					b.Fatal(err)
				}
				if batch.Size() >= 1000 {
					if err := idx.Batch(batch); err != nil {
						b.Fatal(err)
					}
					batch.Reset()
				}
			}
			if err := idx.Batch(batch); err != nil {
				b.Fatal(err)
			}

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

			b.ResetTimer()
			for b.Loop() {
				res, err := searchResourcesByPath(rootID, "./big", idx)
				if err != nil {
					b.Fatal(err)
				}
				if len(res) != n {
					b.Fatalf("expected %d descendants, got %d", n, len(res))
				}
			}
			b.StopTimer()
			close(stop)
			b.ReportMetric(float64(peak.Load()-base.HeapInuse)/1e6, "peak-MB")
		})
	}
}
