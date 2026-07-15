package bleve_test

import (
	"time"

	"github.com/opencloud-eu/opencloud/pkg/conversions"

	bleveSearch "github.com/blevesearch/bleve/v2"
	bquery "github.com/blevesearch/bleve/v2/search/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// Mtime is typed as a date, so range queries are chronological, not a
// lexicographic keyword compare.
var _ = Describe("Mtime date range", func() {
	It("compares chronologically, not lexicographically", func() {
		m, err := bleve.NewMapping()
		Expect(err).ToNot(HaveOccurred())
		idx, err := bleveSearch.NewMemOnly(m)
		Expect(err).ToNot(HaveOccurred())

		r := search.Resource{ID: "x", Document: content.Document{Name: "f", Mtime: conversions.ToPointer(time.Date(2026, 3, 15, 12, 0, 0, 123456789, time.UTC))}}
		doc, err := mapping.PrepareForIndex(r, r.SearchFieldOverrides())
		Expect(err).ToNot(HaveOccurred())
		Expect(idx.Index(r.ID, doc)).To(Succeed())

		hits := func(qs string) uint64 {
			res, err := idx.Search(bleveSearch.NewSearchRequest(bquery.NewQueryStringQuery(qs)))
			Expect(err).ToNot(HaveOccurred(), qs)
			return res.Total
		}
		Expect(hits(`Mtime:>"2026-01-01T00:00:00Z"`)).To(Equal(uint64(1)), "in-range")
		Expect(hits(`Mtime:>"2026-06-01T00:00:00Z"`)).To(Equal(uint64(0)), "out-of-range")
	})
})
