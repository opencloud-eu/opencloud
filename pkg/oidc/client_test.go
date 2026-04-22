package oidc_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
)

type signingKey struct {
	priv any
	jwks *keyfunc.JWKS
}

func TestLogoutVerify(t *testing.T) {
	tests := []logoutVerificationTest{
		{
			name: "good token",
			logoutToken: jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
				"iss": "https://foo",
				"sub": "248289761001",
				"aud": "s6BhdRkqt3",
				"iat": 1471566154,
				"jti": "bWJq",
				"sid": "08a5019c-17e1-4977-8f42-65a12843ea02",
				"events": map[string]any{
					"http://schemas.openid.net/event/backchannel-logout": struct{}{},
				},
			}),
			signKey: newRSAKey(t),
		},
		{
			name:   "invalid issuer",
			issuer: "https://bar",
			logoutToken: jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
				"iss": "https://foo1",
				"sub": "248289761001",
				"events": map[string]any{
					"http://schemas.openid.net/event/backchannel-logout": struct{}{},
				},
			}),
			signKey: newRSAKey(t),
			wantErr: true,
		},
		{
			name: "invalid sig",
			logoutToken: jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
				"iss": "https://foo",
				"sub": "248289761001",
				"aud": "s6BhdRkqt3",
				"iat": 1471566154,
				"jti": "bWJq",
				"sid": "08a5019c-17e1-4977-8f42-65a12843ea02",
				"events": map[string]any{
					"http://schemas.openid.net/event/backchannel-logout": struct{}{},
				},
			}),
			signKey:         newRSAKey(t),
			verificationKey: newRSAKey(t),
			wantErr:         true,
		},
		{
			name: "no sid and no sub",
			logoutToken: jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
				"iss": "https://foo",
				"aud": "s6BhdRkqt3",
				"iat": 1471566154,
				"jti": "bWJq",
				"events": map[string]any{
					"http://schemas.openid.net/event/backchannel-logout": struct{}{},
				},
			}),
			signKey: newRSAKey(t),
			wantErr: true,
		},
		{
			name: "Prohibited nonce present",
			logoutToken: jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
				"iss":   "https://foo",
				"sub":   "248289761001",
				"aud":   "s6BhdRkqt3",
				"iat":   1471566154,
				"jti":   "bWJq",
				"sid":   "08a5019c-17e1-4977-8f42-65a12843ea02",
				"nonce": "123",
				"events": map[string]any{
					"http://schemas.openid.net/event/backchannel-logout": struct{}{},
				},
			}),
			signKey: newRSAKey(t),
			wantErr: true,
		},
		{
			name: "Wrong Event string",
			logoutToken: jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
				"iss": "https://foo",
				"sub": "248289761001",
				"aud": "s6BhdRkqt3",
				"iat": 1471566154,
				"jti": "bWJq",
				"sid": "08a5019c-17e1-4977-8f42-65a12843ea02",
				"events": map[string]any{
					"http://blah.blah.blash/event/backchannel-logout": struct{}{},
				},
			}),
			signKey: newRSAKey(t),
			wantErr: true,
		},
		{
			name: "No Event string",
			logoutToken: jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
				"iss": "https://foo",
				"sub": "248289761001",
				"aud": "s6BhdRkqt3",
				"iat": 1471566154,
				"jti": "bWJq",
				"sid": "08a5019c-17e1-4977-8f42-65a12843ea02",
			}),
			signKey: newRSAKey(t),
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

type logoutVerificationTest struct {
	// Name of the subtest.
	name string

	// If not provided defaults to "https://foo"
	issuer string

	// JWT payload (just the claims).
	logoutToken *jwt.Token

	// Key to sign the ID Token with.
	signKey *signingKey
	// If not provided defaults to signKey. Only useful when
	// testing invalid signatures.
	verificationKey *signingKey

	wantErr bool
}

func (v logoutVerificationTest) runGetToken(t *testing.T) (*oidc.LogoutToken, error) {
	//	token := v.signKey.sign(t, []byte(v.logoutToken))
	v.logoutToken.Header["kid"] = "1"
	token, err := v.logoutToken.SignedString(v.signKey.priv)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	issuer := "https://foo"
	var jwks *keyfunc.JWKS
	if v.verificationKey == nil {
		jwks = v.signKey.jwks
	} else {
		jwks = v.verificationKey.jwks
	}

	pm := oidc.ProviderMetadata{}
	verifier := oidc.NewOIDCClient(
		oidc.WithOidcIssuer(issuer),
		oidc.WithJWKS(jwks),
		oidc.WithProviderMetadata(&pm),
	)

	return verifier.VerifyLogoutToken(ctx, token)
}

func (l logoutVerificationTest) run(t *testing.T) {
	_, err := l.runGetToken(t)
	if err != nil && !l.wantErr {
		t.Errorf("%v", err)
	}
	if err == nil && l.wantErr {
		t.Errorf("expected error")
	}
}

func newRSAKey(t testing.TB) *signingKey {
	priv, err := rsa.GenerateKey(rand.Reader, 1028)
	if err != nil {
		t.Fatal(err)
	}
	givenKey := keyfunc.NewGivenRSA(
		&priv.PublicKey,
		keyfunc.GivenKeyOptions{Algorithm: jwt.SigningMethodRS256.Alg()},
	)
	jwks := keyfunc.NewGiven(
		map[string]keyfunc.GivenKey{
			"1": givenKey,
		},
	)

	return &signingKey{priv, jwks}
}
