package svc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settings "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type assignmentList struct {
	Value []*libregraph.AppRoleAssignment
}

var _ = Describe("AppRoleAssignments", func() {
	var (
		svc             service.Service
		ctx             context.Context
		cfg             *config.Config
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher mocks.Publisher
		roleService     *mocks.RoleService
		identityBackend *identitymocks.Backend

		rr *httptest.ResponseRecorder

		currentUser = &userv1beta1.User{
			Id: &userv1beta1.UserId{
				OpaqueId: "user",
			},
		}
	)

	BeforeEach(func() {
		eventsPublisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		identityBackend = &identitymocks.Backend{}
		roleService = &mocks.RoleService{}

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector = pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		rr = httptest.NewRecorder()
		ctx = context.Background()

		cfg = defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = "" // skip the startup checks, we don't use LDAP at all in this tests
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}
		cfg.Application.ID = "some-application-ID"

		svc, _ = service.NewService(
			service.Config(cfg),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(&eventsPublisher),
			service.WithIdentityBackend(identityBackend),
			service.WithRoleService(roleService),
		)
	})

	Describe("ListAppRoleAssignments", func() {
		It("lists the appRoleAssignments", func() {
			user := &libregraph.User{
				Id: libregraph.PtrString("user1"),
			}
			assignments := []*settingsmsg.UserRoleAssignment{
				{
					Id:          "some-appRoleAssignment-ID",
					AccountUuid: user.GetId(),
					RoleId:      "some-appRole-ID",
				},
			}
			roleService.On("ListRoleAssignments", mock.Anything, mock.Anything, mock.Anything).Return(&settings.ListRoleAssignmentsResponse{Assignments: assignments}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users/user1/appRoleAssignments", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", user.GetId())
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.ListAppRoleAssignments(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			responseList := assignmentList{}
			err = json.Unmarshal(data, &responseList)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(responseList.Value)).To(Equal(1))
			Expect(responseList.Value[0].GetId()).ToNot(BeEmpty())
			Expect(responseList.Value[0].GetAppRoleId()).To(Equal("some-appRole-ID"))
			Expect(responseList.Value[0].GetPrincipalId()).To(Equal(user.GetId()))
			Expect(responseList.Value[0].GetResourceId()).To(Equal(cfg.Application.ID))

		})

	})

	Describe("CreateAppRoleAssignment", func() {
		It("creates an appRoleAssignment", func() {
			user := &libregraph.User{
				Id: libregraph.PtrString("user1"),
			}
			userRoleAssignment := &settingsmsg.UserRoleAssignment{
				Id:          "some-appRoleAssignment-ID",
				AccountUuid: user.GetId(),
				RoleId:      "some-appRole-ID",
			}
			roleService.On("ListRoleAssignments", mock.Anything, mock.Anything, mock.Anything).Return(&settings.ListRoleAssignmentsResponse{
				Assignments: []*settingsmsg.UserRoleAssignment{
					userRoleAssignment,
				},
			}, nil)

			roleService.On("AssignRoleToUser", mock.Anything, mock.Anything, mock.Anything).Return(&settings.AssignRoleToUserResponse{Assignment: userRoleAssignment}, nil)

			ara := libregraph.NewAppRoleAssignmentWithDefaults()
			ara.SetAppRoleId("some-appRole-ID")
			ara.SetPrincipalId(user.GetId())
			ara.SetResourceId(cfg.Application.ID)

			araJson, err := json.Marshal(ara)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/users/user1/appRoleAssignments", bytes.NewBuffer(araJson))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", user.GetId())
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.CreateAppRoleAssignment(rr, r)

			Expect(rr.Code).To(Equal(http.StatusCreated))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			assignment := libregraph.AppRoleAssignment{}
			err = json.Unmarshal(data, &assignment)
			Expect(err).ToNot(HaveOccurred())
			Expect(assignment.GetId()).ToNot(BeEmpty())
			Expect(assignment.GetAppRoleId()).To(Equal("some-appRole-ID"))
			Expect(assignment.GetPrincipalId()).To(Equal("user1"))
			Expect(assignment.GetResourceId()).To(Equal(cfg.Application.ID))
		})

	})

	Describe("DeleteAppRoleAssignment", func() {
		It("deletes an appRoleAssignment", func() {
			user := &libregraph.User{
				Id: libregraph.PtrString("user1"),
			}

			assignments := []*settingsmsg.UserRoleAssignment{
				{
					Id:          "some-appRoleAssignment-ID",
					AccountUuid: user.GetId(),
					RoleId:      "some-appRole-ID",
				},
			}
			roleService.On("ListRoleAssignments", mock.Anything, mock.Anything, mock.Anything).Return(&settings.ListRoleAssignmentsResponse{Assignments: assignments}, nil)

			roleService.On("RemoveRoleFromUser", mock.Anything, mock.Anything, mock.Anything).Return(&empty.Empty{}, nil)

			ara := libregraph.NewAppRoleAssignmentWithDefaults()
			ara.SetAppRoleId("some-appRole-ID")
			ara.SetPrincipalId(user.GetId())
			ara.SetResourceId(cfg.Application.ID)

			araJson, err := json.Marshal(ara)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/users/user1/appRoleAssignments/some-appRoleAssignment-ID", bytes.NewBuffer(araJson))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", user.GetId())
			rctx.URLParams.Add("appRoleAssignmentID", "some-appRoleAssignment-ID")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteAppRoleAssignment(rr, r)

			Expect(rr.Code).To(Equal(http.StatusNoContent))

		})

	})
})
