// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package bleve_test

import (
	"fmt"
	"strings"
	"sync"
	"time"

	bleveSearch "github.com/blevesearch/bleve/v2"
	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	bleveQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

type bleveIndex = bleveSearch.Index

// batchProbe records the batches written to the index and how many ran at the
// same time
type batchProbe struct {
	bleveIndex
	mu      sync.Mutex
	sizes   []int
	running int
	peak    int
}

func (p *batchProbe) Batch(b *bleveSearch.Batch) error {
	p.mu.Lock()
	p.sizes = append(p.sizes, b.Size())
	p.running++
	p.peak = max(p.peak, p.running)
	p.mu.Unlock()

	// long enough for batches pushed together to overlap if nothing serializes them
	time.Sleep(20 * time.Millisecond)
	err := p.bleveIndex.Batch(b)

	p.mu.Lock()
	p.running--
	p.mu.Unlock()
	return err
}

var _ = Describe("Batch", func() {
	var idx bleveSearch.Index

	BeforeEach(func() {
		mapping, err := bleve.NewMapping()
		Expect(err).ToNot(HaveOccurred())
		idx, err = bleveSearch.NewMemOnly(mapping)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { Expect(idx.Close()).To(Succeed()) })
	})

	file := func(i int, text string) search.Resource {
		return search.Resource{
			ID:       fmt.Sprintf("1$2!%d", i),
			RootID:   "1$2!2",
			ParentID: "1$2!2",
			Path:     fmt.Sprintf("./f%d.txt", i),
			Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
			Document: content.Document{Name: fmt.Sprintf("f%d.txt", i), Content: text},
		}
	}

	Describe("Upsert", func() {
		It("pushes the batch before a document that would overflow it", func() {
			batch, err := bleve.NewBatch(idx, 100)
			Expect(err).ToNot(HaveOccurred())
			bleve.SetMaxBytes(batch, 64*1024)
			text := strings.Repeat("lorem ipsum ", 40*1024/12)

			first := file(1, text)
			Expect(batch.Upsert(first.ID, first)).To(Succeed())
			Expect(idx.DocCount()).To(BeZero())

			second := file(2, text)
			Expect(batch.Upsert(second.ID, second)).To(Succeed())
			Expect(idx.DocCount()).To(Equal(uint64(1)))

			Expect(batch.Push()).To(Succeed())
			Expect(idx.DocCount()).To(Equal(uint64(2)))
		})

		It("pushes a document larger than the limit in a batch of its own", func() {
			probe := &batchProbe{bleveIndex: idx}
			batch, err := bleve.NewBatch(probe, 100)
			Expect(err).ToNot(HaveOccurred())
			bleve.SetMaxBytes(batch, 64*1024)

			for i, text := range []string{"lorem ipsum", strings.Repeat("lorem ipsum ", 100*1024/12), "lorem ipsum"} {
				r := file(i, text)
				Expect(batch.Upsert(r.ID, r)).To(Succeed())
			}
			Expect(batch.Push()).To(Succeed())

			Expect(probe.sizes).To(Equal([]int{1, 1, 1}))
		})

		It("collects small documents up to the count limit", func() {
			batch, err := bleve.NewBatch(idx, 3)
			Expect(err).ToNot(HaveOccurred())
			bleve.SetMaxBytes(batch, 64*1024)

			for i := range 2 {
				r := file(i, "lorem ipsum")
				Expect(batch.Upsert(r.ID, r)).To(Succeed())
			}
			Expect(idx.DocCount()).To(BeZero())

			r := file(2, "lorem ipsum")
			Expect(batch.Upsert(r.ID, r)).To(Succeed())
			Expect(idx.DocCount()).To(Equal(uint64(3)))
		})
	})

	Describe("Push", func() {
		It("runs one batch at a time when several are pushed together", func() {
			probe := &batchProbe{bleveIndex: idx}
			eng := bleve.NewBackend(probe, bleveQuery.DefaultCreator, log.NopLogger())

			var wg sync.WaitGroup
			for i := range 4 {
				wg.Go(func() {
					defer GinkgoRecover()
					r := file(i, "lorem ipsum")
					Expect(eng.Upsert(r.ID, r)).To(Succeed())
				})
			}
			wg.Wait()

			Expect(probe.peak).To(Equal(1))
			Expect(eng.DocCount()).To(Equal(uint64(4)))
		})
	})
})
