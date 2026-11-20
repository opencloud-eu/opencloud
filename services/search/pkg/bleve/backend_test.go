package bleve_test

import (
	"fmt"
	"os"

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

var _ = Describe("Bleve", func() {
	var (
		eng *bleve.Backend
		idx bleveSearch.Index

		rootResource   search.Resource
		parentResource search.Resource
		childResource  search.Resource
	)

	BeforeEach(func() {
		mapping, err := bleve.NewMapping()
		Expect(err).ToNot(HaveOccurred())

		idx, err = bleveSearch.NewMemOnly(mapping)
		Expect(err).ToNot(HaveOccurred())

		eng = bleve.NewBackend(idx, bleveQuery.DefaultCreator, log.Logger{})
		Expect(err).ToNot(HaveOccurred())

		rootResource = search.Resource{
			ID:       "1$2!2",
			RootID:   "1$2!2",
			Path:     ".",
			Document: content.Document{},
		}

		parentResource = search.Resource{
			ID:       "1$2!3",
			ParentID: rootResource.ID,
			RootID:   rootResource.ID,
			Path:     "./parent d!r",
			Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_CONTAINER),
			Document: content.Document{Name: "parent d!r"},
		}

		childResource = search.Resource{
			ID:       "1$2!4",
			ParentID: parentResource.ID,
			RootID:   rootResource.ID,
			Path:     "./parent d!r/child.pdf",
			Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
			Document: content.Document{Name: "child.pdf"},
		}

	})

	Describe("PurgeSpace", func() {
		It("takes every record of that space out of the index", func() {
			otherSpace := search.Resource{
				ID:       "1$9!9",
				RootID:   "1$9!9",
				Path:     ".",
				Document: content.Document{Name: "other"},
			}
			for _, resource := range []search.Resource{rootResource, parentResource, childResource, otherSpace} {
				Expect(eng.Upsert(resource.ID, resource)).To(Succeed())
			}

			Expect(eng.PurgeSpace(rootResource.RootID)).To(Succeed())

			count, err := idx.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(uint64(1)), "only the records of that space are gone")
		})

		It("takes a space out that holds more records than one round", func() {
			otherSpace := search.Resource{
				ID:       "1$9!9",
				RootID:   "1$9!9",
				Path:     ".",
				Document: content.Document{Name: "other"},
			}
			Expect(eng.Upsert(otherSpace.ID, otherSpace)).To(Succeed())

			for i := range 120 {
				resource := search.Resource{
					ID:       fmt.Sprintf("%s!file-%d", rootResource.RootID, i),
					RootID:   rootResource.RootID,
					Path:     fmt.Sprintf("./file-%d", i),
					Document: content.Document{Name: fmt.Sprintf("file-%d", i)},
				}
				Expect(eng.Upsert(resource.ID, resource)).To(Succeed())
			}

			Expect(eng.PurgeSpace(rootResource.RootID)).To(Succeed())

			count, err := idx.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(uint64(1)), "only the record of the other space is left")
		})
	})

	Describe("Reload", func() {
		It("reopens the index from disk", func() {
			diskIdx, _, err := bleve.NewIndex(GinkgoT().TempDir(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())

			b := bleve.NewBackend(diskIdx, bleveQuery.DefaultCreator, log.NopLogger())
			DeferCleanup(func() { _ = b.Close() })

			Expect(b.Upsert(rootResource.ID, rootResource)).To(Succeed())
			Expect(b.Reload()).To(Succeed())

			count, err := b.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(uint64(1)))
			Expect(diskIdx.DocCount()).Error().To(HaveOccurred(), "the old index is closed")
		})
	})

	Describe("Reload fails to reopen", func() {
		It("does not panic on Close or a second Reload", func() {
			diskIdx, _, err := bleve.NewIndex(GinkgoT().TempDir(), log.NopLogger())
			Expect(err).ToNot(HaveOccurred())

			b := bleve.NewBackend(diskIdx, bleveQuery.DefaultCreator, log.NopLogger())
			DeferCleanup(func() { _ = b.Close() })

			Expect(os.Rename(diskIdx.Name(), diskIdx.Name()+".moved")).To(Succeed())

			Expect(b.Reload()).ToNot(Succeed())
			Expect(b.Close()).To(Succeed())
			Expect(b.Reload()).To(MatchError(Not(ContainSubstring("panic"))))
		})
	})

	Describe("Close", func() {
		It("closes the index", func() {
			Expect(eng.Close()).To(Succeed())
			Expect(idx.DocCount()).Error().To(HaveOccurred())
		})
	})

	Describe("New", func() {
		It("returns a new index instance", func() {
			b := bleve.NewBackend(idx, bleveQuery.DefaultCreator, log.Logger{})
			Expect(b).ToNot(BeNil())
		})
	})

})
