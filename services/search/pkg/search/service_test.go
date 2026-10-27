package search_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaborationv1beta1 "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	revactx "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v3/tests/cs3mocks/mocks"
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

		var err error
		s, err = search.NewService(gatewaySelector, indexClient, extractor, nil, logger, &config.Config{})
		Expect(err).ToNot(HaveOccurred())

		gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(&gateway.AuthenticateResponse{
			Status: status.NewOK(ctx),
			Token:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjQxMDI0NDQ4MDB9.c2lnbmF0dXJl",
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
			s, err := search.NewService(gatewaySelector, indexClient, extractor, nil, logger, &config.Config{})
			Expect(err).ToNot(HaveOccurred())
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
			indexClient.On("Get", mock.Anything, mock.Anything).Return(nil, search.ErrResourceNotFound)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
				Status: status.NewOK(context.Background()),
				Info:   ri,
			}, nil)

			err := s.IndexSpace(&sprovider.StorageSpaceId{OpaqueId: "storageid$spaceid!spaceid"}, false)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("descends into an unchanged folder and indexes only what is missing or outdated below it", func() {
			indexedAt := time.Unix(4000, 0)
			info := func(id string, t sprovider.ResourceType, seconds uint64) *sprovider.ResourceInfo {
				return &sprovider.ResourceInfo{
					Id:       &sprovider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: id},
					ParentId: &sprovider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "spaceid"},
					Type:     t,
					Path:     id,
					Mtime:    &typesv1beta1.Timestamp{Seconds: seconds},
				}
			}
			root := info("spaceid", sprovider.ResourceType_RESOURCE_TYPE_CONTAINER, 4000)
			root.Path = "."
			children := []*sprovider.ResourceInfo{
				info("current", sprovider.ResourceType_RESOURCE_TYPE_FILE, 4000),
				info("outdated", sprovider.ResourceType_RESOURCE_TYPE_FILE, 5000),
				info("missing", sprovider.ResourceType_RESOURCE_TYPE_FILE, 4000),
				info("trashed", sprovider.ResourceType_RESOURCE_TYPE_FILE, 4000),
				info("failed-due", sprovider.ResourceType_RESOURCE_TYPE_FILE, 4000),
				info("failed-waiting", sprovider.ResourceType_RESOURCE_TYPE_FILE, 4000),
				info("failed-given-up", sprovider.ResourceType_RESOURCE_TYPE_FILE, 4000),
				info("failed-given-up-changed", sprovider.ResourceType_RESOURCE_TYPE_FILE, 5000),
				info("failed-unreachable", sprovider.ResourceType_RESOURCE_TYPE_FILE, 4000),
			}
			indexed := map[string]*search.Resource{
				"storageid$spaceid!spaceid":  {Document: content.Document{Mtime: &indexedAt}},
				"storageid$spaceid!current":  {Document: content.Document{Mtime: &indexedAt}},
				"storageid$spaceid!outdated": {Document: content.Document{Mtime: &indexedAt}},
				"storageid$spaceid!trashed":  {Document: content.Document{Mtime: &indexedAt}, Deleted: true},
				"storageid$spaceid!failed-due": {Document: content.Document{Mtime: &indexedAt},
					ExtractionFailure: &search.ExtractionFailure{Count: 1, RetryAt: ptr(time.Now().Add(-time.Minute))}},
				"storageid$spaceid!failed-waiting": {Document: content.Document{Mtime: &indexedAt},
					ExtractionFailure: &search.ExtractionFailure{Count: 2, RetryAt: ptr(time.Now().Add(time.Hour))}},
				"storageid$spaceid!failed-given-up": {Document: content.Document{Mtime: &indexedAt},
					ExtractionFailure: &search.ExtractionFailure{Count: 3}},
				"storageid$spaceid!failed-given-up-changed": {Document: content.Document{Mtime: &indexedAt},
					ExtractionFailure: &search.ExtractionFailure{Count: 3}},
				"storageid$spaceid!failed-unreachable": {Document: content.Document{Mtime: &indexedAt},
					ExtractionFailure: &search.ExtractionFailure{}},
			}

			gatewayClient.On("GetUserByClaim", mock.Anything, mock.Anything).Return(&userv1beta1.GetUserByClaimResponse{
				Status: status.NewOK(context.Background()),
				User:   user,
			}, nil)
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(
				func(_ context.Context, req *sprovider.StatRequest, _ ...grpc.CallOption) (*sprovider.StatResponse, error) {
					for _, child := range children {
						if req.GetRef().GetPath() == "./"+child.GetPath() {
							return &sprovider.StatResponse{Status: status.NewOK(context.Background()), Info: child}, nil
						}
					}
					return &sprovider.StatResponse{Status: status.NewOK(context.Background()), Info: root}, nil
				})
			gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(&sprovider.ListContainerResponse{
				Status: status.NewOK(context.Background()),
				Infos:  children,
			}, nil)
			indexClient.On("Get", mock.Anything, mock.Anything).Return(
				func(_ context.Context, id string) (*search.Resource, error) {
					if r, ok := indexed[id]; ok {
						return r, nil
					}
					return nil, search.ErrResourceNotFound
				})
			extractor.On("Extract", mock.Anything, mock.Anything, mock.Anything).Return(content.Document{}, nil)

			var upserted []string
			batch := &engineMocks.BatchOperator{}
			batch.EXPECT().Push().Return(nil)
			batch.On("Upsert", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				upserted = append(upserted, args.String(0))
			}).Return(nil)
			indexClient.On("NewBatch", mock.Anything).Return(batch, nil)

			err := s.IndexSpace(&sprovider.StorageSpaceId{OpaqueId: "storageid$spaceid!spaceid"}, false)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(upserted).To(ConsistOf(
				"storageid$spaceid!outdated",
				"storageid$spaceid!missing",
				"storageid$spaceid!trashed",
				"storageid$spaceid!failed-due",
				"storageid$spaceid!failed-given-up-changed",
				"storageid$spaceid!failed-unreachable",
			))
		})

		Context("when tika or the files are out of reach", func() {
			var (
				files     []string
				extract   func(name string, attempt int) (content.Document, error)
				attempts  map[string]int
				upserted  map[string]search.Resource
				unreached = fmt.Errorf("%w: connection refused", content.ErrUnreachable)
			)

			BeforeEach(func() {
				DeferCleanup(search.SetUnreachableRetryDelay(time.Millisecond))
				attempts = map[string]int{}
				upserted = map[string]search.Resource{}

				root := &sprovider.ResourceInfo{
					Id:    &sprovider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "spaceid"},
					Type:  sprovider.ResourceType_RESOURCE_TYPE_CONTAINER,
					Path:  ".",
					Mtime: &typesv1beta1.Timestamp{Seconds: 4000},
				}
				children := func() []*sprovider.ResourceInfo {
					infos := make([]*sprovider.ResourceInfo, 0, len(files))
					for _, name := range files {
						infos = append(infos, &sprovider.ResourceInfo{
							Id:       &sprovider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: name},
							ParentId: root.GetId(),
							Type:     sprovider.ResourceType_RESOURCE_TYPE_FILE,
							Path:     name,
							Mtime:    &typesv1beta1.Timestamp{Seconds: 4000},
						})
					}
					return infos
				}

				gatewayClient.On("GetUserByClaim", mock.Anything, mock.Anything).Return(&userv1beta1.GetUserByClaimResponse{
					Status: status.NewOK(context.Background()),
					User:   user,
				}, nil)
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(
					func(_ context.Context, req *sprovider.StatRequest, _ ...grpc.CallOption) (*sprovider.StatResponse, error) {
						for _, child := range children() {
							if req.GetRef().GetPath() == "./"+child.GetPath() {
								return &sprovider.StatResponse{Status: status.NewOK(context.Background()), Info: child}, nil
							}
						}
						return &sprovider.StatResponse{Status: status.NewOK(context.Background()), Info: root}, nil
					})
				gatewayClient.On("ListContainer", mock.Anything, mock.Anything).Return(
					func(context.Context, *sprovider.ListContainerRequest, ...grpc.CallOption) (*sprovider.ListContainerResponse, error) {
						return &sprovider.ListContainerResponse{Status: status.NewOK(context.Background()), Infos: children()}, nil
					})
				indexClient.On("Get", mock.Anything, mock.Anything).Return(nil, search.ErrResourceNotFound)
				extractor.On("Extract", mock.Anything, mock.Anything).Return(
					func(_ context.Context, info *sprovider.ResourceInfo) (content.Document, error) {
						if info.GetType() == sprovider.ResourceType_RESOURCE_TYPE_CONTAINER {
							return content.Document{}, nil
						}
						attempts[info.GetPath()]++
						return extract(info.GetPath(), attempts[info.GetPath()])
					})

				batch := &engineMocks.BatchOperator{}
				batch.EXPECT().Push().Return(nil)
				batch.On("Upsert", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					upserted[args.String(0)] = args.Get(1).(search.Resource)
				}).Return(nil)
				indexClient.On("NewBatch", mock.Anything).Return(batch, nil)
			})

			It("retries the extraction and indexes the content once tika answers again", func() {
				files = []string{"report.pdf"}
				extract = func(_ string, attempt int) (content.Document, error) {
					if attempt < 3 {
						return content.Document{Name: "report.pdf"}, unreached
					}
					return content.Document{Name: "report.pdf", Content: "quarterly numbers"}, nil
				}

				Expect(s.IndexSpace(&sprovider.StorageSpaceId{OpaqueId: "storageid$spaceid!spaceid"}, false)).To(Succeed())
				Expect(attempts).To(Equal(map[string]int{"report.pdf": 3}))
				r := upserted["storageid$spaceid!report.pdf"]
				Expect(r.Content).To(Equal("quarterly numbers"))
				Expect(r.ExtractionFailure).To(BeNil())
			})

			It("indexes a file still out of reach by name without failing the walk", func() {
				files = []string{"gone.pdf", "report.pdf"}
				extract = func(name string, _ int) (content.Document, error) {
					if name == "gone.pdf" {
						return content.Document{Name: name}, unreached
					}
					return content.Document{Name: name}, nil
				}

				Expect(s.IndexSpace(&sprovider.StorageSpaceId{OpaqueId: "storageid$spaceid!spaceid"}, false)).To(Succeed())
				Expect(attempts).To(Equal(map[string]int{"gone.pdf": 6, "report.pdf": 1}))
				Expect(upserted["storageid$spaceid!gone.pdf"].ExtractionFailure).To(Equal(&search.ExtractionFailure{}))
			})

			It("does not retry a file tika cannot parse", func() {
				files = []string{"scan.png"}
				extract = func(name string, _ int) (content.Document, error) {
					return content.Document{Name: name}, fmt.Errorf("%w %q", content.ErrUnparsable, name)
				}

				Expect(s.IndexSpace(&sprovider.StorageSpaceId{OpaqueId: "storageid$spaceid!spaceid"}, false)).To(Succeed())
				Expect(attempts).To(Equal(map[string]int{"scan.png": 1}))
			})

			It("does not retry a download the storage refuses and counts it as a failure", func() {
				files = []string{"gone.pdf"}
				extract = func(name string, _ int) (content.Document, error) {
					return content.Document{Name: name}, errors.New("could not load resoure: not found")
				}

				Expect(s.IndexSpace(&sprovider.StorageSpaceId{OpaqueId: "storageid$spaceid!spaceid"}, false)).To(Succeed())
				Expect(attempts).To(Equal(map[string]int{"gone.pdf": 1}))
				Expect(upserted["storageid$spaceid!gone.pdf"].ExtractionFailure.Count).To(Equal(1))
			})

			It("stops retrying once tika stays down, indexes the remaining files by name and reports it", func() {
				files = []string{"a.pdf", "b.pdf", "c.pdf", "d.pdf", "e.pdf", "f.pdf", "g.pdf"}
				extract = func(name string, _ int) (content.Document, error) {
					return content.Document{Name: name}, unreached
				}

				err := s.IndexSpace(&sprovider.StorageSpaceId{OpaqueId: "storageid$spaceid!spaceid"}, false)
				Expect(err).To(MatchError(unreached))
				Expect(err).To(MatchError(ContainSubstring("7 files indexed by name only")))
				Expect(attempts).To(Equal(map[string]int{
					"a.pdf": 6, "b.pdf": 6, "c.pdf": 6, "d.pdf": 6, "e.pdf": 6,
					"f.pdf": 1, "g.pdf": 1,
				}))
				for _, name := range files {
					Expect(upserted).To(HaveKey("storageid$spaceid!" + name))
				}
			})

			It("retries again once an extraction succeeds after tika was down", func() {
				files = []string{"a.pdf", "b.pdf", "c.pdf", "d.pdf", "e.pdf", "back.pdf", "z.pdf"}
				extract = func(name string, attempt int) (content.Document, error) {
					switch {
					case name == "back.pdf":
						return content.Document{Name: name}, nil
					case name == "z.pdf" && attempt > 1:
						return content.Document{Name: name}, nil
					}
					return content.Document{Name: name}, unreached
				}

				// tika was down during the walk, which is still reported
				Expect(s.IndexSpace(&sprovider.StorageSpaceId{OpaqueId: "storageid$spaceid!spaceid"}, false)).To(MatchError(unreached))
				Expect(attempts).To(HaveKeyWithValue("back.pdf", 1))
				Expect(attempts).To(HaveKeyWithValue("z.pdf", 2))
			})

			It("skips the extraction while tika does not answer its version", func() {
				files = []string{"a.pdf", "b.pdf", "c.pdf", "d.pdf", "e.pdf", "f.pdf", "g.pdf"}
				extract = func(name string, _ int) (content.Document, error) {
					if name == "g.pdf" {
						return content.Document{Name: name, Content: "back"}, nil
					}
					return content.Document{Name: name}, unreached
				}
				// tika hangs from the first file on and answers again once f.pdf is indexed
				probing := probingExtractor{Extractor: extractor, remote: remoteFiles, probe: func() error {
					if _, back := upserted["storageid$spaceid!f.pdf"]; attempts["a.pdf"] > 0 && !back {
						return unreached
					}
					return nil
				}}
				var err error
				s, err = search.NewService(gatewaySelector, indexClient, probing, nil, logger, &config.Config{})
				Expect(err).ToNot(HaveOccurred())

				Expect(s.IndexSpace(&sprovider.StorageSpaceId{OpaqueId: "storageid$spaceid!spaceid"}, false)).To(MatchError(unreached))
				Expect(attempts).To(Equal(map[string]int{"a.pdf": 1, "g.pdf": 1}))
				for _, name := range files {
					Expect(upserted).To(HaveKey("storageid$spaceid!" + name))
				}
				Expect(upserted["storageid$spaceid!g.pdf"].Content).To(Equal("back"))
			})

			It("keeps tika down across a file the extractor does not send to tika", func() {
				files = []string{"a.pdf", "b.pdf", "c.pdf", "d.pdf", "e.pdf", "huge.pdf", "f.pdf"}
				extract = func(name string, _ int) (content.Document, error) {
					if name == "huge.pdf" {
						return content.Document{Name: name}, nil
					}
					return content.Document{Name: name}, unreached
				}
				probing := probingExtractor{
					Extractor: extractor,
					remote: func(info *sprovider.ResourceInfo) bool {
						return remoteFiles(info) && info.GetPath() != "huge.pdf"
					},
					probe: func() error { return nil },
				}
				var err error
				s, err = search.NewService(gatewaySelector, indexClient, probing, nil, logger, &config.Config{})
				Expect(err).ToNot(HaveOccurred())

				Expect(s.IndexSpace(&sprovider.StorageSpaceId{OpaqueId: "storageid$spaceid!spaceid"}, false)).To(MatchError(unreached))
				Expect(attempts).To(HaveKeyWithValue("huge.pdf", 1))
				Expect(attempts).To(HaveKeyWithValue("f.pdf", 1))
			})
		})
	})

	Describe("indexing a resource in full", func() {
		BeforeEach(func() {
			// a resource the index does not hold yet takes the full path
			indexClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(search.ErrResourceNotFound)
		})

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

				s.UpdateMetadata(&sprovider.Reference{ResourceId: info.GetId()})
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

		DescribeTable("indexes an unparsable resource by name and counts the failures of its version",
			func(previous *search.Resource, count int, retryIn time.Duration) {
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
					Status: status.NewOK(context.Background()),
					Info:   ri,
				}, nil)
				extractor.On("Extract", mock.Anything, mock.Anything, mock.Anything).Return(content.Document{Name: "scan.png"}, fmt.Errorf("%w %q", content.ErrUnparsable, "scan.png"))
				if previous == nil {
					indexClient.On("Get", mock.Anything, mock.Anything).Return(nil, search.ErrResourceNotFound)
				} else {
					indexClient.On("Get", mock.Anything, mock.Anything).Return(previous, nil)
				}

				var indexed search.Resource
				indexClient.On("Upsert", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					indexed = args.Get(1).(search.Resource)
				}).Return(nil)

				s.UpdateMetadata(&sprovider.Reference{ResourceId: ri.GetId()})
				Expect(indexed.Name).To(Equal("scan.png"))
				Expect(indexed.ExtractionFailure).ToNot(BeNil())
				Expect(indexed.ExtractionFailure.Count).To(Equal(count))
				if retryIn == 0 {
					Expect(indexed.ExtractionFailure.RetryAt).To(BeNil())
				} else {
					Expect(indexed.ExtractionFailure.RetryAt).NotTo(BeNil())
					Expect(*indexed.ExtractionFailure.RetryAt).To(BeTemporally("~", time.Now().Add(retryIn), time.Minute))
				}
			},
			Entry("first failure", nil, 1, time.Hour),
			Entry("second failure of the same version",
				&search.Resource{Document: content.Document{Mtime: ptr(time.Unix(4000, 0))}, ExtractionFailure: &search.ExtractionFailure{Count: 1}},
				2, 24*time.Hour),
			Entry("third failure of the same version gives up",
				&search.Resource{Document: content.Document{Mtime: ptr(time.Unix(4000, 0))}, ExtractionFailure: &search.ExtractionFailure{Count: 2}},
				3, time.Duration(0)),
			Entry("a new version starts over",
				&search.Resource{Document: content.Document{Mtime: ptr(time.Unix(3000, 0))}, ExtractionFailure: &search.ExtractionFailure{Count: 3}},
				1, time.Hour),
		)

		DescribeTable("indexes a resource by name without counting when tika or the file is unreachable",
			func(previous *search.Resource, getErr error, written bool) {
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
					Status: status.NewOK(context.Background()),
					Info:   ri,
				}, nil)
				extractor.On("Extract", mock.Anything, mock.Anything, mock.Anything).Return(content.Document{Name: "scan.png"}, fmt.Errorf("%w: connection refused", content.ErrUnreachable))
				indexClient.On("Get", mock.Anything, mock.Anything).Return(previous, getErr)

				var indexed []search.Resource
				indexClient.On("Upsert", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					indexed = append(indexed, args.Get(1).(search.Resource))
				}).Return(nil)

				s.UpdateMetadata(&sprovider.Reference{ResourceId: ri.GetId()})
				if !written {
					Expect(indexed).To(BeEmpty())
					return
				}
				Expect(indexed).To(HaveLen(1))
				Expect(indexed[0].Name).To(Equal("scan.png"))
				Expect(indexed[0].ExtractionFailure).To(Equal(&search.ExtractionFailure{}))
			},
			Entry("a new resource", nil, search.ErrResourceNotFound, true),
			Entry("an outdated entry",
				&search.Resource{Document: content.Document{Mtime: ptr(time.Unix(3000, 0)), Content: "old"}},
				nil, true),
			Entry("an outdated entry with parse failures starts over",
				&search.Resource{Document: content.Document{Mtime: ptr(time.Unix(3000, 0))}, ExtractionFailure: &search.ExtractionFailure{Count: 2}},
				nil, true),
			Entry("a current entry keeps its content",
				&search.Resource{Document: content.Document{Mtime: ptr(time.Unix(4000, 0)), Content: "extracted"}},
				nil, false),
			Entry("a current entry keeps its parse failures",
				&search.Resource{Document: content.Document{Mtime: ptr(time.Unix(4000, 0))}, ExtractionFailure: &search.ExtractionFailure{Count: 3}},
				nil, false),
			Entry("an unknown entry is left alone", nil, errors.New("index unavailable"), false),
		)

		Context("with facets stored on the resource", func() {
			var (
				info  *sprovider.ResourceInfo
				calls []string
				set   map[string]string
				unset []string
			)

			BeforeEach(func() {
				calls, set, unset = nil, nil, nil
				info = &sprovider.ResourceInfo{
					Id:       &sprovider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "photo-opaqueid"},
					ParentId: &sprovider.ResourceId{StorageId: "storageid", OpaqueId: "parentopaqueid"},
					Path:     "./photo.jpg",
					ArbitraryMetadata: &sprovider.ArbitraryMetadata{
						Metadata: map[string]string{
							"libre.graph.photo.cameraMake":    "Canon",
							"libre.graph.photo.iso":           "100",
							"libre.graph.location.altitude":   "12",
							"http://owncloud.org/ns/favorite": "1",
						},
					},
				}
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
					Status: status.NewOK(context.Background()),
					Info:   info,
				}, nil)
				gatewayClient.On("GetPath", mock.Anything, mock.MatchedBy(func(req *sprovider.GetPathRequest) bool {
					return req.GetResourceId().GetOpaqueId() == info.GetId().GetOpaqueId()
				})).Return(&sprovider.GetPathResponse{
					Status: status.NewOK(context.Background()),
					Path:   info.GetPath(),
				}, nil)
				indexClient.On("Upsert", mock.Anything, mock.Anything).Return(nil)
				gatewayClient.On("SetArbitraryMetadata", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					calls = append(calls, "set")
					set = args.Get(1).(*sprovider.SetArbitraryMetadataRequest).GetArbitraryMetadata().GetMetadata()
				}).Return(&sprovider.SetArbitraryMetadataResponse{Status: status.NewOK(context.Background())}, nil)
				gatewayClient.On("UnsetArbitraryMetadata", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					calls = append(calls, "unset")
					unset = args.Get(1).(*sprovider.UnsetArbitraryMetadataRequest).GetArbitraryMetadataKeys()
				}).Return(&sprovider.UnsetArbitraryMetadataResponse{Status: status.NewOK(context.Background())}, nil)
			})

			It("removes the facets the extraction no longer produces after storing the new ones", func() {
				extractor.On("Extract", mock.Anything, mock.Anything, mock.Anything).Return(content.Document{
					Photo: &libregraph.Photo{CameraMake: libregraph.PtrString("Nikon")},
				}, nil)

				s.UpdateMetadata(&sprovider.Reference{ResourceId: info.GetId()})
				Expect(calls).To(Equal([]string{"set", "unset"}))
				Expect(set).To(Equal(map[string]string{"libre.graph.photo.cameraMake": "Nikon"}))
				Expect(unset).To(ConsistOf("libre.graph.photo.iso", "libre.graph.location.altitude"))
			})

			It("removes all facets when the extraction produces none", func() {
				extractor.On("Extract", mock.Anything, mock.Anything, mock.Anything).Return(content.Document{}, nil)

				s.UpdateMetadata(&sprovider.Reference{ResourceId: info.GetId()})
				Expect(calls).To(Equal([]string{"unset"}))
				Expect(unset).To(ConsistOf("libre.graph.photo.cameraMake", "libre.graph.photo.iso", "libre.graph.location.altitude"))
			})

			It("removes keys of facets the indexer no longer knows", func() {
				info.ArbitraryMetadata.Metadata["libre.graph.retiredFacet.field"] = "value"
				extractor.On("Extract", mock.Anything, mock.Anything, mock.Anything).Return(content.Document{
					Photo:    &libregraph.Photo{CameraMake: libregraph.PtrString("Canon"), Iso: libregraph.PtrInt32(100)},
					Location: &libregraph.GeoCoordinates{Altitude: libregraph.PtrFloat64(12)},
				}, nil)

				s.UpdateMetadata(&sprovider.Reference{ResourceId: info.GetId()})
				Expect(calls).To(Equal([]string{"unset"}))
				Expect(unset).To(ConsistOf("libre.graph.retiredFacet.field"))
			})

			It("writes only the keys whose value changed", func() {
				extractor.On("Extract", mock.Anything, mock.Anything, mock.Anything).Return(content.Document{
					Photo:    &libregraph.Photo{CameraMake: libregraph.PtrString("Canon"), Iso: libregraph.PtrInt32(200)},
					Location: &libregraph.GeoCoordinates{Altitude: libregraph.PtrFloat64(12), Latitude: libregraph.PtrFloat64(48.1)},
				}, nil)

				s.UpdateMetadata(&sprovider.Reference{ResourceId: info.GetId()})
				Expect(calls).To(Equal([]string{"set"}))
				Expect(set).To(Equal(map[string]string{
					"libre.graph.photo.iso":         "200",
					"libre.graph.location.latitude": "48.1",
				}))
			})

			It("does not touch the storage when the extraction still produces every stored facet unchanged", func() {
				extractor.On("Extract", mock.Anything, mock.Anything, mock.Anything).Return(content.Document{
					Photo:    &libregraph.Photo{CameraMake: libregraph.PtrString("Canon"), Iso: libregraph.PtrInt32(100)},
					Location: &libregraph.GeoCoordinates{Altitude: libregraph.PtrFloat64(12)},
				}, nil)

				s.UpdateMetadata(&sprovider.Reference{ResourceId: info.GetId()})
				Expect(calls).To(BeEmpty())
			})

			It("keeps the stored facets when the extraction failed", func() {
				extractor.On("Extract", mock.Anything, mock.Anything, mock.Anything).Return(content.Document{}, content.ErrUnparsable)
				indexClient.On("Get", mock.Anything, mock.Anything).Return(nil, search.ErrResourceNotFound)

				s.UpdateMetadata(&sprovider.Reference{ResourceId: info.GetId()})
				Expect(calls).To(BeEmpty())
			})
		})
	})

	DescribeTable("SanitizeHighlights",
		func(highlights, expected string) {
			Expect(search.SanitizeHighlights(highlights)).To(Equal(expected))
		},
		Entry("plain text", "foo <mark>bar</mark> baz", "foo <mark>bar</mark> baz"),
		Entry("raw markup", "<img src=x> <mark>bar</mark>", "&lt;img src=x&gt; <mark>bar</mark>"),
		Entry("already escaped", "&lt;img src=x&gt; <mark>bar</mark>", "&lt;img src=x&gt; <mark>bar</mark>"),
		Entry("escaped text from the file stays as typed", "&amp;lt;img src=x&amp;gt; <mark>bar</mark>", "&amp;lt;img src=x&amp;gt; <mark>bar</mark>"),
		Entry("markup inside a mark", "<mark><b>bar</b></mark>", "<mark>&lt;b&gt;bar&lt;/b&gt;</mark>"),
		Entry("a mark with attributes", `<mark class="x">bar</mark>`, "&lt;mark class=&#34;x&#34;&gt;bar</mark>"),
		Entry("an escaped ampersand stays single", "Tom &amp; Jerry", "Tom &amp; Jerry"),
		Entry("empty", "", ""),
	)

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

	Describe("UpdateMetadata", func() {
		var info *sprovider.ResourceInfo

		BeforeEach(func() {
			info = &sprovider.ResourceInfo{
				Id:       &sprovider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "tagged-opaqueid"},
				ParentId: &sprovider.ResourceId{StorageId: "storageid", OpaqueId: "parentopaqueid"},
				Path:     "./report.pdf",
				Size:     12345,
				ArbitraryMetadata: &sprovider.ArbitraryMetadata{
					Metadata: map[string]string{"tags": "alpha,beta"},
				},
				Opaque: &typesv1beta1.Opaque{
					Map: map[string]*typesv1beta1.OpaqueEntry{
						"favorites": {Decoder: "json", Value: []byte(`["user-1"]`)},
					},
				},
			}
			gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
				Status: status.NewOK(context.Background()),
				Info:   info,
			}, nil)
			gatewayClient.On("GetPath", mock.Anything, mock.MatchedBy(func(req *sprovider.GetPathRequest) bool {
				return req.GetResourceId().GetOpaqueId() == info.GetId().GetOpaqueId()
			})).Return(&sprovider.GetPathResponse{
				Status: status.NewOK(context.Background()),
				Path:   info.GetPath(),
			}, nil)
		})

		It("rewrites the tags and favorites without extracting the content", func() {
			indexClient.On("Update", mock.Anything, "storageid$spaceid!tagged-opaqueid", search.MetadataUpdate{
				Tags:      []string{"alpha", "beta"},
				Favorites: []string{"user-1"},
			}).Return(nil)

			s.UpdateMetadata(&sprovider.Reference{ResourceId: info.GetId()})
			indexClient.AssertNumberOfCalls(GinkgoT(), "Update", 1)
			extractor.AssertNotCalled(GinkgoT(), "Extract", mock.Anything, mock.Anything)
			indexClient.AssertNotCalled(GinkgoT(), "Upsert", mock.Anything, mock.Anything)
		})

		It("indexes the resource in full when the index does not hold it yet", func() {
			indexClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(search.ErrResourceNotFound)
			extractor.On("Extract", mock.Anything, mock.Anything).Return(content.Document{Tags: []string{"alpha", "beta"}}, nil)

			var indexed search.Resource
			indexClient.On("Upsert", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				indexed = args.Get(1).(search.Resource)
			}).Return(nil)

			s.UpdateMetadata(&sprovider.Reference{ResourceId: info.GetId()})
			Expect(indexed.ID).To(Equal("storageid$spaceid!tagged-opaqueid"))
			gatewayClient.AssertNumberOfCalls(GinkgoT(), "Stat", 1)
		})

		It("does not extract the content when the update fails for another reason", func() {
			indexClient.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("cluster unavailable"))

			s.UpdateMetadata(&sprovider.Reference{ResourceId: info.GetId()})
			extractor.AssertNotCalled(GinkgoT(), "Extract", mock.Anything, mock.Anything)
			indexClient.AssertNotCalled(GinkgoT(), "Upsert", mock.Anything, mock.Anything)
		})
	})

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
				indexClient.On("Search", mock.Anything, mock.Anything).Return(&search.IndexResult{
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
								Name:       "Foo.pdf",
								Highlights: `foo <img src=x> <mark>bar</mark>`,
							},
						},
					},
				}, nil)
				gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&sprovider.StatResponse{
					Status: status.NewOK(context.Background()),
					Info:   ri,
				}, nil)
			})

			It("escapes the highlights whatever the engine returned", func() {
				res, err := s.Search(ctx, &searchsvc.SearchRequest{
					Query: "bar",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Matches).To(HaveLen(1))
				Expect(res.Matches[0].Entity.Highlights).To(Equal(`foo &lt;img src=x&gt; <mark>bar</mark>`))
			})

			It("does not mess with field-based searches", func() {
				_, err := s.Search(ctx, &searchsvc.SearchRequest{
					Query: "Size:<10",
				})
				Expect(err).ToNot(HaveOccurred())
				indexClient.AssertCalled(GinkgoT(), "Search", mock.Anything, mock.MatchedBy(func(req *search.IndexQuery) bool {
					return render(req.Query.Nodes) == render(parse("Size:<10").Nodes)
				}))
			})

			It("adds the query filter with the personal space root filled in", func() {
				s, err := search.NewService(gatewaySelector, indexClient, extractor, nil, logger, &config.Config{
					QueryTemplate: `({{.QUERY}}) AND (NOT path:"./.space" AND NOT id:{{.SPACE_ROOT_PERSONAL}})`,
				})
				Expect(err).ToNot(HaveOccurred())
				_, err = s.Search(ctx, &searchsvc.SearchRequest{Query: "foo"})
				Expect(err).ToNot(HaveOccurred())
				indexClient.AssertCalled(GinkgoT(), "Search", mock.Anything, mock.MatchedBy(func(req *search.IndexQuery) bool {
					return render(req.Query.Nodes) == render(parse(`(foo) AND (NOT path:"./.space" AND NOT id:"storageid$personalspace!personalspace")`).Nodes)
				}))
			})

			It("continues with the remaining spaces when one fails", func() {
				failingClient := &engineMocks.Engine{}
				failingClient.On("Search", mock.Anything, mock.Anything).Run(func(mock.Arguments) {
					panic("unexpected")
				}).Return(nil, nil)
				s, err := search.NewService(gatewaySelector, failingClient, extractor, nil, logger, &config.Config{})
				Expect(err).ToNot(HaveOccurred())

				res, err := s.Search(ctx, &searchsvc.SearchRequest{Query: "foo"})
				Expect(err).ToNot(HaveOccurred())
				Expect(res.TotalMatches).To(Equal(int32(0)))
				Expect(res.Matches).To(BeEmpty())
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
				indexClient.On("Search", mock.Anything, mock.Anything).Return(&search.IndexResult{
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
				indexClient.On("Search", mock.Anything, mock.Anything).Return(&search.IndexResult{
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
					indexClient.On("Search", mock.Anything, mock.MatchedBy(func(req *search.IndexQuery) bool {
						return req.Ref.ResourceId.OpaqueId == grantSpace.Root.SpaceId &&
							req.Ref.ResourceId.SpaceId == grantSpace.Root.SpaceId
					})).Return(&search.IndexResult{
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
					indexClient.On("Search", mock.Anything, mock.MatchedBy(func(req *search.IndexQuery) bool {
						return req.Ref.ResourceId.OpaqueId == personalSpace.Root.OpaqueId &&
							req.Ref.ResourceId.SpaceId == personalSpace.Root.SpaceId
					})).Return(&search.IndexResult{
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
	Entry("When scope is joined with AND, the operator goes with it",
		`name:"*mediaTest*" AND mtime:last 30 days AND scope:1234$5678!90`,
		`name:"*mediaTest*" AND mtime:last 30 days`,
		`1234$5678!90`,
	),
)

func ptr[T any](v T) *T {
	return &v
}

// probingExtractor makes the extractor mock one that hands files to tika
type probingExtractor struct {
	*contentMocks.Extractor
	probe  func() error
	remote func(info *sprovider.ResourceInfo) bool
}

func (p probingExtractor) Probe(context.Context) error {
	return p.probe()
}

func (p probingExtractor) Remote(info *sprovider.ResourceInfo) bool {
	return p.remote(info)
}

func remoteFiles(info *sprovider.ResourceInfo) bool {
	return info.GetType() == sprovider.ResourceType_RESOURCE_TYPE_FILE
}
