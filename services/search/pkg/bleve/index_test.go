package bleve_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	bleveSearch "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	bleveMapping "github.com/blevesearch/bleve/v2/mapping"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ = Describe("NewIndex", func() {
	var root string

	BeforeEach(func() {
		root = GinkgoT().TempDir()
	})

	codeMapping := func() *bleveMapping.IndexMappingImpl {
		m, err := bleve.NewMapping()
		Expect(err).ToNot(HaveOccurred())
		impl, ok := m.(*bleveMapping.IndexMappingImpl)
		Expect(ok).To(BeTrue())
		return impl
	}

	// buildIndex simulates an index left behind by an older release
	buildIndex := func(m bleveMapping.IndexMapping, docs map[string]map[string]any) {
		idx, err := bleveSearch.New(filepath.Join(root, fmt.Sprintf("bleve-v%d", search.SchemaVersion)), m)
		Expect(err).ToNot(HaveOccurred())
		for id, doc := range docs {
			Expect(idx.Index(id, doc)).To(Succeed())
		}
		Expect(idx.Close()).To(Succeed())
	}

	It("creates a fresh index", func() {
		idx, classification, err := bleve.NewIndex(root, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(classification.Verdict).To(Equal(searchmapping.VerdictEqual))
		Expect(idx.Close()).To(Succeed())
	})

	It("opens an index with an identical schema", func() {
		buildIndex(codeMapping(), nil)

		idx, classification, err := bleve.NewIndex(root, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(classification.Verdict).To(Equal(searchmapping.VerdictEqual))
		Expect(classification.NewFields).To(BeEmpty())
		Expect(idx.Close()).To(Succeed())
	})

	It("treats a genuinely new field as additive", func() {
		old := codeMapping()
		Expect(old.DefaultMapping.Properties).To(HaveKey("Title"))
		delete(old.DefaultMapping.Properties, "Title")
		buildIndex(old, nil)

		idx, classification, err := bleve.NewIndex(root, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(classification.Verdict).To(Equal(searchmapping.VerdictAdditive))
		Expect(classification.NewFields).To(ConsistOf("Title"))
		Expect(idx.Index("1", map[string]any{"Title": "hello"})).To(Succeed())
		Expect(idx.Close()).To(Succeed())
	})

	It("treats a new nested field as additive", func() {
		old := codeMapping()
		photo := old.DefaultMapping.Properties["photo"]
		Expect(photo).ToNot(BeNil())
		Expect(photo.Properties).To(HaveKey("cameraMake"))
		delete(photo.Properties, "cameraMake")
		buildIndex(old, nil)

		idx, classification, err := bleve.NewIndex(root, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(classification.Verdict).To(Equal(searchmapping.VerdictAdditive))
		Expect(classification.NewFields).To(ConsistOf("photo.cameraMake"))
		Expect(idx.Close()).To(Succeed())
	})

	It("persists an additive schema change so later startups classify it as equal", func() {
		old := codeMapping()
		delete(old.DefaultMapping.Properties, "Title")
		buildIndex(old, nil)

		idx, classification, err := bleve.NewIndex(root, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(classification.Verdict).To(Equal(searchmapping.VerdictAdditive))
		Expect(idx.Index("1", map[string]any{"Title": "hello"})).To(Succeed())
		Expect(idx.Close()).To(Succeed())

		idx, classification, err = bleve.NewIndex(root, log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(classification.Verdict).To(Equal(searchmapping.VerdictEqual))
		Expect(idx.Close()).To(Succeed())
	})

	It("refuses when a new field already has data in the index", func() {
		old := codeMapping()
		Expect(old.DefaultMapping.Properties).To(HaveKey("Mtime"))
		delete(old.DefaultMapping.Properties, "Mtime")
		buildIndex(old, map[string]map[string]any{"1": {"Mtime": "2026-01-02T03:04:05Z"}})

		idx, _, err := bleve.NewIndex(root, log.NopLogger())
		Expect(err).To(MatchError(searchmapping.ErrManualActionRequired))
		Expect(idx).To(BeNil())
	})

	It("refuses when a new object field already has nested data in the index", func() {
		old := codeMapping()
		Expect(old.DefaultMapping.Properties).To(HaveKey("photo"))
		delete(old.DefaultMapping.Properties, "photo")
		buildIndex(old, map[string]map[string]any{"1": {"photo": map[string]any{"cameraMake": "ACME"}}})

		_, _, err := bleve.NewIndex(root, log.NopLogger())
		Expect(err).To(MatchError(searchmapping.ErrManualActionRequired))
	})

	It("refuses on a changed field definition", func() {
		old := codeMapping()
		name := old.DefaultMapping.Properties["Name"]
		Expect(name).ToNot(BeNil())
		Expect(name.Fields).ToNot(BeEmpty())
		name.Fields[0].Analyzer = "fulltext"
		buildIndex(old, nil)

		_, _, err := bleve.NewIndex(root, log.NopLogger())
		Expect(err).To(MatchError(searchmapping.ErrManualActionRequired))
	})

	It("refuses when a stored field was removed from the code schema", func() {
		old := codeMapping()
		old.DefaultMapping.AddFieldMappingsAt("Legacy", bleveSearch.NewTextFieldMapping())
		buildIndex(old, nil)

		_, _, err := bleve.NewIndex(root, log.NopLogger())
		Expect(err).To(MatchError(searchmapping.ErrManualActionRequired))
	})

	It("refuses when a default_mapping attribute changed", func() {
		old := codeMapping()
		old.DefaultMapping.Dynamic = false
		buildIndex(old, nil)

		_, _, err := bleve.NewIndex(root, log.NopLogger())
		Expect(err).To(MatchError(searchmapping.ErrManualActionRequired))
	})

	It("refuses on a changed analyzer definition", func() {
		old := codeMapping()
		Expect(old.CustomAnalysis.Analyzers).To(HaveKey("fulltext"))
		old.CustomAnalysis.Analyzers["fulltext"] = map[string]any{
			"type":          custom.Name,
			"tokenizer":     unicode.Name,
			"token_filters": []string{lowercase.Name},
		}
		buildIndex(old, nil)

		_, _, err := bleve.NewIndex(root, log.NopLogger())
		Expect(err).To(MatchError(searchmapping.ErrManualActionRequired))
	})
})

var _ = Describe("NewMapping", func() {
	It("only references registered analyzers", func() {
		m, err := bleve.NewMapping()
		Expect(err).ToNot(HaveOccurred())
		impl, ok := m.(*bleveMapping.IndexMappingImpl)
		Expect(ok).To(BeTrue())
		Expect(impl.Validate()).To(Succeed())
	})

	// A diff here means existing indexes will classify as breaking (schema or
	// bleve marshaling changed); update the golden file only deliberately.
	It("matches the committed golden mapping", func() {
		m, err := bleve.NewMapping()
		Expect(err).ToNot(HaveOccurred())
		b, err := json.Marshal(m)
		Expect(err).ToNot(HaveOccurred())
		var got, golden map[string]any
		Expect(json.Unmarshal(b, &got)).To(Succeed())

		goldenB, err := os.ReadFile("testdata/mapping.golden.json")
		Expect(err).ToNot(HaveOccurred())
		Expect(json.Unmarshal(goldenB, &golden)).To(Succeed())

		Expect(got).To(Equal(golden))
	})
})
