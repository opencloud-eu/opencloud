package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/log"
	ehmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/eventhistory/v0"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	"github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0/mocks"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/config"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/service"
	httpsvc "github.com/opencloud-eu/opencloud/services/userlog/pkg/service/http"
)

var _ = Describe("Userlog http service", func() {
	var (
		cfg = &config.Config{
			Service: config.Service{
				Name: "userlog",
			},
			ServiceAccount:  config.ServiceAccount{},
			DefaultLanguage: "en",
		}

		ul  *service.UserlogService
		hs  *httpsvc.Service
		sto microstore.Store
		ehc mocks.EventHistoryService

		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	)

	BeforeEach(func() {
		var err error
		sto = store.Create()
		ehc = mocks.EventHistoryService{}

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector = pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(&gateway.AuthenticateResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}}, nil)
		gatewayClient.On("GetUser", mock.Anything, mock.Anything).Return(&user.GetUserResponse{User: &user.User{Id: &user.UserId{OpaqueId: "executinguserid"}, Username: "executinguser", DisplayName: "Executing User"}, Status: &rpc.Status{Code: rpc.Code_CODE_OK}}, nil)
		gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{StorageSpaces: []*provider.StorageSpace{
			{
				Id:        &provider.StorageSpaceId{OpaqueId: "spaceid"},
				SpaceType: "project",
				Root:      &provider.ResourceId{StorageId: "storageid", SpaceId: "spaceid", OpaqueId: "spaceid"},
			},
		}, Status: &rpc.Status{Code: rpc.Code_CODE_OK}}, nil)

		ul, err = service.NewUserlogService(
			service.Store(sto),
			service.Logger(log.NewLogger()),
			service.HistoryClient(&ehc),
			service.TraceProvider(trace.NewNoopTracerProvider()),
		)
		Expect(err).ToNot(HaveOccurred())

		hs, err = httpsvc.New(
			ul,
			httpsvc.Logger(log.NewLogger()),
			httpsvc.Config(cfg),
			httpsvc.GatewaySelector(gatewaySelector),
			httpsvc.RegisteredEvents([]events.Unmarshaller{
				events.SpaceDisabled{},
			}),
			httpsvc.TraceProvider(trace.NewNoopTracerProvider()),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns events for the requesting user", func() {
		ev := events.SpaceDisabled{
			Executant: &user.UserId{OpaqueId: "executinguserid"},
			ID:        &provider.StorageSpaceId{OpaqueId: "spaceid"},
			Timestamp: time.Now(),
		}
		b, err := json.Marshal(ev)
		Expect(err).ToNot(HaveOccurred())
		eventID := "eventid"
		Expect(ul.AddEventToUser("userid", events.Event{ID: eventID})).To(Succeed())

		ehc.On("GetEvents", mock.Anything, mock.Anything).Return(&ehsvc.GetEventsResponse{Events: []*ehmsg.Event{{
			Id:    eventID,
			Type:  reflect.TypeOf(ev).String(),
			Event: b,
		}}}, nil)

		req := httptest.NewRequest(http.MethodGet, "/ocs/v2.php/apps/notifications/api/v1/notifications", nil)
		req = req.WithContext(revactx.ContextSetUser(req.Context(), &user.User{Id: &user.UserId{OpaqueId: "userid"}}))
		rr := httptest.NewRecorder()

		hs.HandleGetEvents(rr, req)

		Expect(rr.Code).To(Equal(http.StatusOK))

		var resp httpsvc.GetEventResponseOC10
		Expect(json.Unmarshal(rr.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.OCS.Data).To(HaveLen(1))
		Expect(resp.OCS.Data[0].EventID).To(Equal(eventID))
	})

	It("responds with unauthorized when no user is in context", func() {
		req := httptest.NewRequest(http.MethodGet, "/ocs/v2.php/apps/notifications/api/v1/notifications", nil)
		rr := httptest.NewRecorder()

		hs.HandleGetEvents(rr, req)

		Expect(rr.Code).To(Equal(http.StatusUnauthorized))
	})
})
