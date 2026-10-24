package middleware

import (
	"net/http"
	"net/http/httptest"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend/mocks"
)

var _ = Describe("Authenticating requests", Label("BasicAuthenticator"), func() {
	var authenticator Authenticator
	ub := mocks.UserBackend{}
	ub.On("Authenticate", mock.Anything, "testuser", "testpassword").Return(
		&userv1beta1.User{
			Id: &userv1beta1.UserId{
				Idp:      "IdpId",
				OpaqueId: "OpaqueId",
			},
			Username: "testuser",
			Mail:     "testuser@example.com",
		},
		"",
		nil,
	)
	ub.On("Authenticate", mock.Anything, mock.Anything, mock.Anything).Return(nil, "", backend.ErrAccountNotFound)

	BeforeEach(func() {
		authenticator = BasicAuthenticator{
			Logger:       log.NewLogger(),
			UserProvider: &ub,
		}
	})

	When("the request contains correct data", func() {
		It("should successfully authenticate", func() {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/example/path", http.NoBody)
			req.SetBasicAuth("testuser", "testpassword")

			result := authenticator.Authenticate(req)

			Expect(result.State).To(Equal(AuthenticationSucceeded))
			Expect(result.Request).ToNot(BeNil())
		})
		It("adds claims to the request context", func() {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/example/path", http.NoBody)
			req.SetBasicAuth("testuser", "testpassword")

			result := authenticator.Authenticate(req)
			Expect(result.State).To(Equal(AuthenticationSucceeded))

			claims := oidc.FromContext(result.Request.Context())
			Expect(claims).ToNot(BeNil())
			Expect(claims[oidc.Iss]).To(Equal("IdpId"))
			Expect(claims[oidc.PreferredUsername]).To(Equal("testuser"))
			Expect(claims[oidc.Email]).To(Equal("testuser@example.com"))
			Expect(claims[oidc.OpenCloudUUID]).To(Equal("OpaqueId"))
		})
	})
})
