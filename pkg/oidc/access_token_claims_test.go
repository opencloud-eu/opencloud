package oidc_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestAccessTokenClaimExtraction(t *testing.T) {
	client, key := newClaimExtractionClient(t, "https://issuer.example")
	t.Run("preserves arbitrary claims and numeric types", func(t *testing.T) {
		claims := jwt.MapClaims{
			"iss": "https://issuer.example", "sub": "alice", "sid": "session",
			"exp": 4102444800.75, "groups": []any{"users", "engineering"},
			"profile": map[string]any{"enabled": true, "score": 1.25}, "custom": nil,
		}
		registered, all, err := client.VerifyAccessToken(context.Background(), signClaimExtractionToken(t, key, claims))
		require.NoError(t, err)
		require.Equal(t, "alice", registered.Subject)
		require.Equal(t, "session", registered.SessionID)
		require.EqualValues(t, 4102444800, registered.ExpiresAt.Unix())
		require.Equal(t, claims, all)
	})
	t.Run("preserves malformed map claim errors", func(t *testing.T) {
		// Typed claims ignore this custom field; decoding MapClaims must still
		// reject its overflowing number and preserve the JWT and JSON errors.
		_, _, err := client.VerifyAccessToken(context.Background(), signClaimExtractionToken(t, key, jwt.MapClaims{
			"iss": "https://issuer.example", "custom": json.Number("1e1000"),
		}))
		require.ErrorIs(t, err, jwt.ErrTokenMalformed)
		var jsonError *json.UnmarshalTypeError
		require.ErrorAs(t, err, &jsonError)
		require.Equal(t, "number 1e1000", jsonError.Value)
	})
	t.Run("retains empty map for null payload", func(t *testing.T) {
		// Without an issuer requirement, a signed null payload previously
		// returned an initialized, non-nil empty map.
		client, key := newClaimExtractionClient(t, "")
		registered, all, err := client.VerifyAccessToken(context.Background(), signClaimExtractionToken(t, key, nil))
		require.NoError(t, err)
		require.Equal(t, oidc.RegClaimsWithSID{}, registered)
		require.NotNil(t, all)
		require.Empty(t, all)
	})
	t.Run("rejects invalid tokens before extracting map claims", func(t *testing.T) {
		_, otherKey := newClaimExtractionClient(t, "https://issuer.example")
		for _, tt := range []struct {
			name    string
			claims  jwt.MapClaims
			key     *rsa.PrivateKey
			wantErr error
		}{
			{name: "invalid signature", claims: jwt.MapClaims{"iss": "https://issuer.example"}, key: otherKey, wantErr: jwt.ErrTokenSignatureInvalid},
			{name: "invalid issuer", claims: jwt.MapClaims{"iss": "https://other.example"}, key: key, wantErr: jwt.ErrTokenInvalidIssuer},
			{name: "expired", claims: jwt.MapClaims{"iss": "https://issuer.example", "exp": time.Now().Add(-time.Hour).Unix()}, key: key, wantErr: jwt.ErrTokenExpired},
			{name: "not yet valid", claims: jwt.MapClaims{"iss": "https://issuer.example", "nbf": time.Now().Add(time.Hour).Unix()}, key: key, wantErr: jwt.ErrTokenNotValidYet},
		} {
			t.Run(tt.name, func(t *testing.T) {
				_, all, err := client.VerifyAccessToken(context.Background(), signClaimExtractionToken(t, tt.key, tt.claims))
				require.ErrorIs(t, err, tt.wantErr)
				require.Empty(t, all, "unverified map claims must not be returned")
			})
		}
	})
}

func BenchmarkVerifyAccessTokenClaims(b *testing.B) {
	client, key := newClaimExtractionClient(b, "https://issuer.example")
	token := signClaimExtractionToken(b, key, jwt.MapClaims{
		"iss": "https://issuer.example", "sub": "alice", "sid": "session",
		"exp": time.Now().Add(time.Hour).Unix(), "preferred_username": "alice",
		"email": "alice@example.com", "groups": []any{"users", "engineering"},
		"profile": map[string]any{"enabled": true, "score": 1.25},
	})
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, _, err := client.VerifyAccessToken(ctx, token); err != nil {
			b.Fatal(err)
		}
	}
}

func newClaimExtractionClient(t testing.TB, issuer string) (oidc.OIDCClient, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	jwks := keyfunc.NewGiven(map[string]keyfunc.GivenKey{
		"1": keyfunc.NewGivenRSA(&key.PublicKey, keyfunc.GivenKeyOptions{Algorithm: jwt.SigningMethodRS256.Alg()}),
	})
	return oidc.NewOIDCClient(
		oidc.WithLogger(log.NopLogger()),
		oidc.WithOidcIssuer(issuer),
		oidc.WithAccessTokenVerifyMethod(config.AccessTokenVerificationJWT),
		oidc.WithJWKS(jwks),
		oidc.WithProviderMetadata(&oidc.ProviderMetadata{}),
	), key
}

func signClaimExtractionToken(t testing.TB, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "1"
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}
