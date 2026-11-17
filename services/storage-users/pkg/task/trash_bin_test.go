package task_test

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	apiGateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	apiUser "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	apiRpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	apiProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	apiTypes "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/task"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v3/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

func MustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

var _ = Describe("trash", func() {
	var (
		gatewayClient             *cs3mocks.GatewayAPIClient
		gatewaySelector           pool.Selectable[gateway.GatewayAPIClient]
		ctx                       context.Context
		now                       time.Time
		genericError              error
		user                      *apiUser.User
		getUserResponse           *apiUser.GetUserResponse
		authenticateResponse      *apiGateway.AuthenticateResponse
		listStorageSpacesResponse *apiProvider.ListStorageSpacesResponse
		personalSpace             *apiProvider.StorageSpace
		projectSpace              *apiProvider.StorageSpace
		virtualSpace              *apiProvider.StorageSpace
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

		ctx = context.Background()
		now = time.Now()
		genericError = errors.New("any")
		getUserResponse = &apiUser.GetUserResponse{
			Status: status.NewOK(ctx),
		}
		authenticateResponse = &apiGateway.AuthenticateResponse{
			Status: status.NewOK(ctx),
			Token:  "",
		}
		listStorageSpacesResponse = &apiProvider.ListStorageSpacesResponse{
			Status:        status.NewOK(ctx),
			StorageSpaces: []*apiProvider.StorageSpace{},
		}
		personalSpace = &apiProvider.StorageSpace{
			SpaceType: "personal",
			Id: &apiProvider.StorageSpaceId{
				OpaqueId: "personal",
			},
			Root: &apiProvider.ResourceId{
				OpaqueId: "personal",
			},
		}
		projectSpace = &apiProvider.StorageSpace{
			SpaceType: "project",
			Id: &apiProvider.StorageSpaceId{
				OpaqueId: "project",
			},
			Root: &apiProvider.ResourceId{
				OpaqueId: "project",
			},
			Opaque: &apiTypes.Opaque{},
		}
		// virtual is here as an example,
		// the task ignores all space types expect `project` and `personal`.
		virtualSpace = &apiProvider.StorageSpace{
			SpaceType: "virtual",
			Id: &apiProvider.StorageSpaceId{
				OpaqueId: "virtual",
			},
			Root: &apiProvider.ResourceId{
				OpaqueId: "virtual",
			},
		}
		user = &apiUser.User{
			Id: &apiUser.UserId{
				OpaqueId: "user",
			},
		}

	})

	Describe("PurgeTrashBin", func() {
		It("throws an error if the user cannot authenticate", func() {
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(nil, genericError)

			err := task.PurgeTrashBin("service-user-id", now, task.Project, gatewaySelector, "")
			Expect(err).To(HaveOccurred())
		})
		It("throws an error if space listing fails", func() {
			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(authenticateResponse, nil)
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(nil, genericError)

			err := task.PurgeTrashBin("service-user-id", now, task.Project, gatewaySelector, "")
			Expect(err).To(HaveOccurred())
		})
		It("only deletes items older than the specified period", func() {
			var (
				recycleItems = map[string][]*apiProvider.RecycleItem{
					"personal": {
						{Key: "now", DeletionTime: utils.TimeToTS(now)},
						{Key: "after", DeletionTime: utils.TimeToTS(now.Add(1 * time.Second))},
						{Key: "before", DeletionTime: utils.TimeToTS(now.Add(-1 * time.Second))},
					},
					"project": {
						{Key: "now", DeletionTime: utils.TimeToTS(now)},
						{Key: "after", DeletionTime: utils.TimeToTS(now.Add(1 * time.Minute))},
						{Key: "before", DeletionTime: utils.TimeToTS(now.Add(-1 * time.Minute))},
					},
					"virtual": {
						{Key: "now", DeletionTime: utils.TimeToTS(now)},
						{Key: "after", DeletionTime: utils.TimeToTS(now.Add(1 * time.Hour))},
						{Key: "before", DeletionTime: utils.TimeToTS(now.Add(-1 * time.Hour))},
					},
				}
			)

			personalSpace.Owner = user
			listStorageSpacesResponse.StorageSpaces = []*apiProvider.StorageSpace{
				personalSpace,
				projectSpace,
				virtualSpace,
			}
			projectSpace.Opaque.Map = map[string]*apiTypes.OpaqueEntry{
				"grants": {
					Decoder: "json",
					Value: MustMarshal(map[string]*apiProvider.ResourcePermissions{
						"admin": {
							Delete: true,
						},
					}),
				},
			}

			gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(getUserResponse, nil)
			gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(authenticateResponse, nil)
			gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(listStorageSpacesResponse, nil)
			gatewayClient.On("ListRecycle", mock.Anything, mock.Anything).Return(
				func(_ context.Context, req *apiProvider.ListRecycleRequest, _ ...grpc.CallOption) *apiProvider.ListRecycleResponse {
					return &apiProvider.ListRecycleResponse{
						RecycleItems: recycleItems[req.Ref.ResourceId.OpaqueId],
					}
				}, nil,
			)
			gatewayClient.On("PurgeRecycle", mock.Anything, mock.Anything).Return(
				func(_ context.Context, req *apiProvider.PurgeRecycleRequest, _ ...grpc.CallOption) *apiProvider.PurgeRecycleResponse {
					var items []*apiProvider.RecycleItem
					for _, item := range recycleItems[req.Ref.ResourceId.OpaqueId] {
						if req.Key == item.Key {
							continue
						}

						items = append(items, item)
					}

					recycleItems[req.Ref.ResourceId.OpaqueId] = items

					return &apiProvider.PurgeRecycleResponse{
						Status: &apiRpc.Status{
							Code: apiRpc.Code_CODE_OK,
						},
					}
				}, nil,
			)

			err := task.PurgeTrashBin("service-user-id", now, task.Project, gatewaySelector, "")
			Expect(err).To(BeNil())
			Expect(recycleItems["personal"]).To(HaveLen(2))
			Expect(recycleItems["project"]).To(HaveLen(2))
			// virtual spaces are ignored
			Expect(recycleItems["virtual"]).To(HaveLen(3))
		})
	})
})
