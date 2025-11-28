package svc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	invitepb "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go/jetstream"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"go-micro.dev/v4/client"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/userstate"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settings "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	settingsmocks "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

type userList struct {
	Value []*libregraph.User
}

var _ = Describe("Users", func() {
	var (
		svc               service.Service
		ctx               context.Context
		cfg               *config.Config
		gatewayClient     *cs3mocks.GatewayAPIClient
		gatewaySelector   pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher   mocks.Publisher
		roleService       *mocks.RoleService
		valueService      *settingsmocks.ValueService
		permissionService *mocks.Permissions
		identityBackend   *identitymocks.Backend
		natsKeyValueMock  *mocks.KeyValue

		rr *httptest.ResponseRecorder

		currentUser = &userv1beta1.User{
			Id: &userv1beta1.UserId{
				OpaqueId: "user",
			},
		}
	)

	BeforeEach(func() {
		eventsPublisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector = pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		identityBackend = &identitymocks.Backend{}
		roleService = &mocks.RoleService{}
		natsKeyValueMock = &mocks.KeyValue{}
		valueService = &settingsmocks.ValueService{}
		permissionService = &mocks.Permissions{}

		rr = httptest.NewRecorder()
		ctx = context.Background()

		cfg = defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = "" // skip the startup checks, we don't use LDAP at all in this tests
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}
		cfg.Application.ID = "some-application-ID"

	})

	When("OCM is disabled", func() {
		BeforeEach(func() {
			var err error
			svc, err = service.NewService(
				service.Config(cfg),
				service.WithGatewaySelector(gatewaySelector),
				service.EventsPublisher(&eventsPublisher),
				service.WithIdentityBackend(identityBackend),
				service.WithRoleService(roleService),
				service.WithValueService(valueService),
				service.PermissionService(permissionService),
				service.WithNatsKeyValue(natsKeyValueMock),
			)
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("GetMe", func() {
			It("handles missing user", func() {
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me", nil)
				svc.GetMe(rr, r)

				Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			})

			It("gets the information", func() {
				valueService.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything, mock.Anything).
					Return(&settings.GetValueResponse{
						Value: &settingsmsg.ValueWithIdentifier{
							Value: &settingsmsg.Value{
								Value: &settingsmsg.Value_ListValue{
									ListValue: &settingsmsg.ListValue{
										Values: []*settingsmsg.ListOptionValue{
											{
												Option: &settingsmsg.ListOptionValue_StringValue{
													StringValue: "it",
												},
											},
										},
									},
								},
							},
						},
					}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me", nil)
				r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
				svc.GetMe(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
			})

			It("expands the memberOf", func() {
				user := &libregraph.User{
					Id: libregraph.PtrString("user1"),
					MemberOf: []libregraph.Group{
						{DisplayName: libregraph.PtrString("somegroup")},
					},
				}
				identityBackend.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)
				valueService.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything, mock.Anything).
					Return(&settings.GetValueResponse{
						Value: &settingsmsg.ValueWithIdentifier{
							Value: &settingsmsg.Value{
								Value: &settingsmsg.Value_ListValue{
									ListValue: &settingsmsg.ListValue{
										Values: []*settingsmsg.ListOptionValue{
											{
												Option: &settingsmsg.ListOptionValue_StringValue{
													StringValue: "it",
												},
											},
										},
									},
								},
							},
						},
					}, nil)
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me?$expand=memberOf", nil)
				r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
				svc.GetMe(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				responseUser := &libregraph.User{}
				err = json.Unmarshal(data, &responseUser)
				Expect(err).ToNot(HaveOccurred())

				Expect(responseUser.GetId()).To(Equal("user1"))
				Expect(responseUser.GetMemberOf()).To(HaveLen(1))
				Expect(responseUser.GetMemberOf()[0].GetDisplayName()).To(Equal("somegroup"))

			})

			It("expands the appRoleAssignments", func() {
				assignments := []*settingsmsg.UserRoleAssignment{
					{
						Id:          "some-appRoleAssignment-ID",
						AccountUuid: "user",
						RoleId:      "some-appRole-ID",
					},
				}
				roleService.On("ListRoleAssignments", mock.Anything, mock.Anything, mock.Anything).Return(&settings.ListRoleAssignmentsResponse{Assignments: assignments}, nil)
				valueService.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything, mock.Anything).
					Return(&settings.GetValueResponse{
						Value: &settingsmsg.ValueWithIdentifier{
							Value: &settingsmsg.Value{
								Value: &settingsmsg.Value_ListValue{
									ListValue: &settingsmsg.ListValue{
										Values: []*settingsmsg.ListOptionValue{
											{
												Option: &settingsmsg.ListOptionValue_StringValue{
													StringValue: "it",
												},
											},
										},
									},
								},
							},
						},
					}, nil)
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me?$expand=appRoleAssignments", nil)
				r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
				svc.GetMe(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))

				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				responseUser := &libregraph.User{}
				err = json.Unmarshal(data, &responseUser)
				Expect(err).ToNot(HaveOccurred())

				Expect(responseUser.GetId()).To(Equal("user"))
				Expect(responseUser.GetAppRoleAssignments()).To(HaveLen(1))
				Expect(responseUser.GetAppRoleAssignments()[0].GetId()).To(Equal("some-appRoleAssignment-ID"))
				Expect(responseUser.GetAppRoleAssignments()[0].GetAppRoleId()).To(Equal("some-appRole-ID"))
				Expect(responseUser.GetAppRoleAssignments()[0].GetPrincipalId()).To(Equal("user"))
				Expect(responseUser.GetAppRoleAssignments()[0].GetResourceId()).To(Equal("some-application-ID"))
			})
		})

		Describe("GetUsers", func() {
			It("handles invalid requests", func() {
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$invalid=true", nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("lists the users", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{
					Permission: &settingsmsg.Permission{
						Operation:  settingsmsg.Permission_OPERATION_UNKNOWN,
						Constraint: settingsmsg.Permission_CONSTRAINT_ALL,
					},
				}, nil)

				user := &libregraph.User{}
				user.SetId("user1")
				users := []*libregraph.User{user}

				identityBackend.On("GetUsers", mock.Anything, mock.Anything, mock.Anything).Return(users, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users", nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				res := userList{}
				err = json.Unmarshal(data, &res)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(res.Value)).To(Equal(1))
				Expect(res.Value[0].GetId()).To(Equal("user1"))
			})
			It("denies listing for unprivileged users", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{}, nil)
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users", nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(http.StatusForbidden))
			})
			It("denies using to short search terms for unprivileged users", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{}, nil)
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$search=a", nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(http.StatusForbidden))
			})
			It("denies using to short quoted search terms for unprivileged users", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{}, nil)
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$search=%22ab%22", nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(http.StatusForbidden))
			})
			It("denies filters other that 'userType eq ...' for unprivileged users", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{}, nil)
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$search=%22abc%22&$filter="+url.QueryEscape("memberOf/any(n:n/id eq 25cb7bc0-3168-4a0c-adbe-396f478ad494)"), nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(http.StatusForbidden))
			})

			It("only returns a restricted set of attributes for unprivileged users", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{}, nil)
				user := libregraph.NewUser("abc user", "username")
				user.SetId("user1")
				user.SetPasswordProfile(
					libregraph.PasswordProfile{
						Password: libregraph.PtrString("password"),
					},
				)
				user.SetUserType("usertype")
				users := []*libregraph.User{user}

				identityBackend.On("GetUsers", mock.Anything, mock.Anything, mock.Anything).Return(users, nil)
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$search=abc", nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				res := userList{}
				err = json.Unmarshal(data, &res)
				Expect(err).ToNot(HaveOccurred())
				userMap, err := res.Value[0].ToMap()
				Expect(err).ToNot(HaveOccurred())
				for k := range userMap {
					Expect(k).Should(BeElementOf([]string{"displayName", "onPremisesSamAccountName", "mail", "id", "userType"}))
				}
			})

			It("sorts", func() {
				user := &libregraph.User{}
				user.SetId("user1")
				user.SetMail("z@example.com")
				user.SetDisplayName("9")
				user.SetOnPremisesSamAccountName("9")
				user2 := &libregraph.User{}
				user2.SetId("user2")
				user2.SetMail("a@example.com")
				user2.SetDisplayName("1")
				user2.SetOnPremisesSamAccountName("1")
				users := []*libregraph.User{user, user2}

				identityBackend.On("GetUsers", mock.Anything, mock.Anything, mock.Anything).Return(users, nil)

				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{
					Permission: &settingsmsg.Permission{
						Operation:  settingsmsg.Permission_OPERATION_UNKNOWN,
						Constraint: settingsmsg.Permission_CONSTRAINT_ALL,
					},
				}, nil)

				getUsers := func(path string) []*libregraph.User {
					r := httptest.NewRequest(http.MethodGet, path, nil)
					rec := httptest.NewRecorder()
					svc.GetUsers(rec, r)

					Expect(rec.Code).To(Equal(http.StatusOK))
					data, err := io.ReadAll(rec.Body)
					Expect(err).ToNot(HaveOccurred())

					res := userList{}
					err = json.Unmarshal(data, &res)
					Expect(err).ToNot(HaveOccurred())
					return res.Value
				}

				unsorted := getUsers("/graph/v1.0/users")
				Expect(len(unsorted)).To(Equal(2))
				Expect(unsorted[0].GetId()).To(Equal("user1"))
				Expect(unsorted[1].GetId()).To(Equal("user2"))

				byMail := getUsers("/graph/v1.0/users?$orderby=mail")
				Expect(len(byMail)).To(Equal(2))
				Expect(byMail[0].GetId()).To(Equal("user2"))
				Expect(byMail[1].GetId()).To(Equal("user1"))
				byMail = getUsers("/graph/v1.0/users?$orderby=mail%20asc")
				Expect(len(byMail)).To(Equal(2))
				Expect(byMail[0].GetId()).To(Equal("user2"))
				Expect(byMail[1].GetId()).To(Equal("user1"))
				byMail = getUsers("/graph/v1.0/users?$orderby=mail%20desc")
				Expect(len(byMail)).To(Equal(2))
				Expect(byMail[0].GetId()).To(Equal("user1"))
				Expect(byMail[1].GetId()).To(Equal("user2"))

				byDisplayName := getUsers("/graph/v1.0/users?$orderby=displayName")
				Expect(len(byDisplayName)).To(Equal(2))
				Expect(byDisplayName[0].GetId()).To(Equal("user2"))
				Expect(byDisplayName[1].GetId()).To(Equal("user1"))
				byDisplayName = getUsers("/graph/v1.0/users?$orderby=displayName%20asc")
				Expect(len(byDisplayName)).To(Equal(2))
				Expect(byDisplayName[0].GetId()).To(Equal("user2"))
				Expect(byDisplayName[1].GetId()).To(Equal("user1"))
				byDisplayName = getUsers("/graph/v1.0/users?$orderby=displayName%20desc")
				Expect(len(byDisplayName)).To(Equal(2))
				Expect(byDisplayName[0].GetId()).To(Equal("user1"))
				Expect(byDisplayName[1].GetId()).To(Equal("user2"))

				byOnPremisesSamAccountName := getUsers("/graph/v1.0/users?$orderby=onPremisesSamAccountName")
				Expect(len(byOnPremisesSamAccountName)).To(Equal(2))
				Expect(byOnPremisesSamAccountName[0].GetId()).To(Equal("user2"))
				Expect(byOnPremisesSamAccountName[1].GetId()).To(Equal("user1"))
				byOnPremisesSamAccountName = getUsers("/graph/v1.0/users?$orderby=onPremisesSamAccountName%20asc")
				Expect(len(byOnPremisesSamAccountName)).To(Equal(2))
				Expect(byOnPremisesSamAccountName[0].GetId()).To(Equal("user2"))
				Expect(byOnPremisesSamAccountName[1].GetId()).To(Equal("user1"))
				byOnPremisesSamAccountName = getUsers("/graph/v1.0/users?$orderby=onPremisesSamAccountName%20desc")
				Expect(len(byOnPremisesSamAccountName)).To(Equal(2))
				Expect(byOnPremisesSamAccountName[0].GetId()).To(Equal("user1"))
				Expect(byOnPremisesSamAccountName[1].GetId()).To(Equal("user2"))

				// Handles invalid sort field
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$orderby=invalid", nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("expands the appRoleAssignments", func() {

				user := &libregraph.User{}
				user.SetId("user1")
				user.SetMail("z@example.com")
				user.SetDisplayName("9")
				user.SetOnPremisesSamAccountName("9")
				user2 := &libregraph.User{}
				user2.SetId("user2")
				user2.SetMail("a@example.com")
				user2.SetDisplayName("1")
				user2.SetOnPremisesSamAccountName("1")
				users := []*libregraph.User{user, user2}
				identityBackend.On("GetUsers", mock.Anything, mock.Anything, mock.Anything).Return(users, nil)

				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{
					Permission: &settingsmsg.Permission{
						Operation:  settingsmsg.Permission_OPERATION_UNKNOWN,
						Constraint: settingsmsg.Permission_CONSTRAINT_ALL,
					},
				}, nil)
				roleService.On("ListRoleAssignments", mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, in *settings.ListRoleAssignmentsRequest, opts ...client.CallOption) *settings.ListRoleAssignmentsResponse {
					return &settings.ListRoleAssignmentsResponse{Assignments: []*settingsmsg.UserRoleAssignment{
						{
							Id:          "some-appRoleAssignment-ID",
							AccountUuid: in.GetAccountUuid(),
							RoleId:      "some-appRole-ID",
						},
					}}
				}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$expand=appRoleAssignments", nil)
				r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))

				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				res := userList{}
				err = json.Unmarshal(data, &res)
				Expect(err).ToNot(HaveOccurred())

				responseUsers := res.Value
				Expect(len(responseUsers)).To(Equal(2))
				Expect(responseUsers[0].GetId()).To(Equal("user1"))
				Expect(responseUsers[0].GetAppRoleAssignments()).To(HaveLen(1))
				Expect(responseUsers[0].GetAppRoleAssignments()[0].GetId()).To(Equal("some-appRoleAssignment-ID"))
				Expect(responseUsers[0].GetAppRoleAssignments()[0].GetAppRoleId()).To(Equal("some-appRole-ID"))
				Expect(responseUsers[0].GetAppRoleAssignments()[0].GetPrincipalId()).To(Equal("user1"))
				Expect(responseUsers[0].GetAppRoleAssignments()[0].GetResourceId()).To(Equal("some-application-ID"))

				Expect(responseUsers[1].GetId()).To(Equal("user2"))
				Expect(responseUsers[1].GetAppRoleAssignments()).To(HaveLen(1))
				Expect(responseUsers[1].GetAppRoleAssignments()[0].GetId()).To(Equal("some-appRoleAssignment-ID"))
				Expect(responseUsers[1].GetAppRoleAssignments()[0].GetAppRoleId()).To(Equal("some-appRole-ID"))
				Expect(responseUsers[1].GetAppRoleAssignments()[0].GetPrincipalId()).To(Equal("user2"))
				Expect(responseUsers[1].GetAppRoleAssignments()[0].GetResourceId()).To(Equal("some-application-ID"))

			})
		})

		DescribeTable("GetUsers handles unsupported or invalid filters",
			func(filter string, status int) {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{
					Permission: &settingsmsg.Permission{
						Operation:  settingsmsg.Permission_OPERATION_UNKNOWN,
						Constraint: settingsmsg.Permission_CONSTRAINT_ALL,
					},
				}, nil)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$filter="+url.QueryEscape(filter), nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(status))
			},
			Entry("with invalid filter", "invalid", http.StatusBadRequest),
			Entry("with unsupported filter for user property", "mail eq 'unsupported'", http.StatusNotImplemented),
			// This error is caugh by godata's parser already
			Entry("with unsupported filter operation", "mail add 10", http.StatusBadRequest),
			Entry("with unsupported logical operation", "memberOf/any(n:n/id eq 1) or memberOf/any(n:n/id eq 2)", http.StatusNotImplemented),
			Entry("with unsupported lambda query ", `drives/any(n:n/id eq '1')`, http.StatusNotImplemented),
			Entry("with unsupported lambda token ", "memberOf/all(n:n/id eq 1)", http.StatusNotImplemented),
			Entry("with unsupported filter operation ", "memberOf/any(n:n/id ne 1)", http.StatusNotImplemented),
			Entry("with unsupported filter operand type", "memberOf/any(n:n/id eq 1)", http.StatusNotImplemented),
			Entry("with unsupported memberOf lambda filter property", "memberOf/any(n:n/name eq 'name')", http.StatusNotImplemented),
			Entry("with unsupported appRoleAssignments lambda filter property", "appRoleAssignments/any(n:n/id eq 'id')", http.StatusNotImplemented),
			Entry("with unsupported appRoleAssignments lambda filter property",
				"appRoleAssignments/any(n:n/id eq 'id') and appRoleAssignments/any(n:n/id eq 'id')", http.StatusNotImplemented),
			Entry("with unsupported appRoleAssignments lambda filter operation",
				"appRoleAssignments/all(n:n/appRoleId eq 'id') and appRoleAssignments/any(n:n/appRoleId eq 'id')", http.StatusNotImplemented),
			Entry("with unsupported appRoleAssignments lambda filter operation",
				"appRoleAssignments/any(n:n/appRoleId ne 'id') and appRoleAssignments/any(n:n/appRoleId eq 'id')", http.StatusNotImplemented),
			Entry("with unsupported userType in filter", "userType eq 'unsupported'", http.StatusNotImplemented),
			Entry("with unsupported property in eq filter", "unsupported eq 'unsupported'", http.StatusNotImplemented),
		)

		DescribeTable("With a valid filter",
			func(filter string, status int) {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{
					Permission: &settingsmsg.Permission{
						Operation:  settingsmsg.Permission_OPERATION_UNKNOWN,
						Constraint: settingsmsg.Permission_CONSTRAINT_ALL,
					},
				}, nil)

				user := &libregraph.User{}
				user.SetId("25cb7bc0-3168-4a0c-adbe-396f478ad494")
				users := []*libregraph.User{user}
				identityBackend.On("GetGroupMembers", mock.Anything, "25cb7bc0-3168-4a0c-adbe-396f478ad494", mock.Anything).Return(users, nil)
				identityBackend.On("GetGroupMembers", mock.Anything, "2713f1d5-6822-42bd-ad56-9f6c55a3a8fa", mock.Anything).Return([]*libregraph.User{}, nil)
				identityBackend.On("GetUsers", mock.Anything, mock.Anything).Return([]*libregraph.User{user}, nil)
				roleService.On("ListRoleAssignmentsFiltered", mock.Anything, mock.Anything, mock.Anything).
					Return(func(ctx context.Context, in *settings.ListRoleAssignmentsFilteredRequest, opts ...client.CallOption) *settings.ListRoleAssignmentsResponse {
						return &settings.ListRoleAssignmentsResponse{Assignments: []*settingsmsg.UserRoleAssignment{
							{
								Id:          "some-appRoleAssignment-ID",
								AccountUuid: user.GetId(),
								RoleId:      "some-appRole-ID",
							},
						}}
					}, nil)
				roleService.On("ListRoleAssignments", mock.Anything, mock.Anything, mock.Anything).
					Return(func(ctx context.Context, in *settings.ListRoleAssignmentsRequest, opts ...client.CallOption) *settings.ListRoleAssignmentsResponse {
						return &settings.ListRoleAssignmentsResponse{Assignments: []*settingsmsg.UserRoleAssignment{
							{
								Id:          "some-appRoleAssignment-ID",
								AccountUuid: user.GetId(),
								RoleId:      "some-appRole-ID",
							},
						}}
					}, nil)
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$filter="+url.QueryEscape(filter), nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(status))
			},
			Entry("with memberOf lambda filter with UUID", "memberOf/any(n:n/id eq 25cb7bc0-3168-4a0c-adbe-396f478ad494)", http.StatusOK),
			Entry("with memberOf lambda filter with UUID string", "memberOf/any(n:n/id eq '25cb7bc0-3168-4a0c-adbe-396f478ad494')", http.StatusOK),
			Entry("with appRoleAssignments lambda filter with appRoleId", "appRoleAssignments/any(n:n/appRoleId eq 'some-appRole-ID')", http.StatusOK),
			Entry("with two memberOf lambda filters combined with and",
				"memberOf/any(n:n/id eq 25cb7bc0-3168-4a0c-adbe-396f478ad494) and memberOf/any(n:n/id eq 2713f1d5-6822-42bd-ad56-9f6c55a3a8fa)",
				http.StatusOK),
			Entry("with two memberOf lambda filters combined with or",
				"memberOf/any(n:n/id eq 25cb7bc0-3168-4a0c-adbe-396f478ad494) or memberOf/any(n:n/id eq 2713f1d5-6822-42bd-ad56-9f6c55a3a8fa)",
				http.StatusOK),
			Entry("with supported appRoleAssignments lambda filter property",
				"appRoleAssignments/any(n:n/appRoleId eq 'some-appRoleAssignment-ID') and memberOf/any(n:n/id eq 2713f1d5-6822-42bd-ad56-9f6c55a3a8fa)",
				http.StatusOK),
		)

		Describe("GetUser", func() {
			It("handles missing userids", func() {
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users", nil)
				svc.GetUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("gets the user", func() {
				user := &libregraph.User{}
				user.SetId("user1")

				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{}, nil)
				identityBackend.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)
				valueService.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything, mock.Anything).
					Return(&settings.GetValueResponse{
						Value: &settingsmsg.ValueWithIdentifier{
							Value: &settingsmsg.Value{
								Value: &settingsmsg.Value_ListValue{
									ListValue: &settingsmsg.ListValue{
										Values: []*settingsmsg.ListOptionValue{
											{
												Option: &settingsmsg.ListOptionValue_StringValue{
													StringValue: "it",
												},
											},
										},
									},
								},
							},
						},
					}, nil)
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users", nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("userID", *user.Id)
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.GetUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
				data, err := io.ReadAll(rr.Body)

				Expect(err).ToNot(HaveOccurred())
				responseUser := &libregraph.User{}
				err = json.Unmarshal(data, &responseUser)
				Expect(err).ToNot(HaveOccurred())
				Expect(responseUser.GetId()).To(Equal("user1"))
				Expect(len(responseUser.GetDrives())).To(Equal(0))
			})

			It("includes the personal space if requested", func() {
				user := &libregraph.User{}
				user.SetId("user1")

				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{}, nil)
				identityBackend.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)
				gatewayClient.On("GetQuota", mock.Anything, mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
					Status:     status.NewOK(ctx),
					TotalBytes: 10,
				}, nil)
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Id:        &provider.StorageSpaceId{OpaqueId: "drive1"},
							Root:      &provider.ResourceId{SpaceId: "space", OpaqueId: "space"},
							SpaceType: "project",
						},
						{
							Id:        &provider.StorageSpaceId{OpaqueId: "personal"},
							Owner:     &userv1beta1.User{Id: &userv1beta1.UserId{OpaqueId: "user1"}},
							Root:      &provider.ResourceId{SpaceId: "personal", OpaqueId: "personal"},
							SpaceType: "personal",
						},
					},
				}, nil)
				valueService.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything, mock.Anything).
					Return(&settings.GetValueResponse{
						Value: &settingsmsg.ValueWithIdentifier{
							Value: &settingsmsg.Value{
								Value: &settingsmsg.Value_ListValue{
									ListValue: &settingsmsg.ListValue{
										Values: []*settingsmsg.ListOptionValue{
											{
												Option: &settingsmsg.ListOptionValue_StringValue{
													StringValue: "it",
												},
											},
										},
									},
								},
							},
						},
					}, nil)
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$expand=drive", nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("userID", *user.Id)
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.GetUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())
				responseUser := &libregraph.User{}
				err = json.Unmarshal(data, &responseUser)
				Expect(err).ToNot(HaveOccurred())
				Expect(responseUser.GetId()).To(Equal("user1"))
				Expect(*responseUser.GetDrive().Id).To(Equal("personal"))
			})

			It("includes the drives if requested", func() {
				user := &libregraph.User{}
				user.SetId("user1")

				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{}, nil)
				identityBackend.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)
				gatewayClient.On("GetQuota", mock.Anything, mock.Anything, mock.Anything).Return(&provider.GetQuotaResponse{
					Status:     status.NewOK(ctx),
					TotalBytes: 10,
				}, nil)
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Id:   &provider.StorageSpaceId{OpaqueId: "drive1"},
							Root: &provider.ResourceId{SpaceId: "space", OpaqueId: "space"},
						},
					},
				}, nil)
				valueService.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything, mock.Anything).
					Return(&settings.GetValueResponse{
						Value: &settingsmsg.ValueWithIdentifier{
							Value: &settingsmsg.Value{
								Value: &settingsmsg.Value_ListValue{
									ListValue: &settingsmsg.ListValue{
										Values: []*settingsmsg.ListOptionValue{
											{
												Option: &settingsmsg.ListOptionValue_StringValue{
													StringValue: "it",
												},
											},
										},
									},
								},
							},
						},
					}, nil)
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$expand=drives", nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("userID", *user.Id)
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.GetUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())
				responseUser := &libregraph.User{}
				err = json.Unmarshal(data, &responseUser)
				Expect(err).ToNot(HaveOccurred())
				Expect(responseUser.GetId()).To(Equal("user1"))
				Expect(len(responseUser.GetDrives())).To(Equal(1))
			})

			It("expands the appRoleAssignments", func() {
				user := &libregraph.User{}
				user.SetId("user1")

				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{}, nil)
				identityBackend.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)

				assignments := []*settingsmsg.UserRoleAssignment{
					{
						Id:          "some-appRoleAssignment-ID",
						AccountUuid: "user1",
						RoleId:      "some-appRole-ID",
					},
				}
				roleService.On("ListRoleAssignments", mock.Anything, mock.Anything, mock.Anything).Return(&settings.ListRoleAssignmentsResponse{Assignments: assignments}, nil)
				valueService.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything, mock.Anything).
					Return(&settings.GetValueResponse{
						Value: &settingsmsg.ValueWithIdentifier{
							Value: &settingsmsg.Value{
								Value: &settingsmsg.Value_ListValue{
									ListValue: &settingsmsg.ListValue{
										Values: []*settingsmsg.ListOptionValue{
											{
												Option: &settingsmsg.ListOptionValue_StringValue{
													StringValue: "it",
												},
											},
										},
									},
								},
							},
						},
					}, nil)
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users/user1?$expand=appRoleAssignments", nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("userID", user.GetId())
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.GetUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))

				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				responseUser := &libregraph.User{}
				err = json.Unmarshal(data, &responseUser)
				Expect(err).ToNot(HaveOccurred())

				Expect(responseUser.GetId()).To(Equal("user1"))
				Expect(responseUser.GetAppRoleAssignments()).To(HaveLen(1))
				Expect(responseUser.GetAppRoleAssignments()[0].GetId()).To(Equal("some-appRoleAssignment-ID"))
				Expect(responseUser.GetAppRoleAssignments()[0].GetAppRoleId()).To(Equal("some-appRole-ID"))
				Expect(responseUser.GetAppRoleAssignments()[0].GetPrincipalId()).To(Equal("user1"))
				Expect(responseUser.GetAppRoleAssignments()[0].GetResourceId()).To(Equal("some-application-ID"))
			})
		})

		Describe("PostUser", func() {
			var (
				user *libregraph.User

				assertHandleBadAttributes = func(user *libregraph.User) {
					userJson, err := json.Marshal(user)
					Expect(err).ToNot(HaveOccurred())

					r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/users", bytes.NewBuffer(userJson))
					svc.PostUser(rr, r)

					Expect(rr.Code).To(Equal(http.StatusBadRequest))
				}
			)

			BeforeEach(func() {
				user = libregraph.NewUser("Display Name", "user")
			})

			It("handles invalid bodies", func() {
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/users?$invalid=true", nil)
				svc.PostUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("handles missing display names", func() {
				user.SetDisplayName("")
				assertHandleBadAttributes(user)
			})

			It("handles missing OnPremisesSamAccountName", func() {
				user.SetOnPremisesSamAccountName("")
				assertHandleBadAttributes(user)
			})

			It("handles bad Mails", func() {
				user.SetMail("not-a-mail-address")
				assertHandleBadAttributes(user)
			})

			It("handles set Ids - they are read-only", func() {
				user.SetId("/users/user")
				assertHandleBadAttributes(user)
			})

			It("handles set memberOf - it's read-only", func() {
				group := libregraph.Group{}
				group.SetId("/groups/group")
				group.SetDisplayName("Group")
				user.SetMemberOf([]libregraph.Group{group})
				assertHandleBadAttributes(user)
			})

			It("handles set userType - it's read-only", func() {
				user.SetUserType("Member")
				assertHandleBadAttributes(user)
			})

			It("creates a user", func() {
				roleService.On("AssignRoleToUser", mock.Anything, mock.Anything).Return(&settings.AssignRoleToUserResponse{}, nil)
				identityBackend.On("CreateUser", mock.Anything, mock.Anything).Return(func(ctx context.Context, user libregraph.User) *libregraph.User {
					user.SetId("/users/user")
					return &user
				}, nil)
				userJson, err := json.Marshal(user)
				Expect(err).ToNot(HaveOccurred())

				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/users", bytes.NewBuffer(userJson))
				r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
				svc.PostUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusCreated))
				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				createdUser := libregraph.User{}
				err = json.Unmarshal(data, &createdUser)
				Expect(err).ToNot(HaveOccurred())
				Expect(createdUser.GetUserType()).To(Equal("Member"))
			})

			It("creates a member user", func() {
				roleService.On("AssignRoleToUser", mock.Anything, mock.Anything).Return(&settings.AssignRoleToUserResponse{}, nil)
				identityBackend.On("CreateUser", mock.Anything, mock.Anything).Return(func(ctx context.Context, user libregraph.User) *libregraph.User {
					user.SetId("/users/user")
					return &user
				}, nil)

				userJson, err := json.Marshal(user)
				Expect(err).ToNot(HaveOccurred())

				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/users", bytes.NewBuffer(userJson))
				r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
				svc.PostUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusCreated))
				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				createdUser := libregraph.User{}
				err = json.Unmarshal(data, &createdUser)
				Expect(err).ToNot(HaveOccurred())
				Expect(createdUser.GetUserType()).To(Equal("Member"))
			})

			Describe("Handling usernames with spaces", func() {
				var (
					newSvc = func(usernameMatch string) service.Service {
						localCfg := defaults.FullDefaultConfig()
						localCfg.Identity.LDAP.CACert = "" // skip the startup checks, we don't use LDAP at all in this tests
						localCfg.TokenManager.JWTSecret = "loremipsum"
						localCfg.Commons = &shared.Commons{}
						localCfg.GRPCClientTLS = &shared.GRPCClientTLS{}

						localCfg.API.UsernameMatch = usernameMatch

						localSvc, err := service.NewService(
							service.Config(localCfg),
							service.WithGatewaySelector(gatewaySelector),
							service.EventsPublisher(&eventsPublisher),
							service.WithIdentityBackend(identityBackend),
							service.WithRoleService(roleService),
						)
						Expect(err).ToNot(HaveOccurred())

						return localSvc
					}
				)

				BeforeEach(func() {
					user.SetOnPremisesSamAccountName("username with spaces")
				})

				It("rejects a username with spaces if match regex is default", func() {
					userJson, err := json.Marshal(user)
					Expect(err).ToNot(HaveOccurred())

					r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/me/users", bytes.NewBuffer(userJson))
					r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))

					newSvc("default").PostUser(rr, r)
					Expect(rr.Code).To(Equal(http.StatusBadRequest))
				})

				It("creates a user with spaces in username if regex is none", func() {
					roleService.On("AssignRoleToUser", mock.Anything, mock.Anything).Return(&settings.AssignRoleToUserResponse{}, nil)
					identityBackend.On("CreateUser", mock.Anything, mock.Anything).Return(func(ctx context.Context, user libregraph.User) *libregraph.User {
						user.SetId("/users/user")
						return &user
					}, nil)
					userJson, err := json.Marshal(user)
					Expect(err).ToNot(HaveOccurred())

					r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/me/users", bytes.NewBuffer(userJson))
					r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
					newSvc("none").PostUser(rr, r)

					Expect(rr.Code).To(Equal(http.StatusCreated))
				})
			})

		})

		Describe("DeleteUser", func() {
			It("handles missing userids", func() {
				r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/users/{userid}", nil)
				svc.DeleteUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("prevents a user from deleting themselves", func() {
				lu := libregraph.User{}
				lu.SetId(currentUser.Id.OpaqueId)
				identityBackend.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(&lu, nil)

				natsKeyValueMock.EXPECT().Get(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, key string) (jetstream.KeyValueEntry, error) {
					byteRep, _ := json.Marshal(userstate.UserState{
						UserId:          lu.GetId(),
						State:           userstate.UserStateSoftDeleted,
						TimeStamp:       time.Now().UTC(),
						RetentionPeriod: 5 * time.Minute,
						Reason:          "unit test",
					})

					kve := &mocks.KeyValueEntry{}
					kve.On("Value").Return(byteRep)

					return kve, nil
				}).Once()

				natsKeyValueMock.EXPECT().Put(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, key string, val []byte) (uint64, error) {
					return 1, nil
				}).Once()

				r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/users/{userid}", nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("userID", currentUser.Id.OpaqueId)
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.DeleteUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusForbidden))
			})

			It("deletes a user from deleting themselves", func() {
				otheruser := &userv1beta1.User{
					Id: &userv1beta1.UserId{
						OpaqueId: "otheruser",
					},
				}

				lu := libregraph.User{}
				lu.SetId(otheruser.Id.OpaqueId)
				identityBackend.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(&lu, nil)
				identityBackend.On("DeleteUser", mock.Anything, mock.Anything).Return(nil)
				gatewayClient.On("DeleteStorageSpace", mock.Anything, mock.Anything).Return(&provider.DeleteStorageSpaceResponse{
					Status: status.NewOK(ctx),
				}, nil)
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Opaque:    &typesv1beta1.Opaque{},
							Id:        &provider.StorageSpaceId{OpaqueId: "drive1"},
							Root:      &provider.ResourceId{SpaceId: "space", OpaqueId: "space"},
							SpaceType: "personal",
							Owner:     otheruser,
						},
					},
				}, nil)

				natsKeyValueMock.EXPECT().Get(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, key string) (jetstream.KeyValueEntry, error) {
					byteRep, _ := json.Marshal(userstate.UserState{
						UserId:          lu.GetId(),
						State:           userstate.UserStateSoftDeleted,
						TimeStamp:       time.Now().UTC(),
						RetentionPeriod: 5 * time.Minute,
						Reason:          "unit test",
					})

					kve := &mocks.KeyValueEntry{}
					kve.On("Value").Return(byteRep)

					return kve, nil
				}).Once()

				natsKeyValueMock.EXPECT().Put(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, key string, val []byte) (uint64, error) {
					return 1, nil
				}).Once()

				r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/users/{userid}", nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("userID", lu.GetId())
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.DeleteUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusNoContent))
				gatewayClient.AssertNumberOfCalls(GinkgoT(), "DeleteStorageSpace", 2) // 2 calls for the home space. first trash, then purge
			})
		})

		Describe("SoftDeleteUser", func() {
			var (
				user *libregraph.User
				//userUpdate   *libregraph.UserUpdate
				expectedUser *libregraph.User
			)

			BeforeEach(func() {
				cfg.UserSoftDeleteRetentionTime = 30 * 24 * time.Hour // 30 days
				user = libregraph.NewUser("Display Name", "user")
				user.SetMail("user@example.com")
				user.SetId("/users/user")

				//userUpdate = libregraph.NewUserUpdate()

				expectedUser = libregraph.NewUser("Display Name", "user")
				expectedUser.SetMail(user.GetMail())
				expectedUser.SetId(user.GetId())

				identityBackend.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)
			})

			It("soft deletes a user", func() {
				otheruser := &userv1beta1.User{
					Id: &userv1beta1.UserId{
						OpaqueId: "otheruser",
					},
				}

				lu := libregraph.User{}
				lu.SetId(otheruser.Id.OpaqueId)
				identityBackend.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(&lu, nil)
				//identityBackend.On("DeleteUser", mock.Anything, mock.Anything).Return(nil)
				identityBackend.On("UpdateUser", mock.Anything, mock.Anything, mock.Anything).Return(&lu, nil)
				gatewayClient.On("DeleteStorageSpace", mock.Anything, mock.Anything).Return(&provider.DeleteStorageSpaceResponse{
					Status: status.NewOK(ctx),
				}, nil)
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Opaque:    &typesv1beta1.Opaque{},
							Id:        &provider.StorageSpaceId{OpaqueId: "drive1"},
							Root:      &provider.ResourceId{SpaceId: "space", OpaqueId: "space"},
							SpaceType: "personal",
							Owner:     otheruser,
						},
					},
				}, nil)

				natsKeyValueMock.EXPECT().Get(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, key string) (jetstream.KeyValueEntry, error) {
					byteRep, _ := json.Marshal(userstate.UserState{
						UserId:          lu.GetId(),
						State:           userstate.UserStateSoftDeleted,
						TimeStamp:       time.Now().UTC(),
						RetentionPeriod: 5 * time.Minute,
						Reason:          "unit test",
					})

					kve := &mocks.KeyValueEntry{}
					kve.On("Value").Return(byteRep)

					return kve, nil
				}).Once()

				natsKeyValueMock.EXPECT().Put(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, key string, val []byte) (uint64, error) {
					return 1, nil
				}).Once()
				r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/users/{userid}", nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("userID", lu.GetId())
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.DeleteUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusNoContent))
				gatewayClient.AssertNumberOfCalls(GinkgoT(), "DeleteStorageSpace", 0) // 2 calls for the home space. first trash, then purge

			})

			It("hard deletes the user", func() {
				otheruser := &userv1beta1.User{
					Id: &userv1beta1.UserId{
						OpaqueId: "otheruser",
					},
				}

				lu := libregraph.User{}
				lu.SetId(otheruser.Id.OpaqueId)
				identityBackend.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(&lu, nil)
				identityBackend.On("DeleteUser", mock.Anything, mock.Anything).Return(nil)
				identityBackend.On("UpdateUser", mock.Anything, mock.Anything, mock.Anything).Return(&lu, nil)
				gatewayClient.On("DeleteStorageSpace", mock.Anything, mock.Anything).Return(&provider.DeleteStorageSpaceResponse{
					Status: status.NewOK(ctx),
				}, nil)
				gatewayClient.On("ListStorageSpaces", mock.Anything, mock.Anything, mock.Anything).Return(&provider.ListStorageSpacesResponse{
					Status: status.NewOK(ctx),
					StorageSpaces: []*provider.StorageSpace{
						{
							Opaque:    &typesv1beta1.Opaque{},
							Id:        &provider.StorageSpaceId{OpaqueId: "drive1"},
							Root:      &provider.ResourceId{SpaceId: "space", OpaqueId: "space"},
							SpaceType: "personal",
							Owner:     otheruser,
						},
					},
				}, nil)

				natsKeyValueMock.EXPECT().Get(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, key string) (jetstream.KeyValueEntry, error) {
					byteRep, _ := json.Marshal(userstate.UserState{
						UserId:          lu.GetId(),
						State:           userstate.UserStateSoftDeleted,
						TimeStamp:       time.Now().UTC(),
						RetentionPeriod: 5 * time.Minute,
						Reason:          "unit test",
					})

					kve := &mocks.KeyValueEntry{}
					kve.On("Value").Return(byteRep)

					return kve, nil
				}).Once()

				natsKeyValueMock.EXPECT().Put(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, key string, val []byte) (uint64, error) {
					return 1, nil
				}).Once()

				r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/users/{userid}", nil)
				r.Header.Set("Prefer", "purge") // this header is used to indicate a hard delete
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("userID", lu.GetId())
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.DeleteUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusNoContent))
				gatewayClient.AssertNumberOfCalls(GinkgoT(), "DeleteStorageSpace", 0) // 0 calls for the home space. since we are "just" soft deleting
			})
		})

		Describe("PatchUser", func() {
			var (
				user                      *libregraph.User
				userUpdate                *libregraph.UserUpdate
				expectedUser              *libregraph.User
				assertHandleBadAttributes = func(userid string, update *libregraph.UserUpdate) {
					userJson, err := json.Marshal(update)
					Expect(err).ToNot(HaveOccurred())

					r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/users/{userid}", bytes.NewBuffer(userJson))
					rctx := chi.NewRouteContext()
					rctx.URLParams.Add("userID", userid)
					r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
					svc.PatchUser(rr, r)

					Expect(rr.Code).To(Equal(http.StatusBadRequest))
				}
			)

			BeforeEach(func() {
				user = libregraph.NewUser("Display Name", "user")
				user.SetMail("user@example.com")
				user.SetId("/users/user")

				userUpdate = libregraph.NewUserUpdate()

				expectedUser = libregraph.NewUser("Display Name", "user")
				expectedUser.SetMail(user.GetMail())
				expectedUser.SetId(user.GetId())

				identityBackend.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)
			})

			It("handles missing userids", func() {
				r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/users/{userid}", nil)
				svc.PatchUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("handles invalid bodies", func() {
				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/users?$invalid=true", nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("userID", user.GetId())
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.PatchUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("handles set Ids - they are read-only", func() {
				userUpdate.SetId("/users/user")
				assertHandleBadAttributes(user.GetId(), userUpdate)
			})

			It("handles set memberOf - it's read-only", func() {
				group := libregraph.Group{}
				group.SetId("/groups/group")
				group.SetDisplayName("Group")
				userUpdate.SetMemberOf([]libregraph.Group{group})
				assertHandleBadAttributes(user.GetId(), userUpdate)
			})

			It("handles set userType - it's read-only", func() {
				userUpdate.SetUserType("Member")
				assertHandleBadAttributes(user.GetId(), userUpdate)
			})

			It("handles invalid email", func() {
				userUpdate.SetMail("invalid")
				data, err := json.Marshal(userUpdate)
				Expect(err).ToNot(HaveOccurred())

				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/users?$invalid=true", bytes.NewBuffer(data))
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("userID", userUpdate.GetId())
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.PatchUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("updates attributes", func() {
				userUpdate.SetMail("mail@mail.test")
				userUpdate.SetDisplayName("New Display Name")

				expectedUser.SetMail("mail@mail.test")
				expectedUser.SetDisplayName("New Display Name")
				identityBackend.On("UpdateUser", mock.Anything, user.GetId(), mock.Anything).Return(expectedUser, nil)

				data, err := json.Marshal(userUpdate)
				Expect(err).ToNot(HaveOccurred())

				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/users", bytes.NewBuffer(data))
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("userID", user.GetId())
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.PatchUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
				data, err = io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				unmarshaledUser := libregraph.User{}
				err = json.Unmarshal(data, &unmarshaledUser)
				Expect(err).ToNot(HaveOccurred())
				Expect(unmarshaledUser.GetMail()).To(Equal("mail@mail.test"))
				Expect(unmarshaledUser.GetDisplayName()).To(Equal("New Display Name"))
			})
		})
	})
	When("OCM is enabled", func() {
		BeforeEach(func() {
			cfg.IncludeOCMSharees = true

			var err error
			svc, err = service.NewService(
				service.Config(cfg),
				service.WithGatewaySelector(gatewaySelector),
				service.EventsPublisher(&eventsPublisher),
				service.WithIdentityBackend(identityBackend),
				service.WithRoleService(roleService),
				service.WithValueService(valueService),
				service.PermissionService(permissionService),
			)
			Expect(err).ToNot(HaveOccurred())
		})
		Describe("GetUsers", func() {
			It("does not list the federated users without a filter", func() {
				gatewayClient.AssertNotCalled(GinkgoT(), "FindAcceptedUsers, mock.Anything, mock.Anything")
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{}, nil)

				identityBackend.On("GetUsers", mock.Anything, mock.Anything, mock.Anything).Return(
					[]*libregraph.User{}, nil,
				)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$search=user", nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				res := userList{}
				err = json.Unmarshal(data, &res)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(res.Value)).To(Equal(0))
			})
			It("does list federated users when the right filter is used", func() {
				gatewayClient.On("FindAcceptedUsers", mock.Anything, mock.Anything).Return(&invitepb.FindAcceptedUsersResponse{
					Status: status.NewOK(ctx),
					AcceptedUsers: []*userv1beta1.User{
						{
							Username: "federated",
							Id: &userv1beta1.UserId{
								OpaqueId: "federated",
								Idp:      "provider",
								Type:     userv1beta1.UserType_USER_TYPE_FEDERATED,
							},
						},
					},
				}, nil)
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{}, nil)

				identityBackend.On("GetUsers", mock.Anything, mock.Anything, mock.Anything).Return(
					[]*libregraph.User{}, nil,
				)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$search=federated&$filter="+url.QueryEscape("userType eq 'Federated'"), nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				res := userList{}
				err = json.Unmarshal(data, &res)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(res.Value)).To(Equal(1))
				Expect(res.Value[0].GetId()).To(Equal("federated@provider"))
				Expect(res.Value[0].GetUserType()).To(Equal("Federated"))
			})
			It("does not list federated users when filtering for 'Member' users", func() {
				permissionService.On("GetPermissionByID", mock.Anything, mock.Anything).Return(&settings.GetPermissionByIDResponse{}, nil)

				user := &libregraph.User{}
				user.SetId("user1")
				user.SetUserType("Member")
				users := []*libregraph.User{user}

				identityBackend.On("GetUsers", mock.Anything, mock.Anything, mock.Anything).Return(
					users, nil,
				)

				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/users?$search=user&$filter="+url.QueryEscape("userType eq 'Member'"), nil)
				svc.GetUsers(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
				data, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				res := userList{}
				err = json.Unmarshal(data, &res)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(res.Value)).To(Equal(1))
				Expect(res.Value[0].GetId()).To(Equal("user1"))
				Expect(res.Value[0].GetUserType()).To(Equal("Member"))
			})
		})
	})
})
