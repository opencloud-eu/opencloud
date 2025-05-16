package svc_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/go-chi/chi/v5"
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
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type applicationList struct {
	Value []*libregraph.Application
}

var _ = Describe("Applications", func() {
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

	Describe("ListApplications", func() {
		It("lists the configured application with appRoles", func() {
			roleService.On("ListRoles", mock.Anything, mock.Anything, mock.Anything).Return(&settings.ListBundlesResponse{
				Bundles: []*settingsmsg.Bundle{
					{
						Id:          "some-appRole-ID",
						Type:        settingsmsg.Bundle_TYPE_ROLE,
						DisplayName: "A human readable name for a role",
					},
				},
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/applications", nil)
			svc.ListApplications(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			responseList := applicationList{}
			err = json.Unmarshal(data, &responseList)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(responseList.Value)).To(Equal(1))
			Expect(responseList.Value[0].Id).To(Equal(cfg.Application.ID))
			Expect(len(responseList.Value[0].GetAppRoles())).To(Equal(1))
			Expect(responseList.Value[0].GetAppRoles()[0].GetId()).To(Equal("some-appRole-ID"))
			Expect(responseList.Value[0].GetAppRoles()[0].GetDisplayName()).To(Equal("A human readable name for a role"))
		})
	})

	Describe("GetApplication", func() {
		It("gets the application with appRoles", func() {
			roleService.On("ListRoles", mock.Anything, mock.Anything, mock.Anything).Return(&settings.ListBundlesResponse{
				Bundles: []*settingsmsg.Bundle{
					{
						Id:          "some-appRole-ID",
						Type:        settingsmsg.Bundle_TYPE_ROLE,
						DisplayName: "A human readable name for a role",
					},
				},
			}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/applications/some-application-ID", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("applicationID", cfg.Application.ID)
			r = r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
			svc.GetApplication(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			application := libregraph.Application{}
			err = json.Unmarshal(data, &application)
			Expect(err).ToNot(HaveOccurred())
			Expect(application.Id).To(Equal(cfg.Application.ID))
			Expect(len(application.GetAppRoles())).To(Equal(1))
			Expect(application.GetAppRoles()[0].GetId()).To(Equal("some-appRole-ID"))
			Expect(application.GetAppRoles()[0].GetDisplayName()).To(Equal("A human readable name for a role"))
		})
	})

})
