package service_test

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	microevents "go-micro.dev/v4/events"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/log"
	ehmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/eventhistory/v0"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	"github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0/mocks"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	settingsmocks "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0/mocks"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/config"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/service"
)

var _ = Describe("UserlogService", func() {
	var (
		cfg = &config.Config{
			MaxConcurrency: 5,
		}

		ul  *service.UserlogService
		bus testBus
		sto microstore.Store

		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]

		ehc mocks.EventHistoryService
		vc  settingsmocks.ValueService
	)

	BeforeEach(func() {
		var err error
		sto = store.Create()
		bus = testBus(make(chan events.Event))

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

		ul, err = service.NewUserlogService(
			service.Config(cfg),
			service.Stream(bus),
			service.Store(sto),
			service.Logger(log.NewLogger()),
			service.Mux(chi.NewMux()),
			service.GatewaySelector(gatewaySelector),
			service.HistoryClient(&ehc),
			service.ValueClient(&vc),
			service.RegisteredEvents([]events.Unmarshaller{
				events.SpaceDisabled{},
			}),
			service.TraceProvider(trace.NewNoopTracerProvider()),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	It("it stores, returns and deletes a couple of events", func() {
		ids := make(map[string]struct{})
		ids[bus.publish(events.SpaceDisabled{Executant: &user.UserId{OpaqueId: "executinguserid"}})] = struct{}{}
		ids[bus.publish(events.SpaceDisabled{Executant: &user.UserId{OpaqueId: "executinguserid"}})] = struct{}{}
		ids[bus.publish(events.SpaceDisabled{Executant: &user.UserId{OpaqueId: "executinguserid"}})] = struct{}{}
		// ids[bus.Publish(events.SpaceMembershipExpired{SpaceOwner: &user.UserId{OpaqueId: "userid"}})] = struct{}{}
		// ids[bus.Publish(events.ShareCreated{Executant: &user.UserId{OpaqueId: "userid"}})] = struct{}{}

		time.Sleep(500 * time.Millisecond)

		var events []*ehmsg.Event
		for id := range ids {
			events = append(events, &ehmsg.Event{Id: id})
		}

		ehc.On("GetEvents", mock.Anything, mock.Anything).Return(&ehsvc.GetEventsResponse{Events: events}, nil)

		evs, err := ul.GetEvents(context.Background(), "userid")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(evs)).To(Equal(len(ids)))

		var evids []string
		for _, e := range evs {
			_, exists := ids[e.Id]
			Expect(exists).To(BeTrue())
			delete(ids, e.Id)
			evids = append(evids, e.Id)
		}

		Expect(len(ids)).To(Equal(0))
		err = ul.DeleteEvents("userid", evids)
		Expect(err).ToNot(HaveOccurred())

		evs, err = ul.GetEvents(context.Background(), "userid")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(evs)).To(Equal(0))
	})

	AfterEach(func() {
		close(bus)
	})
})

type testBus chan events.Event

func (tb testBus) Consume(_ string, _ ...microevents.ConsumeOption) (<-chan microevents.Event, error) {
	ch := make(chan microevents.Event)
	go func() {
		for ev := range tb {
			b, _ := json.Marshal(ev.Event)
			ch <- microevents.Event{
				Payload: b,
				Metadata: map[string]string{
					events.MetadatakeyEventID:   ev.ID,
					events.MetadatakeyEventType: ev.Type,
				},
			}
		}
	}()
	return ch, nil
}

func (tb testBus) Publish(_ string, _ any, _ ...microevents.PublishOption) error {
	return nil
}

func (tb testBus) publish(e any) string {
	ev := events.Event{
		ID:    uuid.New().String(),
		Type:  reflect.TypeOf(e).String(),
		Event: e,
	}

	tb <- ev
	return ev.ID
}
