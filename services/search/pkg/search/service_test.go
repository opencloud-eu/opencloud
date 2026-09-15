package search_test

import (
	"context"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaborationv1beta1 "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	contentMocks "github.com/opencloud-eu/opencloud/services/search/pkg/content/mocks"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
	engineMocks "github.com/opencloud-eu/opencloud/services/search/pkg/search/mocks"
)

var _ = Describe("Searchprovider", func() {
	var (
		s               search.Searcher
		extractor       *contentMocks.Extractor
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		indexClient     *engineMocks.Engine
		ctx             context.Context
		logger          = log.NewLogger()
		user            = &userv1beta1.User{
			Id: &userv1beta1.UserId{
				OpaqueId: "user",
			},
		}
		otherUser = &userv1beta1.User{
			Id: &userv1beta1.UserId{
				OpaqueId: "otheruser",
			},
		}
		personalSpace = &sprovider.StorageSpace{
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte("/foo"),
					},
				},
			},
			Id:        &sprovider.StorageSpaceId{OpaqueId: "storageid$personalspace!personalspace"},
			Root:      &sprovider.ResourceId{StorageId: "storageid", SpaceId: "personalspace", OpaqueId: "personalspace"},
			Name:      "personalspace",
			SpaceType: "personal",
		}

		ri = &sprovider.ResourceInfo{
			Id: &sprovider.ResourceId{
				StorageId: "storageid",
				OpaqueId:  "opaqueid",
			},
			ParentId: &sprovider.ResourceId{
				StorageId: "storageid",
				OpaqueId:  "parentopaqueid",
			},
			Path:  "foo.pdf",
			Size:  12345,
			Mtime: &typesv1beta1.Timestamp{Seconds: 4000},
		}
	)

	BeforeEach(func() {
		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector = pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		ctx = revactx.ContextSetUser(context.Background(), user)
		indexClient = &engineMocks.Engine{}
		extractor = &contentMocks.Extractor{}

		s = search.NewService(gatewaySelector, indexClient, extractor, nil, logger, &config.Config{})

		gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(&gateway.AuthenticateResponse{
			Status: status.NewOK(ctx),
			Token:  "authtoken",
		}, nil)
		gatewayClient.On("GetPath", mock.Anything, mock.MatchedBy(func(req *sprovider.GetPathRequest) bool {
			return req.ResourceId.OpaqueId == ri.Id.OpaqueId
		})).Return(&sprovider.GetPathResponse{
			Status: status.NewOK(context.Background()),
			Path:   ri.Path,
		}, nil)
		gatewayClient.On("GetReceivedShare", mock.Anything, mock.Anything).Return(&collaborationv1beta1.GetReceivedShareResponse{
			Status: status.NewOK(ctx),
		}, nil)
		indexClient.On("DocCount").Return(uint64(1), nil)
	})

	Describe("New", func() {
		It("returns a new instance", func() {
			s := search.NewService(gatewaySelector, indexClient, extractor, nil, logger, &config.Config{})
			Expect(s).ToNot(BeNil())
		})
	})

	Describe("IndexSpace", func() {
		It("walks the space and indexes all files", func() {
			batch := &engineMocks.BatchOperator{}
			batch.EXPECT().Push().Return(nil)
			gatewayClient.On("GetUserByClaim", mock.Anything, mock.Anything).Return(&userv1beta1.GetUserByClaimResponse{
				Status: status.NewOK(context.Background()),
				User:   user,
			}, nil)
			extractor.On("Extract", mock.Anything, mock.Anything, mock.Anything).Return(content.Document{}, nil)
			indexClient.On("NewBatch", mock.Anything).Return(batch, nil)
			batch.On("Upsert", mock.Anything, mock.Anything).Return(nil)
			indexClient.On("Search", mock.Anything, mock.Anything).Return(&searchsvc.SearchIndexResponse{}, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
				Status: status.NewOK(context.Background()),
				Info:   ri,
			}, nil)

			err := s.IndexSpace(&sprovider.StorageSpaceId{OpaqueId: "storageid$spaceid!spaceid"}, false)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("UpsertItem", func() {
		DescribeTable("marks a resource hidden when a dot starts one of its names",
			func(path string, hidden bool) {
				info := &sprovider.ResourceInfo{
					Id:       &sprovider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "hidden-opaqueid"},
					ParentId: &sprovider.ResourceId{StorageId: "storageid", OpaqueId: "parentopaqueid"},
					Path:     path,
				}
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
					Status: status.NewOK(context.Background()),
					Info:   info,
				}, nil)
				gatewayClient.On("GetPath", mock.Anything, mock.MatchedBy(func(req *sprovider.GetPathRequest) bool {
					return req.GetResourceId().GetOpaqueId() == info.GetId().GetOpaqueId()
				})).Return(&sprovider.GetPathResponse{
					Status: status.NewOK(context.Background()),
					Path:   path,
				}, nil)
				extractor.On("Extract", mock.Anything, mock.Anything, mock.Anything).Return(content.Document{}, nil)

				var indexed search.Resource
				indexClient.On("Upsert", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					indexed = args.Get(1).(search.Resource)
				}).Return(nil)

				s.UpsertItem(&sprovider.Reference{ResourceId: info.GetId()})
				Expect(indexed.Hidden).To(Equal(hidden))
			},
			Entry("./documents/notes.txt", "./documents/notes.txt", false),
			Entry("./documents/.notes.txt", "./documents/.notes.txt", true),
			Entry("documents/.notes.txt", "documents/.notes.txt", true),
			Entry("./.git/config", "./.git/config", true),
			Entry(".git/config", ".git/config", true),
			Entry("/.git/config", "/.git/config", true),
			Entry("./", "./", false),
			Entry("/", "/", false),
		)
	})

	DescribeTable("IsHidden",
		func(path string, hidden bool) {
			Expect(search.IsHidden(path)).To(Equal(hidden))
		},
		Entry("a plain path", "./documents/notes.txt", false),
		Entry("a dot file", "./documents/.notes.txt", true),
		Entry("a dot folder above it", "./.git/config", true),
		Entry("the current folder", ".", false),
		Entry("a leading traversal", "../documents/notes.txt", false),
		Entry("a traversal in the middle", "./documents/../notes.txt", false),
		Entry("a traversal out of a dot folder", "./.git/../notes.txt", false),
	)

	Describe("Search", func() {
		It("fails when an empty query is given", func() {
			res, err := s.Search(ctx, &searchsvc.SearchRequest{
				Query: "",
			})
			Expect(err).To(HaveOccurred())
			Expect(res).To(BeNil())
		})

		Context("with a personal space", func() {
			BeforeEach(func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&sprovider.ListStorageSpacesResponse{
					Status:        status.NewOK(ctx),
					StorageSpaces: []*sprovider.StorageSpace{personalSpace},
				}, nil)
				indexClient.On("Search", mock.Anything, mock.Anything).Return(&searchsvc.SearchIndexResponse{
					TotalMatches: 1,
					Matches: []*searchmsg.Match{
						{
							Score: 1,
							Entity: &searchmsg.Entity{
								Ref: &searchmsg.Reference{
									ResourceId: &searchmsg.ResourceID{
										StorageId: personalSpace.Root.StorageId,
										SpaceId:   personalSpace.Root.SpaceId,
										OpaqueId:  personalSpace.Root.OpaqueId,
									},
									Path: "./path/to/Foo.pdf",
								},
								Id: &searchmsg.ResourceID{
									StorageId: personalSpace.Root.StorageId,
									OpaqueId:  "foo-id",
								},
								Name: "Foo.pdf",
							},
						},
					},
				}, nil)
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
					Status: status.NewOK(context.Background()),
					Info:   ri,
				}, nil)
			})

			It("does not mess with field-based searches", func() {
				_, err := s.Search(ctx, &searchsvc.SearchRequest{
					Query: "Size:<10",
				})
				Expect(err).ToNot(HaveOccurred())
				indexClient.AssertCalled(GinkgoT(), "Search", mock.Anything, mock.MatchedBy(func(req *searchsvc.SearchIndexRequest) bool {
					return req.Query == "Size:<10"
				}))
			})

			It("searches the personal user space", func() {
				res, err := s.Search(ctx, &searchsvc.SearchRequest{
					Query: "foo",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(res).ToNot(BeNil())
				Expect(res.TotalMatches).To(Equal(int32(1)))
				Expect(len(res.Matches)).To(Equal(1))
				match := res.Matches[0]
				Expect(match.Entity.Id.OpaqueId).To(Equal("foo-id"))
				Expect(match.Entity.Name).To(Equal("Foo.pdf"))
				Expect(match.Entity.Ref.ResourceId.OpaqueId).To(Equal(personalSpace.Root.OpaqueId))
				Expect(match.Entity.Ref.Path).To(Equal("./path/to/Foo.pdf"))
			})

			It("forwards aggregations to the engine", func() {
				_, err := s.Search(ctx, &searchsvc.SearchRequest{
					Query: "foo",
					Aggregations: []*searchsvc.AggregationOption{
						{Field: "audio.artist", Size: 10},
					},
				})
				Expect(err).ToNot(HaveOccurred())
				indexClient.AssertCalled(GinkgoT(), "Search", mock.Anything, mock.MatchedBy(func(req *searchsvc.SearchIndexRequest) bool {
					return len(req.Aggregations) == 1 &&
						req.Aggregations[0].Field == "audio.artist" &&
						req.Aggregations[0].Size == 10
				}))
			})
		})

		Context("with two personal spaces returning aggregations", func() {
			var (
				spaceA = &sprovider.StorageSpace{
					Id:        &sprovider.StorageSpaceId{OpaqueId: "storageid$a!a"},
					Root:      &sprovider.ResourceId{StorageId: "storageid", SpaceId: "a", OpaqueId: "a"},
					Name:      "space-a",
					SpaceType: "personal",
				}
				spaceB = &sprovider.StorageSpace{
					Id:        &sprovider.StorageSpaceId{OpaqueId: "storageid$b!b"},
					Root:      &sprovider.ResourceId{StorageId: "storageid", SpaceId: "b", OpaqueId: "b"},
					Name:      "space-b",
					SpaceType: "personal",
				}
			)

			BeforeEach(func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&sprovider.ListStorageSpacesResponse{
					Status:        status.NewOK(ctx),
					StorageSpaces: []*sprovider.StorageSpace{spaceA, spaceB},
				}, nil)
				indexClient.On("Search", mock.Anything, mock.MatchedBy(func(req *searchsvc.SearchIndexRequest) bool {
					return req.Ref != nil && req.Ref.ResourceId.SpaceId == "a"
				})).Return(&searchsvc.SearchIndexResponse{
					TotalMatches: 2,
					Aggregations: []*searchsvc.AggregationResult{{
						Field: "audio.artist",
						Buckets: []*searchsvc.Bucket{
							{Key: "Pink Floyd", Count: 2},
							{Key: "Motörhead", Count: 1},
						},
					}},
				}, nil)
				indexClient.On("Search", mock.Anything, mock.MatchedBy(func(req *searchsvc.SearchIndexRequest) bool {
					return req.Ref != nil && req.Ref.ResourceId.SpaceId == "b"
				})).Return(&searchsvc.SearchIndexResponse{
					TotalMatches: 3,
					Aggregations: []*searchsvc.AggregationResult{{
						Field: "audio.artist",
						Buckets: []*searchsvc.Bucket{
							{Key: "Pink Floyd", Count: 3},
							{Key: "Led Zeppelin", Count: 1},
						},
					}},
				}, nil)
			})

			It("merges bucket counts across spaces", func() {
				res, err := s.Search(ctx, &searchsvc.SearchRequest{
					Query: "mediatype:audio",
					Aggregations: []*searchsvc.AggregationOption{
						{Field: "audio.artist", Size: 10},
					},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Aggregations).To(HaveLen(1))
				agg := res.Aggregations[0]
				Expect(agg.Field).To(Equal("audio.artist"))

				counts := map[string]int64{}
				for _, b := range agg.Buckets {
					counts[b.Key] = b.Count
				}
				Expect(counts).To(HaveKeyWithValue("Pink Floyd", int64(5)))
				Expect(counts).To(HaveKeyWithValue("Motörhead", int64(1)))
				Expect(counts).To(HaveKeyWithValue("Led Zeppelin", int64(1)))
			})

			It("sorts buckets by count descending by default", func() {
				res, err := s.Search(ctx, &searchsvc.SearchRequest{
					Query: "mediatype:audio",
					Aggregations: []*searchsvc.AggregationOption{
						{Field: "audio.artist"},
					},
				})
				Expect(err).ToNot(HaveOccurred())
				keys := []string{}
				for _, b := range res.Aggregations[0].Buckets {
					keys = append(keys, b.Key)
				}
				// Pink Floyd:5, Motörhead:1, Led Zeppelin:1 (count desc)
				Expect(keys[0]).To(Equal("Pink Floyd"))
			})

			It("sorts buckets alphabetically ascending with sortBy keyAsString", func() {
				res, err := s.Search(ctx, &searchsvc.SearchRequest{
					Query: "mediatype:audio",
					Aggregations: []*searchsvc.AggregationOption{
						{
							Field: "audio.artist",
							BucketDefinition: &searchsvc.BucketDefinition{
								SortBy: "keyAsString",
							},
						},
					},
				})
				Expect(err).ToNot(HaveOccurred())
				keys := []string{}
				for _, b := range res.Aggregations[0].Buckets {
					keys = append(keys, b.Key)
				}
				Expect(keys).To(Equal([]string{"Led Zeppelin", "Motörhead", "Pink Floyd"}))
			})

			It("applies minimumCount filter and size cap", func() {
				res, err := s.Search(ctx, &searchsvc.SearchRequest{
					Query: "mediatype:audio",
					Aggregations: []*searchsvc.AggregationOption{
						{
							Field: "audio.artist",
							Size:  5,
							BucketDefinition: &searchsvc.BucketDefinition{
								SortBy:       "count",
								IsDescending: true,
								MinimumCount: 2,
							},
						},
					},
				})
				Expect(err).ToNot(HaveOccurred())
				// only Pink Floyd has count >= 2
				Expect(res.Aggregations[0].Buckets).To(HaveLen(1))
				Expect(res.Aggregations[0].Buckets[0].Key).To(Equal("Pink Floyd"))
			})

			It("trims the bucket list to Size", func() {
				res, err := s.Search(ctx, &searchsvc.SearchRequest{
					Query: "mediatype:audio",
					Aggregations: []*searchsvc.AggregationOption{
						{Field: "audio.artist", Size: 1},
					},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Aggregations[0].Buckets).To(HaveLen(1))
				Expect(res.Aggregations[0].Buckets[0].Key).To(Equal("Pink Floyd"))
			})
		})

		Context("with a personal space with a filter", func() {
			BeforeEach(func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&sprovider.ListStorageSpacesResponse{
					Status:        status.NewOK(ctx),
					StorageSpaces: []*sprovider.StorageSpace{personalSpace},
				}, nil)
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
					Status: status.NewOK(context.Background()),
					Info: &sprovider.ResourceInfo{
						Space: &sprovider.StorageSpace{Root: &sprovider.ResourceId{
							StorageId: "storageid",
							SpaceId:   "personalspace",
							OpaqueId:  "personalspace",
						}},
					},
				}, nil)
				gatewayClient.On("GetPath", mock.Anything, mock.Anything).Return(&sprovider.GetPathResponse{
					Status: status.NewOK(ctx),
					Path:   "/path",
				}, nil)
				indexClient.On("Search", mock.Anything, mock.Anything).Return(&searchsvc.SearchIndexResponse{
					TotalMatches: 1,
					Matches: []*searchmsg.Match{
						{
							Score: 1,
							Entity: &searchmsg.Entity{
								Ref: &searchmsg.Reference{
									ResourceId: &searchmsg.ResourceID{
										StorageId: personalSpace.Root.StorageId,
										SpaceId:   personalSpace.Root.SpaceId,
										OpaqueId:  personalSpace.Root.OpaqueId,
									},
									Path: "./path/to/Foo.pdf",
								},
								Id: &searchmsg.ResourceID{
									StorageId: personalSpace.Root.StorageId,
									OpaqueId:  "foo-id",
								},
								Name: "Foo.pdf",
							},
						},
					},
				}, nil)
			})

			It("searches the personal user space", func() {
				res, err := s.Search(ctx, &searchsvc.SearchRequest{
					Query: "foo scope:storageid$personalspace!personalspace/path",
					Ref: &searchmsg.Reference{
						ResourceId: &searchmsg.ResourceID{
							StorageId: "storageid",
							SpaceId:   "personalspace",
							OpaqueId:  "personalspace",
						},
					},
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(res).ToNot(BeNil())
				Expect(res.TotalMatches).To(Equal(int32(1)))
				Expect(len(res.Matches)).To(Equal(1))
				match := res.Matches[0]
				Expect(match.Entity.Id.OpaqueId).To(Equal("foo-id"))
				Expect(match.Entity.Name).To(Equal("Foo.pdf"))
				Expect(match.Entity.Ref.ResourceId.OpaqueId).To(Equal(personalSpace.Root.OpaqueId))
				Expect(match.Entity.Ref.Path).To(Equal("./path/to/Foo.pdf"))
			})
		})

		Context("with received shares", func() {
			var (
				grantSpace      *sprovider.StorageSpace
				mountpointSpace *sprovider.StorageSpace
			)

			BeforeEach(func() {
				grantSpace = &sprovider.StorageSpace{
					SpaceType: "grant",
					Owner:     otherUser,
					Id:        &sprovider.StorageSpaceId{OpaqueId: "storageproviderid$spaceid!otherspacegrant"},
					Root:      &sprovider.ResourceId{StorageId: "storageproviderid", SpaceId: "spaceid", OpaqueId: "otherspacegrant"},
					Name:      "grantspace",
					RootInfo: &sprovider.ResourceInfo{
						Id: &sprovider.ResourceId{
							StorageId: "storageid",
							SpaceId:   "spaceid",
							OpaqueId:  "opaqueid",
						},
					},
				}
				mountpointSpace = &sprovider.StorageSpace{
					SpaceType: "mountpoint",
					Owner:     otherUser,
					Id:        &sprovider.StorageSpaceId{OpaqueId: "storageproviderid$spaceid!otherspacemountpoint"},
					Root:      &sprovider.ResourceId{StorageId: "storageproviderid", SpaceId: "spaceid", OpaqueId: "otherspacemountpoint"},
					Name:      "mountpointspace",
					Opaque: &typesv1beta1.Opaque{
						Map: map[string]*typesv1beta1.OpaqueEntry{
							"grantStorageID": {Decoder: "plain", Value: []byte("storageproviderid")},
							"grantSpaceID":   {Decoder: "plain", Value: []byte("spaceid")},
							"grantOpaqueID":  {Decoder: "plain", Value: []byte("otherspacegrant")},
						},
					},
				}
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
					Status: status.NewOK(context.Background()),
					Info:   ri,
				}, nil)
				gatewayClient.On("GetPath", mock.Anything, mock.Anything).Return(&sprovider.GetPathResponse{
					Status: status.NewOK(ctx),
					Path:   "/grant/path",
				}, nil)
			})

			It("searches the received spaces", func() {
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&sprovider.ListStorageSpacesResponse{
					Status:        status.NewOK(ctx),
					StorageSpaces: []*sprovider.StorageSpace{grantSpace, mountpointSpace},
				}, nil)
				indexClient.On("Search", mock.Anything, mock.Anything).Return(&searchsvc.SearchIndexResponse{
					TotalMatches: 1,
					Matches: []*searchmsg.Match{
						{
							Entity: &searchmsg.Entity{
								Ref: &searchmsg.Reference{
									ResourceId: &searchmsg.ResourceID{
										StorageId: grantSpace.Root.StorageId,
										SpaceId:   grantSpace.Root.SpaceId,
										OpaqueId:  grantSpace.Root.OpaqueId,
									},
									Path: "./grant/path/to/Shared.pdf",
								},
								Id: &searchmsg.ResourceID{
									StorageId: grantSpace.Root.StorageId,
									OpaqueId:  "grant-shared-id",
								},
								Name: "Shared.pdf",
							},
						},
					},
				}, nil)

				res, err := s.Search(ctx, &searchsvc.SearchRequest{
					Query: "Foo",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(res).ToNot(BeNil())
				Expect(len(res.Matches)).To(Equal(1))
				match := res.Matches[0]
				Expect(match.Entity.Id.OpaqueId).To(Equal("grant-shared-id"))
				Expect(match.Entity.Name).To(Equal("Shared.pdf"))
				Expect(match.Entity.Ref.ResourceId.OpaqueId).To(Equal(mountpointSpace.Root.OpaqueId))
				Expect(match.Entity.Ref.Path).To(Equal("./to/Shared.pdf"))
				Expect(match.Entity.RemoteItemId).To(Equal(&searchmsg.ResourceID{
					StorageId: grantSpace.RootInfo.Id.StorageId,
					SpaceId:   grantSpace.RootInfo.Id.SpaceId,
					OpaqueId:  grantSpace.RootInfo.Id.OpaqueId,
				}))
			})

			Context("when searching both spaces", func() {
				BeforeEach(func() {
					gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&sprovider.ListStorageSpacesResponse{
						Status:        status.NewOK(ctx),
						StorageSpaces: []*sprovider.StorageSpace{personalSpace, grantSpace, mountpointSpace},
					}, nil)
					indexClient.On("Search", mock.Anything, mock.MatchedBy(func(req *searchsvc.SearchIndexRequest) bool {
						return req.Ref.ResourceId.OpaqueId == grantSpace.Root.SpaceId &&
							req.Ref.ResourceId.SpaceId == grantSpace.Root.SpaceId
					})).Return(&searchsvc.SearchIndexResponse{
						TotalMatches: 2,
						Matches: []*searchmsg.Match{
							{
								Score: 2,
								Entity: &searchmsg.Entity{
									Ref: &searchmsg.Reference{
										ResourceId: &searchmsg.ResourceID{
											StorageId: grantSpace.Root.StorageId,
											SpaceId:   grantSpace.Root.SpaceId,
											OpaqueId:  grantSpace.Root.OpaqueId,
										},
										Path: "./grant/path/to/Shared.pdf",
									},
									Id: &searchmsg.ResourceID{
										StorageId: grantSpace.Root.StorageId,
										OpaqueId:  "grant-shared-id",
									},
									Name: "Shared.pdf",
								},
							},
							{
								Score: 0.01,
								Entity: &searchmsg.Entity{
									Ref: &searchmsg.Reference{
										ResourceId: &searchmsg.ResourceID{
											StorageId: grantSpace.Root.StorageId,
											SpaceId:   grantSpace.Root.SpaceId,
											OpaqueId:  grantSpace.Root.OpaqueId,
										},
										Path: "./grant/path/to/Irrelevant.pdf",
									},
									Id: &searchmsg.ResourceID{
										StorageId: grantSpace.Root.StorageId,
										OpaqueId:  "grant-irrelevant-id",
									},
									Name: "Irrelevant.pdf",
								},
							},
						},
					}, nil)
					indexClient.On("Search", mock.Anything, mock.MatchedBy(func(req *searchsvc.SearchIndexRequest) bool {
						return req.Ref.ResourceId.OpaqueId == personalSpace.Root.OpaqueId &&
							req.Ref.ResourceId.SpaceId == personalSpace.Root.SpaceId
					})).Return(&searchsvc.SearchIndexResponse{
						TotalMatches: 1,
						Matches: []*searchmsg.Match{
							{
								Score: 1,
								Entity: &searchmsg.Entity{
									Ref: &searchmsg.Reference{
										ResourceId: &searchmsg.ResourceID{
											StorageId: personalSpace.Root.StorageId,
											SpaceId:   personalSpace.Root.SpaceId,
											OpaqueId:  personalSpace.Root.OpaqueId,
										},
										Path: "./path/to/Foo.pdf",
									},
									Id: &searchmsg.ResourceID{
										StorageId: personalSpace.Root.StorageId,
										OpaqueId:  "foo-id",
									},
									Name: "Foo.pdf",
								},
							},
						},
					}, nil)
				})

				It("prunes the fan-out when the query pins a drive", func() {
					res, err := s.Search(ctx, &searchsvc.SearchRequest{
						Query: `driveId:"storageid$personalspace" AND foo`,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(res).ToNot(BeNil())
					Expect(len(res.Matches)).To(Equal(1))
					Expect(res.Matches[0].Entity.Id.OpaqueId).To(Equal("foo-id"))
					indexClient.AssertNumberOfCalls(GinkgoT(), "Search", 1)
				})

				It("keeps the full fan-out for an OR query", func() {
					res, err := s.Search(ctx, &searchsvc.SearchRequest{
						Query: `driveId:"storageid$personalspace" OR foo`,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(res).ToNot(BeNil())
					Expect(len(res.Matches)).To(Equal(3))
					indexClient.AssertNumberOfCalls(GinkgoT(), "Search", 2)
				})

				It("considers the search Ref parameter", func() {
					res, err := s.Search(ctx, &searchsvc.SearchRequest{
						Query: "foo",
						Ref: &searchmsg.Reference{
							ResourceId: &searchmsg.ResourceID{
								StorageId: "storageid",
								SpaceId:   "personalspace",
								OpaqueId:  "personalspace",
							},
						},
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(res).ToNot(BeNil())
					Expect(len(res.Matches)).To(Equal(1))
					Expect(res.Matches[0].Entity.Id.OpaqueId).To(Equal("foo-id"))
				})

				It("finds matches in both the personal space AND the grant", func() {
					res, err := s.Search(ctx, &searchsvc.SearchRequest{
						Query: "foo",
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(res).ToNot(BeNil())
					Expect(len(res.Matches)).To(Equal(3))
					ids := []string{res.Matches[0].Entity.Id.OpaqueId, res.Matches[1].Entity.Id.OpaqueId, res.Matches[2].Entity.Id.OpaqueId}
					Expect(ids).To(ConsistOf("foo-id", "grant-shared-id", "grant-irrelevant-id"))
				})

				It("sorts and limits the combined results from all spaces", func() {
					res, err := s.Search(ctx, &searchsvc.SearchRequest{
						Query:    "foo",
						PageSize: 2,
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(res).ToNot(BeNil())
					Expect(len(res.Matches)).To(Equal(2))
					ids := []string{res.Matches[0].Entity.Id.OpaqueId, res.Matches[1].Entity.Id.OpaqueId}
					Expect(ids).To(Equal([]string{"grant-shared-id", "foo-id"}))
				})
			})
		})
	})
})

var _ = DescribeTable("Parse Scope",
	func(pattern, wantSearch, wantScope string) {
		gotSearch, gotScope := search.ParseScope(pattern)
		Expect(gotSearch).To(Equal(wantSearch))
		Expect(gotScope).To(Equal(wantScope))
	},
	Entry("When scope is at the end of the line",
		`+Name:*file* +Tags:&quot;foo&quot; scope:<uuid>/folder/subfolder`,
		`+Name:*file* +Tags:&quot;foo&quot;`,
		`<uuid>/folder/subfolder`,
	),
	Entry("When scope is at the end of the line 2",
		`+Name:*file* +Tags:&quot;foo&quot; scope:<uuid>/folder`,
		`+Name:*file* +Tags:&quot;foo&quot;`,
		`<uuid>/folder`,
	),
	Entry("When scope is at the end of the line 3",
		`file scope:<uuid>/folder/subfolder`,
		`file`,
		`<uuid>/folder/subfolder`,
	),
	Entry("When scope is at the end of the line with a space",
		`+Name:*file* +Tags:&quot;foo&quot; scope: <uuid>/folder/subfolder`,
		`+Name:*file* +Tags:&quot;foo&quot;`,
		`<uuid>/folder/subfolder`,
	),
	Entry("When scope is in the middle of the line",
		`+Name:*file* scope:<uuid>/folder/subfolder +Tags:&quot;foo&quot;`,
		`+Name:*file*  +Tags:&quot;foo&quot;`,
		`<uuid>/folder/subfolder`,
	),
	Entry("When scope is at the end of the line",
		`scope:<uuid>/folder/subfolder +Name:*file*`,
		`+Name:*file*`,
		`<uuid>/folder/subfolder`,
	),
	Entry("When scope is at the begging of the line",
		`scope:<uuid>/folder/subfolder file`,
		`file`,
		`<uuid>/folder/subfolder`,
	),
	Entry("When no scope",
		`+Name:*file* +Tags:&quot;foo&quot;`,
		`+Name:*file* +Tags:&quot;foo&quot;`,
		``,
	),
)
