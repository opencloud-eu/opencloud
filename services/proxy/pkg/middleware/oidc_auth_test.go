package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	oidcmocks "github.com/opencloud-eu/opencloud/pkg/oidc/mocks"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"go-micro.dev/v4/store"
)

var _ = Describe("Authenticating requests", Label("OIDCAuthenticator"), func() {
	var authenticator Authenticator

	oc := oidcmocks.OIDCClient{}
	oc.On("VerifyAccessToken", mock.Anything, mock.Anything).Return(
		oidc.RegClaimsWithSID{
			SessionID: "a-session-id",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Unix(1147483647, 0)),
			},
		}, jwt.MapClaims{
			"sid": "a-session-id",
			"exp": 1147483647,
		},
		nil,
	)
	/*
		// to test with skipUserInfo:  true, we need to also use an interface so we can mock the UserInfo.Claim call
		oc.On("UserInfo", mock.Anything, mock.Anything).Return(
			&oidc.UserInfo{
				Subject:       "my-sub",
				EmailVerified: true,
				Email:         "test@example.org",
			},
			nil,
		)
	*/

	BeforeEach(func() {
		authenticator = &OIDCAuthenticator{
			OIDCIss:       "http://idp.example.com",
			Logger:        log.NewLogger(),
			oidcClient:    &oc,
			userInfoCache: store.NewMemoryStore(),
			skipUserInfo:  true,
		}
	})

	It("should apply the default access token ttl option", func() {
		ttl := 5 * time.Minute

		authenticator := NewOIDCAuthenticator(DefaultAccessTokenTTL(ttl))

		Expect(authenticator.DefaultTokenCacheTTL).To(Equal(ttl))
	})

	When("the request contains correct data", func() {
		It("should successfully authenticate", func() {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/example/path", http.NoBody)
			req.Header.Set(_headerAuthorization, "Bearer jwt.token.sig")

			req2, valid := authenticator.Authenticate(req)

			Expect(valid).To(Equal(true))
			Expect(req2).ToNot(BeNil())
		})
		It("should successfully authenticate", func() {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/dav/public-files", http.NoBody)
			req.Header.Set(_headerAuthorization, "Bearer jwt.token.sig")

			req2, valid := authenticator.Authenticate(req)

			Expect(valid).To(Equal(true))
			Expect(req2).ToNot(BeNil())
		})
		It("should skip authenticate if the header ShareToken is set", func() {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/dav/public-files/", http.NoBody)
			req.Header.Set(_headerAuthorization, "Bearer jwt.token.sig")
			req.Header.Set(headerShareToken, "sharetoken")

			req2, valid := authenticator.Authenticate(req)

			// TODO Should the authentication of public path requests is handled by another authenticator?
			//Expect(valid).To(Equal(false))
			//Expect(req2).To(BeNil())
			Expect(valid).To(Equal(true))
			Expect(req2).ToNot(BeNil())
		})
		It("should skip authenticate if the 'public-token' is set", func() {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/dav/public-files/?public-token=sharetoken", http.NoBody)
			req.Header.Set(_headerAuthorization, "Bearer jwt.token.sig")

			req2, valid := authenticator.Authenticate(req)

			// TODO Should the authentication of public path requests is handled by another authenticator?
			//Expect(valid).To(Equal(false))
			//Expect(req2).To(BeNil())
			Expect(valid).To(Equal(true))
			Expect(req2).ToNot(BeNil())
		})
	})

	When("the request contains an expired token", func() {
		var buf bytes.Buffer
		expiredAuthenticator := &OIDCAuthenticator{
			OIDCIss:       "http://idp.example.com",
			Logger:        log.Logger{zerolog.New(&buf).Level(zerolog.DebugLevel)},
			oidcClient: &oidcmocks.OIDCClient{},
			userInfoCache: store.NewMemoryStore(),
			skipUserInfo:  true,
		}

		BeforeEach(func() {
			expiredAuthenticator.oidcClient.(*oidcmocks.OIDCClient).On("VerifyAccessToken", mock.Anything, mock.Anything).Return(
				oidc.RegClaimsWithSID{}, jwt.MapClaims{}, jwt.ErrTokenExpired,
			)
		})

		It("should reject the request", func() {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/example/path", http.NoBody)
			req.Header.Set(_headerAuthorization, "Bearer jwt.token.sig")

			req2, valid := expiredAuthenticator.Authenticate(req)

			Expect(valid).To(Equal(false))
			Expect(req2).To(BeNil())
		})

		It("should log the authentication failure at debug level", func() {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/example/path", http.NoBody)
			req.Header.Set(_headerAuthorization, "Bearer jwt.token.sig")

			expiredAuthenticator.Authenticate(req)

			Expect(buf.String()).To(ContainSubstring("failed to authenticate the request"))
			Expect(buf.String()).To(ContainSubstring(`"level":"debug"`))
			Expect(buf.String()).ToNot(ContainSubstring(`"level":"error"`))
		})
	})

	When("the request contains a token that cannot be verified", func() {
		var buf bytes.Buffer
		failingAuthenticator := &OIDCAuthenticator{
			OIDCIss:       "http://idp.example.com",
			Logger:        log.Logger{zerolog.New(&buf).Level(zerolog.DebugLevel)},
			oidcClient: &oidcmocks.OIDCClient{},
			userInfoCache: store.NewMemoryStore(),
			skipUserInfo:  true,
		}

		BeforeEach(func() {
			failingAuthenticator.oidcClient.(*oidcmocks.OIDCClient).On("VerifyAccessToken", mock.Anything, mock.Anything).Return(
				oidc.RegClaimsWithSID{}, jwt.MapClaims{}, jwt.ErrTokenMalformed,
			)
		})

		It("should log the authentication failure at error level", func() {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/example/path", http.NoBody)
			req.Header.Set(_headerAuthorization, "Bearer jwt.token.sig")

			failingAuthenticator.Authenticate(req)

			Expect(buf.String()).To(ContainSubstring("failed to authenticate the request"))
			Expect(buf.String()).To(ContainSubstring(`"level":"error"`))
		})
	})
})
