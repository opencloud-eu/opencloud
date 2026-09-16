package bleve_test

import (
	"context"
	"time"

	bleveSearch "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	bleveQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// Open extensions bypass the mapping (IndexAdvanced), so the invariants the
// rest of the engine relies on are pinned here: the typed siblings exist as
// fields, they answer typed queries, the mapping is untouched, and the stored
// values survive a Move, which re-indexes the resource from a hit.
var _ = Describe("Open extensions", func() {
	var (
		idx bleveSearch.Index
		eng *bleve.Backend
	)

	key := func(property string) string {
		return "http://opencloud.eu/ns/extensions/com.example.project/" + property
	}
	stored := map[string]string{
		key("state"):    "s:Open",
		key("priority"): "n:3",
		key("done"):     "b:false",
		key("due"):      "d:2026-10-01T00:00:00Z",
		key("site"):     "g:52.5,13.4",
	}

	resource := func(id, path string, exts map[string]string) search.Resource {
		return search.Resource{
			ID:       id,
			RootID:   "1$2!2",
			ParentID: "1$2!2",
			Path:     path,
			Type:     1,
			Document: content.Document{Name: path[2:], OpenExtensions: exts},
		}
	}

	hits := func(q query.Query) []string {
		GinkgoHelper()
		req := bleveSearch.NewSearchRequest(q)
		req.Fields = []string{"*"}
		res, err := idx.Search(req)
		Expect(err).ToNot(HaveOccurred())
		ids := make([]string, 0, len(res.Hits))
		for _, h := range res.Hits {
			ids = append(ids, h.ID)
		}
		return ids
	}

	term := func(field, value string) query.Query {
		q := query.NewTermQuery(value)
		q.SetField(field)
		return q
	}

	BeforeEach(func() {
		m, err := bleve.NewMapping()
		Expect(err).ToNot(HaveOccurred())
		idx, err = bleveSearch.NewMemOnly(m)
		Expect(err).ToNot(HaveOccurred())
		eng = bleve.NewBackend(idx, bleveQuery.DefaultCreator, log.Logger{})

		Expect(eng.Upsert("1$2!a", resource("1$2!a", "./a.txt", stored))).To(Succeed())
		Expect(eng.Upsert("1$2!b", resource("1$2!b", "./b.txt", map[string]string{key("state"): "s:closed", key("priority"): "s:high"}))).To(Succeed())
		Expect(eng.Upsert("1$2!c", resource("1$2!c", "./c.txt", nil))).To(Succeed())
	})

	It("indexes every property under the sibling of its kind", func() {
		fields, err := idx.Fields()
		Expect(err).ToNot(HaveOccurred())
		Expect(fields).To(ContainElements(
			"ext.com.example.project.state.@keyword",
			"ext.com.example.project.state.@lower",
			"ext.com.example.project.priority.@number",
			"ext.com.example.project.priority.@keyword",
			"ext.com.example.project.done.@bool",
			"ext.com.example.project.due.@date",
			"ext.com.example.project.site.@geo",
		))
	})

	It("answers typed queries on the siblings", func() {
		Expect(hits(term("ext.com.example.project.state.@lower", "open"))).To(ConsistOf("1$2!a"))
		Expect(hits(term("ext.com.example.project.state.@keyword", "Open"))).To(ConsistOf("1$2!a"))
		Expect(hits(term("ext.com.example.project.state.@keyword", "open"))).To(BeEmpty(), "the keyword sibling is case-sensitive")

		lo, hi := 2.0, 4.0
		inclusive := true
		nq := query.NewNumericRangeInclusiveQuery(&lo, &hi, &inclusive, &inclusive)
		nq.SetField("ext.com.example.project.priority.@number")
		Expect(hits(nq)).To(ConsistOf("1$2!a"), "b's priority is a string and lives in the keyword sibling")
		Expect(hits(term("ext.com.example.project.priority.@lower", "high"))).To(ConsistOf("1$2!b"))

		bq := query.NewBoolFieldQuery(false)
		bq.SetField("ext.com.example.project.done.@bool")
		Expect(hits(bq)).To(ConsistOf("1$2!a"))

		start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
		dq := query.NewDateRangeQuery(start, end)
		dq.SetField("ext.com.example.project.due.@date")
		Expect(hits(dq)).To(ConsistOf("1$2!a"))

		gq := query.NewGeoDistanceQuery(13.4, 52.5, "1km")
		gq.SetField("ext.com.example.project.site.@geo")
		Expect(hits(gq)).To(ConsistOf("1$2!a"))
	})

	It("stores the values with the hit and keeps them through a Move", func() {
		req := bleveSearch.NewSearchRequest(bleveSearch.NewDocIDQuery([]string{"1$2!a"}))
		req.Fields = []string{"*"}
		res, err := idx.Search(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Hits).To(HaveLen(1))
		Expect(res.Hits[0].Fields).To(HaveKeyWithValue("OpenExtensions."+key("state"), "s:Open"))
		Expect(res.Hits[0].Fields).To(HaveKeyWithValue("OpenExtensions."+key("site"), "g:52.5,13.4"))

		Expect(eng.Move("1$2!a", "1$2!2", "./moved.txt")).To(Succeed())
		Expect(hits(term("ext.com.example.project.state.@lower", "open"))).To(ConsistOf("1$2!a"), "the re-indexed resource keeps its extensions")

		sr, err := eng.Search(context.Background(), &searchService.SearchIndexRequest{Query: `name:moved.txt`})
		Expect(err).ToNot(HaveOccurred())
		Expect(sr.Matches).To(HaveLen(1))
	})

	It("replaces the siblings on upsert instead of accumulating them", func() {
		Expect(eng.Upsert("1$2!a", resource("1$2!a", "./a.txt", map[string]string{key("state"): "s:closed"}))).To(Succeed())
		Expect(hits(term("ext.com.example.project.state.@lower", "open"))).To(BeEmpty())
		Expect(hits(term("ext.com.example.project.state.@lower", "closed"))).To(ConsistOf("1$2!a", "1$2!b"))

		lo, hi := 2.0, 4.0
		inclusive := true
		nq := query.NewNumericRangeInclusiveQuery(&lo, &hi, &inclusive, &inclusive)
		nq.SetField("ext.com.example.project.priority.@number")
		Expect(hits(nq)).To(BeEmpty(), "a property that is gone leaves no sibling behind")
	})

	It("leaves the stored mapping untouched", func() {
		fields, err := idx.Fields()
		Expect(err).ToNot(HaveOccurred())
		Expect(fields).To(ContainElement("ext.com.example.project.state.@lower"))

		// the mapping only knows the resource struct; an extension field never
		// becomes part of it, so the reconciler never sees one
		m := idx.Mapping()
		Expect(m.AnalyzerNameForPath("ext.com.example.project.state.@lower")).To(Equal(m.AnalyzerNameForPath("does.not.exist")))
	})
})
