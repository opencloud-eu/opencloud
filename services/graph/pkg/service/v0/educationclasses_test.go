package svc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

var _ = Describe("EducationClass", func() {
	var (
		svc                      service.Service
		ctx                      context.Context
		cfg                      *config.Config
		gatewayClient            *cs3mocks.GatewayAPIClient
		gatewaySelector          pool.Selectable[gateway.GatewayAPIClient]
		eventsPublisher          mocks.Publisher
		identityBackend          *identitymocks.Backend
		identityEducationBackend *identitymocks.EducationBackend

		rr *httptest.ResponseRecorder

		newClass    *libregraph.EducationClass
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
		identityBackend = &identitymocks.Backend{}
		newClass = libregraph.NewEducationClass("math", "course")
		newClass.SetMembersodataBind([]string{"/users/user1"})
		newClass.SetId("math")

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
			service.WithIdentityBackend(identityBackend),
			service.WithIdentityEducationBackend(identityEducationBackend),
		)
	})

	Describe("GetEducationClasses", func() {
		It("handles invalid ODATA parameters", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/classes?§foo=bar", nil)
			svc.GetEducationClasses(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles invalid sorting queries", func() {
			identityEducationBackend.On("GetEducationClasses", ctx, mock.Anything).Return([]*libregraph.EducationClass{newClass}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/classes?$orderby=invalid", nil)
			svc.GetEducationClasses(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			odataerr := libregraph.OdataError{}
			err = json.Unmarshal(data, &odataerr)
			Expect(err).ToNot(HaveOccurred())
			Expect(odataerr.Error.Code).To(Equal("invalidRequest"))
		})

		It("handles unknown backend errors", func() {
			identityEducationBackend.On("GetEducationClasses", ctx, mock.Anything).Return(nil, errors.New("failed"))

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/classes", nil)
			svc.GetEducationClasses(rr, r)
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			odataerr := libregraph.OdataError{}
			err = json.Unmarshal(data, &odataerr)
			Expect(err).ToNot(HaveOccurred())
			Expect(odataerr.Error.Code).To(Equal("generalException"))
		})

		It("handles backend errors", func() {
			identityEducationBackend.On("GetEducationClasses", ctx, mock.Anything).Return(nil, errorcode.New(errorcode.AccessDenied, "access denied"))

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/classes", nil)
			svc.GetEducationClasses(rr, r)

			Expect(rr.Code).To(Equal(http.StatusForbidden))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			odataerr := libregraph.OdataError{}
			err = json.Unmarshal(data, &odataerr)
			Expect(err).ToNot(HaveOccurred())
			Expect(odataerr.Error.Code).To(Equal("accessDenied"))
		})

		It("renders an empty list of classes", func() {
			identityEducationBackend.On("GetEducationClasses", ctx, mock.Anything).Return([]*libregraph.EducationClass{}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/classes", nil)
			svc.GetEducationClasses(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := service.ListResponse{}
			err = json.Unmarshal(data, &res)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Value).To(Equal([]interface{}{}))
		})

		It("renders a list of classes", func() {
			identityEducationBackend.On("GetEducationClasses", ctx, mock.Anything).Return([]*libregraph.EducationClass{newClass}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/classes", nil)
			svc.GetEducationClasses(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := groupList{}
			err = json.Unmarshal(data, &res)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(res.Value)).To(Equal(1))
			Expect(res.Value[0].GetId()).To(Equal("math"))
		})
	})

	Describe("GetEducationClass", func() {
		It("handles missing or empty class id", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/classes", nil)
			svc.GetEducationClass(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))

			r = httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/classes", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", "")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			svc.GetEducationClass(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})
		Context("with an existing class", func() {
			BeforeEach(func() {
				identityEducationBackend.On("GetEducationClass", mock.Anything, mock.Anything, mock.Anything).Return(newClass, nil)
			})

			It("gets the class", func() {
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/classes/"+*newClass.Id, nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("classID", *newClass.Id)
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))

				svc.GetEducationClass(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("PostEducationClass", func() {
		It("handles invalid body", func() {
			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/classes/", bytes.NewBufferString("{invalid"))

			svc.PostEducationClass(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles missing display name", func() {
			newClass = libregraph.NewEducationClassWithDefaults()
			newClass.SetId("disallowed")
			newClass.SetMembersodataBind([]string{"/non-users/user"})
			newClassJson, err := json.Marshal(newClass)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/classes/", bytes.NewBuffer(newClassJson))

			svc.PostEducationClass(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("disallows group create ids", func() {
			newClass = libregraph.NewEducationClassWithDefaults()
			newClass.SetId("disallowed")
			newClass.SetDisplayName("New Class")
			newClass.SetMembersodataBind([]string{"/non-users/user"})
			newClassJson, err := json.Marshal(newClass)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/classes/", bytes.NewBuffer(newClassJson))

			svc.PostEducationClass(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("creates the Class", func() {
			newClass = libregraph.NewEducationClassWithDefaults()
			newClass.SetDisplayName("New Class")
			newClassJson, err := json.Marshal(newClass)
			Expect(err).ToNot(HaveOccurred())

			identityEducationBackend.On("CreateEducationClass", mock.Anything, mock.Anything).Return(newClass, nil)

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/class/", bytes.NewBuffer(newClassJson))

			svc.PostEducationClass(rr, r)

			Expect(rr.Code).To(Equal(http.StatusCreated))
		})
	})

	Describe("PatchEducationClass", func() {
		It("handles invalid body", func() {
			r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/education/classes/", bytes.NewBufferString("{invalid"))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PatchEducationClass(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles missing or empty group id", func() {
			r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/education/classes", nil)
			svc.PatchEducationClass(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))

			r = httptest.NewRequest(http.MethodPatch, "/graph/v1.0/education/classes", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", "")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PatchEducationClass(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		Context("with an existing group", func() {
			BeforeEach(func() {
				identityEducationBackend.On("GetEducationClass", mock.Anything, mock.Anything, mock.Anything).Return(newClass, nil)
				identityEducationBackend.On("UpdateEducationClass", mock.Anything, mock.Anything, mock.Anything).Return(newClass, nil)
			})

			It("fails when the number of users is exceeded - spec says 20 max", func() {
				updatedClass := libregraph.NewEducationClassWithDefaults()
				updatedClass.SetDisplayName("class updated")
				updatedClass.SetMembersodataBind([]string{
					"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18",
					"19", "20", "21",
				})
				updatedClassJson, err := json.Marshal(updatedClass)
				Expect(err).ToNot(HaveOccurred())

				r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/education/classes", bytes.NewBuffer(updatedClassJson))
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("classID", *newClass.Id)
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.PatchEducationClass(rr, r)

				resp, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				Expect(string(resp)).To(ContainSubstring("Request is limited to 20"))
				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("succeeds when the number of users is over 20 but the limit is raised to 21", func() {
				updatedClass := libregraph.NewEducationClassWithDefaults()
				updatedClass.SetDisplayName("group1 updated")
				updatedClass.SetMembersodataBind([]string{
					"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18",
					"19", "20", "21",
				})
				updatedClassJson, err := json.Marshal(updatedClass)
				Expect(err).ToNot(HaveOccurred())

				cfg.API.GroupMembersPatchLimit = 21
				svc, _ = service.NewService(
					service.Config(cfg),
					service.WithGatewaySelector(gatewaySelector),
					service.EventsPublisher(&eventsPublisher),
					service.WithIdentityBackend(identityBackend),
					service.WithIdentityEducationBackend(identityEducationBackend),
				)

				r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/education/classes", bytes.NewBuffer(updatedClassJson))
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("classID", *newClass.Id)
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.PatchEducationClass(rr, r)

				resp, err := io.ReadAll(rr.Body)
				Expect(err).ToNot(HaveOccurred())

				Expect(string(resp)).To(ContainSubstring("Error parsing member@odata.bind values"))
				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("fails on invalid user refs", func() {
				updatedClass := libregraph.NewEducationClassWithDefaults()
				updatedClass.SetDisplayName("class updated")
				updatedClass.SetMembersodataBind([]string{"invalid"})
				updatedClassJson, err := json.Marshal(updatedClass)
				Expect(err).ToNot(HaveOccurred())

				r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/education/classes", bytes.NewBuffer(updatedClassJson))
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("classID", *newClass.Id)
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.PatchEducationClass(rr, r)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("fails when the adding non-users users", func() {
				updatedClass := libregraph.NewEducationClassWithDefaults()
				updatedClass.SetDisplayName("group1 updated")
				updatedClass.SetMembersodataBind([]string{"/non-users/user1"})
				updatedClassJson, err := json.Marshal(updatedClass)
				Expect(err).ToNot(HaveOccurred())

				r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/education/classes", bytes.NewBuffer(updatedClassJson))
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("classID", *newClass.Id)
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.PatchEducationClass(rr, r)

				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})

			It("adds members to the class", func() {
				identityBackend.On("AddMembersToGroup", mock.Anything, mock.Anything, mock.Anything).Return(nil)

				updatedClass := libregraph.NewEducationClassWithDefaults()
				updatedClass.SetDisplayName("Class updated")
				updatedClass.SetMembersodataBind([]string{"/users/user1"})
				updatedClassJson, err := json.Marshal(updatedClass)
				Expect(err).ToNot(HaveOccurred())

				r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/education/classes", bytes.NewBuffer(updatedClassJson))
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("classID", *newClass.Id)
				r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.PatchEducationClass(rr, r)

				Expect(rr.Code).To(Equal(http.StatusNoContent))
				identityBackend.AssertNumberOfCalls(GinkgoT(), "AddMembersToGroup", 1)
			})
		})
	})

	Describe("DeleteEducationClass", func() {
		Context("with an existing EducationClass", func() {
			BeforeEach(func() {
				identityEducationBackend.On("GetEducationClass", mock.Anything, mock.Anything, mock.Anything).Return(newClass, nil)
			})
		})

		It("deletes the EducationClass", func() {
			identityEducationBackend.On("DeleteEducationClass", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/education/classes", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationClass(rr, r)

			Expect(rr.Code).To(Equal(http.StatusNoContent))
			identityEducationBackend.AssertNumberOfCalls(GinkgoT(), "DeleteEducationClass", 1)
			// eventsPublisher.AssertNumberOfCalls(GinkgoT(), "Publish", 1)
		})
	})

	Describe("GetEducationClassMembers", func() {
		It("gets the list of members", func() {
			user := libregraph.NewEducationUser()
			user.SetId("user")
			identityEducationBackend.On("GetEducationClassMembers", mock.Anything, mock.Anything, mock.Anything).
				Return([]*libregraph.EducationUser{user}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/classes/{classID}/members", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.GetEducationClassMembers(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			var members []*libregraph.EducationUser
			err = json.Unmarshal(data, &members)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(members)).To(Equal(1))
			Expect(members[0].GetId()).To(Equal("user"))
		})
	})

	Describe("PostEducationClassMember", func() {
		It("fails on invalid body", func() {
			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/classes/{classID}/members", bytes.NewBufferString("{invalid"))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationClassMember(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("fails on missing member refs", func() {
			member := libregraph.NewMemberReference()
			data, err := json.Marshal(member)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/classes/{classID}/members", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationClassMember(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("fails on invalid member refs", func() {
			member := libregraph.NewMemberReference()
			member.SetOdataId("/invalidtype/user")
			data, err := json.Marshal(member)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/classes/{classID}/members", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationClassMember(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("adds a new member", func() {
			member := libregraph.NewMemberReference()
			member.SetOdataId("/users/user")
			data, err := json.Marshal(member)
			Expect(err).ToNot(HaveOccurred())
			identityBackend.On("AddMembersToGroup", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/classes/{classID}/members", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationClassMember(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNoContent))

			identityBackend.AssertNumberOfCalls(GinkgoT(), "AddMembersToGroup", 1)
		})
	})

	Describe("DeleteEducationClassMembers", func() {
		It("handles missing or empty member id", func() {
			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/classes/{classID}/members/{memberID}/$ref", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationClassMember(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles missing or empty member id", func() {
			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/classes/{classID}/members/{memberID}/$ref", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("memberID", "/users/user")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationClassMember(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("deletes members", func() {
			identityBackend.On("RemoveMemberFromGroup", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/classes/{classID}/members/{memberID}/$ref", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			rctx.URLParams.Add("memberID", "/users/user1")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationClassMember(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNoContent))

			identityBackend.AssertNumberOfCalls(GinkgoT(), "RemoveMemberFromGroup", 1)
		})
	})

	Describe("GetEducationClassTeachers", func() {
		It("gets the list of teachers", func() {
			user := libregraph.NewEducationUser()
			user.SetId("user")
			identityEducationBackend.On("GetEducationClassTeachers", mock.Anything, mock.Anything, mock.Anything).
				Return([]*libregraph.EducationUser{user}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/classes/{classID}/teachers", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.GetEducationClassTeachers(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			var teachers []*libregraph.EducationUser
			err = json.Unmarshal(data, &teachers)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(teachers)).To(Equal(1))
			Expect(teachers[0].GetId()).To(Equal("user"))
		})
	})

	Describe("PostEducationClassTeacher", func() {
		It("fails on invalid body", func() {
			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/classes/{classID}/teachers", bytes.NewBufferString("{invalid"))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationClassTeacher(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("fails on missing teacher refs", func() {
			member := libregraph.NewMemberReference()
			data, err := json.Marshal(member)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/classes/{classID}/teachers", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationClassTeacher(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("fails on invalid teacher refs", func() {
			member := libregraph.NewMemberReference()
			member.SetOdataId("/invalidtype/user")
			data, err := json.Marshal(member)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/classes/{classID}/teachers", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationClassTeacher(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("adds a new teacher", func() {
			member := libregraph.NewMemberReference()
			member.SetOdataId("/users/user")
			data, err := json.Marshal(member)
			Expect(err).ToNot(HaveOccurred())
			identityEducationBackend.On("AddTeacherToEducationClass", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/classes/{classID}/teachers", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationClassTeacher(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNoContent))

			identityEducationBackend.AssertNumberOfCalls(GinkgoT(), "AddTeacherToEducationClass", 1)
		})
	})

	Describe("DeleteEducationClassTeacher", func() {
		It("handles missing or empty teacher id", func() {
			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/classes/{classID}/teachers/{teacherID}/$ref", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationClassTeacher(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles missing or empty teacher id", func() {
			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/classes/{classID}/teachers/{teacherID}/$ref", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("teacherID", "/users/user")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationClassTeacher(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("deletes teacher", func() {
			identityEducationBackend.On("RemoveTeacherFromEducationClass", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/classes/{classID}/teachers/{teacherID}/$ref", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("classID", *newClass.Id)
			rctx.URLParams.Add("teacherID", "/users/user1")
			r = r.WithContext(context.WithValue(revactx.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationClassTeacher(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNoContent))

			identityEducationBackend.AssertNumberOfCalls(GinkgoT(), "RemoveTeacherFromEducationClass", 1)
		})
	})
})
