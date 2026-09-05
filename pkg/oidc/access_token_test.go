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

func TestAccessTokenAudiences(t *testing.T) {
	key := newRSAKey(t)
	tests := []struct {
		name      string
		audiences []string
		aud       any
		missing   bool
		wantErr   error
	}{
		{name: "disabled missing", missing: true},
		{name: "disabled foreign", aud: "immich"},
		{name: "disabled null", aud: nil},
		{name: "disabled empty array", aud: []string{}},
		{name: "explicitly empty configuration", audiences: []string{}, aud: "immich"},
		{name: "single audience", audiences: []string{"opencloud"}, aud: "opencloud"},
		{name: "array first match", audiences: []string{"opencloud"}, aud: []string{"opencloud", "immich"}},
		{name: "array last match", audiences: []string{"opencloud"}, aud: []string{"immich", "opencloud"}},
		{name: "any allowed audience", audiences: []string{"opencloud", "opencloud-api"}, aud: "opencloud-api"},
		{name: "any allowed audience in array", audiences: []string{"opencloud", "opencloud-api"}, aud: []string{"immich", "opencloud-api"}},
		{name: "duplicate audiences", audiences: []string{"opencloud", "opencloud"}, aud: []string{"opencloud", "opencloud"}},
		{name: "URI audience", audiences: []string{"https://cloud.example/api"}, aud: "https://cloud.example/api"},
		{name: "foreign", audiences: []string{"opencloud"}, aud: "immich", wantErr: jwt.ErrTokenInvalidAudience},
		{name: "foreign array", audiences: []string{"opencloud"}, aud: []string{"immich", "account"}, wantErr: jwt.ErrTokenInvalidAudience},
		{name: "case sensitive", audiences: []string{"opencloud"}, aud: "OpenCloud", wantErr: jwt.ErrTokenInvalidAudience},
		{name: "exact match", audiences: []string{"opencloud"}, aud: "opencloud-api", wantErr: jwt.ErrTokenInvalidAudience},
		{name: "no token normalization", audiences: []string{"opencloud"}, aud: " opencloud ", wantErr: jwt.ErrTokenInvalidAudience},
		{name: "no wildcard matching", audiences: []string{"*"}, aud: "opencloud", wantErr: jwt.ErrTokenInvalidAudience},
		{name: "missing", audiences: []string{"opencloud"}, missing: true, wantErr: jwt.ErrTokenRequiredClaimMissing},
		{name: "null", audiences: []string{"opencloud"}, aud: nil, wantErr: jwt.ErrTokenRequiredClaimMissing},
		{name: "empty string", audiences: []string{"opencloud"}, aud: "", wantErr: jwt.ErrTokenRequiredClaimMissing},
		{name: "empty array", audiences: []string{"opencloud"}, aud: []string{}, wantErr: jwt.ErrTokenRequiredClaimMissing},
		{name: "array empty string", audiences: []string{"opencloud"}, aud: []string{""}, wantErr: jwt.ErrTokenRequiredClaimMissing},
		{name: "number", audiences: []string{"opencloud"}, aud: 123, wantErr: jwt.ErrTokenMalformed},
		{name: "object", audiences: []string{"opencloud"}, aud: map[string]string{"aud": "opencloud"}, wantErr: jwt.ErrTokenMalformed},
		{name: "mixed array", audiences: []string{"opencloud"}, aud: []any{"opencloud", 123}, wantErr: jwt.ErrTokenMalformed},
		{name: "null array entry", audiences: []string{"opencloud"}, aud: []any{"opencloud", nil}, wantErr: jwt.ErrTokenMalformed},
		{name: "disabled still rejects invalid type", aud: 123, wantErr: jwt.ErrTokenMalformed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := jwt.MapClaims{
				"iss": "https://issuer.example",
				"sub": "alice",
				"sid": "session",
				"exp": time.Now().Add(time.Hour).Unix(),
			}
			if !tt.missing {
				claims["aud"] = tt.aud
			}
			client := newAccessTokenTestClient(key, tt.audiences, &oidc.ProviderMetadata{})
			registered, all, err := client.VerifyAccessToken(context.Background(), signAccessToken(t, key, claims))
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Empty(t, all, "unverified claims must not be returned")
				return
			}
			require.NoError(t, err)
			require.Equal(t, "alice", registered.Subject)
			require.Equal(t, "session", registered.SessionID)
			require.Equal(t, "alice", all["sub"])
		})
	}
}

func TestAccessTokenValidationWithAudiences(t *testing.T) {
	key, otherKey := newRSAKey(t), newRSAKey(t)
	tests := []struct {
		name       string
		issuer     string
		provider   *oidc.ProviderMetadata
		signingKey *signingKey
		exp        time.Time
		nbf        time.Time
		wantErr    error
	}{
		{name: "invalid signature", signingKey: otherKey, wantErr: jwt.ErrTokenSignatureInvalid},
		{name: "invalid issuer", issuer: "https://other.example", wantErr: jwt.ErrTokenInvalidIssuer},
		{name: "expired", exp: time.Now().Add(-time.Hour), wantErr: jwt.ErrTokenExpired},
		{name: "not yet valid", nbf: time.Now().Add(time.Hour), wantErr: jwt.ErrTokenNotValidYet},
		{name: "AD FS access token issuer", issuer: "https://adfs.example", provider: &oidc.ProviderMetadata{AccessTokenIssuer: "https://adfs.example"}},
		{name: "AD FS rejects discovery issuer", provider: &oidc.ProviderMetadata{AccessTokenIssuer: "https://adfs.example"}, wantErr: jwt.ErrTokenInvalidIssuer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.issuer == "" {
				tt.issuer = "https://issuer.example"
			}
			if tt.provider == nil {
				tt.provider = &oidc.ProviderMetadata{}
			}
			if tt.signingKey == nil {
				tt.signingKey = key
			}
			if tt.exp.IsZero() {
				tt.exp = time.Now().Add(time.Hour)
			}
			claims := jwt.MapClaims{"iss": tt.issuer, "sub": "alice", "aud": "opencloud", "exp": tt.exp.Unix()}
			if !tt.nbf.IsZero() {
				claims["nbf"] = tt.nbf.Unix()
			}
			client := newAccessTokenTestClient(key, []string{"opencloud"}, tt.provider)
			_, _, err := client.VerifyAccessToken(context.Background(), signAccessToken(t, tt.signingKey, claims))
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestAccessTokenAudienceConfiguration(t *testing.T) {
	for _, method := range []string{config.AccessTokenVerificationNone, ""} {
		t.Run("incompatible method "+method, func(t *testing.T) {
			// No HTTP client is supplied: invalid configuration must fail before discovery.
			client := oidc.NewOIDCClient(
				oidc.WithAccessTokenVerifyMethod(method),
				oidc.WithAccessTokenAudiences([]string{"opencloud"}),
			)
			_, _, err := client.VerifyAccessToken(context.Background(), "opaque-token")
			require.ErrorContains(t, err, "requires the jwt verification method")
		})
	}
	for _, audiences := range [][]string{{""}, {" \t"}, {"opencloud", ""}} {
		client := oidc.NewOIDCClient(
			oidc.WithAccessTokenVerifyMethod(config.AccessTokenVerificationJWT),
			oidc.WithAccessTokenAudiences(audiences),
		)
		_, _, err := client.VerifyAccessToken(context.Background(), "token")
		require.ErrorContains(t, err, "empty or whitespace-only")
	}
	t.Run("none remains compatible when disabled", func(t *testing.T) {
		client := oidc.NewOIDCClient(
			oidc.WithLogger(log.NopLogger()),
			oidc.WithAccessTokenVerifyMethod(config.AccessTokenVerificationNone),
			oidc.WithProviderMetadata(&oidc.ProviderMetadata{}),
		)
		_, _, err := client.VerifyAccessToken(context.Background(), "opaque-token")
		require.NoError(t, err)
	})
	t.Run("caller cannot mutate the policy", func(t *testing.T) {
		key := newRSAKey(t)
		audiences := []string{"opencloud"}
		client := newAccessTokenTestClient(key, audiences, &oidc.ProviderMetadata{})
		audiences[0] = "immich"
		_, _, err := client.VerifyAccessToken(context.Background(), signAccessToken(t, key,
			jwt.MapClaims{"iss": "https://issuer.example", "aud": "immich"}))
		require.ErrorIs(t, err, jwt.ErrTokenInvalidAudience)
	})
}

func TestAccessTokenAudiencesDoNotApplyToLogoutTokens(t *testing.T) {
	key := newRSAKey(t)
	client := newAccessTokenTestClient(key, []string{"opencloud"}, &oidc.ProviderMetadata{})
	token := signAccessToken(t, key, jwt.MapClaims{
		"iss": "https://issuer.example",
		"sub": "alice",
		"aud": "web-client",
		"events": map[string]any{
			"http://schemas.openid.net/event/backchannel-logout": map[string]any{},
		},
	})
	_, err := client.VerifyLogoutToken(context.Background(), token)
	require.NoError(t, err)
}

func newAccessTokenTestClient(key *signingKey, audiences []string, provider *oidc.ProviderMetadata) oidc.OIDCClient {
	return oidc.NewOIDCClient(
		oidc.WithLogger(log.NopLogger()),
		oidc.WithOidcIssuer("https://issuer.example"),
		oidc.WithAccessTokenVerifyMethod(config.AccessTokenVerificationJWT),
		oidc.WithAccessTokenAudiences(audiences),
		oidc.WithJWKS(key.jwks),
		oidc.WithProviderMetadata(provider),
	)
}

func signAccessToken(t *testing.T, key *signingKey, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "1"
	signed, err := token.SignedString(key.priv)
	require.NoError(t, err)
	return signed
}
