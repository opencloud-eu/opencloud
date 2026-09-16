package bleve

import (
	"fmt"

	"github.com/blevesearch/bleve/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func indexResources(idx bleve.Index, resources ...search.Resource) {
	batch := idx.NewBatch()
	for _, r := range resources {
		Expect(batch.Index(r.ID, r)).To(Succeed())
		if batch.Size() >= 1000 {
			Expect(idx.Batch(batch)).To(Succeed())
			batch.Reset()
		}
	}
	Expect(idx.Batch(batch)).To(Succeed())
}

var _ = Describe("forEachResourceByPath", func() {
	var idx bleve.Index

	BeforeEach(func() {
		var err error
		idx, _, err = NewIndex(GinkgoT().TempDir(), log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { Expect(idx.Close()).To(Succeed()) })
	})

	pageSize := func(n int) {
		old := descendantPageSize
		descendantPageSize = n
		DeferCleanup(func() { descendantPageSize = old })
	}

	count := func(path string, deleted *bool) uint64 {
		pq := bleve.NewTermQuery(path)
		pq.SetField("Path")
		q := bleve.NewConjunctionQuery(pq)
		if deleted != nil {
			dq := bleve.NewBoolFieldQuery(*deleted)
			dq.SetField("Deleted")
			q.AddQuery(dq)
		}
		req := bleve.NewSearchRequest(q)
		req.Size = 0
		res, err := idx.Search(req)
		Expect(err).ToNot(HaveOccurred())
		return res.Total
	}
	deleted := true

	DescribeTable("walks a folder larger than one page while the batch pushes between pages",
		func(op func(*Batch) error, check func()) {
			pageSize(1000)
			const n, rootID = 6000, "s$op!root"
			docs := []search.Resource{{ID: "s$op!big", RootID: rootID, Path: "./big", Type: 2}}
			for i := 0; i < n; i++ {
				docs = append(docs, search.Resource{ID: fmt.Sprintf("s$op!f%05d", i), RootID: rootID, Path: fmt.Sprintf("./big/f%05d.txt", i), Type: 1})
			}
			indexResources(idx, docs...)

			batch, err := NewBatch(idx, 100)
			Expect(err).ToNot(HaveOccurred())
			Expect(op(batch)).To(Succeed())
			Expect(batch.Push()).To(Succeed())
			check()
		},
		Entry("move", func(b *Batch) error { return b.Move("s$op!big", "s$op!root", "./moved") }, func() {
			Expect(count("./big", nil)).To(BeZero())
			Expect(count("./moved", nil)).To(Equal(uint64(6001)))
		}),
		Entry("delete", func(b *Batch) error { return b.Delete("s$op!big") }, func() {
			Expect(count("./big", &deleted)).To(Equal(uint64(6001)))
		}),
		Entry("purge", func(b *Batch) error {
			if err := b.Delete("s$op!big"); err != nil {
				return err
			}
			if err := b.Push(); err != nil {
				return err
			}
			return b.Purge("s$op!big", true)
		}, func() {
			Expect(count("./big", nil)).To(BeZero())
		}),
	)
})
