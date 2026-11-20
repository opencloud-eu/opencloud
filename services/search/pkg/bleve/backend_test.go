package bleve_test

import (
	"context"
	"fmt"
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

	Describe("Get", func() {
		It("returns the indexed resource with its exact mtime", func() {
			mtime := time.Unix(1700000000, 123456789).UTC()
			childResource.Mtime = &mtime
			Expect(eng.Upsert(childResource.ID, childResource)).To(Succeed())

			r, err := eng.Get(context.Background(), childResource.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.ID).To(Equal(childResource.ID))
			Expect(r.Mtime).ToNot(BeNil())
			Expect(r.Mtime.Equal(mtime)).To(BeTrue(), "got %s", r.Mtime)
		})

		It("returns a trashed resource", func() {
			childResource.Deleted = true
			Expect(eng.Upsert(childResource.ID, childResource)).To(Succeed())

			r, err := eng.Get(context.Background(), childResource.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Deleted).To(BeTrue())
		})

		It("fails with ErrResourceNotFound for an unknown id", func() {
			_, err := eng.Get(context.Background(), "1$2!unknown")
			Expect(err).To(MatchError(search.ErrResourceNotFound))
		})
	})

	Describe("New", func() {
		It("returns a new index instance", func() {
			b := bleve.NewBackend(idx, bleveQuery.DefaultCreator, log.Logger{})
			Expect(b).ToNot(BeNil())
		})
	})

})
