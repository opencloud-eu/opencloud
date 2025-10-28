package svc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	identitymocks "github.com/opencloud-eu/opencloud/services/graph/pkg/identity/mocks"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

type schoolList struct {
	Value []*libregraph.EducationSchool
}

var _ = Describe("Schools", func() {
	var (
		svc                      service.Service
		ctx                      context.Context
		cfg                      *config.Config
		gatewayClient            *cs3mocks.GatewayAPIClient
		gatewaySelector          pool.Selectable[gateway.GatewayAPIClient]
		identityEducationBackend *identitymocks.EducationBackend

		rr *httptest.ResponseRecorder

		newSchool   *libregraph.EducationSchool
		currentUser = &userv1beta1.User{
			Id: &userv1beta1.UserId{
				OpaqueId: "user",
			},
		}
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

		identityEducationBackend = &identitymocks.EducationBackend{}
		newSchool = libregraph.NewEducationSchool()
		newSchool.SetId("school1")

		rr = httptest.NewRecorder()
		ctx = context.Background()

		cfg = defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = "" // skip the startup checks, we don't use LDAP at all in this tests
		cfg.Identity.LDAP.EducationConfig.SchoolTerminationGraceDays = 30
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}

		var err error
		svc, err = service.NewService(
			service.Config(cfg),
			service.WithGatewaySelector(gatewaySelector),
			service.WithIdentityEducationBackend(identityEducationBackend),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("GetEducationSchools", func() {
		It("handles invalid ODATA parameters", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/schools?§foo=bar", nil)
			svc.GetEducationSchools(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles invalid sorting queries", func() {
			identityEducationBackend.On("GetEducationSchools", ctx, mock.Anything).Return([]*libregraph.EducationSchool{newSchool}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/schools?$orderby=invalid", nil)
			svc.GetEducationSchools(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			odataerr := libregraph.OdataError{}
			err = json.Unmarshal(data, &odataerr)
			Expect(err).ToNot(HaveOccurred())
			Expect(odataerr.Error.Code).To(Equal("invalidRequest"))
		})

		It("handles unknown backend errors", func() {
			identityEducationBackend.On("GetEducationSchools", ctx, mock.Anything).Return(nil, errors.New("failed"))

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/schools", nil)
			svc.GetEducationSchools(rr, r)
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			odataerr := libregraph.OdataError{}
			err = json.Unmarshal(data, &odataerr)
			Expect(err).ToNot(HaveOccurred())
			Expect(odataerr.Error.Code).To(Equal("generalException"))
		})

		It("handles backend errors", func() {
			identityEducationBackend.On("GetEducationSchools", ctx, mock.Anything).Return(nil, errorcode.New(errorcode.AccessDenied, "access denied"))

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/schools", nil)
			svc.GetEducationSchools(rr, r)

			Expect(rr.Code).To(Equal(http.StatusForbidden))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			odataerr := libregraph.OdataError{}
			err = json.Unmarshal(data, &odataerr)
			Expect(err).ToNot(HaveOccurred())
			Expect(odataerr.Error.Code).To(Equal("accessDenied"))
		})

		It("renders an empty list of schools", func() {
			identityEducationBackend.On("GetEducationSchools", ctx, mock.Anything).Return([]*libregraph.EducationSchool{}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/schools", nil)
			svc.GetEducationSchools(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := service.ListResponse{}
			err = json.Unmarshal(data, &res)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Value).To(Equal([]interface{}{}))
		})

		It("renders a list of schools", func() {
			identityEducationBackend.On("GetEducationSchools", ctx, mock.Anything).Return([]*libregraph.EducationSchool{newSchool}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/schools", nil)
			svc.GetEducationSchools(rr, r)

			Expect(rr.Code).To(Equal(http.StatusOK))
			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			res := schoolList{}
			err = json.Unmarshal(data, &res)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(res.Value)).To(Equal(1))
			Expect(res.Value[0].GetId()).To(Equal("school1"))
		})
	})

	Describe("GetEducationSchool", func() {
		It("handles missing or empty school id", func() {
			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/schools", nil)
			svc.GetEducationSchool(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))

			r = httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/schools", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", "")
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))
			svc.GetEducationSchool(rr, r)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		Context("with an existing school", func() {
			BeforeEach(func() {
				identityEducationBackend.On("GetEducationSchool", mock.Anything, mock.Anything, mock.Anything).Return(newSchool, nil)
			})

			It("gets the school", func() {
				r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/schools/"+*newSchool.Id, nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("schoolID", *newSchool.Id)
				r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, nil), chi.RouteCtxKey, rctx))

				svc.GetEducationSchool(rr, r)

				Expect(rr.Code).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("PostEducationSchool", func() {
		It("handles invalid body", func() {
			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/", bytes.NewBufferString("{invalid"))

			svc.PostEducationSchool(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles missing display name", func() {
			newSchool = libregraph.NewEducationSchool()
			newSchool.SetSchoolNumber("0034")
			newSchoolJson, err := json.Marshal(newSchool)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/", bytes.NewBuffer(newSchoolJson))

			svc.PostEducationSchool(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("disallows school create ids", func() {
			newSchool = libregraph.NewEducationSchool()
			newSchool.SetId("disallowed")
			newSchool.SetDisplayName("New School")
			newSchool.SetSchoolNumber("0034")
			newSchoolJson, err := json.Marshal(newSchool)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/", bytes.NewBuffer(newSchoolJson))

			svc.PostEducationSchool(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("handles backend errors", func() {
			newSchool = libregraph.NewEducationSchool()
			newSchool.SetDisplayName("New School")
			newSchool.SetSchoolNumber("0034")
			newSchoolJson, err := json.Marshal(newSchool)
			Expect(err).ToNot(HaveOccurred())

			identityEducationBackend.On("CreateEducationSchool", mock.Anything, mock.Anything).Return(nil, errorcode.New(errorcode.AccessDenied, "access denied"))

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/", bytes.NewBuffer(newSchoolJson))

			svc.PostEducationSchool(rr, r)

			Expect(rr.Code).To(Equal(http.StatusForbidden))
		})

		It("creates the school", func() {
			newSchool = libregraph.NewEducationSchool()
			newSchool.SetDisplayName("New School")
			newSchool.SetSchoolNumber("0034")
			newSchoolJson, err := json.Marshal(newSchool)
			Expect(err).ToNot(HaveOccurred())

			identityEducationBackend.On("CreateEducationSchool", mock.Anything, mock.Anything).Return(newSchool, nil)

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/", bytes.NewBuffer(newSchoolJson))

			svc.PostEducationSchool(rr, r)

			Expect(rr.Code).To(Equal(http.StatusCreated))
		})
	})

	Describe("updating a School", func() {
		schoolUpdate := libregraph.NewEducationSchool()
		schoolUpdate.SetDisplayName("New School Name")
		schoolUpdateJson, _ := json.Marshal(schoolUpdate)

		schoolUpdatePast := libregraph.NewEducationSchool()
		schoolUpdatePast.SetTerminationDate(time.Now().Add(-time.Hour * 1))
		schoolUpdatePastJson, _ := json.Marshal(schoolUpdatePast)

		schoolUpdateBeforeGrace := libregraph.NewEducationSchool()
		schoolUpdateBeforeGrace.SetTerminationDate(time.Now().Add(24 * 10 * time.Hour))
		schoolUpdateBeforeGraceJson, _ := json.Marshal(schoolUpdateBeforeGrace)

		schoolUpdatePastGrace := libregraph.NewEducationSchool()
		schoolUpdatePastGrace.SetTerminationDate(time.Now().Add(24 * 31 * time.Hour))
		schoolUpdatePastGraceJson, _ := json.Marshal(schoolUpdatePastGrace)

		BeforeEach(func() {
			identityEducationBackend.On("UpdateEducationSchool", mock.Anything, mock.Anything, mock.Anything).Return(schoolUpdate, nil)
		})
		DescribeTable("PatchEducationSchool",
			func(schoolId string, body io.Reader, statusCode int) {
				r := httptest.NewRequest(http.MethodPatch, "/graph/v1.0/education/schools/", body)
				rctx := chi.NewRouteContext()
				if schoolId != "" {
					rctx.URLParams.Add("schoolID", schoolId)
				}
				r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.PatchEducationSchool(rr, r)
				Expect(rr.Code).To(Equal(statusCode))
			},
			Entry("handles invalid body", "school-id", bytes.NewBufferString("{invalid"), http.StatusBadRequest),
			Entry("handles missing or empty school id", "", bytes.NewBufferString(""), http.StatusBadRequest),
			Entry("handles malformed school id", "school%id", bytes.NewBuffer(schoolUpdateJson), http.StatusBadRequest),
			Entry("updates the school", "school-id", bytes.NewBuffer(schoolUpdateJson), http.StatusOK),
			Entry("fails to set a termination date in the past", "school-id", bytes.NewBuffer(schoolUpdatePastJson), http.StatusBadRequest),
			Entry("fails to set a termination date before grace period", "school-id", bytes.NewBuffer(schoolUpdateBeforeGraceJson), http.StatusBadRequest),
			Entry("succeeds to set a termination date past the grace period", "school-id", bytes.NewBuffer(schoolUpdatePastGraceJson), http.StatusOK),
		)
	})

	Describe("DeleteEducationSchool", func() {
		schoolWithFutureTermination := libregraph.NewEducationSchool()
		schoolWithFutureTermination.SetId("schoolWithFutureTermination")
		schoolWithFutureTermination.SetTerminationDate(time.Now().Add(time.Hour * 24))

		schoolWithPastTermination := libregraph.NewEducationSchool()
		schoolWithPastTermination.SetId("schoolWithPastTermination")
		schoolWithPastTermination.SetTerminationDate(time.Now().Add(-time.Hour * 24))

		Context("with an existing school", func() {
			BeforeEach(func() {
				identityEducationBackend.On("GetEducationSchool", mock.Anything, "school1").Return(newSchool, nil)
				identityEducationBackend.On("GetEducationSchool", mock.Anything, "schoolWithFutureTermination", mock.Anything).Return(schoolWithFutureTermination, nil)
				identityEducationBackend.On("GetEducationSchool", mock.Anything, "schoolWithPastTermination", mock.Anything).Return(schoolWithPastTermination, nil)
			})

			DescribeTable("checks terminnation date",
				func(schoolId string, statusCode int) {
					identityEducationBackend.On("DeleteEducationSchool", mock.Anything, mock.Anything, mock.Anything).Return(nil)
					identityEducationBackend.On("GetEducationSchoolUsers", mock.Anything, mock.Anything, mock.Anything).Return([]*libregraph.EducationUser{}, nil)
					r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/schools", nil)
					rctx := chi.NewRouteContext()
					rctx.URLParams.Add("schoolID", schoolId)
					r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
					svc.DeleteEducationSchool(rr, r)

					Expect(rr.Code).To(Equal(statusCode))
					if rr.Code == http.StatusNoContent {
						identityEducationBackend.AssertNumberOfCalls(GinkgoT(), "DeleteEducationSchool", 1)
					} else {
						identityEducationBackend.AssertNumberOfCalls(GinkgoT(), "DeleteEducationSchool", 0)
					}
				},
				Entry("fails when school has no termination date", "school1", http.StatusMethodNotAllowed),
				Entry("fails when school has a termination date in the future", "schoolWithFutureTermination", http.StatusMethodNotAllowed),
				Entry("succeeds when school has a termination date in the past", "schoolWithPastTermination", http.StatusNoContent),
			)

			It("removes the users from the school", func() {
				user1 := libregraph.NewEducationUser()
				user1.SetId("user1")
				user2 := libregraph.NewEducationUser()
				user2.SetId("user2")
				identityEducationBackend.On("GetEducationSchoolUsers", mock.Anything, mock.Anything, mock.Anything).Return([]*libregraph.EducationUser{user1, user2}, nil)
				identityEducationBackend.On("DeleteEducationSchool", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				identityEducationBackend.On("RemoveUserFromEducationSchool", mock.Anything, mock.Anything, *user1.Id).Return(nil)
				identityEducationBackend.On("RemoveUserFromEducationSchool", mock.Anything, mock.Anything, *user2.Id).Return(nil)

				r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/schools", nil)
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("schoolID", "schoolWithPastTermination")
				r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
				svc.DeleteEducationSchool(rr, r)

				Expect(rr.Code).To(Equal(http.StatusNoContent))
				identityEducationBackend.AssertNumberOfCalls(GinkgoT(), "DeleteEducationSchool", 1)
				identityEducationBackend.AssertNumberOfCalls(GinkgoT(), "RemoveUserFromEducationSchool", 2)
				identityEducationBackend.AssertNumberOfCalls(GinkgoT(), "GetEducationSchoolUsers", 1)
			})
		})
	})

	Describe("GetEducationSchoolUsers", func() {
		It("gets the list of members", func() {
			user := libregraph.NewEducationUser()
			user.SetOnPremisesSamAccountName("user")
			user.SetDisplayName("display name")
			user.SetMail("mail")
			user.SetId("user")
			identityEducationBackend.On("GetEducationSchoolUsers", mock.Anything, mock.Anything, mock.Anything).Return([]*libregraph.EducationUser{user}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/schools/{schoolID}/users", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.GetEducationSchoolUsers(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			var members []*libregraph.User
			err = json.Unmarshal(data, &members)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(members)).To(Equal(1))
			Expect(members[0].GetId()).To(Equal("user"))
		})
	})

	Describe("PostEducationSchoolUsers", func() {
		It("fails on invalid body", func() {
			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/{schoolID}/members", bytes.NewBufferString("{invalid"))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationSchoolUser(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("fails on missing member refs", func() {
			member := libregraph.NewMemberReference()
			data, err := json.Marshal(member)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/{schoolID}/members", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationSchoolUser(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("fails on invalid member refs", func() {
			member := libregraph.NewMemberReference()
			member.SetOdataId("/invalidtype/user")
			data, err := json.Marshal(member)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/{schoolID}/members", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationSchoolUser(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("adds a new member", func() {
			member := libregraph.NewMemberReference()
			member.SetOdataId("/users/user")
			data, err := json.Marshal(member)
			Expect(err).ToNot(HaveOccurred())
			identityEducationBackend.On("AddUsersToEducationSchool", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/{schoolID}/members", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationSchoolUser(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNoContent))

			identityEducationBackend.AssertNumberOfCalls(GinkgoT(), "AddUsersToEducationSchool", 1)
		})
	})

	Describe("DeleteEducationSchoolUsers", func() {
		It("handles missing or empty member id", func() {
			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/schools/{schoolID}/members/{userID}/$ref", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationSchoolUser(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})
		It("handles missing or empty member id", func() {
			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/schools/{schoolID}/members/{userID}/$ref", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userID", "/users/user")
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationSchoolUser(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("deletes members", func() {
			identityEducationBackend.On("RemoveUserFromEducationSchool", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/schools/{schoolID}/members/{userID}/$ref", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			rctx.URLParams.Add("userID", "/users/user1")
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationSchoolUser(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNoContent))

			identityEducationBackend.AssertNumberOfCalls(GinkgoT(), "RemoveUserFromEducationSchool", 1)
		})
	})

	Describe("GetEducationSchoolClasses", func() {
		It("gets the list of classes", func() {
			class := libregraph.NewEducationClassWithDefaults()
			class.SetId("class")
			identityEducationBackend.On("GetEducationSchoolClasses", mock.Anything, *newSchool.Id).Return([]*libregraph.EducationClass{class}, nil)

			r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/education/schools/{schoolID}/classes", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.GetEducationSchoolClasses(rr, r)
			Expect(rr.Code).To(Equal(http.StatusOK))

			data, err := io.ReadAll(rr.Body)
			Expect(err).ToNot(HaveOccurred())

			var classes []*libregraph.EducationClass
			err = json.Unmarshal(data, &classes)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(classes)).To(Equal(1))
			Expect(classes[0].GetId()).To(Equal("class"))
		})
	})

	Describe("PostEducationSchoolClasses", func() {
		It("fails on invalid body", func() {
			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/{schoolID}/classes", bytes.NewBufferString("{invalid"))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationSchoolClass(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("fails on missing member refs", func() {
			member := libregraph.NewMemberReference()
			data, err := json.Marshal(member)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/{schoolID}/classes", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationSchoolClass(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("fails on invalid class refs", func() {
			class := libregraph.NewMemberReference()
			class.SetOdataId("/invalidtype/class")
			data, err := json.Marshal(class)
			Expect(err).ToNot(HaveOccurred())

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/{schoolID}/classesa", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationSchoolUser(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("adds a new class", func() {
			member := libregraph.NewMemberReference()
			member.SetOdataId("/classes/class")
			data, err := json.Marshal(member)
			Expect(err).ToNot(HaveOccurred())
			identityEducationBackend.On("AddClassesToEducationSchool", mock.Anything, *newSchool.Id, []string{"class"}).Return(nil)

			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/education/schools/{schoolID}/members", bytes.NewBuffer(data))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.PostEducationSchoolClass(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNoContent))

			identityEducationBackend.AssertNumberOfCalls(GinkgoT(), "AddClassesToEducationSchool", 1)
		})
	})

	Describe("DeleteEducationSchoolClass", func() {
		It("handles missing or empty member id", func() {
			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/schools/{schoolID}/classes/{classID}/$ref", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationSchoolClass(rr, r)
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("deletes members", func() {
			identityEducationBackend.On("RemoveClassFromEducationSchool", mock.Anything, *newSchool.Id, "/classes/class1").Return(nil)

			r := httptest.NewRequest(http.MethodDelete, "/graph/v1.0/education/schools/{schoolID}/classes/{classID}/$ref", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("schoolID", *newSchool.Id)
			rctx.URLParams.Add("classID", "/classes/class1")
			r = r.WithContext(context.WithValue(ctxpkg.ContextSetUser(ctx, currentUser), chi.RouteCtxKey, rctx))
			svc.DeleteEducationSchoolClass(rr, r)
			Expect(rr.Code).To(Equal(http.StatusNoContent))

			identityEducationBackend.AssertNumberOfCalls(GinkgoT(), "RemoveClassFromEducationSchool", 1)
		})
	})
})
