package bleve_test

import (
	"context"
	"fmt"
	"os"
	"time"

	bleveSearch "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/index/scorch"
	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	bleveQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ = Describe("Bleve", func() {
	var (
		eng *bleve.Backend
		idx bleveSearch.Index

		doSearch = func(id string, query, path string) (*searchsvc.SearchIndexResponse, error) {
			rID, err := storagespace.ParseID(id)
			if err != nil {
				return nil, err
			}

			return eng.Search(context.Background(), &searchsvc.SearchIndexRequest{
				Query: query,
				Ref: &searchmsg.Reference{
					ResourceId: &searchmsg.ResourceID{
						StorageId: rID.StorageId,
						SpaceId:   rID.SpaceId,
						OpaqueId:  rID.OpaqueId,
					},
					Path: path,
				},
			})
		}

		assertDocCount = func(id string, query string, expectedCount int) []*searchmsg.Match {
			res, err := doSearch(id, query, "")

			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			ExpectWithOffset(1, len(res.Matches)).To(Equal(expectedCount), "query returned unexpected number of results: "+query)
			return res.Matches
		}

		rootResource   search.Resource
		parentResource search.Resource
		childResource  search.Resource
		childResource2 search.Resource
	)

	BeforeEach(func() {
		mapping, err := bleve.NewMapping()
		Expect(err).ToNot(HaveOccurred())

		tmpDir, err := os.MkdirTemp("", "bleve-test-")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpDir)
		idx, err = bleveSearch.NewUsing(tmpDir, mapping, scorch.Name, bleveSearch.Config.DefaultKVStore, nil)
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

		childResource2 = search.Resource{
			ID:       "1$2!5",
			ParentID: parentResource.ID,
			RootID:   rootResource.ID,
			Path:     "./parent d!r/child2.pdf",
			Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
			Document: content.Document{Name: "child2.pdf"},
		}
	})

	Describe("New", func() {
		It("returns a new index instance", func() {
			b := bleve.NewBackend(idx, bleveQuery.DefaultCreator, log.Logger{})
			Expect(b).ToNot(BeNil())
		})
	})

	Describe("Search", func() {
		Context("by other fields than filename", func() {
			It("finds files by tags", func() {
				parentResource.Document.Tags = []string{"foo", "bar"}
				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				assertDocCount(rootResource.ID, "Tags:foo", 1)
				assertDocCount(rootResource.ID, "Tags:bar", 1)
				assertDocCount(rootResource.ID, "Tags:foo Tags:bar", 1)
				assertDocCount(rootResource.ID, "Tags:foo Tags:bar Tags:baz", 1)
				assertDocCount(rootResource.ID, "Tags:foo Tags:bar Tags:baz", 1)
				assertDocCount(rootResource.ID, "Tags:baz", 0)
			})

			It("finds files by tags case-insensitively", func() {
				// exercises the []string/[]any sibling-lowercasing branch end-to-end.
				parentResource.Document.Tags = []string{"Work", "Urgent"}
				Expect(eng.Upsert(parentResource.ID, parentResource)).To(Succeed())

				assertDocCount(rootResource.ID, "tag:work", 1) // stored "Work", queried lower
				assertDocCount(rootResource.ID, "tag:WORK", 1) // queried upper
				assertDocCount(rootResource.ID, "Tags:Urgent", 1)
				assertDocCount(rootResource.ID, "tag:missing", 0)
			})

			It("binds a leading NOT to the term right after it, combined with AND", func() {
				// regression: a leading NOT next to AND dropped the AND'd term, so
				// `NOT tag:x AND name:y` matched nothing (a self-contradicting clause).
				parentResource.Document.Tags = []string{"physik"}
				childResource.Document.Tags = []string{"mathe"}
				Expect(eng.Upsert(parentResource.ID, parentResource)).To(Succeed())
				Expect(eng.Upsert(childResource.ID, childResource)).To(Succeed())

				assertDocCount(rootResource.ID, "NOT tag:physik AND name:child.pdf", 1) // the mathe child
				assertDocCount(rootResource.ID, "NOT tag:mathe AND name:parent*", 1)    // the physik parent
			})

			It("finds files by size", func() {
				parentResource.Document.Size = 12345
				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				assertDocCount(rootResource.ID, "Size:12345", 1)
				assertDocCount(rootResource.ID, "Size:>1000", 1)
				assertDocCount(rootResource.ID, "Size:<100000", 1)
				assertDocCount(rootResource.ID, "Size:12344", 0)
				assertDocCount(rootResource.ID, "Size:<1000", 0)
				assertDocCount(rootResource.ID, "Size:>100000", 0)
			})

			It("preserves value case for fields not explicitly marked lowercase", func() {
				parentResource.Document.Audio = &libregraph.Audio{
					Artist: libregraph.PtrString("Some Artist"),
				}
				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				assertDocCount(rootResource.ID, `audio.artist:"Some Artist"`, 1)
				assertDocCount(rootResource.ID, `audio.artist:"some artist"`, 0)
			})
		})

		Context("by filename", func() {
			It("finds files with spaces in the filename", func() {
				parentResource.Document.Name = "Foo oo.pdf"
				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				assertDocCount(rootResource.ID, `name:"foo o*"`, 1)
			})

			It("finds files by digits in the filename", func() {
				parentResource.Document.Name = "12345.pdf"
				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				assertDocCount(rootResource.ID, "Name:1234*", 1)
			})

			It("filters hidden files", func() {
				childResource.Hidden = true
				err := eng.Upsert(childResource.ID, childResource)
				Expect(err).ToNot(HaveOccurred())

				assertDocCount(rootResource.ID, "Hidden:T", 1)
				assertDocCount(rootResource.ID, "Hidden:F", 0)
			})

			Context("with a file in the root of the space", func() {
				It("scopes the search to the specified space", func() {
					parentResource.Document.Name = "foo.pdf"
					err := eng.Upsert(parentResource.ID, parentResource)
					Expect(err).ToNot(HaveOccurred())

					assertDocCount(rootResource.ID, "Name:foo.pdf", 1)
					assertDocCount("9$8!7", "Name:foo.pdf", 0)
				})
			})

			It("limits the search to the specified fields", func() {
				parentResource.Document.Name = "bar.pdf"
				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				assertDocCount(rootResource.ID, "Name:bar.pdf", 1)
				assertDocCount(rootResource.ID, "Unknown:field", 0)
			})

			It("returns the total number of hits", func() {
				parentResource.Document.Name = "bar.pdf"
				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				res, err := doSearch(rootResource.ID, "Name:bar*", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(res.TotalMatches).To(Equal(int32(1)))
			})

			It("returns all desired fields", func() {
				parentResource.Document.Name = "bar.pdf"
				parentResource.Type = 3
				parentResource.MimeType = "application/pdf"

				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				matches := assertDocCount(rootResource.ID, fmt.Sprintf("Name:%s", parentResource.Name), 1)
				match := matches[0]
				Expect(match.Entity.Ref.Path).To(Equal(parentResource.Path))
				Expect(match.Entity.Name).To(Equal(parentResource.Name))
				Expect(match.Entity.Size).To(Equal(parentResource.Size))
				Expect(match.Entity.Type).To(Equal(parentResource.Type))
				Expect(match.Entity.MimeType).To(Equal(parentResource.MimeType))
				Expect(match.Entity.Deleted).To(BeFalse())
				Expect(match.Score > 0).To(BeTrue())
			})

			It("finds files by name, prefix or substring match", func() {
				parentResource.Document.Name = "foo.pdf"

				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				queries := []string{"foo.pdf", "foo*", "*oo.p*"}
				for _, query := range queries {
					err := eng.Upsert(parentResource.ID, parentResource)
					Expect(err).ToNot(HaveOccurred())

					assertDocCount(rootResource.ID, query, 1)
				}
			})

			It("does a case-insensitive search", func() {
				parentResource.Document.Name = "foo.pdf"

				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				assertDocCount(rootResource.ID, "Name:foo*", 1)
				assertDocCount(rootResource.ID, "Name:Foo*", 1)
			})

			Context("and an additional file in a subdirectory", func() {
				BeforeEach(func() {
					err := eng.Upsert(parentResource.ID, parentResource)
					Expect(err).ToNot(HaveOccurred())

					err = eng.Upsert(childResource.ID, childResource)
					Expect(err).ToNot(HaveOccurred())
				})

				It("finds files living deeper in the tree by filename, prefix or substring match", func() {
					queries := []string{"child.pdf", "child*", "*ld.*"}
					for _, query := range queries {
						assertDocCount(rootResource.ID, query, 1)
					}
				})
			})
		})

		Context("by path", func() {
			BeforeEach(func() {
				for _, r := range []search.Resource{parentResource, childResource, childResource2} {
					Expect(eng.Upsert(r.ID, r)).To(Succeed())
				}
			})

			It("matches a folder and its descendants", func() {
				assertDocCount(rootResource.ID, `path:"./parent d!r"`, 3)
			})

			It("matches a descendant path only itself", func() {
				assertDocCount(rootResource.ID, `path:"./parent d!r/child.pdf"`, 1)
			})

			It("matches case-sensitively", func() {
				// paths act as references: /Foo and /foo are distinct siblings,
				// so a wrong-cased path must not match
				assertDocCount(rootResource.ID, `path:"./PARENT D!R"`, 0)
			})

			It("applies an AND filter to the folder itself, not only descendants", func() {
				// regression: the folder-itself clause used to match unconditionally
				// under an AND, so the parent leaked in despite the name filter.
				matches := assertDocCount(rootResource.ID, `path:"./parent d!r" AND name:child.pdf`, 1)
				Expect(matches[0].Entity.Name).To(Equal("child.pdf"))
			})
		})

		Context("by content", func() {
			It("matches full-text case-insensitively and stemmed", func() {
				parentResource.Document.Content = "Running Foxes"
				Expect(eng.Upsert(parentResource.ID, parentResource)).To(Succeed())

				assertDocCount(rootResource.ID, "content:running", 1)
				assertDocCount(rootResource.ID, "content:RUNNING", 1) // case-insensitive
				assertDocCount(rootResource.ID, "content:run", 1)     // porter stemming
				assertDocCount(rootResource.ID, "content:run*", 1)    // wildcard over the stemmed term
				assertDocCount(rootResource.ID, "content:cat", 0)
			})
		})

		Context("by mediatype", func() {
			It("matches categories and literal MIME types (incl. + and /)", func() {
				childResource.Document.MimeType = "image/svg+xml"
				childResource2.Document.MimeType = "image/png"
				for _, r := range []search.Resource{childResource, childResource2} {
					Expect(eng.Upsert(r.ID, r)).To(Succeed())
				}

				assertDocCount(rootResource.ID, "mediatype:image", 2) // image/* wildcard -> both
				assertDocCount(rootResource.ID, "mediatype:IMAGE", 2) // categories are case-insensitive
				assertDocCount(rootResource.ID, "mediatype:pdf", 0)
				// literal MIME with + and /, must hit only the svg doc, not the png
				assertDocCount(rootResource.ID, "mediatype:image/svg+xml", 1)
				assertDocCount(rootResource.ID, "mediatype:image/png", 1)
				// the same literal via the raw field name (no mediatype alias)
				assertDocCount(rootResource.ID, "MimeType:image/svg+xml", 1)
				assertDocCount(rootResource.ID, "MimeType:image/png", 1)
			})

			It("combines mediatype:file with another term", func() {
				// regression: mediatype:file (a NOT) next to an operator dropped the
				// other operand, so mediatype:file AND name:x matched nothing.
				parentResource.Document.MimeType = "httpd/unix-directory" // a folder
				childResource.Document.MimeType = "image/png"             // a file
				for _, r := range []search.Resource{parentResource, childResource} {
					Expect(eng.Upsert(r.ID, r)).To(Succeed())
				}
				assertDocCount(rootResource.ID, "mediatype:file", 1)                    // only the file
				assertDocCount(rootResource.ID, "mediatype:file AND name:child.pdf", 1) // file AND its name
				assertDocCount(rootResource.ID, "mediatype:file AND name:nope", 0)
			})
		})

		Context("Highlights", func() {

			It("highlights only for content searches", func() {
				parentResource.Document.Name = "baz.pdf"
				parentResource.Document.Content = "foo bar baz"
				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				res, err := doSearch(rootResource.ID, "Name:baz*", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(res.TotalMatches).To(Equal(int32(1)))
				Expect(res.Matches[0].Entity.Highlights).To(Equal(""))
			})

			It("highlights search terms", func() {
				parentResource.Document.Name = "baz.pdf"
				parentResource.Document.Content = "foo bar baz"
				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				res, err := doSearch(rootResource.ID, "Content:bar", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(res.TotalMatches).To(Equal(int32(1)))
				Expect(res.Matches[0].Entity.Highlights).To(Equal("foo <mark>bar</mark> baz"))
			})

		})

		Context("with a file in the root of the space and folder with a file. all of them have the same name", func() {
			BeforeEach(func() {
				parentResource := search.Resource{
					ID:       "1$2!3",
					ParentID: rootResource.ID,
					RootID:   rootResource.ID,
					Path:     "./doc",
					Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_CONTAINER),
					Document: content.Document{Name: "doc"},
				}

				childResource := search.Resource{
					ID:       "1$2!4",
					ParentID: parentResource.ID,
					RootID:   rootResource.ID,
					Path:     "./doc/doc.pdf",
					Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
					Document: content.Document{Name: "doc.pdf"},
				}

				childResource2 := search.Resource{
					ID:       "1$2!7",
					ParentID: parentResource.ID,
					RootID:   rootResource.ID,
					Path:     "./doc/file.pdf",
					Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
					Document: content.Document{Name: "file.pdf"},
				}

				rootChildResource := search.Resource{
					ID:       "1$2!5",
					ParentID: rootResource.ID,
					RootID:   rootResource.ID,
					Path:     "./doc.pdf",
					Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
					Document: content.Document{Name: "doc.pdf"},
				}

				rootChildResource2 := search.Resource{
					ID:       "1$2!6",
					ParentID: rootResource.ID,
					RootID:   rootResource.ID,
					Path:     "./file.pdf",
					Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
					Document: content.Document{Name: "file.pdf"},
				}
				err := eng.Upsert(parentResource.ID, parentResource)
				Expect(err).ToNot(HaveOccurred())

				err = eng.Upsert(rootChildResource.ID, rootChildResource)
				Expect(err).ToNot(HaveOccurred())
				err = eng.Upsert(rootChildResource2.ID, rootChildResource2)
				Expect(err).ToNot(HaveOccurred())

				err = eng.Upsert(childResource.ID, childResource)
				Expect(err).ToNot(HaveOccurred())
				err = eng.Upsert(childResource2.ID, childResource2)
				Expect(err).ToNot(HaveOccurred())
			})
			It("search *doc* in a root", func() {
				res, err := doSearch(rootResource.ID, "Name:*doc*", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(res.TotalMatches).To(Equal(int32(3)))
			})
			It("search *doc* in a subfolder", func() {
				res, err := doSearch(rootResource.ID, "Name:*doc*", "./doc")
				Expect(err).ToNot(HaveOccurred())
				Expect(res.TotalMatches).To(Equal(int32(2)))
			})
			It("search *file* in a root", func() {
				res, err := doSearch(rootResource.ID, "Name:*file*", "")
				Expect(err).ToNot(HaveOccurred())
				Expect(res.TotalMatches).To(Equal(int32(2)))
			})
			It("search *file* in a subfolder", func() {
				res, err := doSearch(rootResource.ID, "Name:*file*", "./doc")
				Expect(err).ToNot(HaveOccurred())
				Expect(res.TotalMatches).To(Equal(int32(1)))
			})
		})

	})

	Describe("path scoped searches", func() {
		BeforeEach(func() {
			Expect(eng.Upsert(parentResource.ID, parentResource)).To(Succeed())
			Expect(eng.Upsert(childResource.ID, childResource)).To(Succeed())
			Expect(eng.Upsert(childResource2.ID, childResource2)).To(Succeed())
			outside := search.Resource{
				ID:       "1$2!6",
				ParentID: rootResource.ID,
				RootID:   rootResource.ID,
				Path:     "./other/child3.pdf",
				Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
				Document: content.Document{Name: "child3.pdf"},
			}
			Expect(eng.Upsert(outside.ID, outside)).To(Succeed())
		})

		It("restricts hits and totals to the scope at query level", func() {
			// without the scope all three children match
			res, err := doSearch(rootResource.ID, "name:child*", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.TotalMatches).To(Equal(int32(3)))

			res, err = doSearch(rootResource.ID, "name:child*", "./parent d!r")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.TotalMatches).To(Equal(int32(2)))
			Expect(len(res.Matches)).To(Equal(2))
		})

		It("keeps totals right on a small page", func() {
			// the scope is part of the query, so totals cover the full scope
			// even when the page holds a single hit
			rID, err := storagespace.ParseID(rootResource.ID)
			Expect(err).ToNot(HaveOccurred())
			res, err := eng.Search(context.Background(), &searchsvc.SearchIndexRequest{
				Query:    "name:child*",
				PageSize: 1,
				Ref: &searchmsg.Reference{
					ResourceId: &searchmsg.ResourceID{
						StorageId: rID.StorageId, SpaceId: rID.SpaceId, OpaqueId: rID.OpaqueId,
					},
					Path: "./parent d!r",
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(res.TotalMatches).To(Equal(int32(2)))
			Expect(len(res.Matches)).To(Equal(1))
		})

		It("matches the scope case-sensitively", func() {
			res, err := doSearch(rootResource.ID, "name:child*", "./PARENT D!R")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.TotalMatches).To(Equal(int32(0)))
		})
	})

	Describe("Upsert", func() {
		It("adds a resourceInfo to the index", func() {
			err := eng.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())

			count, err := idx.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(uint64(1)))

			query := bleveSearch.NewMatchQuery("child.pdf")
			res, err := idx.Search(bleveSearch.NewSearchRequest(query))
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Hits.Len()).To(Equal(1))
		})

		It("updates an existing resource in the index", func() {

			err := eng.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())

			countA, err := idx.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(countA).To(Equal(uint64(1)))

			err = eng.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())

			countB, err := idx.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(countB).To(Equal(uint64(1)))
		})
	})

	Describe("Delete", func() {
		It("marks a resource as deleted", func() {
			err := eng.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())

			assertDocCount(rootResource.ID, "Name:*child*", 1)

			err = eng.Delete(childResource.ID)
			Expect(err).ToNot(HaveOccurred())

			assertDocCount(rootResource.ID, "Name:*child*", 0)
		})

		It("marks a child resources as deleted", func() {
			err := eng.Upsert(parentResource.ID, parentResource)
			Expect(err).ToNot(HaveOccurred())

			err = eng.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())

			assertDocCount(rootResource.ID, `"`+parentResource.Document.Name+`"`, 1)
			assertDocCount(rootResource.ID, `"`+childResource.Document.Name+`"`, 1)

			err = eng.Delete(parentResource.ID)
			Expect(err).ToNot(HaveOccurred())

			assertDocCount(rootResource.ID, `"`+parentResource.Document.Name+`"`, 0)
			assertDocCount(rootResource.ID, `"`+childResource.Document.Name+`"`, 0)
		})
	})

	Describe("Restore", func() {
		It("also marks child resources as restored", func() {
			err := eng.Upsert(parentResource.ID, parentResource)
			Expect(err).ToNot(HaveOccurred())

			err = eng.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())

			err = eng.Delete(parentResource.ID)
			Expect(err).ToNot(HaveOccurred())

			assertDocCount(rootResource.ID, `"`+parentResource.Name+`"`, 0)
			assertDocCount(rootResource.ID, `"`+childResource.Name+`"`, 0)

			err = eng.Restore(parentResource.ID)
			Expect(err).ToNot(HaveOccurred())

			assertDocCount(rootResource.ID, `"`+parentResource.Name+`"`, 1)
			assertDocCount(rootResource.ID, `"`+childResource.Name+`"`, 1)
		})
	})

	Describe("Purge", func() {
		It("removes a resource from the index", func() {
			err := eng.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())
			assertDocCount(rootResource.ID, "Name:child.pdf", 1)

			err = eng.Purge(childResource.ID, false)

			Expect(err).ToNot(HaveOccurred())
			assertDocCount(rootResource.ID, "Name:child.pdf", 0)
		})
		It("removes a resource and its children from the index", func() {
			err := eng.Upsert(parentResource.ID, parentResource)
			Expect(err).ToNot(HaveOccurred())
			err = eng.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())

			assertDocCount(rootResource.ID, `"`+parentResource.Document.Name+`"`, 1)
			assertDocCount(rootResource.ID, `"`+childResource.Document.Name+`"`, 1)

			err = eng.Purge(parentResource.ID, false)
			Expect(err).ToNot(HaveOccurred())

			assertDocCount(rootResource.ID, `"`+parentResource.Document.Name+`"`, 0)
			assertDocCount(rootResource.ID, `"`+childResource.Document.Name+`"`, 0)
		})
		It("removes a resource and ignores its children from the index", func() {
			err := eng.Upsert(parentResource.ID, parentResource)
			Expect(err).ToNot(HaveOccurred())

			assertDocCount(rootResource.ID, `"`+parentResource.Document.Name+`"`, 1)

			err = eng.Delete(parentResource.ID)
			Expect(err).ToNot(HaveOccurred())

			err = eng.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())

			assertDocCount(rootResource.ID, `"`+childResource.Document.Name+`"`, 1)

			err = eng.Purge(parentResource.ID, true)
			Expect(err).ToNot(HaveOccurred())

			assertDocCount(rootResource.ID, `"`+parentResource.Document.Name+`"`, 0)
			assertDocCount(rootResource.ID, `"`+childResource.Document.Name+`"`, 1)
		})
	})

	Describe("Move", func() {
		It("renames the parent and its child resources", func() {
			err := eng.Upsert(parentResource.ID, parentResource)
			Expect(err).ToNot(HaveOccurred())

			err = eng.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())

			parentResource.Path = "newname"
			err = eng.Move(parentResource.ID, parentResource.ParentID, "./my/newname")
			Expect(err).ToNot(HaveOccurred())

			assertDocCount(rootResource.ID, parentResource.Name, 0)

			matches := assertDocCount(rootResource.ID, "Name:child.pdf", 1)
			Expect(matches[0].Entity.ParentId.OpaqueId).To(Equal("3"))
			Expect(matches[0].Entity.Ref.Path).To(Equal("./my/newname/child.pdf"))
		})

		It("moves the parent and its child resources", func() {
			err := eng.Upsert(parentResource.ID, parentResource)
			Expect(err).ToNot(HaveOccurred())

			err = eng.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())

			parentResource.Path = " "
			parentResource.ParentID = "1$2!somewhereopaqueid"

			err = eng.Move(parentResource.ID, parentResource.ParentID, "./somewhere/else/newname")
			Expect(err).ToNot(HaveOccurred())
			assertDocCount(rootResource.ID, `parent d!r`, 0)

			matches := assertDocCount(rootResource.ID, "Name:child.pdf", 1)
			Expect(matches[0].Entity.ParentId.OpaqueId).To(Equal("3"))
			Expect(matches[0].Entity.Ref.Path).To(Equal("./somewhere/else/newname/child.pdf"))

			matches = assertDocCount(rootResource.ID, `newname`, 1)
			Expect(matches[0].Entity.ParentId.OpaqueId).To(Equal("somewhereopaqueid"))
			Expect(matches[0].Entity.Ref.Path).To(Equal("./somewhere/else/newname"))

		})

		It("keeps case-insensitive search working after a move", func() {
			Expect(eng.Upsert(parentResource.ID, parentResource)).To(Succeed())
			Expect(eng.Upsert(childResource.ID, childResource)).To(Succeed())

			Expect(eng.Move(parentResource.ID, parentResource.ParentID, "./my/NewName")).To(Succeed())

			// the lowercased name sibling is rebuilt, so a case-insensitive
			// name query still works; the path is case-sensitive by design, so
			// only the exact new path matches (and the old one no longer does).
			assertDocCount(rootResource.ID, "name:NEWNAME", 1)
			assertDocCount(rootResource.ID, `path:"./my/NewName"`, 2)
			assertDocCount(rootResource.ID, `path:"./MY/NEWNAME"`, 0)
			assertDocCount(rootResource.ID, `path:"./parent d!r"`, 0)
		})
	})

	Describe("StartBatch", func() {
		It("starts a new batch", func() {
			b, err := eng.NewBatch(100)
			Expect(err).ToNot(HaveOccurred())

			err = b.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())

			count, err := idx.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(uint64(0)))

			err = b.Push()
			Expect(err).ToNot(HaveOccurred())

			count, err = idx.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(uint64(1)))

			query := bleveSearch.NewMatchQuery("child.pdf")
			res, err := idx.Search(bleveSearch.NewSearchRequest(query))
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Hits.Len()).To(Equal(1))
		})

		It("doesn't intertwine different batches", func() {
			b, err := eng.NewBatch(100)
			Expect(err).ToNot(HaveOccurred())

			err = b.Upsert(childResource.ID, childResource)
			Expect(err).ToNot(HaveOccurred())

			count, err := idx.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(uint64(0)))

			b2, err := eng.NewBatch(100)
			Expect(err).ToNot(HaveOccurred())

			err = b2.Upsert(childResource2.ID, childResource2)
			Expect(err).ToNot(HaveOccurred())

			Expect(b.Push()).To(Succeed())
			count, err = idx.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(uint64(1)))

			Expect(b2.Push()).To(Succeed())
			count, err = idx.DocCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(uint64(2)))
		})
	})

	Describe("File type specific metadata", func() {

		Context("with audio metadata", func() {
			BeforeEach(func() {
				resource := search.Resource{
					ID:       "1$2!7",
					ParentID: rootResource.ID,
					RootID:   rootResource.ID,
					Path:     "./some_song.mp3",
					Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
					Document: content.Document{
						Name:     "some_song.mp3",
						MimeType: "audio/mpeg",
						Audio: &libregraph.Audio{
							Album:             libregraph.PtrString("Some Album"),
							AlbumArtist:       libregraph.PtrString("Some AlbumArtist"),
							Artist:            libregraph.PtrString("Some Artist"),
							Bitrate:           libregraph.PtrInt64(192),
							Composers:         libregraph.PtrString("Some Composers"),
							Copyright:         libregraph.PtrString(""),
							Disc:              libregraph.PtrInt32(2),
							DiscCount:         libregraph.PtrInt32(5),
							Duration:          libregraph.PtrInt64(225000),
							Genre:             libregraph.PtrString("Some Genre"),
							HasDrm:            libregraph.PtrBool(false),
							IsVariableBitrate: libregraph.PtrBool(true),
							Title:             libregraph.PtrString("Some Title"),
							Track:             libregraph.PtrInt32(34),
							TrackCount:        libregraph.PtrInt32(99),
							Year:              libregraph.PtrInt32(2004),
						},
					},
				}
				err := eng.Upsert(resource.ID, resource)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns audio metadata for search", func() {
				matches := assertDocCount(rootResource.ID, `*song*`, 1)
				audio := matches[0].Entity.Audio

				Expect(audio).ToNot(BeNil())

				Expect(audio.Album).To(Equal(libregraph.PtrString("Some Album")))
				Expect(audio.AlbumArtist).To(Equal(libregraph.PtrString("Some AlbumArtist")))
				Expect(audio.Artist).To(Equal(libregraph.PtrString("Some Artist")))
				Expect(audio.Bitrate).To(Equal(libregraph.PtrInt64(192)))
				Expect(audio.Composers).To(Equal(libregraph.PtrString("Some Composers")))
				Expect(audio.Copyright).To(Equal(libregraph.PtrString("")))
				Expect(audio.Disc).To(Equal(libregraph.PtrInt32(2)))
				Expect(audio.DiscCount).To(Equal(libregraph.PtrInt32(5)))
				Expect(audio.Duration).To(Equal(libregraph.PtrInt64(225000)))
				Expect(audio.Genre).To(Equal(libregraph.PtrString("Some Genre")))
				Expect(audio.HasDrm).To(Equal(libregraph.PtrBool(false)))
				Expect(audio.IsVariableBitrate).To(Equal(libregraph.PtrBool(true)))
				Expect(audio.Title).To(Equal(libregraph.PtrString("Some Title")))
				Expect(audio.Track).To(Equal(libregraph.PtrInt32(34)))
				Expect(audio.TrackCount).To(Equal(libregraph.PtrInt32(99)))
				Expect(audio.Year).To(Equal(libregraph.PtrInt32(2004)))
			})
		})

		Context("with location metadata", func() {
			BeforeEach(func() {
				resource := search.Resource{
					ID:       "1$2!7",
					ParentID: rootResource.ID,
					RootID:   rootResource.ID,
					Path:     "./team.jpg",
					Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
					Document: content.Document{
						Name:     "team.jpg",
						MimeType: "image/jpeg",
						Location: &libregraph.GeoCoordinates{
							Altitude:  libregraph.PtrFloat64(1047.7),
							Latitude:  libregraph.PtrFloat64(49.48675890884328),
							Longitude: libregraph.PtrFloat64(11.103870357204285),
						},
					},
				}
				err := eng.Upsert(resource.ID, resource)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns audio metadata for search", func() {
				matches := assertDocCount(rootResource.ID, `*team*`, 1)
				location := matches[0].Entity.Location

				Expect(location).ToNot(BeNil())

				Expect(location.Altitude).To(Equal(libregraph.PtrFloat64(1047.7)))
				Expect(location.Latitude).To(Equal(libregraph.PtrFloat64(49.48675890884328)))
				Expect(location.Longitude).To(Equal(libregraph.PtrFloat64(11.103870357204285)))
			})
		})
	})

	Describe("Aggregations", func() {
		upsertAudio := func(id, name, artist, album, title string) {
			r := search.Resource{
				ID:       id,
				ParentID: rootResource.ID,
				RootID:   rootResource.ID,
				Path:     "./" + name,
				Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
				Document: content.Document{
					Name:     name,
					MimeType: "audio/mpeg",
					Audio: &libregraph.Audio{
						Artist: libregraph.PtrString(artist),
						Album:  libregraph.PtrString(album),
						Title:  libregraph.PtrString(title),
					},
				},
			}
			Expect(eng.Upsert(r.ID, r)).To(Succeed())
		}

		searchWithAggs := func(query string, aggs ...*searchsvc.AggregationOption) *searchsvc.SearchIndexResponse {
			rID, err := storagespace.ParseID(rootResource.ID)
			Expect(err).ToNot(HaveOccurred())
			res, err := eng.Search(context.Background(), &searchsvc.SearchIndexRequest{
				Query: query,
				Ref: &searchmsg.Reference{
					ResourceId: &searchmsg.ResourceID{
						StorageId: rID.StorageId,
						SpaceId:   rID.SpaceId,
						OpaqueId:  rID.OpaqueId,
					},
				},
				Aggregations: aggs,
			})
			Expect(err).ToNot(HaveOccurred())
			return res
		}

		BeforeEach(func() {
			Expect(eng.Upsert(rootResource.ID, rootResource)).To(Succeed())
			upsertAudio("1$2!1001", "a.mp3", "Pink Floyd", "The Wall", "Brick")
			upsertAudio("1$2!1002", "b.mp3", "Pink Floyd", "The Wall", "Comfortably Numb")
			upsertAudio("1$2!1003", "c.mp3", "Motörhead", "Bomber", "Bomber")
			upsertAudio("1$2!1004", "d.mp3", "Motörhead", "Bomber", "Stone Dead Forever")
			upsertAudio("1$2!1005", "e.mp3", "Motörhead", "Ace of Spades", "Ace of Spades")
		})

		It("returns term buckets on audio.artist", func() {
			res := searchWithAggs("mediatype:audio", &searchsvc.AggregationOption{
				Field: "audio.artist",
				Size:  10,
			})
			Expect(res.Aggregations).To(HaveLen(1))
			agg := res.Aggregations[0]
			Expect(agg.Field).To(Equal("audio.artist"))

			counts := map[string]int64{}
			for _, b := range agg.Buckets {
				counts[b.Key] = b.Count
			}
			Expect(counts).To(HaveKeyWithValue("Pink Floyd", int64(2)))
			Expect(counts).To(HaveKeyWithValue("Motörhead", int64(3)))
		})

		It("returns nil aggregations when none are requested", func() {
			res := searchWithAggs("mediatype:audio")
			Expect(res.Aggregations).To(BeNil())
		})

		It("handles multiple aggregations in one request", func() {
			res := searchWithAggs("mediatype:audio",
				&searchsvc.AggregationOption{Field: "audio.artist"},
				&searchsvc.AggregationOption{Field: "audio.album"},
			)
			fields := []string{}
			for _, a := range res.Aggregations {
				fields = append(fields, a.Field)
			}
			Expect(fields).To(ConsistOf("audio.artist", "audio.album"))
		})

		Describe("numeric range aggregations", func() {
			upsertWithYear := func(id, name string, year int32) {
				r := search.Resource{
					ID:       id,
					ParentID: rootResource.ID,
					RootID:   rootResource.ID,
					Path:     "./" + name,
					Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
					Document: content.Document{
						Name:     name,
						MimeType: "audio/mpeg",
						Audio:    &libregraph.Audio{Year: libregraph.PtrInt32(year)},
					},
				}
				Expect(eng.Upsert(r.ID, r)).To(Succeed())
			}

			BeforeEach(func() {
				upsertWithYear("1$2!2001", "70a.mp3", 1971)
				upsertWithYear("1$2!2002", "70b.mp3", 1975)
				upsertWithYear("1$2!2003", "80a.mp3", 1982)
				upsertWithYear("1$2!2004", "90a.mp3", 1999)
				upsertWithYear("1$2!2005", "00a.mp3", 2001)
				upsertWithYear("1$2!2006", "00b.mp3", 2005)
				upsertWithYear("1$2!2007", "00c.mp3", 2009)
			})

			It("returns buckets per decade range", func() {
				res := searchWithAggs("mediatype:audio",
					&searchsvc.AggregationOption{
						Field: "audio.year",
						BucketDefinition: &searchsvc.BucketDefinition{
							Ranges: []*searchsvc.BucketRange{
								{From: "1970", To: "1980"},
								{From: "1980", To: "1990"},
								{From: "1990", To: "2000"},
								{From: "2000", To: "2010"},
							},
						},
					},
				)
				Expect(res.Aggregations).To(HaveLen(1))
				counts := map[string]int64{}
				for _, b := range res.Aggregations[0].Buckets {
					counts[b.Key] = b.Count
				}
				Expect(counts).To(HaveKeyWithValue("1970-1980", int64(2)))
				Expect(counts).To(HaveKeyWithValue("1980-1990", int64(1)))
				Expect(counts).To(HaveKeyWithValue("1990-2000", int64(1)))
				Expect(counts).To(HaveKeyWithValue("2000-2010", int64(3)))
			})

			It("supports open-ended ranges", func() {
				res := searchWithAggs("mediatype:audio",
					&searchsvc.AggregationOption{
						Field: "audio.year",
						BucketDefinition: &searchsvc.BucketDefinition{
							Ranges: []*searchsvc.BucketRange{
								{To: "1990"},
								{From: "2000"},
							},
						},
					},
				)
				Expect(res.Aggregations).To(HaveLen(1))
				counts := map[string]int64{}
				for _, b := range res.Aggregations[0].Buckets {
					counts[b.Key] = b.Count
				}
				Expect(counts).To(HaveKeyWithValue("-1990", int64(3)))
				Expect(counts).To(HaveKeyWithValue("2000-", int64(3)))
			})

			It("computes top-level metric aggregations by scanning hits", func() {
				res := searchWithAggs("mediatype:audio",
					&searchsvc.AggregationOption{Field: "audio.year", MetricKind: searchsvc.MetricKind_METRIC_KIND_SUM},
					&searchsvc.AggregationOption{Field: "audio.year", MetricKind: searchsvc.MetricKind_METRIC_KIND_MIN},
					&searchsvc.AggregationOption{Field: "audio.year", MetricKind: searchsvc.MetricKind_METRIC_KIND_MAX},
					&searchsvc.AggregationOption{Field: "audio.year", MetricKind: searchsvc.MetricKind_METRIC_KIND_AVG},
				)
				Expect(res.Aggregations).To(HaveLen(4))
				// upserted years: 1971, 1975, 1982, 1999, 2001, 2005, 2009
				Expect(res.Aggregations[0].MetricKind).To(Equal(searchsvc.MetricKind_METRIC_KIND_SUM))
				Expect(res.Aggregations[0].Value).To(Equal(float64(13942)))
				Expect(res.Aggregations[1].Value).To(Equal(float64(1971)))
				Expect(res.Aggregations[2].Value).To(Equal(float64(2009)))
				Expect(res.Aggregations[3].MetricKind).To(Equal(searchsvc.MetricKind_METRIC_KIND_AVG))
				Expect(res.Aggregations[3].Sum).To(Equal(float64(13942)))
				Expect(res.Aggregations[3].Count).To(Equal(int64(7)))
			})
		})

		Describe("date range aggregations", func() {
			upsertWithTakenDateTime := func(id, name, taken string) {
				t, err := time.Parse(time.RFC3339, taken)
				Expect(err).ToNot(HaveOccurred())
				r := search.Resource{
					ID:       id,
					ParentID: rootResource.ID,
					RootID:   rootResource.ID,
					Path:     "./" + name,
					Type:     uint64(sprovider.ResourceType_RESOURCE_TYPE_FILE),
					Document: content.Document{
						Name:     name,
						MimeType: "image/jpeg",
						Photo:    &libregraph.Photo{TakenDateTime: &t},
					},
				}
				Expect(eng.Upsert(r.ID, r)).To(Succeed())
			}

			BeforeEach(func() {
				upsertWithTakenDateTime("1$2!3001", "a.jpg", "2018-08-11T09:15:00Z")
				upsertWithTakenDateTime("1$2!3002", "b.jpg", "2018-08-11T19:42:00Z")
				upsertWithTakenDateTime("1$2!3003", "c.jpg", "2018-09-01T12:00:00Z")
				upsertWithTakenDateTime("1$2!3004", "d.jpg", "2021-08-11T08:00:00Z")
			})

			It("returns buckets per date range", func() {
				res := searchWithAggs("mediatype:image",
					&searchsvc.AggregationOption{
						Field: "photo.takenDateTime",
						BucketDefinition: &searchsvc.BucketDefinition{
							Ranges: []*searchsvc.BucketRange{
								{From: "2018-08-11T00:00:00Z", To: "2018-08-12T00:00:00Z"},
								{From: "2018-08-01", To: "2018-09-01"},
								{From: "2021-01-01", To: "2022-01-01"},
								{From: "2023-01-01", To: "2024-01-01"},
							},
						},
					},
				)
				Expect(res.Aggregations).To(HaveLen(1))
				counts := map[string]int64{}
				for _, b := range res.Aggregations[0].Buckets {
					counts[b.Key] = b.Count
				}
				Expect(counts).To(HaveKeyWithValue("2018-08-11T00:00:00Z-2018-08-12T00:00:00Z", int64(2)))
				Expect(counts).To(HaveKeyWithValue("2018-08-01-2018-09-01", int64(2)))
				Expect(counts).To(HaveKeyWithValue("2021-01-01-2022-01-01", int64(1)))
			})

			It("supports open-ended date ranges", func() {
				res := searchWithAggs("mediatype:image",
					&searchsvc.AggregationOption{
						Field: "photo.takenDateTime",
						BucketDefinition: &searchsvc.BucketDefinition{
							Ranges: []*searchsvc.BucketRange{
								{To: "2019-01-01"},
								{From: "2019-01-01"},
							},
						},
					},
				)
				Expect(res.Aggregations).To(HaveLen(1))
				counts := map[string]int64{}
				for _, b := range res.Aggregations[0].Buckets {
					counts[b.Key] = b.Count
				}
				Expect(counts).To(HaveKeyWithValue("-2019-01-01", int64(3)))
				Expect(counts).To(HaveKeyWithValue("2019-01-01-", int64(1)))
			})

			It("rejects malformed date range bounds", func() {
				rID, err := storagespace.ParseID(rootResource.ID)
				Expect(err).ToNot(HaveOccurred())
				_, err = eng.Search(context.Background(), &searchsvc.SearchIndexRequest{
					Query: "mediatype:image",
					Ref: &searchmsg.Reference{ResourceId: &searchmsg.ResourceID{
						StorageId: rID.StorageId, SpaceId: rID.SpaceId, OpaqueId: rID.OpaqueId,
					}},
					Aggregations: []*searchsvc.AggregationOption{
						{
							Field: "photo.takenDateTime",
							BucketDefinition: &searchsvc.BucketDefinition{
								Ranges: []*searchsvc.BucketRange{
									{From: "2018-08-11T00:00:00Z", To: "not-a-date"},
								},
							},
						},
					},
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
