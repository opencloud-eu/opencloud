package middleware

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
	"go-micro.dev/v4/store"
	"golang.org/x/crypto/sha3"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	oidcmocks "github.com/opencloud-eu/opencloud/pkg/oidc/mocks"
)

func TestOIDCCacheTokenVerification(t *testing.T) {
	for _, validateCacheHit := range []bool{false, true} {
		for _, cached := range []bool{false, true} {
			t.Run(fmt.Sprintf("validate_cache_hit=%t/cached=%t", validateCacheHit, cached), func(t *testing.T) {
				client := &oidcmocks.OIDCClient{}
				expiresAt := time.Now().Add(time.Hour)
				claims := jwt.MapClaims{"sub": "alice", "exp": expiresAt.Unix()}
				if validateCacheHit || !cached {
					client.On("VerifyAccessToken", mock.Anything, "token").Return(oidc.RegClaimsWithSID{
						SessionID: "session",
						RegisteredClaims: jwt.RegisteredClaims{
							Subject: "alice", ExpiresAt: jwt.NewNumericDate(expiresAt),
						},
					}, claims, nil).Once()
				}
				cache := store.NewMemoryStore()
				if cached {
					hash := make([]byte, 64)
					sha3.ShakeSum256(hash, []byte("token"))
					data, err := msgpack.Marshal(claims)
					require.NoError(t, err)
					require.NoError(t, cache.Write(&store.Record{
						Key: base64.URLEncoding.EncodeToString(hash), Value: data,
					}))
				}
				authenticator := NewOIDCAuthenticator(
					Logger(log.NopLogger()), OIDCClient(client), UserInfoCache(cache),
					SkipUserInfo(true), ValidateAccessTokenOnCacheHit(validateCacheHit),
				)
				got, newSession, err := authenticator.getClaims("token", httptest.NewRequest(http.MethodGet, "/", http.NoBody))
				require.NoError(t, err)
				require.Equal(t, "alice", got["sub"])
				require.Equal(t, !cached, newSession)
				client.AssertExpectations(t)
				if !validateCacheHit && cached {
					client.AssertNotCalled(t, "VerifyAccessToken", mock.Anything, mock.Anything)
				}
				client.AssertNotCalled(t, "UserInfo", mock.Anything, mock.Anything)
			})
		}
	}
}
