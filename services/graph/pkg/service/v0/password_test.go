package svc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-ldap/ldap/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity"
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

var _ = Describe("Users changing their own password", func() {
	var (
		svc             service.Service
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		ldapClient      *identitymocks.Client
		ldapConfig      config.LDAP
		identityBackend identity.Backend
		eventsPublisher mocks.Publisher
		ctx             context.Context
		cfg             *config.Config
		user            *userv1beta1.User
		err             error
	)

	JustBeforeEach(func() {
		ctx = context.Background()
		cfg = defaults.FullDefaultConfig()
		cfg.TokenManager.JWTSecret = "loremipsum"
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

		ldapClient = mockedLDAPClient()

		ldapConfig = config.LDAP{
			WriteEnabled:             true,
			UserDisplayNameAttribute: "displayName",
			UserNameAttribute:        "uid",
			UserEmailAttribute:       "mail",
			UserIDAttribute:          "openCloudUUID",
			UserSearchScope:          "sub",
			GroupNameAttribute:       "cn",
			GroupIDAttribute:         "openCloudUUID",
			GroupSearchScope:         "sub",
		}
		loggger := log.NewLogger()
		identityBackend, err = identity.NewLDAPBackend(ldapClient, ldapConfig, &loggger)
		Expect(err).To(BeNil())

		eventsPublisher = mocks.Publisher{}
		svc, _ = service.NewService(
			service.Config(cfg),
			service.WithGatewaySelector(gatewaySelector),
			service.WithIdentityBackend(identityBackend),
			service.EventsPublisher(&eventsPublisher),
		)
		user = &userv1beta1.User{
			Id: &userv1beta1.UserId{
				OpaqueId: "user",
			},
		}
		ctx = revactx.ContextSetUser(ctx, user)
		eventsPublisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	})

	It("fails if no user in context", func() {
		r := httptest.NewRequest(http.MethodGet, "/graph/v1.0/me/changePassword", nil)
		rr := httptest.NewRecorder()
		svc.ChangeOwnPassword(rr, r)
		Expect(rr.Code).To(Equal(http.StatusInternalServerError))
	})

	DescribeTable("changing the password",
		func(current string, newpw string, authresult string, expected int) {
			switch authresult {
			case "error":
				gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(nil, errors.New("fail"))
			case "deny":
				gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(&gateway.AuthenticateResponse{
					Status: status.NewPermissionDenied(ctx, errors.New("wrong password"), "wrong password"),
					Token:  "authtoken",
				}, nil)
			default:
				gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(&gateway.AuthenticateResponse{
					Status: status.NewOK(ctx),
					Token:  "authtoken",
				}, nil)
			}
			cpw := libregraph.NewPasswordChangeWithDefaults()
			cpw.SetCurrentPassword(current)
			cpw.SetNewPassword(newpw)
			body, _ := json.Marshal(cpw)
			b := bytes.NewBuffer(body)
			r := httptest.NewRequest(http.MethodPost, "/graph/v1.0/me/changePassword", b).WithContext(ctx)
			rr := httptest.NewRecorder()
			svc.ChangeOwnPassword(rr, r)
			Expect(rr.Code).To(Equal(expected))
		},
		Entry("fails when current password is empty", "", "newpassword", "", http.StatusBadRequest),
		Entry("fails when new password is empty", "currentpassword", "", "", http.StatusBadRequest),
		Entry("fails when current and new password are equal", "password", "password", "", http.StatusBadRequest),
		Entry("fails authentication with current password errors", "currentpassword", "newpassword", "error", http.StatusInternalServerError),
		Entry("fails when current password is wrong", "currentpassword", "newpassword", "deny", http.StatusBadRequest),
		Entry("succeeds when current password is correct", "currentpassword", "newpassword", "", http.StatusNoContent),
	)
})

func mockedLDAPClient() *identitymocks.Client {
	lm := &identitymocks.Client{}

	userEntry := ldap.NewEntry("uid=test", map[string][]string{
		"uid":         {"test"},
		"displayName": {"test"},
		"mail":        {"test@example.org"},
	})

	lm.On("Search", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(
			&ldap.SearchResult{Entries: []*ldap.Entry{userEntry}},
			nil)

	mr := ldap.NewModifyRequest("uid=test", nil)
	mr.Changes = []ldap.Change{
		{
			Operation: ldap.ReplaceAttribute,
			Modification: ldap.PartialAttribute{
				Type: "userPassword",
				Vals: []string{"newpassword"},
			},
		},
	}
	lm.On("Modify", mr).Return(nil)
	return lm
}
