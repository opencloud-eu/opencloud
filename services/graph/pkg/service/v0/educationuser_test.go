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
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type educationUserList struct {
	Value []*libregraph.EducationUser
}

var _ = Describe("EducationUsers", func() {
	var (
		svc                      service.Service
		ctx                      context.Context
		cfg                      *config.Config
		gatewayClient            *cs3mocks.GatewayAPIClient
		gatewaySelector          pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher          mocks.Publisher
		roleService              *mocks.RoleService
		identityEducationBackend *identitymocks.EducationBackend

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

		identityEducationBackend = &identitymocks.EducationBackend{}
		roleService = &mocks.RoleService{}

		rr = httptest.NewRecorder()
		ctx = context.Background()

		cfg = defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = "" // skip the startup checks, we don't use LDAP at all in this tests
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		svc, _ = service.NewService(
			service.Config(cfg),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(&eventsPublisher),
			service.WithIdentityEducationBackend(identityEducationBackend),
			service.WithRoleService(roleService),
		)
	})

	Describe("GetEducationUsers", func() {
		It("handles invalid requests", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/users?$invalid=true", nil)
			svc.GetEducationUsers(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("lists the users", func() {
			user := &libregraph.EducationUser{}
			user.SetId("user1")
			users := []*libregraph.EducationUser{user}

			identityEducationBackend.On("GetEducationUsers", mock.Anything, mock.Anything, mock.Anything).Return(users, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/users", nil)
			svc.GetEducationUsers(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := educationUserList{}
			err = json.Unmarshal(data, &res)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(res.Value)).To(Equal(1))
			Expect(res.Value[0].GetId()).To(Equal("user1"))
		})

		It("sorts", func() {
			user := &libregraph.EducationUser{}
			user.SetId("user1")
			user.SetMail("z@example.com")
			user.SetDisplayName("9")
			user.SetOnPremisesSamAccountName("9")
			user2 := &libregraph.EducationUser{}
			user2.SetId("user2")
			user2.SetMail("a@example.com")
			user2.SetDisplayName("1")
			user2.SetOnPremisesSamAccountName("1")
			users := []*libregraph.EducationUser{user, user2}

			identityEducationBackend.On("GetEducationUsers", mock.Anything, mock.Anything, mock.Anything).Return(users, nil)

			getUsers := func(path string) []*libregraph.EducationUser {
				r := httptest.NewRequest(http.MethodGet, path, nil)
				rec := httptest.NewRecorder()
				svc.GetEducationUsers(rec, r)

				Expect(rec.Code).To(Equal(http.StatusOK))
				data, err := io.ReadAll(rec.Body)
				Expect(err).ToNot(HaveOccurred())

				res := educationUserList{}
				err = json.Unmarshal(data, &res)
				Expect(err).ToNot(HaveOccurred())
				return res.Value
			}

			unsorted := getUsers("/graph/v1.0/education/users")
			Expect(len(unsorted)).To(Equal(2))
			Expect(unsorted[0].GetId()).To(Equal("user1"))
			Expect(unsorted[1].GetId()).To(Equal("user2"))

			byMail := getUsers("/graph/v1.0/education/users?$orderby=mail")
			Expect(len(byMail)).To(Equal(2))
			Expect(byMail[0].GetId()).To(Equal("user2"))
			Expect(byMail[1].GetId()).To(Equal("user1"))
			byMail = getUsers("/graph/v1.0/education/users?$orderby=mail%20asc")
			Expect(len(byMail)).To(Equal(2))
			Expect(byMail[0].GetId()).To(Equal("user2"))
			Expect(byMail[1].GetId()).To(Equal("user1"))
			byMail = getUsers("/graph/v1.0/education/users?$orderby=mail%20desc")
			Expect(len(byMail)).To(Equal(2))
			Expect(byMail[0].GetId()).To(Equal("user1"))
			Expect(byMail[1].GetId()).To(Equal("user2"))

			byDisplayName := getUsers("/graph/v1.0/education/users?$orderby=displayName")
			Expect(len(byDisplayName)).To(Equal(2))
			Expect(byDisplayName[0].GetId()).To(Equal("user2"))
			Expect(byDisplayName[1].GetId()).To(Equal("user1"))
			byDisplayName = getUsers("/graph/v1.0/education/users?$orderby=displayName%20asc")
			Expect(len(byDisplayName)).To(Equal(2))
			Expect(byDisplayName[0].GetId()).To(Equal("user2"))
			Expect(byDisplayName[1].GetId()).To(Equal("user1"))
			byDisplayName = getUsers("/graph/v1.0/education/users?$orderby=displayName%20desc")
			Expect(len(byDisplayName)).To(Equal(2))
			Expect(byDisplayName[0].GetId()).To(Equal("user1"))
			Expect(byDisplayName[1].GetId()).To(Equal("user2"))

			byOnPremisesSamAccountName := getUsers("/graph/v1.0/education/users?$orderby=onPremisesSamAccountName")
			Expect(len(byOnPremisesSamAccountName)).To(Equal(2))
			Expect(byOnPremisesSamAccountName[0].GetId()).To(Equal("user2"))
			Expect(byOnPremisesSamAccountName[1].GetId()).To(Equal("user1"))
			byOnPremisesSamAccountName = getUsers("/graph/v1.0/education/users?$orderby=onPremisesSamAccountName%20asc")
			Expect(len(byOnPremisesSamAccountName)).To(Equal(2))
			Expect(byOnPremisesSamAccountName[0].GetId()).To(Equal("user2"))
			Expect(byOnPremisesSamAccountName[1].GetId()).To(Equal("user1"))
			byOnPremisesSamAccountName = getUsers("/graph/v1.0/education/users?$orderby=onPremisesSamAccountName%20desc")
			Expect(len(byOnPremisesSamAccountName)).To(Equal(2))
			Expect(byOnPremisesSamAccountName[0].GetId()).To(Equal("user1"))
			Expect(byOnPremisesSamAccountName[1].GetId()).To(Equal("user2"))

			// Handles invalid sort field
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/users?$orderby=invalid", nil)
			svc.GetEducationUsers(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("GetEducationUser", func() {
		It("handles missing userids", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/users", nil)
			svc.GetEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("gets the user", func() {
			user := &libregraph.EducationUser{}
			user.SetId("user1")

			identityEducationBackend.On("GetEducationUser", mock.Anything, mock.Anything, mock.Anything).Return(user, nil)
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/users", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", *user.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.GetEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())
			responseUser := &libregraph.EducationUser{}
			err = json.Unmarshal(data, &responseUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(responseUser.GetId()).To(Equal("user1"))
		})

	})

	Describe("PostEducationUser", func() {
		var (
			user *libregraph.EducationUser

			assertHandleBadAttributes = func(user *libregraph.EducationUser) {
				userJson, err := json.Marshal(user)
				Expect(err).ToNot(HaveOccurred())

				r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/users", bytes.NewBuffer(userJson))
				svc.PostEducationUser(rr, r)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			}
		)

		BeforeEach(func() {
			identity := libregraph.ObjectIdentity{}
			identity.SetIssuer("issu.er")
			identity.SetIssuerAssignedId("our-user.1")

			user = &libregraph.EducationUser{}
			user.SetDisplayName("Display Name")
			user.SetOnPremisesSamAccountName("user")
			user.SetMail("user@example.com")
			user.SetAccountEnabled(true)
			user.SetIdentities([]libregraph.ObjectIdentity{identity})
			user.SetPrimaryRole("student")
		})

		It("handles invalid bodies", func() {
			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/users?$invalid=true", nil)
			svc.PostEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles missing display names", func() {
			user.DisplayName = nil
			assertHandleBadAttributes(user)

		})

		It("handles missing OnPremisesSamAccountName", func() {
			user.OnPremisesSamAccountName = nil
			assertHandleBadAttributes(user)

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

		It("creates a user", func() {
			roleService.On("AssignRoleToUser", mock.Anything, mock.Anything).Return(&settingssvc.AssignRoleToUserResponse{}, nil)
			identityEducationBackend.On("CreateEducationUser", mock.Anything, mock.Anything).Return(func(ctx context.Context, user libregraph.EducationUser) *libregraph.EducationUser {
				user.SetId("/users/user")
				return &user
			}, nil)
			userJson, err := json.Marshal(user)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/users", bytes.NewBuffer(userJson))
			r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
			svc.PostEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			createdUser := libregraph.EducationUser{}
			err = json.Unmarshal(data, &createdUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(createdUser.GetUserType()).To(Equal("Member"))
		})

		It("creates a guest user", func() {
			roleService.On("AssignRoleToUser", mock.Anything, mock.Anything).Return(&settingssvc.AssignRoleToUserResponse{}, nil)
			identityEducationBackend.On("CreateEducationUser", mock.Anything, mock.Anything).Return(func(ctx context.Context, user libregraph.EducationUser) *libregraph.EducationUser {
				user.SetId("/users/user")
				return &user
			}, nil)

			user.SetUserType("Guest")
			userJson, err := json.Marshal(user)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/users", bytes.NewBuffer(userJson))
			r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
			svc.PostEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			createdUser := libregraph.EducationUser{}
			err = json.Unmarshal(data, &createdUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(createdUser.GetUserType()).To(Equal("Guest"))
		})

		It("creates a member user", func() {
			roleService.On("AssignRoleToUser", mock.Anything, mock.Anything).Return(&settingssvc.AssignRoleToUserResponse{}, nil)
			identityEducationBackend.On("CreateEducationUser", mock.Anything, mock.Anything).Return(func(ctx context.Context, user libregraph.EducationUser) *libregraph.EducationUser {
				user.SetId("/users/user")
				return &user
			}, nil)

			user.SetUserType("Member")
			userJson, err := json.Marshal(user)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/users", bytes.NewBuffer(userJson))
			r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
			svc.PostEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			createdUser := libregraph.EducationUser{}
			err = json.Unmarshal(data, &createdUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(createdUser.GetUserType()).To(Equal("Member"))
		})

		It("creates a user without email", func() {
			roleService.On("AssignRoleToUser", mock.Anything, mock.Anything).Return(&settingssvc.AssignRoleToUserResponse{}, nil)
			identityEducationBackend.On("CreateEducationUser", mock.Anything, mock.Anything).Return(func(ctx context.Context, user libregraph.EducationUser) *libregraph.EducationUser {
				user.SetId("/users/user")
				return &user
			}, nil)
			user.Mail = nil
			userJson, err := json.Marshal(user)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/users", bytes.NewBuffer(userJson))
			r = r.WithContext(revactx.ContextSetUser(ctx, currentUser))
			svc.PostEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			createdUser := libregraph.EducationUser{}
			err = json.Unmarshal(data, &createdUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(createdUser.GetMail()).To(Equal(""))
		})
	})

	Describe("DeleteEducationUser", func() {
		It("handles missing userids", func() {
			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/users/{userid}", nil)
			svc.DeleteEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("prevents a user from deleting themselves", func() {
			lu := libregraph.EducationUser{}
			lu.SetId(currentUser.Id.OpaqueId)
			identityEducationBackend.On("GetEducationUser", mock.Anything, mock.Anything, mock.Anything).Return(&lu, nil)

			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/users/{userid}", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", currentUser.Id.OpaqueId)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusForbidden))
		})

		It("deletes a user from deleting themselves", func() {
			otheruser := &userv1beta1.User{
				Id: &userv1beta1.UserId{
					OpaqueId: "otheruser",
				},
			}

			lu := libregraph.EducationUser{}
			lu.SetId(otheruser.Id.OpaqueId)
			identityEducationBackend.On("GetEducationUser", mock.Anything, mock.Anything, mock.Anything).Return(&lu, nil)
			identityEducationBackend.On("DeleteEducationUser", mock.Anything, mock.Anything).Return(nil)
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

			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/users/{userid}", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", lu.GetId())
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusNoContent))
			gatewayClient.AssertNumberOfCalls(GinkgoT(), "DeleteStorageSpace", 2) // 2 calls for the home space. first trash, then purge
		})
	})

	Describe("PatchEducationUser", func() {
		var (
			user *libregraph.EducationUser
		)

		BeforeEach(func() {
			user = &libregraph.EducationUser{}
			user.SetDisplayName("Display Name")
			user.SetOnPremisesSamAccountName("user")
			user.SetMail("user@example.com")
			user.SetId("/users/user")
			user.SetAccountEnabled(true)

			identityEducationBackend.On("GetEducationUser", mock.Anything, mock.Anything, mock.Anything).Return(&user, nil)
		})

		It("handles missing userids", func() {
			r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/education/users/{userid}", nil)
			svc.PatchEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles invalid bodies", func() {
			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/users?$invalid=true", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", user.GetId())
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PatchEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles invalid email", func() {
			user.SetMail("invalid")
			data, err := json.Marshal(user)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/users?$invalid=true", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", user.GetId())
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PatchEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles invalid userType", func() {
			user.SetUserType("Clown")
			data, err := json.Marshal(user)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/users?$invalid=true", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", user.GetId())
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PatchEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("updates attributes", func() {
			identityEducationBackend.On("UpdateEducationUser", mock.Anything, user.GetId(), mock.Anything).Return(user, nil)

			user.SetDisplayName("New Display Name")
			user.SetAccountEnabled(false)
			data, err := json.Marshal(user)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/users?$invalid=true", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", user.GetId())
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PatchEducationUser(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err = io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			updatedUser := libregraph.EducationUser{}
			err = json.Unmarshal(data, &updatedUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedUser.GetDisplayName()).To(Equal("New Display Name"))
			Expect(updatedUser.GetAccountEnabled()).To(Equal(false))
		})
	})
})
