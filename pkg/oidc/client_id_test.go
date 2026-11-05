package oidc_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestAccessTokenClientIDs(t *testing.T) {
	key := newRSAKey(t)
	tests := []struct {
		name      string
		clientIDs []string
		claims    jwt.MapClaims
		wantErr   error
	}{
		{name: "disabled accepts any client", claims: jwt.MapClaims{"client_id": "other"}},
		{name: "disabled accepts no client", claims: jwt.MapClaims{}},
		{name: "explicitly empty configuration", clientIDs: []string{}, claims: jwt.MapClaims{"azp": "other"}},
		{name: "client_id", clientIDs: []string{"web"}, claims: jwt.MapClaims{"client_id": "web"}},
		{name: "azp", clientIDs: []string{"web"}, claims: jwt.MapClaims{"azp": "web"}},
		{name: "any allowed client", clientIDs: []string{"web", "desktop"}, claims: jwt.MapClaims{"client_id": "desktop"}},
		{name: "client_id wins over azp", clientIDs: []string{"web"}, claims: jwt.MapClaims{"client_id": "web", "azp": "other"}},
		{name: "client_id not allowed despite allowed azp", clientIDs: []string{"web"}, claims: jwt.MapClaims{"client_id": "other", "azp": "web"}, wantErr: oidc.ErrTokenInvalidClientID},
		{name: "foreign client", clientIDs: []string{"web"}, claims: jwt.MapClaims{"client_id": "other"}, wantErr: oidc.ErrTokenInvalidClientID},
		{name: "foreign azp", clientIDs: []string{"web"}, claims: jwt.MapClaims{"azp": "other"}, wantErr: oidc.ErrTokenInvalidClientID},
		{name: "allowed audience does not help", clientIDs: []string{"web"}, claims: jwt.MapClaims{"aud": "web", "azp": "other"}, wantErr: oidc.ErrTokenInvalidClientID},
		{name: "case sensitive", clientIDs: []string{"web"}, claims: jwt.MapClaims{"client_id": "Web"}, wantErr: oidc.ErrTokenInvalidClientID},
		{name: "exact match", clientIDs: []string{"web"}, claims: jwt.MapClaims{"client_id": "web-legacy"}, wantErr: oidc.ErrTokenInvalidClientID},
		{name: "no wildcard matching", clientIDs: []string{"*"}, claims: jwt.MapClaims{"client_id": "web"}, wantErr: oidc.ErrTokenInvalidClientID},
		{name: "missing", clientIDs: []string{"web"}, claims: jwt.MapClaims{}, wantErr: jwt.ErrTokenRequiredClaimMissing},
		{name: "null", clientIDs: []string{"web"}, claims: jwt.MapClaims{"client_id": nil}, wantErr: jwt.ErrTokenRequiredClaimMissing},
		{name: "empty", clientIDs: []string{"web"}, claims: jwt.MapClaims{"client_id": ""}, wantErr: jwt.ErrTokenRequiredClaimMissing},
		{name: "number", clientIDs: []string{"web"}, claims: jwt.MapClaims{"client_id": 123}, wantErr: jwt.ErrTokenMalformed},
		{name: "array", clientIDs: []string{"web"}, claims: jwt.MapClaims{"azp": []string{"web"}}, wantErr: jwt.ErrTokenMalformed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := jwt.MapClaims{
				"iss": "https://issuer.example",
				"sub": "alice",
				"exp": time.Now().Add(time.Hour).Unix(),
			}
			for k, v := range tt.claims {
				claims[k] = v
			}
			client := newClientIDTestClient(t, key, tt.clientIDs)
			registered, all, err := client.VerifyAccessToken(context.Background(), signAccessToken(t, key, claims))
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Empty(t, all, "claims of a rejected token must not be returned")
				require.Empty(t, registered.Subject, "claims of a rejected token must not be returned")
				return
			}
			require.NoError(t, err)
			require.Equal(t, "alice", registered.Subject)
			require.Equal(t, "alice", all["sub"])
		})
	}
}

func TestAccessTokenClientIDsWithAudiences(t *testing.T) {
	key := newRSAKey(t)
	client, err := oidc.NewOIDCClient(
		oidc.WithLogger(log.NopLogger()),
		oidc.WithOidcIssuer("https://issuer.example"),
		oidc.WithAccessTokenVerifyMethod(config.AccessTokenVerificationJWT),
		oidc.WithAccessTokenAudiences([]string{"opencloud"}),
		oidc.WithAccessTokenClientIDs([]string{"web"}),
		oidc.WithJWKS(key.jwks),
		oidc.WithProviderMetadata(&oidc.ProviderMetadata{}),
	)
	require.NoError(t, err)
	verify := func(claims jwt.MapClaims) error {
		claims["iss"] = "https://issuer.example"
		_, _, err := client.VerifyAccessToken(context.Background(), signAccessToken(t, key, claims))
		return err
	}
	require.NoError(t, verify(jwt.MapClaims{"aud": "opencloud", "azp": "web"}))
	require.ErrorIs(t, verify(jwt.MapClaims{"aud": "opencloud", "azp": "other"}), oidc.ErrTokenInvalidClientID)
	require.ErrorIs(t, verify(jwt.MapClaims{"aud": "other", "azp": "web"}), jwt.ErrTokenInvalidAudience)
}

func TestAccessTokenClientIDConfiguration(t *testing.T) {
	for _, method := range []string{config.AccessTokenVerificationNone, ""} {
		t.Run("incompatible method "+method, func(t *testing.T) {
			// No HTTP client is supplied: invalid configuration must fail before discovery.
			client, err := oidc.NewOIDCClient(
				oidc.WithAccessTokenVerifyMethod(method),
				oidc.WithAccessTokenClientIDs([]string{"web"}),
			)
			require.ErrorContains(t, err, "requires the jwt verification method")
			require.Nil(t, client)
		})
	}
	for _, clientIDs := range [][]string{{""}, {" \t"}, {"web", ""}} {
		client, err := oidc.NewOIDCClient(
			oidc.WithAccessTokenVerifyMethod(config.AccessTokenVerificationJWT),
			oidc.WithAccessTokenClientIDs(clientIDs),
		)
		require.ErrorContains(t, err, "empty or whitespace-only")
		require.Nil(t, client)
	}
	t.Run("caller cannot mutate the policy", func(t *testing.T) {
		key := newRSAKey(t)
		clientIDs := []string{"web"}
		client := newClientIDTestClient(t, key, clientIDs)
		clientIDs[0] = "other"
		_, _, err := client.VerifyAccessToken(context.Background(), signAccessToken(t, key,
			jwt.MapClaims{"iss": "https://issuer.example", "client_id": "other"}))
		require.ErrorIs(t, err, oidc.ErrTokenInvalidClientID)
	})
}

func TestAccessTokenClientIDsDoNotApplyToLogoutTokens(t *testing.T) {
	key := newRSAKey(t)
	client := newClientIDTestClient(t, key, []string{"web"})
	token := signAccessToken(t, key, jwt.MapClaims{
		"iss": "https://issuer.example",
		"sub": "alice",
		"aud": "other",
		"events": map[string]any{
			"http://schemas.openid.net/event/backchannel-logout": map[string]any{},
		},
	})
	_, err := client.VerifyLogoutToken(context.Background(), token)
	require.NoError(t, err)
}

func newClientIDTestClient(t *testing.T, key *signingKey, clientIDs []string) oidc.OIDCClient {
	t.Helper()
	client, err := oidc.NewOIDCClient(
		oidc.WithLogger(log.NopLogger()),
		oidc.WithOidcIssuer("https://issuer.example"),
		oidc.WithAccessTokenVerifyMethod(config.AccessTokenVerificationJWT),
		oidc.WithAccessTokenClientIDs(clientIDs),
		oidc.WithJWKS(key.jwks),
		oidc.WithProviderMetadata(&oidc.ProviderMetadata{}),
	)
	require.NoError(t, err)
	return client
}
