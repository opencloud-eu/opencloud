package consumer_test

import (
	"context"
	"encoding/json"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/internal/eventstest"
	"github.com/opencloud-eu/opencloud/pkg/log"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	settingsmocks "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0/mocks"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/config"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/service"
	consumersvc "github.com/opencloud-eu/opencloud/services/userlog/pkg/service/consumer"
)

var _ = Describe("Userlog consumer", func() {
	var (
		cfg = &config.Config{
			MaxConcurrency: 5,
			DisableSSE:     true,
			Service: config.Service{
				Name: "userlog",
			},
		}

		cs  *consumersvc.Service
		bus eventstest.TestBus
		sto microstore.Store

		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]

		vc settingsmocks.ValueService
	)

	BeforeEach(func() {
		var err error
		sto = store.Create()
		bus = eventstest.NewTestBus()

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector = pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		o := utils.AppendJSONToOpaque(nil, "grants", map[string]*provider.ResourcePermissions{"userid": {Stat: true}})
		gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{StorageSpaces: []*provider.StorageSpace{
			{
				Opaque:    o,
				SpaceType: "project",
			},
		}, Status: &rpc.Status{Code: rpc.Code_CODE_OK}}, nil)
		gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(&user.GetUserResponse{User: &user.User{Id: &user.UserId{OpaqueId: "userid"}}, Status: &rpc.Status{Code: rpc.Code_CODE_OK}}, nil)
		gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(&gateway.AuthenticateResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}}, nil)
		vc.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything).Return(&settingssvc.GetValueResponse{
			Value: &settingsmsg.ValueWithIdentifier{
				Value: &settingsmsg.Value{
					Value: &settingsmsg.Value_CollectionValue{
						CollectionValue: &settingsmsg.CollectionValue{
							Values: []*settingsmsg.CollectionOption{
								{
									Key:    "in-app",
									Option: &settingsmsg.CollectionOption_BoolValue{BoolValue: true},
								},
							},
						},
					},
				},
			},
		}, nil)

		ul, err := service.NewUserlogService(
			service.Store(sto),
			service.Logger(log.NewLogger()),
			service.TraceProvider(trace.NewNoopTracerProvider()),
		)
		Expect(err).ToNot(HaveOccurred())

		cs, err = consumersvc.New(
			ul,
			bus,
			consumersvc.Context(context.Background()),
			consumersvc.Logger(log.NewLogger()),
			consumersvc.Config(cfg),
			consumersvc.GatewaySelector(gatewaySelector),
			consumersvc.ValueClient(&vc),
			consumersvc.RegisteredEvents([]events.Unmarshaller{
				events.SpaceDisabled{},
			}),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	It("stores events without HTTP (consumer-only mode)", func() {
		go func() {
			Expect(cs.Run()).To(Succeed())
		}()

		ids := make(map[string]struct{})
		ids[bus.Push(events.SpaceDisabled{Executant: &user.UserId{OpaqueId: "executinguserid"}, ID: &provider.StorageSpaceId{OpaqueId: "spaceid"}})] = struct{}{}

		time.Sleep(500 * time.Millisecond)

		cs.Close()

		recs, err := sto.Read("userid")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(recs)).To(Equal(1))

		var storedIDs []string
		err = json.Unmarshal(recs[0].Value, &storedIDs)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(storedIDs)).To(Equal(1))
		_, exists := ids[storedIDs[0]]
		Expect(exists).To(BeTrue())
	})

	AfterEach(func() {
		close(bus)
	})
})
