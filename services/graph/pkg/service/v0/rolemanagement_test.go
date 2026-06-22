package svc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

var _ = Describe("RoleManagement", func() {
	var (
		svc             service.Service
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher mocks.Publisher
		permSvc         mocks.Permissions
		cfg             *config.Config
		rr              *httptest.ResponseRecorder
		ctx             context.Context
	)

	BeforeEach(func() {
		rr = httptest.NewRecorder()
		ctx = context.Background()

		cfg = defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = ""
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector = pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)
		eventsPublisher = mocks.Publisher{}
		permSvc = mocks.Permissions{}

		var err error
		svc, err = service.NewService(
			service.Config(cfg),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(&eventsPublisher),
			service.PermissionService(&permSvc),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("GetRoleDefinitions", func() {
		It("returns all available roles in English when no Accept-Language is set", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/roleManagement/permissions/roleDefinitions", nil)
			r = r.WithContext(ctx)
			svc.ServeHTTP(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))

			var roles []libregraph.UnifiedRoleDefinition
			Expect(json.Unmarshal(rr.Body.Bytes(), &roles)).To(Succeed())
			Expect(roles).NotTo(BeEmpty())

			viewer := findRoleByID(roles, unifiedrole.UnifiedRoleViewerID)
			Expect(viewer).NotTo(BeNil())
			Expect(viewer.GetDisplayName()).To(Equal("Can view"))
		})

		It("returns translated roles when Accept-Language is German", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/roleManagement/permissions/roleDefinitions", nil)
			r.Header.Set("Accept-Language", "de")
			r = r.WithContext(ctx)
			svc.ServeHTTP(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Header().Get("Content-Language")).To(Equal("de"))

			var roles []libregraph.UnifiedRoleDefinition
			Expect(json.Unmarshal(rr.Body.Bytes(), &roles)).To(Succeed())

			viewer := findRoleByID(roles, unifiedrole.UnifiedRoleViewerID)
			Expect(viewer).NotTo(BeNil())
			Expect(viewer.GetDisplayName()).To(Equal("Kann anzeigen"))
			Expect(viewer.GetDescription()).To(Equal("Ansehen und herunterladen."))
		})

		It("does not mutate the global buildInRoles after a German request", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/roleManagement/permissions/roleDefinitions", nil)
			r.Header.Set("Accept-Language", "de")
			r = r.WithContext(ctx)
			svc.ServeHTTP(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			// A second request without a locale must still return English
			rr2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/roleManagement/permissions/roleDefinitions", nil)
			r2 = r2.WithContext(ctx)
			svc.ServeHTTP(rr2, r2)
			Expect(rr2.Code).To(Equal(http.StatusOK))

			var roles []libregraph.UnifiedRoleDefinition
			Expect(json.Unmarshal(rr2.Body.Bytes(), &roles)).To(Succeed())
			viewer := findRoleByID(roles, unifiedrole.UnifiedRoleViewerID)
			Expect(viewer).NotTo(BeNil())
			Expect(viewer.GetDisplayName()).To(Equal("Can view"))
		})
	})

	Describe("GetRoleDefinition", func() {
		It("returns a single role in English when no Accept-Language is set", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/roleManagement/permissions/roleDefinitions/"+unifiedrole.UnifiedRoleViewerID, nil)
			r = r.WithContext(ctx)
			svc.ServeHTTP(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			var role libregraph.UnifiedRoleDefinition
			Expect(json.Unmarshal(rr.Body.Bytes(), &role)).To(Succeed())
			Expect(role.GetDisplayName()).To(Equal("Can view"))
		})

		It("returns a single role translated when Accept-Language is German", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/roleManagement/permissions/roleDefinitions/"+unifiedrole.UnifiedRoleViewerID, nil)
			r.Header.Set("Accept-Language", "de")
			r = r.WithContext(ctx)
			svc.ServeHTTP(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Header().Get("Content-Language")).To(Equal("de"))
			var role libregraph.UnifiedRoleDefinition
			Expect(json.Unmarshal(rr.Body.Bytes(), &role)).To(Succeed())
			Expect(role.GetDisplayName()).To(Equal("Kann anzeigen"))
			Expect(role.GetDescription()).To(Equal("Ansehen und herunterladen."))
		})

		It("returns 404 for an unknown roleID", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1beta1/roleManagement/permissions/roleDefinitions/unknown-role-id", nil)
			r = r.WithContext(ctx)
			svc.ServeHTTP(rr, r)

			Expect(rr.Code).To(Equal(http.StatusNotFound))
		})
	})
})

// findRoleByID returns the first role with the given ID from the slice, or nil.
func findRoleByID(roles []libregraph.UnifiedRoleDefinition, id string) *libregraph.UnifiedRoleDefinition {
	for i := range roles {
		if roles[i].GetId() == id {
			return &roles[i]
		}
	}
	return nil
}
