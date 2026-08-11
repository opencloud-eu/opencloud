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

		rootResource   search.Resource
		parentResource search.Resource
		childResource  search.Resource
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

	Describe("New", func() {
		It("returns a new index instance", func() {
			b := bleve.NewBackend(idx, bleveQuery.DefaultCreator, log.Logger{})
			Expect(b).ToNot(BeNil())
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
