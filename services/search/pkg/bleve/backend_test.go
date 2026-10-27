package bleve_test

import (
	"context"
	"fmt"
	"time"

	bleveSearch "github.com/blevesearch/bleve/v2"
	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config/defaults"
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

		eng = bleve.NewBackend(idx, bleveQuery.Compiler{}, log.Logger{})
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

	Describe("Search", func() {
		It("applies the default query template", func() {
			keep := func(id, path, name string) search.Resource {
				return search.Resource{ID: id, RootID: rootResource.ID, Path: path, Document: content.Document{Name: name, Tags: []string{"keep"}}}
			}
			for _, r := range []search.Resource{
				keep(rootResource.ID, ".", "root"),
				keep("1$2!5", "./.space/readme.md", "readme.md"),
				keep("1$2!6", "./child.pdf", "child.pdf"),
			} {
				Expect(eng.Upsert(r.ID, r)).To(Succeed())
			}
			template, err := search.ParseQueryTemplate(defaults.DefaultConfig().QueryTemplate)
			Expect(err).ToNot(HaveOccurred())
			query, err := search.ParseQuery(`tag:keep`)
			Expect(err).ToNot(HaveOccurred())
			space := &sprovider.StorageSpace{SpaceType: "personal", Root: &sprovider.ResourceId{StorageId: "1", SpaceId: "2", OpaqueId: "2"}}

			res, err := eng.Search(context.Background(), &search.IndexQuery{Query: template.Expand(query, space)})
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Matches).To(HaveLen(1))
			Expect(res.Matches[0].Entity.Name).To(Equal("child.pdf"))
		})
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

	Describe("Update", func() {
		tagHits := func(tag string) []string {
			query, err := kql.Builder{}.Build(`tag:("` + tag + `")`)
			Expect(err).ToNot(HaveOccurred())
			res, err := eng.Search(context.Background(), &search.IndexQuery{Query: query})
			Expect(err).ToNot(HaveOccurred())
			var ids []string
			for _, match := range res.Matches {
				ids = append(ids, match.GetEntity().GetId().GetOpaqueId())
			}
			return ids
		}

		It("rewrites tags and favorites and keeps the extracted fields", func() {
			childResource.Content = "quarterly report"
			childResource.Tags = []string{"alpha"}
			Expect(eng.Upsert(childResource.ID, childResource)).To(Succeed())

			Expect(eng.Update(context.Background(), childResource.ID, search.MetadataUpdate{
				Tags:      []string{"Beta"},
				Favorites: []string{"user-1"},
			})).To(Succeed())

			r, err := eng.Get(context.Background(), childResource.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Tags).To(Equal([]string{"Beta"}))
			Expect(r.Favorites).To(Equal([]string{"user-1"}))
			Expect(r.Content).To(Equal("quarterly report"))

			Expect(tagHits("beta")).To(ConsistOf("4"), "the lowercase sibling follows the new tag")
			Expect(tagHits("alpha")).To(BeEmpty())
		})

		It("clears the tags", func() {
			childResource.Tags = []string{"alpha"}
			Expect(eng.Upsert(childResource.ID, childResource)).To(Succeed())

			Expect(eng.Update(context.Background(), childResource.ID, search.MetadataUpdate{})).To(Succeed())

			r, err := eng.Get(context.Background(), childResource.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Tags).To(BeEmpty())
			Expect(tagHits("alpha")).To(BeEmpty())
		})

		It("fails with ErrResourceNotFound for an unknown id", func() {
			err := eng.Update(context.Background(), "1$2!unknown", search.MetadataUpdate{Tags: []string{"alpha"}})
			Expect(err).To(MatchError(search.ErrResourceNotFound))
		})
	})

	Describe("New", func() {
		It("returns a new index instance", func() {
			b := bleve.NewBackend(idx, bleveQuery.Compiler{}, log.Logger{})
			Expect(b).ToNot(BeNil())
		})
	})

})
