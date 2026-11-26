package bleve_test

import (
	"context"
	"fmt"
	"time"

	bleveSearch "github.com/blevesearch/bleve/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	bleveQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ = Describe("Optimize", func() {
	// segments reads the counters scorch keeps of the segments in its root
	segments := func(idx bleveSearch.Index) (memory, file uint64) {
		stats, ok := idx.StatsMap()["index"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(stats).To(HaveKeyWithValue("TotMemorySegmentsAtRoot", BeAssignableToTypeOf(uint64(0))))
		Expect(stats).To(HaveKeyWithValue("TotFileSegmentsAtRoot", BeAssignableToTypeOf(uint64(0))))
		return stats["TotMemorySegmentsAtRoot"].(uint64), stats["TotFileSegmentsAtRoot"].(uint64)
	}

	// the persister and merger of scorch run in the background, a busy machine
	// needs longer than the default second
	const settle = 10 * time.Second

	newDiskBackend := func() (*bleve.Backend, bleveSearch.Index) {
		idx, _, err := bleve.NewIndex(GinkgoT().TempDir(), log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(idx.Close)
		return bleve.NewBackend(idx, bleveQuery.DefaultCreator, log.NopLogger()), idx
	}

	It("merges the segments of the index into one and keeps the documents", func() {
		eng, idx := newDiskBackend()

		const writes = 5
		for i := range writes {
			id := fmt.Sprintf("1$2!%d", i)
			Expect(eng.Upsert(id, search.Resource{ID: id, RootID: "1$2!2", Path: ".", Document: content.Document{Name: id}})).To(Succeed())
			// persisted one by one, scorch would otherwise merge the writes in memory
			Eventually(func() uint64 { m, _ := segments(idx); return m }).WithTimeout(settle).Should(BeZero())
		}
		_, before := segments(idx)
		Expect(before).To(BeNumerically(">", 1))

		Expect(eng.Optimize(context.Background())).To(Succeed())

		Eventually(func() uint64 { _, f := segments(idx); return f }).WithTimeout(settle).Should(Equal(uint64(1)))
		Expect(eng.DocCount()).To(Equal(uint64(writes)))
	})

	It("reports a context cancelled before the merge", func() {
		eng, _ := newDiskBackend()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		Expect(eng.Optimize(ctx)).To(MatchError(context.Canceled))
	})

	It("refuses a second optimization while one runs", func() {
		eng, _ := newDiskBackend()
		unlock := bleve.LockOptimize(eng)
		defer unlock()

		Expect(eng.Optimize(context.Background())).To(MatchError(search.ErrOptimizeInProgress))
	})

	It("reports an index that is not a scorch index as unsupported", func() {
		mapping, err := bleve.NewMapping()
		Expect(err).ToNot(HaveOccurred())
		idx, err := bleveSearch.NewMemOnly(mapping)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(idx.Close)

		eng := bleve.NewBackend(idx, bleveQuery.DefaultCreator, log.NopLogger())
		Expect(eng.Optimize(context.Background())).To(MatchError(search.ErrOptimizeUnsupported))
	})
})
